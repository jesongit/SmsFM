package protocol

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"testing"
)

// TestCalcSign 对照参照值：Node crypto 以相同输入计算（2026-10-24 生成）
// node: createHmac('sha256', secret).update(ts+'\n'+secret) → base64 → encodeURIComponent
func TestCalcSign(t *testing.T) {
	got := CalcSign("1761356800000", "demo-key-123")
	want := "xcueGziYfl2ELdBwPQiz8%2BwrZWzmQzktnrSWa%2BzcHIs%3D"
	if got != want {
		t.Fatalf("签名与参照值不符：\n got  %s\n want %s", got, want)
	}
}

// TestSM4Roundtrip SM4/CBC/PKCS7 固定 IV 往返
func TestSM4Roundtrip(t *testing.T) {
	key, _ := hex.DecodeString("0123456789ABCDEFFEDCBA9876543210")
	plain := []byte(`{"code":200,"msg":"success","data":{"version_name":"3.3.2"},"timestamp":1761356800000}`)
	wire, err := SM4EncryptHex(plain, key)
	if err != nil {
		t.Fatal(err)
	}
	back, err := SM4DecryptHex(wire, key)
	if err != nil {
		t.Fatal(err)
	}
	if string(back) != string(plain) {
		t.Fatalf("SM4 往返不一致：\n got  %s\n want %s", back, plain)
	}
}

// TestSM4CiphertextShape 密文长度 = 明文 PKCS7 对齐后长度；hex 大写
func TestSM4CiphertextShape(t *testing.T) {
	key := make([]byte, 16)
	plain := []byte("abc") // 3 → 填充到 16
	wire, err := SM4EncryptHex(plain, key)
	if err != nil {
		t.Fatal(err)
	}
	if len(wire) != 32 { // 16 字节 = 32 hex 字符
		t.Fatalf("密文 hex 长度 %d，期望 32", len(wire))
	}
	for _, c := range wire {
		if c >= 'a' && c <= 'f' {
			t.Fatalf("hex 应为大写：%s", wire)
		}
	}
}

// TestRSAPubEncryptPrivDecrypt 公钥加密（我的实现）→ Go 标准库私钥解密（上游手机的视角）
func TestRSAPubEncryptPrivDecrypt(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	pubDER, _ := x509.MarshalPKIXPublicKey(&key.PublicKey)
	pubB64 := base64.StdEncoding.EncodeToString(pubDER)

	pub, err := ParseRSAPublicKey(pubB64)
	if err != nil {
		t.Fatal(err)
	}
	// 500 字节明文（跨 245 分段边界）
	plain := make([]byte, 500)
	for i := range plain {
		plain[i] = byte('a' + i%26)
	}
	wire, err := RSAEncryptByPublicKey(plain, pub)
	if err != nil {
		t.Fatal(err)
	}

	// 标准库私钥解密：Base64 → 分段 256 解开
	cipherAll, _ := base64.StdEncoding.DecodeString(wire)
	var out []byte
	for off := 0; off < len(cipherAll); off += 256 {
		end := off + 256
		if end > len(cipherAll) {
			end = len(cipherAll)
		}
		block, err := rsa.DecryptPKCS1v15(nil, key, cipherAll[off:end])
		if err != nil {
			t.Fatalf("标准库私钥解密失败（块 %d）：%v", off/256, err)
		}
		out = append(out, block...)
	}
	if string(out) != string(plain) {
		t.Fatal("RSA 公钥加密/私钥解密不一致")
	}
}

// TestRSAPubDecryptBt1 私钥 bt1 加密（手机的响应路径，测试内手工构造）→ 我的公钥解密
func TestRSAPubDecryptBt1(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	msg := []byte(base64.StdEncoding.EncodeToString([]byte(`{"code":200}`))) // 上游：Base64(响应) 再私钥加密
	k := 256

	em := make([]byte, k)
	em[0], em[1] = 0x00, 0x01
	i := 2
	for ; i < k-len(msg)-1; i++ {
		em[i] = 0xFF
	}
	em[i] = 0x00
	copy(em[i+1:], msg)

	c := new(big.Int).Exp(new(big.Int).SetBytes(em), key.D, key.N)
	cipher := make([]byte, k)
	c.FillBytes(cipher)
	wire := base64.StdEncoding.EncodeToString(cipher)

	got, err := RSADecryptByPublicKey(wire, &key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	// 上游客户端：decryptByPublicKey 得到 Base64(响应) 字符串 → 再 Base64 解码
	inner, err := base64.StdEncoding.DecodeString(got)
	if err != nil {
		t.Fatal(err)
	}
	var resp BaseResponse
	if err := json.Unmarshal(inner, &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Code != 200 {
		t.Fatalf("解开后的响应 code=%d", resp.Code)
	}
}

// TestSM4FixedIV 断言固定 IV 与上游 SM4_CBC_IV 一致
func TestSM4FixedIV(t *testing.T) {
	want := []byte{3, 5, 6, 9, 6, 9, 5, 9, 3, 5, 6, 9, 6, 9, 5, 9}
	for i := range want {
		if sm4FixedIV[i] != want[i] {
			t.Fatalf("IV[%d]=%d 期望 %d", i, sm4FixedIV[i], want[i])
		}
	}
}
