package data

import (
	"time"
)

// DataType 定义数据类型
type DataType byte

const (
	TypeString DataType = iota
	TypeList
	TypeHash
	TypeSet
	TypeZSet
)

// Data 接口，所有数据类型都需要实现
type Data interface {
	Type() DataType
}

// String 类型
type String struct {
	Value string
}

func (s *String) Type() DataType { return TypeString }

// List 类型 (双向链表)
type List struct {
	Items []string
}

func (l *List) Type() DataType { return TypeList }

// Hash 类型 (field-value 映射)
type Hash struct {
	Fields map[string]string
}

func (h *Hash) Type() DataType { return TypeHash }

// Set 类型 (无序唯一集合)
type Set struct {
	Members map[string]struct{}
}

func (s *Set) Type() DataType { return TypeSet }

// ZSetItem 有序集合项
type ZSetItem struct {
	Member string
	Score  float64
}

// ZSet 类型 (有序集合)
type ZSet struct {
	Members map[string]float64 // member -> score
}

func (z *ZSet) Type() DataType { return TypeZSet }

// Entry 存储在数据库中的条目
type Entry struct {
	Data       Data
	ExpireTime *time.Time // nil 表示永不过期
}

// IsExpired 检查条目是否已过期
func (e *Entry) IsExpired() bool {
	if e.ExpireTime == nil {
		return false
	}
	return time.Now().After(*e.ExpireTime)
}

// TTL 获取剩余生存时间（秒），-1 表示永不过期，-2 表示已过期
func (e *Entry) TTL() int64 {
	if e.ExpireTime == nil {
		return -1
	}
	if e.IsExpired() {
		return -2
	}
	return int64(e.ExpireTime.Sub(time.Now()).Seconds())
}
