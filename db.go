package main

import "sync"

// DB 结构体用来存储所有的 key-value
type DB struct {
	data sync.Map
}

// NewDB 创建一个新的数据库实例
func NewDB() *DB {
	return &DB{}
}

// Set 存储数据
func (db *DB) Set(key string, value string) {
	db.data.Store(key, value)
}

// Get 获取数据
func (db *DB) Get(key string) (string, bool) {
	val, ok := db.data.Load(key)
	if !ok {
		return "", false
	}
	// 这里又是类型断言，把 any 转回 string
	return val.(string), true
}
