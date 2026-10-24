// Package auth: 单管理员账号认证。
// bcrypt 密码哈希、内存 session（重启失效，重新登录即可）、TOTP 两步验证
// （Google Authenticator 兼容，RFC 6238）、登录失败限速。
package auth

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"
)

const (
	sessionTTL    = 12 * time.Hour      // 未勾选记住：关浏览器级会话
	sessionTTLRem = 30 * 24 * time.Hour // 记住我：30 天
	failLockMax   = 5                   // 连续失败次数
	failLockTime  = time.Minute         // 锁定时长
	totpIssuer    = "SmsFM"
)

// User 持久化的账号信息（data/auth.json）
type User struct {
	Username     string `json:"username"`
	PasswordHash string `json:"passwordHash"`
	TOTPSecret   string `json:"totpSecret,omitempty"`
	TOTPEnabled  bool   `json:"totpEnabled"`
	IsDefault    bool   `json:"isDefault"` // 仍为默认密码 admin/admin
}

// Store 账号 + 会话
type Store struct {
	mu       sync.Mutex
	path     string
	user     User
	sessions map[string]time.Time // token -> 过期时间
	failCnt  int
	lockTill time.Time
	// 待确认的 TOTP 绑定（setup 生成，enable 验证后生效）
	pendingTOTP map[string]time.Time // secret -> 创建时间
}

// Open 打开（或初始化默认 admin/admin）
func Open(dataDir string) (*Store, error) {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, err
	}
	s := &Store{
		path:        filepath.Join(dataDir, "auth.json"),
		sessions:    map[string]time.Time{},
		pendingTOTP: map[string]time.Time{},
	}
	raw, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		hash, _ := bcrypt.GenerateFromPassword([]byte("admin"), bcrypt.DefaultCost)
		s.user = User{Username: "admin", PasswordHash: string(hash), IsDefault: true}
		return s, s.persist()
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(raw, &s.user); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) persist() error {
	raw, err := json.MarshalIndent(s.user, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

/* ── 登录 ── */

var (
	ErrLocked        = errors.New("失败次数过多，账号已临时锁定，请稍后再试")
	ErrBadCredential = errors.New("用户名或密码错误")
	ErrNeed2FA       = errors.New("需要两步验证码")
	ErrBad2FA        = errors.New("两步验证码错误")
	ErrBadTicket     = errors.New("登录会话已过期，请重新登录")
)

// LoginStep1 校验用户名密码；启用 2FA 时返回待验证票据（第二步用），否则直接签发 token。
// remember=true 时会话 30 天，否则 12 小时。
func (s *Store) LoginStep1(username, password string, remember bool) (token string, ticket string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if time.Now().Before(s.lockTill) {
		return "", "", ErrLocked
	}
	if bcrypt.CompareHashAndPassword([]byte(s.user.PasswordHash), []byte(password)) != nil || username != s.user.Username {
		s.failCnt++
		if s.failCnt >= failLockMax {
			s.lockTill = time.Now().Add(failLockTime)
			s.failCnt = 0
		}
		return "", "", ErrBadCredential
	}
	s.failCnt = 0
	if s.user.TOTPEnabled {
		tk, err := randomToken()
		if err != nil {
			return "", "", err
		}
		// 票据也放 sessions，2 分钟有效
		s.sessions["2fa:"+tk] = time.Now().Add(2 * time.Minute)
		return "", tk, nil
	}
	tk, err := randomToken()
	if err != nil {
		return "", "", err
	}
	s.issue(tk, remember)
	return tk, "", nil
}

func (s *Store) issue(token string, remember bool) {
	ttl := sessionTTL
	if remember {
		ttl = sessionTTLRem
	}
	s.sessions[token] = time.Now().Add(ttl)
}

// LoginStep2 校验 TOTP 码，换取正式 token
func (s *Store) LoginStep2(ticket, code string, remember bool) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	exp, ok := s.sessions["2fa:"+ticket]
	if !ok || time.Now().After(exp) {
		return "", ErrBadTicket
	}
	delete(s.sessions, "2fa:"+ticket)
	if !totp.Validate(code, s.user.TOTPSecret) {
		return "", ErrBad2FA
	}
	tk, err := randomToken()
	if err != nil {
		return "", err
	}
	s.issue(tk, remember)
	return tk, nil
}

// Logout 作废 token
func (s *Store) Logout(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, token)
}

// Valid token 是否有效（顺带清理过期）
func (s *Store) Valid(token string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for t, exp := range s.sessions {
		if now.After(exp) {
			delete(s.sessions, t)
		}
	}
	exp, ok := s.sessions[token]
	return ok && now.Before(exp)
}

func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

/* ── 账号管理 ── */

// Me 当前状态（不含敏感字段）
func (s *Store) Me() (username string, totpEnabled bool, isDefault bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.user.Username, s.user.TOTPEnabled, s.user.IsDefault
}

// ErrBadPassword 旧密码错误
var ErrBadPassword = errors.New("当前密码错误")

// ChangePassword 修改密码（需当前密码）
func (s *Store) ChangePassword(oldPwd, newPwd string) error {
	if len(newPwd) < 6 {
		return errors.New("新密码至少 6 位")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if bcrypt.CompareHashAndPassword([]byte(s.user.PasswordHash), []byte(oldPwd)) != nil {
		return ErrBadPassword
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPwd), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	s.user.PasswordHash = string(hash)
	s.user.IsDefault = false
	return s.persist()
}

// ChangeUsername 修改用户名（需密码）
func (s *Store) ChangeUsername(password, newUsername string) error {
	if newUsername == "" || len(newUsername) > 32 {
		return errors.New("用户名须为 1-32 个字符")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if bcrypt.CompareHashAndPassword([]byte(s.user.PasswordHash), []byte(password)) != nil {
		return ErrBadPassword
	}
	s.user.Username = newUsername
	s.user.IsDefault = false
	return s.persist()
}

/* ── TOTP ── */

// TOTPSetup 生成待绑定密钥（otpauth URI 供二维码/手动输入）
func (s *Store) TOTPSetup() (secret string, uri string, err error) {
	key, err := totp.Generate(totp.GenerateOpts{Issuer: totpIssuer, AccountName: s.username()})
	if err != nil {
		return "", "", err
	}
	s.mu.Lock()
	s.pendingTOTP[key.Secret()] = time.Now().Add(10 * time.Minute)
	// 只保留最新的一个待绑定密钥
	for k, t := range s.pendingTOTP {
		if k != key.Secret() && time.Now().After(t) {
			delete(s.pendingTOTP, k)
		}
	}
	s.mu.Unlock()
	return key.Secret(), key.URL(), nil
}

// TOTPEnable 校验一次验证码后启用
func (s *Store) TOTPEnable(secret, code string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	created, ok := s.pendingTOTP[secret]
	if !ok || time.Now().After(created.Add(10*time.Minute)) {
		return errors.New("绑定会话过期，请重新生成密钥")
	}
	if !totp.Validate(code, secret) {
		return ErrBad2FA
	}
	delete(s.pendingTOTP, secret)
	s.user.TOTPSecret = secret
	s.user.TOTPEnabled = true
	return s.persist()
}

// TOTPDisable 关闭两步验证（需密码）
func (s *Store) TOTPDisable(password string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if bcrypt.CompareHashAndPassword([]byte(s.user.PasswordHash), []byte(password)) != nil {
		return ErrBadPassword
	}
	s.user.TOTPSecret = ""
	s.user.TOTPEnabled = false
	return s.persist()
}

func (s *Store) username() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.user.Username
}
