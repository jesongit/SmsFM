package auth

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

// totpCode 参照 RFC 6238 生成当前 6 位码（SHA1/6 位/30s，与库一致）
func totpCode(t *testing.T, secret string) string {
	t.Helper()
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secret)
	if err != nil {
		t.Fatalf("密钥解码失败：%v", err)
	}
	c := make([]byte, 8)
	binary.BigEndian.PutUint64(c, uint64(time.Now().Unix())/30)
	h := hmac.New(sha1.New, key)
	h.Write(c)
	sum := h.Sum(nil)
	o := sum[len(sum)-1] & 0xF
	code := (binary.BigEndian.Uint32(sum[o:o+4]) & 0x7FFFFFFF) % 1000000
	return fmt.Sprintf("%06d", code)
}

func TestLoginAndSession(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	// 错误密码
	if _, _, err := s.LoginStep1("admin", "wrong", false); err != ErrBadCredential {
		t.Fatalf("期望密码错误，得到 %v", err)
	}
	// 正确登录
	token, ticket, err := s.LoginStep1("admin", "admin", false)
	if err != nil || ticket != "" || token == "" {
		t.Fatalf("登录异常：token=%q ticket=%q err=%v", token, ticket, err)
	}
	if !s.Valid(token) {
		t.Fatal("token 应有效")
	}
	s.Logout(token)
	if s.Valid(token) {
		t.Fatal("退出后 token 应失效")
	}
}

func TestChangePasswordAndUsername(t *testing.T) {
	s, _ := Open(t.TempDir())
	if err := s.ChangePassword("bad", "newpass1"); err != ErrBadPassword {
		t.Fatalf("期望旧密码错误，得到 %v", err)
	}
	if err := s.ChangePassword("admin", "12345"); err == nil {
		t.Fatal("短密码应被拒绝")
	}
	if err := s.ChangePassword("admin", "newpass1"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.LoginStep1("admin", "admin", false); err != ErrBadCredential {
		t.Fatal("旧密码应失效")
	}
	if err := s.ChangeUsername("newpass1", "boss"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.LoginStep1("admin", "newpass1", false); err != ErrBadCredential {
		t.Fatal("旧用户名应失效")
	}
	if tk, _, err := s.LoginStep1("boss", "newpass1", false); err != nil || tk == "" {
		t.Fatalf("新凭据登录失败：%v", err)
	}
	_, totp, isDefault := s.Me()
	if totp || isDefault {
		t.Fatal("改密后 isDefault 应清除")
	}
}

func TestTOTPFlow(t *testing.T) {
	s, _ := Open(t.TempDir())
	secret, uri, err := s.TOTPSetup()
	if err != nil || secret == "" || uri == "" {
		t.Fatalf("setup 异常：%v", err)
	}
	// 错误码不能启用
	if err := s.TOTPEnable(secret, "000000"); err == nil {
		t.Fatal("错误验证码不应启用")
	}
	if err := s.TOTPEnable(secret, totpCode(t, secret)); err != nil {
		t.Fatalf("正确验证码启用失败：%v", err)
	}
	// 登录走 2FA 分支
	_, ticket, err := s.LoginStep1("admin", "admin", false)
	if err != nil || ticket == "" {
		t.Fatalf("应返回 2FA 票据：%v", err)
	}
	if _, err := s.LoginStep2(ticket, "000000", false); err == nil {
		t.Fatal("错误 2FA 码应被拒")
	}
	// 票据一次性：上一步失败后已消耗
	if _, err := s.LoginStep2(ticket, totpCode(t, secret), false); err == nil {
		t.Fatal("票据应一次性")
	}
	_, ticket2, _ := s.LoginStep1("admin", "admin", false)
	tk, err := s.LoginStep2(ticket2, totpCode(t, secret), false)
	if err != nil || tk == "" {
		t.Fatalf("2FA 登录失败：%v", err)
	}
	// 关闭
	if err := s.TOTPDisable("admin"); err != nil {
		t.Fatal(err)
	}
	tk2, ticket3, _ := s.LoginStep1("admin", "admin", false)
	if ticket3 != "" || tk2 == "" {
		t.Fatal("关闭 2FA 后应直接登录")
	}
}

func TestLockAfterFails(t *testing.T) {
	s, _ := Open(t.TempDir())
	for i := 0; i < 5; i++ {
		_, _, _ = s.LoginStep1("admin", "bad", false)
	}
	if _, _, err := s.LoginStep1("admin", "admin", false); err != ErrLocked {
		t.Fatalf("连续失败后应锁定，得到 %v", err)
	}
}

func TestPersistAcrossReopen(t *testing.T) {
	dir := t.TempDir()
	s, _ := Open(dir)
	_ = s.ChangePassword("admin", "newpass1")
	// 重新打开（模拟重启）：账号保留，session 清空
	s2, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := s2.LoginStep1("admin", "newpass1", false); err != nil {
		t.Fatalf("重启后新密码应有效：%v", err)
	}
	raw, _ := json.Marshal(s2)
	if len(raw) == 0 {
		t.Fatal("状态序列化异常")
	}
}
