package command

import (
	"go-redis/data"
	"go-redis/resp"
	"strconv"
)

// LPushCommand 将一个或多个值插入到列表头部
type LPushCommand struct{}

func (c *LPushCommand) Name() string  { return "LPUSH" }
func (c *LPushCommand) ArgCount() int { return 2 }
func (c *LPushCommand) Execute(db *data.DB, args []string, writer *resp.Writer) error {
	key := args[0]
	values := args[1:]

	var list *data.List
	existing, ok := db.Get(key)
	if ok {
		var typeOk bool
		list, typeOk = existing.(*data.List)
		if !typeOk {
			return writer.WriteError("WRONGTYPE Operation against a key holding the wrong kind of value")
		}
	} else {
		list = &data.List{Items: []string{}}
	}

	// 在头部插入所有值（逆序插入，使第一个值在最前面）
	for i := len(values) - 1; i >= 0; i-- {
		list.Items = append([]string{values[i]}, list.Items...)
	}

	db.Set(key, list)
	return writer.WriteInteger(int64(len(list.Items)))
}

// RPushCommand 将一个或多个值插入到列表尾部
type RPushCommand struct{}

func (c *RPushCommand) Name() string  { return "RPUSH" }
func (c *RPushCommand) ArgCount() int { return 2 }
func (c *RPushCommand) Execute(db *data.DB, args []string, writer *resp.Writer) error {
	key := args[0]
	values := args[1:]

	var list *data.List
	existing, ok := db.Get(key)
	if ok {
		var typeOk bool
		list, typeOk = existing.(*data.List)
		if !typeOk {
			return writer.WriteError("WRONGTYPE Operation against a key holding the wrong kind of value")
		}
	} else {
		list = &data.List{Items: []string{}}
	}

	list.Items = append(list.Items, values...)
	db.Set(key, list)
	return writer.WriteInteger(int64(len(list.Items)))
}

// LPopCommand 移除并返回列表的第一个元素
type LPopCommand struct{}

func (c *LPopCommand) Name() string  { return "LPOP" }
func (c *LPopCommand) ArgCount() int { return 1 }
func (c *LPopCommand) Execute(db *data.DB, args []string, writer *resp.Writer) error {
	key := args[0]

	existing, ok := db.Get(key)
	if !ok {
		return writer.WriteNullBulkString()
	}

	list, ok := existing.(*data.List)
	if !ok {
		return writer.WriteError("WRONGTYPE Operation against a key holding the wrong kind of value")
	}

	if len(list.Items) == 0 {
		db.Delete(key)
		return writer.WriteNullBulkString()
	}

	value := list.Items[0]
	list.Items = list.Items[1:]

	if len(list.Items) == 0 {
		db.Delete(key)
	} else {
		db.Set(key, list)
	}

	return writer.WriteBulkString(value)
}

// RPopCommand 移除并返回列表的最后一个元素
type RPopCommand struct{}

func (c *RPopCommand) Name() string  { return "RPOP" }
func (c *RPopCommand) ArgCount() int { return 1 }
func (c *RPopCommand) Execute(db *data.DB, args []string, writer *resp.Writer) error {
	key := args[0]

	existing, ok := db.Get(key)
	if !ok {
		return writer.WriteNullBulkString()
	}

	list, ok := existing.(*data.List)
	if !ok {
		return writer.WriteError("WRONGTYPE Operation against a key holding the wrong kind of value")
	}

	if len(list.Items) == 0 {
		db.Delete(key)
		return writer.WriteNullBulkString()
	}

	value := list.Items[len(list.Items)-1]
	list.Items = list.Items[:len(list.Items)-1]

	if len(list.Items) == 0 {
		db.Delete(key)
	} else {
		db.Set(key, list)
	}

	return writer.WriteBulkString(value)
}

// LLenCommand 获取列表长度
type LLenCommand struct{}

func (c *LLenCommand) Name() string  { return "LLEN" }
func (c *LLenCommand) ArgCount() int { return 1 }
func (c *LLenCommand) Execute(db *data.DB, args []string, writer *resp.Writer) error {
	existing, ok := db.Get(args[0])
	if !ok {
		return writer.WriteInteger(0)
	}

	list, ok := existing.(*data.List)
	if !ok {
		return writer.WriteError("WRONGTYPE Operation against a key holding the wrong kind of value")
	}

	return writer.WriteInteger(int64(len(list.Items)))
}

// LRangeCommand 获取列表指定范围内的元素
type LRangeCommand struct{}

func (c *LRangeCommand) Name() string  { return "LRANGE" }
func (c *LRangeCommand) ArgCount() int { return 3 }
func (c *LRangeCommand) Execute(db *data.DB, args []string, writer *resp.Writer) error {
	key, startStr, stopStr := args[0], args[1], args[2]

	start, err := strconv.Atoi(startStr)
	if err != nil {
		return writer.WriteError("ERR value is not an integer or out of range")
	}

	stop, err := strconv.Atoi(stopStr)
	if err != nil {
		return writer.WriteError("ERR value is not an integer or out of range")
	}

	existing, ok := db.Get(key)
	if !ok {
		return writer.WriteStringArray([]string{})
	}

	list, ok := existing.(*data.List)
	if !ok {
		return writer.WriteError("WRONGTYPE Operation against a key holding the wrong kind of value")
	}

	length := len(list.Items)
	if length == 0 {
		return writer.WriteStringArray([]string{})
	}

	// 处理负数索引
	if start < 0 {
		start = length + start
	}
	if stop < 0 {
		stop = length + stop
	}

	// 边界检查
	if start < 0 {
		start = 0
	}
	if stop >= length {
		stop = length - 1
	}
	if start > stop {
		return writer.WriteStringArray([]string{})
	}

	result := list.Items[start : stop+1]
	return writer.WriteStringArray(result)
}

// LIndexCommand 通过索引获取列表中的元素
type LIndexCommand struct{}

func (c *LIndexCommand) Name() string  { return "LINDEX" }
func (c *LIndexCommand) ArgCount() int { return 2 }
func (c *LIndexCommand) Execute(db *data.DB, args []string, writer *resp.Writer) error {
	key, indexStr := args[0], args[1]

	index, err := strconv.Atoi(indexStr)
	if err != nil {
		return writer.WriteError("ERR value is not an integer or out of range")
	}

	existing, ok := db.Get(key)
	if !ok {
		return writer.WriteNullBulkString()
	}

	list, ok := existing.(*data.List)
	if !ok {
		return writer.WriteError("WRONGTYPE Operation against a key holding the wrong kind of value")
	}

	length := len(list.Items)
	if index < 0 {
		index = length + index
	}

	if index < 0 || index >= length {
		return writer.WriteNullBulkString()
	}

	return writer.WriteBulkString(list.Items[index])
}

// LRemCommand 移除列表元素
type LRemCommand struct{}

func (c *LRemCommand) Name() string  { return "LREM" }
func (c *LRemCommand) ArgCount() int { return 3 }
func (c *LRemCommand) Execute(db *data.DB, args []string, writer *resp.Writer) error {
	key, countStr, value := args[0], args[1], args[2]

	count, err := strconv.Atoi(countStr)
	if err != nil {
		return writer.WriteError("ERR value is not an integer or out of range")
	}

	existing, ok := db.Get(key)
	if !ok {
		return writer.WriteInteger(0)
	}

	list, ok := existing.(*data.List)
	if !ok {
		return writer.WriteError("WRONGTYPE Operation against a key holding the wrong kind of value")
	}

	removed := 0
	if count == 0 {
		// 移除所有匹配的元素
		newItems := make([]string, 0, len(list.Items))
		for _, item := range list.Items {
			if item != value {
				newItems = append(newItems, item)
			} else {
				removed++
			}
		}
		list.Items = newItems
	} else if count > 0 {
		// 从头部开始移除 count 个
		newItems := make([]string, 0, len(list.Items))
		removedCount := 0
		for _, item := range list.Items {
			if item == value && removedCount < count {
				removedCount++
				removed++
			} else {
				newItems = append(newItems, item)
			}
		}
		list.Items = newItems
	} else {
		// 从尾部开始移除 |count| 个
		count = -count
		newItems := make([]string, 0, len(list.Items))
		removedCount := 0
		for i := len(list.Items) - 1; i >= 0; i-- {
			if list.Items[i] == value && removedCount < count {
				removedCount++
				removed++
			} else {
				newItems = append([]string{list.Items[i]}, newItems...)
			}
		}
		list.Items = newItems
	}

	if len(list.Items) == 0 {
		db.Delete(key)
	} else {
		db.Set(key, list)
	}

	return writer.WriteInteger(int64(removed))
}

// LSetCommand 通过索引设置列表元素的值
type LSetCommand struct{}

func (c *LSetCommand) Name() string  { return "LSET" }
func (c *LSetCommand) ArgCount() int { return 3 }
func (c *LSetCommand) Execute(db *data.DB, args []string, writer *resp.Writer) error {
	key, indexStr, value := args[0], args[1], args[2]

	index, err := strconv.Atoi(indexStr)
	if err != nil {
		return writer.WriteError("ERR value is not an integer or out of range")
	}

	existing, ok := db.Get(key)
	if !ok {
		return writer.WriteError("ERR no such key")
	}

	list, ok := existing.(*data.List)
	if !ok {
		return writer.WriteError("WRONGTYPE Operation against a key holding the wrong kind of value")
	}

	length := len(list.Items)
	if index < 0 {
		index = length + index
	}

	if index < 0 || index >= length {
		return writer.WriteError("ERR index out of range")
	}

	list.Items[index] = value
	db.Set(key, list)

	return writer.WriteOK()
}
