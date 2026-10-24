// Package store: 已保存连接的本地持久化（JSON 文件）。
// 计划 §3.2：连接条目使用控制台内部标识；IP 与备注均可变化，不凭备注合并设备。
package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"smsfm/server/internal/protocol"
)

// Entry 一条保存的连接
type Entry struct {
	ID         string          `json:"id"` // 内部标识，不随地址/备注变化
	Host       string          `json:"host"`
	Port       int             `json:"port"`
	SafetyMode protocol.SafetyMode `json:"safetyMode"`
	Secret     string          `json:"secret,omitempty"`
	Remark     string          `json:"remark,omitempty"`
	CreatedAt  time.Time       `json:"createdAt"`
	UpdatedAt  time.Time       `json:"updatedAt"`
}

// Store 文件持久化的连接仓库
type Store struct {
	mu      sync.RWMutex
	path    string
	next    int
	entries []Entry
}

// Open 打开（或创建）数据目录中的 connections.json
func Open(dataDir string) (*Store, error) {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, err
	}
	s := &Store{path: filepath.Join(dataDir, "connections.json")}
	return s, s.load()
}

type fileFormat struct {
	NextID int     `json:"nextId"`
	Items  []Entry `json:"items"`
}

func (s *Store) load() error {
	raw, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	var f fileFormat
	if err := json.Unmarshal(raw, &f); err != nil {
		return err
	}
	s.next = f.NextID
	s.entries = f.Items
	return nil
}

func (s *Store) persistLocked() error {
	f := fileFormat{NextID: s.next, Items: s.entries}
	raw, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

// List 全部连接（副本）
func (s *Store) List() []Entry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Entry, len(s.entries))
	copy(out, s.entries)
	return out
}

// Get 按 ID 查找
func (s *Store) Get(id string) (Entry, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, e := range s.entries {
		if e.ID == id {
			return e, true
		}
	}
	return Entry{}, false
}

// ConnOf 按 ID 取连接参数
func (s *Store) ConnOf(id string) (protocol.Conn, bool) {
	e, ok := s.Get(id)
	if !ok {
		return protocol.Conn{}, false
	}
	return protocol.Conn{Host: e.Host, Port: e.Port, SafetyMode: e.SafetyMode, Secret: e.Secret}, true
}

// FindByAddr 按地址找已保存连接（扫描时优先用其凭据）
func (s *Store) FindByAddr(host string, port int) (Entry, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, e := range s.entries {
		if e.Host == host && e.Port == port {
			return e, true
		}
	}
	return Entry{}, false
}

// Create 新建并保存（测试连接成功后调用）
func (s *Store) Create(e Entry) (Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.next++
	e.ID = fmt.Sprintf("c%d", s.next)
	now := time.Now()
	e.CreatedAt = now
	e.UpdatedAt = now
	s.entries = append(s.entries, e)
	return e, s.persistLocked()
}

// Update 更新（地址/凭据/备注可变；ID 不变）
func (s *Store) Update(id string, e Entry) (Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.entries {
		if s.entries[i].ID == id {
			e.ID = id
			e.CreatedAt = s.entries[i].CreatedAt
			e.UpdatedAt = time.Now()
			s.entries[i] = e
			return e, s.persistLocked()
		}
	}
	return Entry{}, os.ErrNotExist
}

// Delete 删除
func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.entries {
		if s.entries[i].ID == id {
			s.entries = append(s.entries[:i], s.entries[i+1:]...)
			return s.persistLocked()
		}
	}
	return os.ErrNotExist
}
