package command

import (
	"go-redis/data"
	"go-redis/resp"
	"strconv"
	"time"
)

// DelCommand 删除键
type DelCommand struct{}

func (c *DelCommand) Name() string  { return "DEL" }
func (c *DelCommand) ArgCount() int { return 1 }
func (c *DelCommand) Execute(db *data.DB, args []string, writer *resp.Writer) error {
	count := db.DeleteKeys(args)
	return writer.WriteInteger(int64(count))
}

// ExistsCommand 检查键是否存在
type ExistsCommand struct{}

func (c *ExistsCommand) Name() string  { return "EXISTS" }
func (c *ExistsCommand) ArgCount() int { return 1 }
func (c *ExistsCommand) Execute(db *data.DB, args []string, writer *resp.Writer) error {
	count := db.Exists(args)
	return writer.WriteInteger(int64(count))
}

// ExpireCommand 设置键的过期时间
type ExpireCommand struct{}

func (c *ExpireCommand) Name() string  { return "EXPIRE" }
func (c *ExpireCommand) ArgCount() int { return 2 }
func (c *ExpireCommand) Execute(db *data.DB, args []string, writer *resp.Writer) error {
	seconds, err := strconv.Atoi(args[1])
	if err != nil {
		return writer.WriteError("ERR value is not an integer or out of range")
	}

	if seconds <= 0 {
		// 如果 seconds <= 0，删除键
		if db.Delete(args[0]) {
			return writer.WriteInteger(1)
		}
		return writer.WriteInteger(0)
	}

	if db.Expire(args[0], time.Duration(seconds)*time.Second) {
		return writer.WriteInteger(1)
	}
	return writer.WriteInteger(0)
}

// TTLCommand 获取键的剩余生存时间
type TTLCommand struct{}

func (c *TTLCommand) Name() string  { return "TTL" }
func (c *TTLCommand) ArgCount() int { return 1 }
func (c *TTLCommand) Execute(db *data.DB, args []string, writer *resp.Writer) error {
	ttl := db.TTL(args[0])
	return writer.WriteInteger(ttl)
}

// PersistCommand 移除键的过期时间
type PersistCommand struct{}

func (c *PersistCommand) Name() string  { return "PERSIST" }
func (c *PersistCommand) ArgCount() int { return 1 }
func (c *PersistCommand) Execute(db *data.DB, args []string, writer *resp.Writer) error {
	if db.Persist(args[0]) {
		return writer.WriteInteger(1)
	}
	return writer.WriteInteger(0)
}

// KeysCommand 查找所有符合给定模式的键
type KeysCommand struct{}

func (c *KeysCommand) Name() string  { return "KEYS" }
func (c *KeysCommand) ArgCount() int { return 1 }
func (c *KeysCommand) Execute(db *data.DB, args []string, writer *resp.Writer) error {
	keys := db.Keys(args[0])
	return writer.WriteStringArray(keys)
}

// TypeCommand 返回键的类型
type TypeCommand struct{}

func (c *TypeCommand) Name() string  { return "TYPE" }
func (c *TypeCommand) ArgCount() int { return 1 }
func (c *TypeCommand) Execute(db *data.DB, args []string, writer *resp.Writer) error {
	keyType := db.Type(args[0])
	return writer.WriteSimpleString(keyType)
}

// RenameCommand 重命名键
type RenameCommand struct{}

func (c *RenameCommand) Name() string  { return "RENAME" }
func (c *RenameCommand) ArgCount() int { return 2 }
func (c *RenameCommand) Execute(db *data.DB, args []string, writer *resp.Writer) error {
	oldKey, newKey := args[0], args[1]

	entry, ok := db.GetEntry(oldKey)
	if !ok {
		return writer.WriteError("ERR no such key")
	}

	db.Set(newKey, entry.Data)
	if entry.ExpireTime != nil {
		db.SetWithExpire(newKey, entry.Data, time.Until(*entry.ExpireTime))
	}
	db.Delete(oldKey)

	return writer.WriteOK()
}

// RandomKeyCommand 随机返回一个键
type RandomKeyCommand struct{}

func (c *RandomKeyCommand) Name() string  { return "RANDOMKEY" }
func (c *RandomKeyCommand) ArgCount() int { return 0 }
func (c *RandomKeyCommand) Execute(db *data.DB, args []string, writer *resp.Writer) error {
	key := db.RandomKey()
	if key == "" {
		return writer.WriteNullBulkString()
	}
	return writer.WriteBulkString(key)
}

// DBSizeCommand 返回当前数据库的键的数量
type DBSizeCommand struct{}

func (c *DBSizeCommand) Name() string  { return "DBSIZE" }
func (c *DBSizeCommand) ArgCount() int { return 0 }
func (c *DBSizeCommand) Execute(db *data.DB, args []string, writer *resp.Writer) error {
	return writer.WriteInteger(int64(db.DBSize()))
}

// FlushDBCommand 清空当前数据库
type FlushDBCommand struct{}

func (c *FlushDBCommand) Name() string  { return "FLUSHDB" }
func (c *FlushDBCommand) ArgCount() int { return 0 }
func (c *FlushDBCommand) Execute(db *data.DB, args []string, writer *resp.Writer) error {
	db.FlushDB()
	return writer.WriteOK()
}

// FlushAllCommand 清空所有数据库
type FlushAllCommand struct{}

func (c *FlushAllCommand) Name() string  { return "FLUSHALL" }
func (c *FlushAllCommand) ArgCount() int { return 0 }
func (c *FlushAllCommand) Execute(db *data.DB, args []string, writer *resp.Writer) error {
	// 这里需要通过 Redis 实例来清空所有数据库
	// 暂时只清空当前数据库
	db.FlushDB()
	return writer.WriteOK()
}
