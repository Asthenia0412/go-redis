package command

import (
	"go-redis/data"
	"go-redis/resp"
	"strconv"
)

// HSetCommand 设置哈希表字段的值
type HSetCommand struct{}

func (c *HSetCommand) Name() string  { return "HSET" }
func (c *HSetCommand) ArgCount() int { return 3 }
func (c *HSetCommand) Execute(db *data.DB, args []string, writer *resp.Writer) error {
	key, field, value := args[0], args[1], args[2]

	var hash *data.Hash
	existing, ok := db.Get(key)
	if ok {
		var typeOk bool
		hash, typeOk = existing.(*data.Hash)
		if !typeOk {
			return writer.WriteError("WRONGTYPE Operation against a key holding the wrong kind of value")
		}
	} else {
		hash = &data.Hash{Fields: make(map[string]string)}
	}

	// 检查字段是否已存在
	_, existed := hash.Fields[field]
	hash.Fields[field] = value
	db.Set(key, hash)

	if existed {
		return writer.WriteInteger(0)
	}
	return writer.WriteInteger(1)
}

// HGetCommand 获取哈希表字段的值
type HGetCommand struct{}

func (c *HGetCommand) Name() string  { return "HGET" }
func (c *HGetCommand) ArgCount() int { return 2 }
func (c *HGetCommand) Execute(db *data.DB, args []string, writer *resp.Writer) error {
	key, field := args[0], args[1]

	existing, ok := db.Get(key)
	if !ok {
		return writer.WriteNullBulkString()
	}

	hash, ok := existing.(*data.Hash)
	if !ok {
		return writer.WriteError("WRONGTYPE Operation against a key holding the wrong kind of value")
	}

	value, ok := hash.Fields[field]
	if !ok {
		return writer.WriteNullBulkString()
	}

	return writer.WriteBulkString(value)
}

// HDelCommand 删除哈希表字段
type HDelCommand struct{}

func (c *HDelCommand) Name() string  { return "HDEL" }
func (c *HDelCommand) ArgCount() int { return 2 }
func (c *HDelCommand) Execute(db *data.DB, args []string, writer *resp.Writer) error {
	key := args[0]
	fields := args[1:]

	existing, ok := db.Get(key)
	if !ok {
		return writer.WriteInteger(0)
	}

	hash, ok := existing.(*data.Hash)
	if !ok {
		return writer.WriteError("WRONGTYPE Operation against a key holding the wrong kind of value")
	}

	deleted := 0
	for _, field := range fields {
		if _, ok := hash.Fields[field]; ok {
			delete(hash.Fields, field)
			deleted++
		}
	}

	if len(hash.Fields) == 0 {
		db.Delete(key)
	} else {
		db.Set(key, hash)
	}

	return writer.WriteInteger(int64(deleted))
}

// HExistsCommand 检查哈希表字段是否存在
type HExistsCommand struct{}

func (c *HExistsCommand) Name() string  { return "HEXISTS" }
func (c *HExistsCommand) ArgCount() int { return 2 }
func (c *HExistsCommand) Execute(db *data.DB, args []string, writer *resp.Writer) error {
	key, field := args[0], args[1]

	existing, ok := db.Get(key)
	if !ok {
		return writer.WriteInteger(0)
	}

	hash, ok := existing.(*data.Hash)
	if !ok {
		return writer.WriteError("WRONGTYPE Operation against a key holding the wrong kind of value")
	}

	if _, ok := hash.Fields[field]; ok {
		return writer.WriteInteger(1)
	}
	return writer.WriteInteger(0)
}

// HLenCommand 获取哈希表字段数量
type HLenCommand struct{}

func (c *HLenCommand) Name() string  { return "HLEN" }
func (c *HLenCommand) ArgCount() int { return 1 }
func (c *HLenCommand) Execute(db *data.DB, args []string, writer *resp.Writer) error {
	existing, ok := db.Get(args[0])
	if !ok {
		return writer.WriteInteger(0)
	}

	hash, ok := existing.(*data.Hash)
	if !ok {
		return writer.WriteError("WRONGTYPE Operation against a key holding the wrong kind of value")
	}

	return writer.WriteInteger(int64(len(hash.Fields)))
}

// HGetAllCommand 获取哈希表所有字段和值
type HGetAllCommand struct{}

func (c *HGetAllCommand) Name() string  { return "HGETALL" }
func (c *HGetAllCommand) ArgCount() int { return 1 }
func (c *HGetAllCommand) Execute(db *data.DB, args []string, writer *resp.Writer) error {
	existing, ok := db.Get(args[0])
	if !ok {
		return writer.WriteStringArray([]string{})
	}

	hash, ok := existing.(*data.Hash)
	if !ok {
		return writer.WriteError("WRONGTYPE Operation against a key holding the wrong kind of value")
	}

	// 返回 field value field value ... 的数组
	result := make([]string, 0, len(hash.Fields)*2)
	for field, value := range hash.Fields {
		result = append(result, field, value)
	}
	return writer.WriteStringArray(result)
}

// HKeysCommand 获取哈希表所有字段
type HKeysCommand struct{}

func (c *HKeysCommand) Name() string  { return "HKEYS" }
func (c *HKeysCommand) ArgCount() int { return 1 }
func (c *HKeysCommand) Execute(db *data.DB, args []string, writer *resp.Writer) error {
	existing, ok := db.Get(args[0])
	if !ok {
		return writer.WriteStringArray([]string{})
	}

	hash, ok := existing.(*data.Hash)
	if !ok {
		return writer.WriteError("WRONGTYPE Operation against a key holding the wrong kind of value")
	}

	keys := make([]string, 0, len(hash.Fields))
	for field := range hash.Fields {
		keys = append(keys, field)
	}
	return writer.WriteStringArray(keys)
}

// HValsCommand 获取哈希表所有值
type HValsCommand struct{}

func (c *HValsCommand) Name() string  { return "HVALS" }
func (c *HValsCommand) ArgCount() int { return 1 }
func (c *HValsCommand) Execute(db *data.DB, args []string, writer *resp.Writer) error {
	existing, ok := db.Get(args[0])
	if !ok {
		return writer.WriteStringArray([]string{})
	}

	hash, ok := existing.(*data.Hash)
	if !ok {
		return writer.WriteError("WRONGTYPE Operation against a key holding the wrong kind of value")
	}

	values := make([]string, 0, len(hash.Fields))
	for _, value := range hash.Fields {
		values = append(values, value)
	}
	return writer.WriteStringArray(values)
}

// HMSetCommand 同时设置多个哈希表字段
type HMSetCommand struct{}

func (c *HMSetCommand) Name() string  { return "HMSET" }
func (c *HMSetCommand) ArgCount() int { return 3 }
func (c *HMSetCommand) Execute(db *data.DB, args []string, writer *resp.Writer) error {
	key := args[0]

	if len(args[1:])%2 != 0 {
		return writer.WriteError("ERR wrong number of arguments for 'hmset' command")
	}

	var hash *data.Hash
	existing, ok := db.Get(key)
	if ok {
		var typeOk bool
		hash, typeOk = existing.(*data.Hash)
		if !typeOk {
			return writer.WriteError("WRONGTYPE Operation against a key holding the wrong kind of value")
		}
	} else {
		hash = &data.Hash{Fields: make(map[string]string)}
	}

	for i := 1; i < len(args); i += 2 {
		hash.Fields[args[i]] = args[i+1]
	}

	db.Set(key, hash)
	return writer.WriteOK()
}

// HMGetCommand 获取多个哈希表字段的值
type HMGetCommand struct{}

func (c *HMGetCommand) Name() string  { return "HMGET" }
func (c *HMGetCommand) ArgCount() int { return 2 }
func (c *HMGetCommand) Execute(db *data.DB, args []string, writer *resp.Writer) error {
	key := args[0]
	fields := args[1:]

	existing, ok := db.Get(key)
	if !ok {
		// 所有字段都返回 nil
		values := make([]resp.Value, len(fields))
		for i := range values {
			values[i] = &resp.BulkString{Content: nil}
		}
		return writer.Write(&resp.Array{Items: values})
	}

	hash, ok := existing.(*data.Hash)
	if !ok {
		return writer.WriteError("WRONGTYPE Operation against a key holding the wrong kind of value")
	}

	values := make([]resp.Value, len(fields))
	for i, field := range fields {
		if value, ok := hash.Fields[field]; ok {
			values[i] = &resp.BulkString{Content: []byte(value)}
		} else {
			values[i] = &resp.BulkString{Content: nil}
		}
	}
	return writer.Write(&resp.Array{Items: values})
}

// HIncrByCommand 为哈希表字段值加上指定增量
type HIncrByCommand struct{}

func (c *HIncrByCommand) Name() string  { return "HINCRBY" }
func (c *HIncrByCommand) ArgCount() int { return 3 }
func (c *HIncrByCommand) Execute(db *data.DB, args []string, writer *resp.Writer) error {
	key, field, incrementStr := args[0], args[1], args[2]

	increment, err := strconv.ParseInt(incrementStr, 10, 64)
	if err != nil {
		return writer.WriteError("ERR value is not an integer or out of range")
	}

	var hash *data.Hash
	existing, ok := db.Get(key)
	if ok {
		var typeOk bool
		hash, typeOk = existing.(*data.Hash)
		if !typeOk {
			return writer.WriteError("WRONGTYPE Operation against a key holding the wrong kind of value")
		}
	} else {
		hash = &data.Hash{Fields: make(map[string]string)}
	}

	current := int64(0)
	if value, ok := hash.Fields[field]; ok {
		current, err = strconv.ParseInt(value, 10, 64)
		if err != nil {
			return writer.WriteError("ERR hash value is not an integer")
		}
	}

	current += increment
	hash.Fields[field] = strconv.FormatInt(current, 10)
	db.Set(key, hash)

	return writer.WriteInteger(current)
}
