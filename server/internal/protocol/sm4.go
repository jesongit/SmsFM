package protocol

import (
	"encoding/hex"
	"errors"
	"strings"

	"github.com/tjfoc/gmsm/sm4"
)

// 上游 SM4Crypt：SM4/CBC/PKCS7Padding，固定 IV（SM4_CBC_IV 字节数组）。
var sm4FixedIV = []byte{3, 5, 6, 9, 6, 9, 5, 9, 3, 5, 6, 9, 6, 9, 5, 9}

// SM4DecryptHex 解开手机响应：hex（大小写均可）→ SM4/CBC/PKCS7 解密。
func SM4DecryptHex(input string, key []byte) ([]byte, error) {
	if len(key) != 16 {
		return nil, errors.New("SM4 密钥须为 16 字节（32 位十六进制）")
	}
	raw, err := hex.DecodeString(stripWhitespace(input))
	if err != nil {
		return nil, err
	}
	plain, err := sm4DecryptCBC(raw, key)
	if err != nil {
		return nil, err
	}
	return plain, nil
}

// SM4EncryptHex 加密请求并输出十六进制（xUtil bytes2HexString 为大写）。
func SM4EncryptHex(input []byte, key []byte) (string, error) {
	if len(key) != 16 {
		return "", errors.New("SM4 密钥须为 16 字节（32 位十六进制）")
	}
	out, err := sm4EncryptCBC(input, key)
	if err != nil {
		return "", err
	}
	// 上游 xUtil bytes2HexString 输出大写；手机端解码大小写不敏感，但保持字节级一致
	return strings.ToUpper(hex.EncodeToString(out)), nil
}

func sm4EncryptCBC(plain, key []byte) ([]byte, error) {
	padded := pkcs7Pad(plain, sm4.BlockSize)
	block, err := sm4.NewCipher(key)
	if err != nil {
		return nil, err
	}
	out := make([]byte, len(padded))
	prev := append([]byte(nil), sm4FixedIV...)
	buf := make([]byte, sm4.BlockSize)
	for off := 0; off < len(padded); off += sm4.BlockSize {
		for i := 0; i < sm4.BlockSize; i++ {
			buf[i] = padded[off+i] ^ prev[i]
		}
		block.Encrypt(out[off:off+sm4.BlockSize], buf)
		copy(prev, out[off:off+sm4.BlockSize])
	}
	return out, nil
}

func sm4DecryptCBC(cipherText, key []byte) ([]byte, error) {
	if len(cipherText) == 0 || len(cipherText)%sm4.BlockSize != 0 {
		return nil, errors.New("SM4 密文长度非 16 字节对齐")
	}
	block, err := sm4.NewCipher(key)
	if err != nil {
		return nil, err
	}
	out := make([]byte, len(cipherText))
	prev := append([]byte(nil), sm4FixedIV...)
	buf := make([]byte, sm4.BlockSize)
	for off := 0; off < len(cipherText); off += sm4.BlockSize {
		block.Decrypt(buf, cipherText[off:off+sm4.BlockSize])
		for i := 0; i < sm4.BlockSize; i++ {
			out[off+i] = buf[i] ^ prev[i]
		}
		copy(prev, cipherText[off:off+sm4.BlockSize])
	}
	return pkcs7Unpad(out, sm4.BlockSize)
}

func pkcs7Pad(data []byte, size int) []byte {
	pad := size - len(data)%size
	out := make([]byte, len(data)+pad)
	copy(out, data)
	for i := len(data); i < len(out); i++ {
		out[i] = byte(pad)
	}
	return out
}

func pkcs7Unpad(data []byte, size int) ([]byte, error) {
	if len(data) == 0 || len(data)%size != 0 {
		return nil, errors.New("填充长度非法")
	}
	pad := int(data[len(data)-1])
	if pad == 0 || pad > size || pad > len(data) {
		return nil, errors.New("填充内容非法")
	}
	for _, b := range data[len(data)-pad:] {
		if int(b) != pad {
			return nil, errors.New("填充内容非法")
		}
	}
	return data[:len(data)-pad], nil
}

func stripWhitespace(s string) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c != '\n' && c != '\r' && c != ' ' && c != '\t' {
			out = append(out, c)
		}
	}
	return string(out)
}
