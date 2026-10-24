package protocol

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/url"
)

// CalcSign 复刻上游 HttpServerUtils.calcSign：
// stringToSign = "timestamp\n" + secret；HMAC-SHA256(key=secret)；Base64(标准)；URL 编码。
// 注意（开发计划 §6）：时间戳签名不是完整的防篡改/防重放机制，仅按上游语义兼容。
func CalcSign(timestamp string, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(timestamp + "\n" + secret))
	sum := mac.Sum(nil)
	b64 := base64.StdEncoding.EncodeToString(sum)
	return url.QueryEscape(b64)
}
