package protocol

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// 错误分级（开发计划 §3.2：连接失败、鉴权失败、功能关闭、业务错误须区分展示）
type FailureKind string

const (
	FailConnection FailureKind = "connection"
	FailAuth       FailureKind = "auth"
	FailFeatureOff FailureKind = "feature-off"
	FailBusiness   FailureKind = "business"
	FailProtocol   FailureKind = "protocol"
)

func (k FailureKind) String() string {
	switch k {
	case FailConnection:
		return "连接失败"
	case FailAuth:
		return "鉴权失败"
	case FailFeatureOff:
		return "功能未开启"
	case FailBusiness:
		return "业务错误"
	case FailProtocol:
		return "协议不符"
	}
	return "未知错误"
}

// PhoneError 分类错误；HTTP 200 + code 500 属业务层失败，须看 Msg
type PhoneError struct {
	Kind FailureKind
	Msg  string
	Code int
}

func (e *PhoneError) Error() string {
	if e.Code != 0 && e.Code != 200 {
		return fmt.Sprintf("%s：%s（code %d）", e.Kind, e.Msg, e.Code)
	}
	return fmt.Sprintf("%s：%s", e.Kind, e.Msg)
}

// Conn 一次手机连接的参数（安全模式与密钥）
type Conn struct {
	Host       string     `json:"host"`
	Port       int        `json:"port"`
	SafetyMode SafetyMode `json:"safetyMode"`
	Secret     string     `json:"secret,omitempty"` // 签名密钥 / RSA 公钥 / SM4 十六进制密钥
}

func (c Conn) Addr() string { return fmt.Sprintf("%s:%d", c.Host, c.Port) }

// Client 按上游协议与一台手机通信
type Client struct {
	HTTP *http.Client
	conn Conn
}

// NewClient 默认客户端；超时覆盖业务调用的最长等待
func NewClient(conn Conn, timeout time.Duration) *Client {
	return &Client{
		HTTP: &http.Client{Timeout: timeout},
		conn: conn,
	}
}

// Call 发送一次请求并解包 BaseResponse；dataIn 为业务参数对象。
// 返回业务 code==200 时的 data 原始 JSON；网络/解包错误以 *PhoneError 分类。
func (c *Client) Call(ctx context.Context, path string, dataIn any) (json.RawMessage, error) {
	body, err := c.buildRequestBody(dataIn)
	if err != nil {
		return nil, &PhoneError{Kind: FailAuth, Msg: "请求加密失败：" + err.Error()}
	}
	url := fmt.Sprintf("http://%s%s", c.conn.Addr(), path)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, &PhoneError{Kind: FailConnection, Msg: err.Error()}
	}
	req.Header.Set("Content-Type", contentTypeFor(c.conn.SafetyMode))
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, &PhoneError{Kind: FailConnection, Msg: dialErrMsg(err)}
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return nil, &PhoneError{Kind: FailConnection, Msg: "读取响应失败：" + err.Error()}
	}
	if resp.StatusCode != http.StatusOK {
		return nil, &PhoneError{Kind: FailProtocol, Msg: fmt.Sprintf("HTTP %d（上游业务异常均返回 200，非 200 说明不是 SmsForwarder 服务）", resp.StatusCode)}
	}
	jsonBody, err := c.decodeResponseBody(raw)
	if err != nil {
		// 加密模式下密钥不匹配的首要表现就是解不开密文
		return nil, &PhoneError{Kind: FailAuth, Msg: "响应解密失败（密钥或安全模式可能不匹配）：" + err.Error()}
	}
	var base BaseResponse
	if err := json.Unmarshal(jsonBody, &base); err != nil {
		return nil, &PhoneError{Kind: FailProtocol, Msg: "响应不是 SmsForwarder 报文（非 JSON）"}
	}
	if base.Code != 200 {
		kind := FailBusiness
		if isAuthMsg(base.Msg) {
			kind = FailAuth
		} else if isFeatureOffMsg(base.Msg) {
			kind = FailFeatureOff
		}
		return nil, &PhoneError{Kind: kind, Msg: base.Msg, Code: base.Code}
	}
	return base.Data, nil
}

// ConfigQuery 连接识别：读设备信息、SIM、版本与功能开关
func (c *Client) ConfigQuery(ctx context.Context) (*ConfigData, error) {
	data, err := c.Call(ctx, "/config/query", map[string]any{})
	if err != nil {
		return nil, err
	}
	var cfg ConfigData
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, &PhoneError{Kind: FailProtocol, Msg: "配置结构解析失败：" + err.Error()}
	}
	return &cfg, nil
}

// buildRequestBody 按安全模式组装请求体
func (c *Client) buildRequestBody(dataIn any) ([]byte, error) {
	dataJSON, err := json.Marshal(dataIn)
	if err != nil {
		return nil, err
	}
	env := BaseRequest{Data: dataJSON, Timestamp: time.Now().UnixMilli()}
	switch c.conn.SafetyMode {
	case SafetyNone:
		return json.Marshal(env)
	case SafetySign:
		if c.conn.Secret == "" {
			return nil, errors.New("签名模式缺少密钥")
		}
		env.Sign = CalcSign(fmt.Sprintf("%d", env.Timestamp), c.conn.Secret)
		return json.Marshal(env)
	case SafetyRSA:
		if c.conn.Secret == "" {
			return nil, errors.New("RSA 模式缺少公钥")
		}
		pub, err := ParseRSAPublicKey(c.conn.Secret)
		if err != nil {
			return nil, errors.New("RSA 公钥无效：" + err.Error())
		}
		envJSON, err := json.Marshal(env)
		if err != nil {
			return nil, err
		}
		// 上游：Base64(信封) → 公钥分段加密 → Base64
		wire, err := RSAEncryptByPublicKey([]byte(base64.StdEncoding.EncodeToString(envJSON)), pub)
		if err != nil {
			return nil, err
		}
		return []byte(wire), nil
	case SafetySM4:
		if c.conn.Secret == "" {
			return nil, errors.New("SM4 模式缺少密钥")
		}
		key, err := hex.DecodeString(c.conn.Secret)
		if err != nil || len(key) != 16 {
			return nil, errors.New("SM4 密钥须为 32 位十六进制（16 字节）")
		}
		envJSON, err := json.Marshal(env)
		if err != nil {
			return nil, err
		}
		wire, err := SM4EncryptHex(envJSON, key)
		if err != nil {
			return nil, err
		}
		return []byte(wire), nil
	}
	return nil, errors.New("未知安全模式")
}

// decodeResponseBody 按模式解开响应密文，得到 BaseResponse JSON
func (c *Client) decodeResponseBody(raw []byte) ([]byte, error) {
	switch c.conn.SafetyMode {
	case SafetyRSA:
		pub, err := ParseRSAPublicKey(c.conn.Secret)
		if err != nil {
			return nil, err
		}
		// 上游：decryptByPublicKey 得到 Base64(响应)，再 Base64 解码
		inner, err := RSADecryptByPublicKey(string(raw), pub)
		if err != nil {
			return nil, err
		}
		return base64.StdEncoding.DecodeString(inner)
	case SafetySM4:
		key, err := hex.DecodeString(c.conn.Secret)
		if err != nil || len(key) != 16 {
			return nil, errors.New("SM4 密钥无效")
		}
		return SM4DecryptHex(string(raw), key)
	default:
		return raw, nil
	}
}

func contentTypeFor(mode SafetyMode) string {
	if mode == SafetyRSA || mode == SafetySM4 {
		// 上游 upString：纯文本体
		return "text/plain; charset=utf-8"
	}
	return "application/json; charset=utf-8"
}

// isAuthMsg 上游 checkSign 抛 HttpException(500)，文案含签名/timestamp；据此归入鉴权失败
func isAuthMsg(msg string) bool {
	m := strings.ToLower(msg)
	return strings.Contains(msg, "签名") ||
		strings.Contains(m, "sign") ||
		strings.Contains(msg, "时间戳") ||
		strings.Contains(m, "timestamp")
}

// isFeatureOffMsg 上游功能开关关闭时返回「服务端禁用此功能」类文案
func isFeatureOffMsg(msg string) bool {
	return strings.Contains(msg, "禁用") || strings.Contains(msg, "未开启") || strings.Contains(msg, "未启用")
}

func dialErrMsg(err error) string {
	msg := err.Error()
	if strings.Contains(msg, "context deadline exceeded") || strings.Contains(msg, "Client.Timeout") {
		return "连接超时：" + msg
	}
	if strings.Contains(msg, "connection refused") {
		return "连接被拒绝（端口未开放或服务已停止）"
	}
	return msg
}
