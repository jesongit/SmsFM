// Package scan: 局域网设备发现（开发计划 §3.1）。
// 流程：并发快扫（128 并发、350ms TCP 超时）→ 端口开放即并行调 /config/query 识别
// → 后台复查快扫超时的地址（64 并发、1000ms）。不把 Ping 成功作为前提。
package scan

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"smsfm/server/internal/protocol"
)

const (
	FastConcurrency    = 128
	FastTimeout        = 350 * time.Millisecond
	RecheckConcurrency = 64
	RecheckTimeout     = 1000 * time.Millisecond
	IdentifyTimeout    = 3 * time.Second
	MaxAddresses       = 65536
)

// FoundDevice 一台被发现的候选/已识别设备
type FoundDevice struct {
	Host   string               `json:"host"`
	Port   int                  `json:"port"`
	State  string               `json:"state"` // identified | need-credentials | unverified
	Error  string               `json:"error,omitempty"`
	Config *protocol.ConfigData `json:"config,omitempty"`
}

// Progress 扫描进度（前端轮询）
type Progress struct {
	Phase     string        `json:"phase"` // fast | recheck | done | cancelled | error
	CIDR      string        `json:"cidr"`
	Port      int           `json:"port"`
	Scanned   int           `json:"scanned"`
	Total     int           `json:"total"`
	Found     []FoundDevice `json:"found"`
	StartedAt time.Time     `json:"startedAt"`
	ElapsedMs int64         `json:"elapsedMs"`
	Error     string        `json:"error,omitempty"`
}

// Scanner 单次扫描任务；Cancel 后保留已发现结果
type Scanner struct {
	cidr     string
	port     int
	credSink func(host string, port int) *protocol.Conn

	ctx       context.Context
	cancel    context.CancelFunc
	mu        sync.Mutex
	progress  Progress
	foundMap  map[string]FoundDevice // host → 去重
}

// New 建立一次扫描；credSink 可为 nil（无凭据探测），非 nil 时已保存连接的凭据优先使用
func New(cidr string, port int, credSink func(host string, port int) *protocol.Conn) *Scanner {
	ctx, cancel := context.WithCancel(context.Background())
	return &Scanner{
		cidr:     cidr,
		port:     port,
		credSink: credSink,
		ctx:      ctx,
		cancel:   cancel,
		progress: Progress{
			Phase: "fast", CIDR: cidr, Port: port,
			Found: []FoundDevice{}, StartedAt: time.Now(),
		},
		foundMap: map[string]FoundDevice{},
	}
}

// Cancel 请求取消（不阻塞）
func (s *Scanner) Cancel() { s.cancel() }

// Snapshot 线程安全进度快照
func (s *Scanner) Snapshot() Progress {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := s.progress
	p.Found = make([]FoundDevice, 0, len(s.foundMap))
	for _, d := range s.foundMap {
		p.Found = append(p.Found, d)
	}
	p.ElapsedMs = time.Since(s.progress.StartedAt).Milliseconds()
	return p
}

// Run 阻塞执行完整流程；应在独立 goroutine 中调用
func (s *Scanner) Run() Progress {
	ips, err := expandCIDR(s.cidr)
	if err != nil {
		s.mu.Lock()
		s.progress.Phase = "error"
		s.progress.Error = err.Error()
		s.mu.Unlock()
		return s.Snapshot()
	}
	s.mu.Lock()
	s.progress.Total = len(ips)
	s.mu.Unlock()

	// 阶段 1：快扫 + 发现即并行识别（识别不阻塞剩余地址扫描）
	var recheck []net.IP
	var wg sync.WaitGroup
	identifyWG := &sync.WaitGroup{}
	sem := make(chan struct{}, FastConcurrency)
	var mu sync.Mutex
	for i, ip := range ips {
		if s.cancelled() {
			s.setPhase("cancelled")
			identifyWG.Wait()
			return s.Snapshot()
		}
		sem <- struct{}{}
		wg.Add(1)
		go func(ip net.IP, idx int) {
			defer wg.Done()
			defer func() { <-sem }()
			open := tcpOpen(s.ctx, ip, s.port, FastTimeout)
			mu.Lock()
			if !open {
				recheck = append(recheck, ip) // 快扫超时/失败 → 待复查
			}
			s.mu.Lock()
			s.progress.Scanned = idx + 1
			s.mu.Unlock()
			mu.Unlock()
			if open {
				identifyWG.Add(1)
				go func() {
					defer identifyWG.Done()
					s.identify(ip)
				}()
			}
		}(ip, i)
	}
	wg.Wait()

	// 阶段 2：后台复查快扫未开放的地址（64 并发、1000ms）
	if !s.cancelled() && len(recheck) > 0 {
		s.setPhase("recheck")
		var wg2 sync.WaitGroup
		sem2 := make(chan struct{}, RecheckConcurrency)
		for _, ip := range recheck {
			if s.cancelled() {
				break
			}
			sem2 <- struct{}{}
			wg2.Add(1)
			go func(ip net.IP) {
				defer wg2.Done()
				defer func() { <-sem2 }()
				if tcpOpen(s.ctx, ip, s.port, RecheckTimeout) {
					s.identify(ip)
				}
			}(ip)
		}
		wg2.Wait()
	}

	identifyWG.Wait() // 识别与剩余扫描并行，但收尾前必须完成
	if s.cancelled() {
		s.setPhase("cancelled")
	} else {
		s.setPhase("done")
	}
	return s.Snapshot()
}

// identify 端口开放后的配置识别；已保存凭据优先，其次无凭据探测
func (s *Scanner) identify(ip net.IP) {
	if s.cancelled() {
		return
	}
	conns := []protocol.Conn{{Host: ip.String(), Port: s.port, SafetyMode: protocol.SafetyNone}}
	if s.credSink != nil {
		if saved := s.credSink(ip.String(), s.port); saved != nil {
			conns = append([]protocol.Conn{*saved}, conns...)
		}
	}
	var dev FoundDevice
	var lastErr string
	identified := false
	for _, conn := range conns {
		if s.cancelled() {
			return
		}
		c := protocol.NewClient(conn, IdentifyTimeout)
		cfg, err := c.ConfigQuery(s.ctx)
		if err == nil {
			dev = FoundDevice{Host: ip.String(), Port: s.port, State: "identified", Config: cfg}
			identified = true
			break
		}
		lastErr = err.Error()
		if pe, ok := err.(*protocol.PhoneError); ok && pe.Kind == protocol.FailAuth {
			// 明确签名/鉴权错误 → 待配置凭据的候选（计划 §3.1）
			dev = FoundDevice{Host: ip.String(), Port: s.port, State: "need-credentials", Error: pe.Msg}
			identified = true
			break
		}
	}
	if !identified {
		// 非 JSON / 解密失败等 → 未验证候选；不能仅凭密文形状判断产品或安全模式
		dev = FoundDevice{Host: ip.String(), Port: s.port, State: "unverified", Error: lastErr}
	}
	s.mu.Lock()
	s.foundMap[ip.String()] = dev
	s.mu.Unlock()
}

func (s *Scanner) cancelled() bool {
	select {
	case <-s.ctx.Done():
		return true
	default:
		return false
	}
}

func (s *Scanner) setPhase(p string) {
	s.mu.Lock()
	s.progress.Phase = p
	s.mu.Unlock()
}

func tcpOpen(ctx context.Context, ip net.IP, port int, timeout time.Duration) bool {
	d := net.Dialer{Timeout: timeout}
	cctx, cancel := context.WithTimeout(ctx, timeout+100*time.Millisecond)
	defer cancel()
	conn, err := d.DialContext(cctx, "tcp", fmt.Sprintf("%s:%d", ip.String(), port))
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

// LocalCIDRs 枚举本机 IPv4 /24 网段（排除回环/链路本地），供默认扫描范围
func LocalCIDRs() ([]string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	var out []string
	seen := map[string]bool{}
	for _, ifc := range ifaces {
		if ifc.Flags&net.FlagUp == 0 || ifc.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := ifc.Addrs()
		if err != nil {
			continue
		}
		for _, a := range addrs {
			ipnet, ok := a.(*net.IPNet)
			if !ok {
				continue
			}
			ip4 := ipnet.IP.To4()
			if ip4 == nil || ip4.IsLoopback() || ip4.IsLinkLocalUnicast() {
				continue
			}
			c := fmt.Sprintf("%s/24", ip4.Mask(ipnet.Mask).String())
			if !seen[c] {
				seen[c] = true
				out = append(out, c)
			}
		}
	}
	return out, nil
}

// expandCIDR 展开网段为可扫地址（去掉网络/广播地址）
func expandCIDR(cidr string) ([]net.IP, error) {
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, fmt.Errorf("无效网段 %q：%w", cidr, err)
	}
	var out []net.IP
	for cur := ip.Mask(ipnet.Mask); ipnet.Contains(cur); incIP(cur) {
		dup := make(net.IP, len(cur))
		copy(dup, cur)
		out = append(out, dup)
		if len(out) > MaxAddresses {
			return nil, fmt.Errorf("网段过大：最多 %d 个地址", MaxAddresses)
		}
	}
	if ones, _ := ipnet.Mask.Size(); ones <= 30 && len(out) > 2 {
		out = out[1 : len(out)-1]
	}
	return out, nil
}

func incIP(ip net.IP) {
	for i := len(ip) - 1; i >= 0; i-- {
		ip[i]++
		if ip[i] != 0 {
			break
		}
	}
}
