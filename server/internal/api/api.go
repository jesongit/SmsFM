// Package api: 控制台 HTTP API。
// 前端 ↔ 后端：连接管理、扫描、业务调用代理（按连接的安全模式与密钥由后端代调手机）。
package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"smsfm/server/internal/protocol"
	"smsfm/server/internal/scan"
	"smsfm/server/internal/store"
)

// Server 控制台后端
type Server struct {
	store       *store.Store
	mux         *http.ServeMux
	callTimeout time.Duration

	mu  sync.Mutex
	cur *scan.Scanner // 单扫描任务（单管理员场景）
}

// New 建立服务；staticFS 为前端构建产物（可为 nil：仅 API 模式）
func New(st *store.Store, staticFS fs.FS) *Server {
	s := &Server{store: st, mux: http.NewServeMux(), callTimeout: 30 * time.Second}
	s.routes()
	if staticFS != nil {
		s.mux.Handle("/", spaHandler(staticFS))
	}
	return s
}

func (s *Server) Handler() http.Handler { return s.mux }

func (s *Server) routes() {
	s.mux.HandleFunc("GET /api/health", s.health)
	s.mux.HandleFunc("GET /api/networks", s.networks)

	s.mux.HandleFunc("GET /api/connections", s.listConns)
	s.mux.HandleFunc("POST /api/connections", s.createConn)
	s.mux.HandleFunc("PUT /api/connections/{id}", s.updateConn)
	s.mux.HandleFunc("DELETE /api/connections/{id}", s.deleteConn)
	s.mux.HandleFunc("POST /api/connections/{id}/probe", s.probeConn)

	s.mux.HandleFunc("POST /api/scan/start", s.scanStart)
	s.mux.HandleFunc("GET /api/scan/status", s.scanStatus)
	s.mux.HandleFunc("POST /api/scan/cancel", s.scanCancel)

	for _, path := range []string{
		"/config/query", "/sms/query", "/sms/send", "/call/query",
		"/contact/query", "/contact/add", "/battery/query",
		"/location/query", "/wol/send", "/clone/pull", "/clone/push",
	} {
		p := path
		s.mux.HandleFunc("POST /api/connections/{id}"+p, s.proxy(p))
	}
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "service": "smsfm-server", "time": time.Now().Format(time.RFC3339)})
}

// networks 返回默认扫描网段，供前端。
// 优先读 SMSFM_NETWORK_CIDRS（逗号分隔，如 "192.168.1.0/24,10.0.0.0/24"）——
// 容器 bridge 网络下容器网卡是 docker 内网（如 172.17.0.x），枚举不到宿主局域网，
// 用环境变量显式指定；Linux host 网络模式则无需设置。
func (s *Server) networks(w http.ResponseWriter, _ *http.Request) {
	if env := os.Getenv("SMSFM_NETWORK_CIDRS"); env != "" {
		var cidrs []string
		for _, c := range strings.Split(env, ",") {
			if c = strings.TrimSpace(c); c != "" {
				if _, _, err := net.ParseCIDR(c); err != nil {
					writeErr(w, http.StatusBadRequest, fmt.Errorf("SMSFM_NETWORK_CIDRS 含非法网段 %q", c))
					return
				}
				cidrs = append(cidrs, c)
			}
		}
		if len(cidrs) > 0 {
			writeJSON(w, http.StatusOK, map[string]any{"cidrs": cidrs, "source": "env"})
			return
		}
	}
	cidrs, err := scan.LocalCIDRs()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"cidrs": cidrs, "source": "auto"})
}

/* ── 连接管理 ── */

func (s *Server) listConns(w http.ResponseWriter, _ *http.Request) {
	items := s.store.List()
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.Before(items[j].CreatedAt) })
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

type connBody struct {
	Host       string              `json:"host"`
	Port       int                 `json:"port"`
	SafetyMode protocol.SafetyMode `json:"safetyMode"`
	Secret     string              `json:"secret"`
	Remark     string              `json:"remark"`
}

func (b connBody) validate() error {
	if b.Host == "" {
		return errors.New("主机地址不能为空")
	}
	if b.Port <= 0 || b.Port > 65535 {
		return errors.New("端口非法")
	}
	if b.SafetyMode < protocol.SafetyNone || b.SafetyMode > protocol.SafetySM4 {
		return errors.New("安全模式非法")
	}
	return nil
}

// createConn 手动连接：先测试配置查询，成功才保存（计划 §3.2）
func (s *Server) createConn(w http.ResponseWriter, r *http.Request) {
	var b connBody
	if !readJSON(w, r, &b) {
		return
	}
	if err := b.validate(); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	// 查重：同一 host:port 已保存则返回现有条目（幂等；凭据/备注以现有为准提示更新）
	if exist, ok := s.store.FindByAddr(b.Host, b.Port); ok {
		cfg, err := protocol.NewClient(protocol.Conn{
			Host: exist.Host, Port: exist.Port, SafetyMode: exist.SafetyMode, Secret: exist.Secret,
		}, 8*time.Second).ConfigQuery(r.Context())
		if err != nil {
			writePhoneErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"entry": exist, "config": cfg, "existed": true})
		return
	}
	conn := protocol.Conn{Host: b.Host, Port: b.Port, SafetyMode: b.SafetyMode, Secret: b.Secret}
	cfg, err := protocol.NewClient(conn, 8*time.Second).ConfigQuery(r.Context())
	if err != nil {
		writePhoneErr(w, err)
		return
	}
	entry, err := s.store.Create(store.Entry{
		Host: b.Host, Port: b.Port, SafetyMode: b.SafetyMode,
		Secret: b.Secret, Remark: b.Remark,
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"entry": entry, "config": cfg, "existed": false})
}

func (s *Server) updateConn(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var b connBody
	if !readJSON(w, r, &b) {
		return
	}
	if err := b.validate(); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	entry, err := s.store.Update(id, store.Entry{
		Host: b.Host, Port: b.Port, SafetyMode: b.SafetyMode,
		Secret: b.Secret, Remark: b.Remark,
	})
	if err != nil {
		writeErr(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"entry": entry})
}

func (s *Server) deleteConn(w http.ResponseWriter, r *http.Request) {
	if err := s.store.Delete(r.PathValue("id")); err != nil {
		writeErr(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// probeConn 连接测试 + 读配置（刷新状态）
func (s *Server) probeConn(w http.ResponseWriter, r *http.Request) {
	conn, ok := s.store.ConnOf(r.PathValue("id"))
	if !ok {
		writeErr(w, http.StatusNotFound, os.ErrNotExist)
		return
	}
	cfg, err := protocol.NewClient(conn, 8*time.Second).ConfigQuery(r.Context())
	if err != nil {
		writePhoneErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"config": cfg})
}

/* ── 扫描 ── */

type scanBody struct {
	CIDR string `json:"cidr"`
	Port int    `json:"port"`
}

func (s *Server) scanStart(w http.ResponseWriter, r *http.Request) {
	var b scanBody
	if !readJSON(w, r, &b) {
		return
	}
	if b.Port == 0 {
		b.Port = 5000
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cur != nil {
		switch p := s.cur.Snapshot().Phase; p {
		case "fast", "recheck":
			writeErr(w, http.StatusConflict, errors.New("扫描进行中，请先取消或等待完成"))
			return
		}
	}
	sc := scan.New(b.CIDR, b.Port, func(host string, port int) *protocol.Conn {
		if e, ok := s.store.FindByAddr(host, port); ok {
			c := protocol.Conn{Host: e.Host, Port: e.Port, SafetyMode: e.SafetyMode, Secret: e.Secret}
			return &c
		}
		return nil
	})
	s.cur = sc
	go sc.Run()
	writeJSON(w, http.StatusAccepted, sc.Snapshot())
}

func (s *Server) scanStatus(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	sc := s.cur
	s.mu.Unlock()
	if sc == nil {
		writeJSON(w, http.StatusOK, scan.Progress{Phase: "idle"})
		return
	}
	writeJSON(w, http.StatusOK, sc.Snapshot())
}

func (s *Server) scanCancel(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	sc := s.cur
	s.mu.Unlock()
	if sc != nil {
		sc.Cancel()
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

/* ── 业务代理：后端按连接凭据代调手机接口 ── */

func (s *Server) proxy(phonePath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, ok := s.store.ConnOf(r.PathValue("id"))
		if !ok {
			writeErr(w, http.StatusNotFound, os.ErrNotExist)
			return
		}
		var dataIn any
		if r.Body != nil {
			raw, _ := io.ReadAll(io.LimitReader(r.Body, 4<<20))
			if len(raw) > 0 {
				if err := json.Unmarshal(raw, &dataIn); err != nil {
					writeErr(w, http.StatusBadRequest, fmt.Errorf("请求体须为 JSON 对象：%w", err))
					return
				}
			}
		}
		ctx, cancel := context.WithTimeout(r.Context(), s.callTimeout)
		defer cancel()
		out, err := protocol.NewClient(conn, s.callTimeout).Call(ctx, phonePath, dataIn)
		if err != nil {
			writePhoneErr(w, err)
			return
		}
		writeRaw(w, http.StatusOK, out)
	}
}

/* ── 工具 ── */

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeRaw(w http.ResponseWriter, code int, raw json.RawMessage) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	if len(raw) == 0 {
		raw = json.RawMessage("null")
	}
	_, _ = w.Write(raw)
}

func writeErr(w http.ResponseWriter, code int, err error) {
	writeJSON(w, code, map[string]any{"error": err.Error()})
}

// writePhoneErr 把 PhoneError 的分级带回前端（kind + 中文标签 + 消息）
func writePhoneErr(w http.ResponseWriter, err error) {
	var pe *protocol.PhoneError
	if errors.As(err, &pe) {
		writeJSON(w, http.StatusOK, map[string]any{
			"error": pe.Error(), "kind": string(pe.Kind), "kindLabel": pe.Kind.String(), "code": pe.Code,
		})
		return
	}
	writeErr(w, http.StatusInternalServerError, err)
}

func readJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return false
	}
	if err := json.Unmarshal(raw, v); err != nil {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("JSON 解析失败：%w", err))
		return false
	}
	return true
}

// spaHandler 前端单页应用：静态文件 + 非 /api 路径回退 index.html
func spaHandler(f fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(f))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		if p != "/" {
			if _, err := fs.Stat(f, p[1:]); err != nil {
				// 回退到入口（前端路由）
				r2 := new(http.Request)
				*r2 = *r
				r2.URL.Path = "/"
				http.ServeFileFS(w, r2, f, "index.html")
				return
			}
		}
		fileServer.ServeHTTP(w, r)
	})
}
