package data

import (
	"sync"
)

// Redis 服务器结构体
type Redis struct {
	dbs       []*DB
	dbCount   int
	mu        sync.RWMutex
	currentDB int // 默认选择的数据库
}

// NewRedis 创建新的 Redis 服务器实例
func NewRedis(dbCount int) *Redis {
	if dbCount <= 0 {
		dbCount = 16 // 默认16个数据库
	}

	redis := &Redis{
		dbs:       make([]*DB, dbCount),
		dbCount:   dbCount,
		currentDB: 0,
	}

	for i := 0; i < dbCount; i++ {
		redis.dbs[i] = NewDB(i)
	}

	return redis
}

// SelectDB 选择数据库
func (r *Redis) SelectDB(index int) (*DB, bool) {
	if index < 0 || index >= r.dbCount {
		return nil, false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.currentDB = index
	return r.dbs[index], true
}

// GetDB 获取指定索引的数据库
func (r *Redis) GetDB(index int) (*DB, bool) {
	if index < 0 || index >= r.dbCount {
		return nil, false
	}
	return r.dbs[index], true
}

// GetCurrentDB 获取当前选择的数据库
func (r *Redis) GetCurrentDB() *DB {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.dbs[r.currentDB]
}

// GetCurrentDBIndex 获取当前数据库索引
func (r *Redis) GetCurrentDBIndex() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.currentDB
}

// FlushAll 清空所有数据库
func (r *Redis) FlushAll() {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, db := range r.dbs {
		db.FlushDB()
	}
}

// GetDBCount 获取数据库数量
func (r *Redis) GetDBCount() int {
	return r.dbCount
}
