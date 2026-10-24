package scan

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"smsfm/server/internal/protocol"
)

// fakePhoneHandler 最小可识别手机（无安全模式，/config/query 返回有效配置）
func fakePhoneHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/config/query" {
			_, _ = w.Write([]byte(`{"code":200,"msg":"success","data":{"enable_api_battery_query":true,"extra_device_mark":"scan-e2e","version_name":"3.3.2"},"timestamp":1}`))
			return
		}
		_, _ = w.Write([]byte(`{"code":200,"msg":"success","timestamp":1}`))
	}
}

func localhostCIDR(t *testing.T) string {
	t.Helper()
	return "127.0.0.1/32"
}

func TestScanFindsFakePhone(t *testing.T) {
	srv := httptest.NewServer(fakePhoneHandler())
	defer srv.Close()
	port := portOf(t, srv.URL)

	sc := New(localhostCIDR(t), port, nil)
	done := make(chan Progress, 1)
	go func() { done <- sc.Run() }()
	select {
	case p := <-done:
		if p.Phase != "done" {
			t.Fatalf("phase=%s error=%s", p.Phase, p.Error)
		}
		if p.Total != 1 {
			t.Fatalf("/32 应只含 1 个地址，实际 %d", p.Total)
		}
		found := false
		for _, d := range p.Found {
			if d.Port == port && d.State == "identified" && d.Config != nil && d.Config.ExtraDeviceMark == "scan-e2e" {
				found = true
			}
		}
		if !found {
			t.Fatalf("未识别到假手机：%+v", p.Found)
		}
	case <-time.After(20 * time.Second):
		t.Fatal("扫描超时")
	}
}

// TestScanCancel 取消保留已完成部分
func TestScanCancel(t *testing.T) {
	srv := httptest.NewServer(fakePhoneHandler())
	defer srv.Close()
	port := portOf(t, srv.URL)

	sc := New("127.0.0.0/29", port, nil)
	go func() { _ = sc.Run() }()
	time.Sleep(30 * time.Millisecond)
	sc.Cancel()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if p := sc.Snapshot(); p.Phase == "cancelled" || p.Phase == "done" {
			return // 取消后收敛
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("取消未生效")
}

func TestExpandCIDR(t *testing.T) {
	ips, err := expandCIDR("192.168.1.0/24")
	if err != nil {
		t.Fatal(err)
	}
	if len(ips) != 254 {
		t.Fatalf("/24 应展开 254 地址，实际 %d", len(ips))
	}
	if ips[0].String() != "192.168.1.1" || ips[253].String() != "192.168.1.254" {
		t.Fatalf("首尾地址不符：%s ~ %s", ips[0], ips[253])
	}
	one, err := expandCIDR("127.0.0.5/32")
	if err != nil {
		t.Fatal(err)
	}
	if len(one) != 1 || one[0].String() != "127.0.0.5" {
		t.Fatalf("/32 应保留单地址：%v", one)
	}
}

func TestLocalCIDRs(t *testing.T) {
	cidrs, err := LocalCIDRs()
	if err != nil {
		t.Fatal(err)
	}
	if len(cidrs) == 0 {
		t.Log("警告：未发现可用 IPv4 网段")
	}
	for _, c := range cidrs {
		if _, _, err := net.ParseCIDR(c); err != nil {
			t.Fatalf("非法网段 %s：%v", c, err)
		}
	}
}

func portOf(t *testing.T, url string) int {
	t.Helper()
	parts := strings.Split(strings.TrimPrefix(url, "http://"), ":")
	var port int
	if _, err := fmt.Sscanf(parts[len(parts)-1], "%d", &port); err != nil {
		t.Fatalf("端口解析失败：%v", err)
	}
	return port
}

// 编译期保证类型被引用（Progress JSON 序列化供 API 使用）
var _ = func() error {
	_, _ = json.Marshal(Progress{})
	_, _ = json.Marshal(protocol.ConfigData{})
	_ = context.Background()
	return nil
}()
