package protocol

import (
	"context"
	"fmt"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// fakePhone 完整模拟上游 SmsForwarder 服务端的一种安全模式，用于端到端协议验证。
// 行为对齐 AppMessageConverter + HttpServerUtils.checkSign + AppExceptionResolver。
type fakePhone struct {
	mode     SafetyMode
	signKey  string
	sm4Key   []byte
	rsaKey   *rsa.PrivateKey // 手机侧密钥对（客户端只拿公钥）
	timeTol  time.Duration

	sawEnvelope *BaseRequest // 记录最近一次解开的请求信封
}

func newFakePhone(mode SafetyMode) *fakePhone {
	fp := &fakePhone{mode: mode, timeTol: 60 * time.Second}
	if mode == SafetySign {
		fp.signKey = "e2e-sign-key"
	}
	if mode == SafetySM4 {
		fp.sm4Key, _ = hex.DecodeString("00112233445566778899AABBCCDDEEFF")
	}
	if mode == SafetyRSA {
		fp.rsaKey, _ = rsa.GenerateKey(rand.Reader, 2048)
	}
	return fp
}

func (fp *fakePhone) publicKeyB64() string {
	der, _ := x509.MarshalPKIXPublicKey(&fp.rsaKey.PublicKey)
	return base64.StdEncoding.EncodeToString(der)
}

func (fp *fakePhone) secretForClient() string {
	switch fp.mode {
	case SafetySign:
		return fp.signKey
	case SafetySM4:
		return strings.ToUpper(hex.EncodeToString(fp.sm4Key))
	case SafetyRSA:
		return fp.publicKeyB64()
	}
	return ""
}

// handler 模拟各接口：/config/query 返回配置；其他返回固定数据；异常走 code:500 包装
func (fp *fakePhone) handler(t *testing.T) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		envJSON, err := fp.openRequest(body)
		if err != nil {
			fp.writeWrapped(w, "请求解密失败："+err.Error())
			return
		}
		var env BaseRequest
		if err := json.Unmarshal(envJSON, &env); err != nil {
			fp.writeWrapped(w, "请求不是合法信封")
			return
		}
		fp.sawEnvelope = &env
		if fp.mode == SafetySign && fp.signKey != "" {
			if env.Sign == "" || env.Timestamp == 0 {
				fp.writeWrapped(w, "签名校验失败：缺少 sign 或 timestamp")
				return
			}
			if time.Since(time.UnixMilli(env.Timestamp)) > fp.timeTol {
				fp.writeWrapped(w, "时间戳校验失败")
				return
			}
			if CalcSign(itoa(env.Timestamp), fp.signKey) != env.Sign {
				fp.writeWrapped(w, "签名校验失败")
				return
			}
		}
		switch r.URL.Path {
		case "/config/query":
			fp.writeData(w, map[string]any{
				"enable_api_sms_query": true, "enable_api_sms_send": true,
				"enable_api_location": false,
				"extra_device_mark":   "e2e-fake", "version_name": "3.3.2-e2e",
				"sim_info_list":       map[string]any{"0": map[string]any{"mCarrierName": "测试运营商", "mSimSlotIndex": 0}},
			})
		case "/sms/query":
			fp.writeData(w, []map[string]any{{"name": "张三", "number": "13800000000", "content": "验证码 246810", "date": 1761356800000, "type": 1, "sim_id": 0}})
		default:
			fp.writeData(w, map[string]any{"path": r.URL.Path})
		}
	}
}

func itoa(v int64) string {
	return strings.TrimSpace(strings.ReplaceAll(json.Number(fmtInt(v)).String(), "\"", ""))
}

func fmtInt(v int64) string {
	b, _ := json.Marshal(v)
	return string(b)
}

// openRequest 按模式解开请求体 → 信封 JSON
func (fp *fakePhone) openRequest(body []byte) ([]byte, error) {
	switch fp.mode {
	case SafetyRSA:
		cipherAll, err := base64.StdEncoding.DecodeString(string(body))
		if err != nil {
			return nil, err
		}
		var out []byte
		for off := 0; off < len(cipherAll); off += 256 {
			end := off + 256
			if end > len(cipherAll) {
				end = len(cipherAll)
			}
			// 公钥加密的块用私钥解（标准 PKCS1 v1.5 bt2）
			m := new(big.Int).Exp(new(big.Int).SetBytes(cipherAll[off:end]), fp.rsaKey.D, fp.rsaKey.N)
			em := make([]byte, 256)
			m.FillBytes(em)
			plain, err := unpadPKCS1BlockType2(em)
			if err != nil {
				return nil, err
			}
			out = append(out, plain...)
		}
		return base64.StdEncoding.DecodeString(string(out))
	case SafetySM4:
		return SM4DecryptHex(string(body), fp.sm4Key)
	default:
		return body, nil
	}
}

// writeWrapped 模拟 HttpServerUtils.response + 按模式加密（异常路径）
func (fp *fakePhone) writeWrapped(w http.ResponseWriter, msg string) {
	resp := map[string]any{"code": 500, "msg": msg, "timestamp": time.Now().UnixMilli()}
	raw, _ := json.Marshal(resp)
	fp.writeWire(w, raw)
}

func (fp *fakePhone) writeData(w http.ResponseWriter, data any) {
	resp := map[string]any{"code": 200, "msg": "success", "data": data, "timestamp": time.Now().UnixMilli()}
	raw, _ := json.Marshal(resp)
	fp.writeWire(w, raw)
}

// writeWire 按模式加密响应体（对齐 AppMessageConverter.convert）
func (fp *fakePhone) writeWire(w http.ResponseWriter, jsonBody []byte) {
	switch fp.mode {
	case SafetyRSA:
		inner := base64.StdEncoding.EncodeToString(jsonBody)
		wire := fp.privEncryptBT1([]byte(inner))
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte(wire))
	case SafetySM4:
		wire, _ := SM4EncryptHex(jsonBody, fp.sm4Key)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte(wire))
	default:
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = w.Write(jsonBody)
	}
}

// privEncryptBT1 私钥加密（块类型 1，模拟上游 encryptByPrivateKey）
func (fp *fakePhone) privEncryptBT1(input []byte) string {
	k := 256
	var out []byte
	for off := 0; off < len(input); off += 245 {
		end := off + 245
		if end > len(input) {
			end = len(input)
		}
		msg := input[off:end]
		em := make([]byte, k)
		em[0], em[1] = 0x00, 0x01
		i := 2
		for ; i < k-len(msg)-1; i++ {
			em[i] = 0xFF
		}
		em[i] = 0x00
		copy(em[i+1:], msg)
		c := new(big.Int).Exp(new(big.Int).SetBytes(em), fp.rsaKey.D, fp.rsaKey.N)
		block := make([]byte, k)
		c.FillBytes(block)
		out = append(out, block...)
	}
	return base64.StdEncoding.EncodeToString(out)
}

func unpadPKCS1BlockType2(em []byte) ([]byte, error) {
	if len(em) < 11 || em[0] != 0x00 || em[1] != 0x02 {
		return nil, errTest("bt2 头不符")
	}
	i := 2
	for i < len(em) && em[i] != 0x00 {
		i++
	}
	if i == len(em) || i < 10 {
		return nil, errTest("bt2 分隔不符")
	}
	return em[i+1:], nil
}

type errTest string

func (e errTest) Error() string { return string(e) }

/* ── 端到端用例 ── */

func TestE2ENoneMode(t *testing.T) {
	runE2E(t, SafetyNone)
}

func TestE2ESignMode(t *testing.T) {
	runE2E(t, SafetySign)
}

func TestE2ERSAMode(t *testing.T) {
	runE2E(t, SafetyRSA)
}

func TestE2ESM4Mode(t *testing.T) {
	runE2E(t, SafetySM4)
}

func runE2E(t *testing.T, mode SafetyMode) {
	fp := newFakePhone(mode)
	srv := httptest.NewServer(fp.handler(t))
	defer srv.Close()
	host, port := splitHostPort(t, srv.URL)

	conn := Conn{Host: host, Port: port, SafetyMode: mode, Secret: fp.secretForClient()}
	c := NewClient(conn, 5*time.Second)

	cfg, err := c.ConfigQuery(context.Background())
	if err != nil {
		t.Fatalf("[%v] config/query 失败：%v", mode, err)
	}
	if cfg.VersionName != "3.3.2-e2e" || cfg.ExtraDeviceMark != "e2e-fake" {
		t.Fatalf("[%v] 配置内容不符：%+v", mode, cfg)
	}
	if !cfg.EnableApiSmsQuery || cfg.EnableApiLocation {
		t.Fatalf("[%v] 功能开关不符", mode)
	}

	data, err := c.Call(context.Background(), "/sms/query", map[string]any{"type": 1, "page_num": 1, "page_size": 10, "keyword": ""})
	if err != nil {
		t.Fatalf("[%v] sms/query 失败：%v", mode, err)
	}
	var list []map[string]any
	if err := json.Unmarshal(data, &list); err != nil {
		t.Fatalf("[%v] sms 数据解析失败：%v", mode, err)
	}
	if list[0]["content"] != "验证码 246810" {
		t.Fatalf("[%v] 短信内容不符：%v", mode, list[0])
	}

	// 信封应被手机正确解开（timestamp 存在）
	if fp.sawEnvelope == nil || fp.sawEnvelope.Timestamp == 0 {
		t.Fatalf("[%v] 手机未解开请求信封", mode)
	}
}

// TestE2EWrongSignKey 错误密钥 → 鉴权失败分类
func TestE2EWrongSignKey(t *testing.T) {
	fp := newFakePhone(SafetySign)
	srv := httptest.NewServer(fp.handler(t))
	defer srv.Close()
	host, port := splitHostPort(t, srv.URL)

	c := NewClient(Conn{Host: host, Port: port, SafetyMode: SafetySign, Secret: "wrong-key"}, 5*time.Second)
	_, err := c.ConfigQuery(context.Background())
	if err == nil {
		t.Fatal("错误密钥应失败")
	}
	pe, ok := err.(*PhoneError)
	if !ok || pe.Kind != FailAuth {
		t.Fatalf("应分类为鉴权失败，实际：%v", err)
	}
}

// TestE2EConnectionRefused 连接失败分类
func TestE2EConnectionRefused(t *testing.T) {
	// 127.0.0.1:1 一般无监听
	c := NewClient(Conn{Host: "127.0.0.1", Port: 1}, 1500*time.Millisecond)
	_, err := c.ConfigQuery(context.Background())
	pe, ok := err.(*PhoneError)
	if !ok || pe.Kind != FailConnection {
		t.Fatalf("应分类为连接失败，实际：%v", err)
	}
}

// TestE2ENotSmsForwarder 非 JSON 响应 → 协议不符
func TestE2ENotSmsForwarder(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("<html>not a phone</html>"))
	}))
	defer srv.Close()
	host, port := splitHostPort(t, srv.URL)
	c := NewClient(Conn{Host: host, Port: port}, 3*time.Second)
	_, err := c.ConfigQuery(context.Background())
	pe, ok := err.(*PhoneError)
	if !ok || (pe.Kind != FailProtocol && pe.Kind != FailAuth) {
		t.Fatalf("非 JSON 应分类为协议不符，实际：%v", err)
	}
}

func splitHostPort(t *testing.T, url string) (string, int) {
	t.Helper()
	u := strings.TrimPrefix(url, "http://")
	parts := strings.Split(u, ":")
	if len(parts) != 2 {
		t.Fatalf("测试地址异常：%s", url)
	}
	var port int
	if _, err := fmt.Sscanf(parts[1], "%d", &port); err != nil {
		t.Fatalf("端口解析失败：%v", err)
	}
	return parts[0], port
}
