package protocol

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"math/big"
)

// 上游 RSACrypt（RSA/ECB/PKCS1Padding，2048 位）的客户端侧实现。
// 请求：分段（≤245 字节）公钥加密（块类型 2，PKCS1 v1.5）→ Base64。
// 响应：手机用私钥加密（块类型 1），本端公钥解密，256 字节分段。
// Go 标准库不提供"公钥解密"，块类型 1 的解开在此手工实现。

const (
	rsaEncryptBlock = 245 // 2048/8 - 11
	rsaDecryptBlock = 256
)

// ParseRSAPublicKey 接受 X.509 SubjectPublicKeyInfo 的 Base64（上游 getPublicKey）或 PEM。
func ParseRSAPublicKey(key string) (*rsa.PublicKey, error) {
	raw, err := base64.StdEncoding.DecodeString(stripPEM(key))
	if err != nil {
		return nil, err
	}
	if pub, err := x509.ParsePKIXPublicKey(raw); err == nil {
		if p, ok := pub.(*rsa.PublicKey); ok {
			return p, nil
		}
		return nil, errors.New("非 RSA 公钥")
	}
	return x509.ParsePKCS1PublicKey(raw)
}

func stripPEM(key string) string {
	if len(key) > 40 && key[:10] == "-----BEGIN" {
		// 找 PEM 体内的 Base64
		start := 0
		for i := 0; i < len(key); i++ {
			if key[i] == '\n' {
				start = i + 1
				break
			}
		}
		end := len(key)
		for i := start; i < len(key); i++ {
			if key[i] == '-' && i+4 <= len(key) && key[i:i+4] == "----" {
				end = i
				break
			}
		}
		return replaceWhitespace(key[start:end])
	}
	return replaceWhitespace(key)
}

func replaceWhitespace(s string) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c != '\n' && c != '\r' && c != ' ' && c != '\t' {
			out = append(out, c)
		}
	}
	return string(out)
}

// RSAEncryptByPublicKey 分段公钥加密，返回 Base64（与上游 encryptByPublicKey 输出一致）。
func RSAEncryptByPublicKey(input []byte, pub *rsa.PublicKey) (string, error) {
	k := (pub.N.BitLen() + 7) / 8
	maxBlock := k - 11 // 2048 位 → 245
	var out []byte
	for offset := 0; offset < len(input); offset += maxBlock {
		end := offset + maxBlock
		if end > len(input) {
			end = len(input)
		}
		block, err := rsa.EncryptPKCS1v15(rand.Reader, pub, input[offset:end])
		if err != nil {
			return "", err
		}
		out = append(out, block...)
	}
	return base64.StdEncoding.EncodeToString(out), nil
}

// RSADecryptByPublicKey 公钥解开"私钥加密（块类型 1）"的密文，分段 256 字节。
// 返回明文字符串（上游 decryptByPublicKey 的返回）。
func RSADecryptByPublicKey(inputB64 string, pub *rsa.PublicKey) (string, error) {
	cipherAll, err := base64.StdEncoding.DecodeString(stripPEM(inputB64))
	if err != nil {
		return "", err
	}
	k := (pub.N.BitLen() + 7) / 8
	var out []byte
	for offset := 0; offset < len(cipherAll); offset += k {
		end := offset + k
		if end > len(cipherAll) {
			end = len(cipherAll)
		}
		block := cipherAll[offset:end]
		if len(block) != k {
			return "", errors.New("RSA 密文块长度不符")
		}
		c := new(big.Int).SetBytes(block)
		if c.Cmp(pub.N) >= 0 {
			return "", errors.New("RSA 密文超出模数")
		}
		m := new(big.Int).Exp(c, big.NewInt(int64(pub.E)), pub.N)
		em := make([]byte, k)
		m.FillBytes(em)
		plain, err := unpadPKCS1BlockType1(em)
		if err != nil {
			return "", err
		}
		out = append(out, plain...)
	}
	return string(out), nil
}

// unpadPKCS1BlockType1 撤销签名式填充：00 01 FF..FF 00 || message
func unpadPKCS1BlockType1(em []byte) ([]byte, error) {
	if len(em) < 11 || em[0] != 0x00 || em[1] != 0x01 {
		return nil, errors.New("RSA 块类型 1 填充不符")
	}
	i := 2
	for i < len(em) && em[i] == 0xFF {
		i++
	}
	if i == len(em) || em[i] != 0x00 || i < 10 {
		return nil, errors.New("RSA 填充分隔符不符")
	}
	return em[i+1:], nil
}
