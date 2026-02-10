package data

import (
	"sync"
	"time"
)

// DB 数据库结构体
type DB struct {
	data  map[string]*Entry
	mu    sync.RWMutex
	index int // 数据库索引
}

// NewDB 创建新的数据库实例
func NewDB(index int) *DB {
	return &DB{
		data:  make(map[string]*Entry),
		index: index,
	}
}

// GetIndex 获取数据库索引
func (db *DB) GetIndex() int {
	return db.index
}

// Set 设置键值对
func (db *DB) Set(key string, value Data) {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.data[key] = &Entry{Data: value, ExpireTime: nil}
}

// SetWithExpire 设置键值对并指定过期时间
func (db *DB) SetWithExpire(key string, value Data, expire time.Duration) {
	db.mu.Lock()
	defer db.mu.Unlock()
	expireTime := time.Now().Add(expire)
	db.data[key] = &Entry{Data: value, ExpireTime: &expireTime}
}

// Get 获取键值对
func (db *DB) Get(key string) (Data, bool) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	entry, ok := db.data[key]
	if !ok {
		return nil, false
	}

	if entry.IsExpired() {
		return nil, false
	}

	return entry.Data, true
}

// GetEntry 获取完整的 Entry
func (db *DB) GetEntry(key string) (*Entry, bool) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	entry, ok := db.data[key]
	if !ok {
		return nil, false
	}

	if entry.IsExpired() {
		return nil, false
	}

	return entry, true
}

// Delete 删除键值对
func (db *DB) Delete(key string) bool {
	db.mu.Lock()
	defer db.mu.Unlock()

	_, ok := db.data[key]
	if ok {
		delete(db.data, key)
		return true
	}
	return false
}

// DeleteKeys 批量删除键值对，返回删除的数量
func (db *DB) DeleteKeys(keys []string) int {
	db.mu.Lock()
	defer db.mu.Unlock()

	count := 0
	for _, key := range keys {
		if _, ok := db.data[key]; ok {
			delete(db.data, key)
			count++
		}
	}
	return count
}

// Exists 检查键是否存在
func (db *DB) Exists(keys []string) int {
	db.mu.RLock()
	defer db.mu.RUnlock()

	count := 0
	for _, key := range keys {
		if entry, ok := db.data[key]; ok && !entry.IsExpired() {
			count++
		}
	}
	return count
}

// Expire 设置过期时间
func (db *DB) Expire(key string, duration time.Duration) bool {
	db.mu.Lock()
	defer db.mu.Unlock()

	entry, ok := db.data[key]
	if !ok || entry.IsExpired() {
		return false
	}

	expireTime := time.Now().Add(duration)
	entry.ExpireTime = &expireTime
	return true
}

// TTL 获取键的剩余生存时间
func (db *DB) TTL(key string) int64 {
	db.mu.RLock()
	defer db.mu.RUnlock()

	entry, ok := db.data[key]
	if !ok {
		return -2 // 键不存在
	}

	return entry.TTL()
}

// Persist 移除键的过期时间
func (db *DB) Persist(key string) bool {
	db.mu.Lock()
	defer db.mu.Unlock()

	entry, ok := db.data[key]
	if !ok || entry.IsExpired() {
		return false
	}

	entry.ExpireTime = nil
	return true
}

// Keys 获取所有键（支持简单的模式匹配）
func (db *DB) Keys(pattern string) []string {
	db.mu.RLock()
	defer db.mu.RUnlock()

	var keys []string
	for key, entry := range db.data {
		if entry.IsExpired() {
			continue
		}
		// 简单实现：支持 * 通配符
		if matchPattern(key, pattern) {
			keys = append(keys, key)
		}
	}
	return keys
}

// Type 获取键的类型
func (db *DB) Type(key string) string {
	db.mu.RLock()
	defer db.mu.RUnlock()

	entry, ok := db.data[key]
	if !ok || entry.IsExpired() {
		return "none"
	}

	switch entry.Data.Type() {
	case TypeString:
		return "string"
	case TypeList:
		return "list"
	case TypeHash:
		return "hash"
	case TypeSet:
		return "set"
	case TypeZSet:
		return "zset"
	default:
		return "unknown"
	}
}

// RandomKey 随机返回一个键
func (db *DB) RandomKey() string {
	db.mu.RLock()
	defer db.mu.RUnlock()

	for key, entry := range db.data {
		if !entry.IsExpired() {
			return key
		}
	}
	return ""
}

// DBSize 返回键的数量
func (db *DB) DBSize() int {
	db.mu.RLock()
	defer db.mu.RUnlock()

	count := 0
	for _, entry := range db.data {
		if !entry.IsExpired() {
			count++
		}
	}
	return count
}

// FlushDB 清空当前数据库
func (db *DB) FlushDB() {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.data = make(map[string]*Entry)
}

// GetExpiredKeys 获取所有已过期的键（用于清理）
func (db *DB) GetExpiredKeys() []string {
	db.mu.RLock()
	defer db.mu.RUnlock()

	var expired []string
	for key, entry := range db.data {
		if entry.IsExpired() {
			expired = append(expired, key)
		}
	}
	return expired
}

// matchPattern 简单的模式匹配
func matchPattern(key, pattern string) bool {
	if pattern == "*" {
		return true
	}
	if pattern == key {
		return true
	}
	// 简单的通配符匹配实现
	// 这里可以实现更复杂的匹配逻辑
	return false
}
