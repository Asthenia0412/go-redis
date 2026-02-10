package command

import (
	"strconv"
	"strings"
	"time"

	"go-redis/data"
	"go-redis/resp"
)

// Command 命令接口
type Command interface {
	Name() string
	Execute(db *data.DB, args []string, writer *resp.Writer) error
	ArgCount() int // 最小参数数量（不包括命令本身）
}

// BaseCommand 基础命令结构
type BaseCommand struct {
	name     string
	argCount int
}

func (c *BaseCommand) Name() string  { return c.name }
func (c *BaseCommand) ArgCount() int { return c.argCount }

// CommandRegistry 命令注册表
type CommandRegistry struct {
	commands map[string]Command
}

// NewCommandRegistry 创建新的命令注册表
func NewCommandRegistry() *CommandRegistry {
	registry := &CommandRegistry{
		commands: make(map[string]Command),
	}
	registry.registerCommands()
	return registry
}

// Register 注册命令
func (r *CommandRegistry) Register(cmd Command) {
	r.commands[strings.ToUpper(cmd.Name())] = cmd
}

// Get 获取命令
func (r *CommandRegistry) Get(name string) (Command, bool) {
	cmd, ok := r.commands[strings.ToUpper(name)]
	return cmd, ok
}

// registerCommands 注册所有命令
func (r *CommandRegistry) registerCommands() {
	// String 命令
	r.Register(&GetCommand{})
	r.Register(&SetCommand{})
	r.Register(&SetEXCommand{})
	r.Register(&SetNXCommand{})
	r.Register(&AppendCommand{})
	r.Register(&StrLenCommand{})
	r.Register(&IncrCommand{})
	r.Register(&DecrCommand{})
	r.Register(&IncrByCommand{})
	r.Register(&DecrByCommand{})
	r.Register(&GetSetCommand{})
	r.Register(&MGetCommand{})
	r.Register(&MSetCommand{})

	// Key 命令
	r.Register(&DelCommand{})
	r.Register(&ExistsCommand{})
	r.Register(&ExpireCommand{})
	r.Register(&TTLCommand{})
	r.Register(&PersistCommand{})
	r.Register(&KeysCommand{})
	r.Register(&TypeCommand{})
	r.Register(&RenameCommand{})
	r.Register(&RandomKeyCommand{})
	r.Register(&DBSizeCommand{})
	r.Register(&FlushDBCommand{})
	r.Register(&FlushAllCommand{})

	// List 命令
	r.Register(&LPushCommand{})
	r.Register(&RPushCommand{})
	r.Register(&LPopCommand{})
	r.Register(&RPopCommand{})
	r.Register(&LLenCommand{})
	r.Register(&LRangeCommand{})
	r.Register(&LIndexCommand{})
	r.Register(&LRemCommand{})
	r.Register(&LSetCommand{})

	// Hash 命令
	r.Register(&HSetCommand{})
	r.Register(&HGetCommand{})
	r.Register(&HDelCommand{})
	r.Register(&HExistsCommand{})
	r.Register(&HLenCommand{})
	r.Register(&HGetAllCommand{})
	r.Register(&HKeysCommand{})
	r.Register(&HValsCommand{})
	r.Register(&HMSetCommand{})
	r.Register(&HMGetCommand{})
	r.Register(&HIncrByCommand{})

	// Set 命令
	r.Register(&SAddCommand{})
	r.Register(&SRemCommand{})
	r.Register(&SIsMemberCommand{})
	r.Register(&SCardCommand{})
	r.Register(&SMembersCommand{})
	r.Register(&SPopCommand{})
	r.Register(&SRandMemberCommand{})
	r.Register(&SUnionCommand{})
	r.Register(&SInterCommand{})
	r.Register(&SDiffCommand{})

	// Sorted Set 命令
	r.Register(&ZAddCommand{})
	r.Register(&ZRemCommand{})
	r.Register(&ZScoreCommand{})
	r.Register(&ZCardCommand{})
	r.Register(&ZRangeCommand{})
	r.Register(&ZRevRangeCommand{})
	r.Register(&ZRankCommand{})
	r.Register(&ZRevRankCommand{})
	r.Register(&ZIncrByCommand{})
	r.Register(&ZRangeByScoreCommand{})
	r.Register(&ZRemRangeByRankCommand{})
	r.Register(&ZRemRangeByScoreCommand{})

	// 连接命令
	r.Register(&PingCommand{})
	r.Register(&EchoCommand{})
	r.Register(&SelectCommand{})
	r.Register(&QuitCommand{})
}

// ==================== String 命令 ====================

type GetCommand struct{}

func (c *GetCommand) Name() string  { return "GET" }
func (c *GetCommand) ArgCount() int { return 1 }
func (c *GetCommand) Execute(db *data.DB, args []string, writer *resp.Writer) error {
	val, ok := db.Get(args[0])
	if !ok {
		return writer.WriteNullBulkString()
	}
	str, ok := val.(*data.String)
	if !ok {
		return writer.WriteError("WRONGTYPE Operation against a key holding the wrong kind of value")
	}
	return writer.WriteBulkString(str.Value)
}

type SetCommand struct{}

func (c *SetCommand) Name() string  { return "SET" }
func (c *SetCommand) ArgCount() int { return 2 }
func (c *SetCommand) Execute(db *data.DB, args []string, writer *resp.Writer) error {
	key, value := args[0], args[1]

	// 处理可选参数 (EX, PX, NX, XX)
	var expire time.Duration = 0
	nx, xx := false, false

	for i := 2; i < len(args); i++ {
		switch strings.ToUpper(args[i]) {
		case "EX":
			if i+1 < len(args) {
				seconds, _ := strconv.Atoi(args[i+1])
				expire = time.Duration(seconds) * time.Second
				i++
			}
		case "PX":
			if i+1 < len(args) {
				millis, _ := strconv.Atoi(args[i+1])
				expire = time.Duration(millis) * time.Millisecond
				i++
			}
		case "NX":
			nx = true
		case "XX":
			xx = true
		}
	}

	// 检查 NX/XX 条件
	_, exists := db.Get(key)
	if nx && exists {
		return writer.WriteNullBulkString() // 键已存在，不设置
	}
	if xx && !exists {
		return writer.WriteNullBulkString() // 键不存在，不设置
	}

	if expire > 0 {
		db.SetWithExpire(key, &data.String{Value: value}, expire)
	} else {
		db.Set(key, &data.String{Value: value})
	}
	return writer.WriteOK()
}

type SetEXCommand struct{}

func (c *SetEXCommand) Name() string  { return "SETEX" }
func (c *SetEXCommand) ArgCount() int { return 3 }
func (c *SetEXCommand) Execute(db *data.DB, args []string, writer *resp.Writer) error {
	key, secondsStr, value := args[0], args[1], args[2]
	seconds, err := strconv.Atoi(secondsStr)
	if err != nil || seconds <= 0 {
		return writer.WriteError("ERR invalid expire time in 'setex' command")
	}
	db.SetWithExpire(key, &data.String{Value: value}, time.Duration(seconds)*time.Second)
	return writer.WriteOK()
}

type SetNXCommand struct{}

func (c *SetNXCommand) Name() string  { return "SETNX" }
func (c *SetNXCommand) ArgCount() int { return 2 }
func (c *SetNXCommand) Execute(db *data.DB, args []string, writer *resp.Writer) error {
	key, value := args[0], args[1]
	_, exists := db.Get(key)
	if exists {
		return writer.WriteInteger(0)
	}
	db.Set(key, &data.String{Value: value})
	return writer.WriteInteger(1)
}

type AppendCommand struct{}

func (c *AppendCommand) Name() string  { return "APPEND" }
func (c *AppendCommand) ArgCount() int { return 2 }
func (c *AppendCommand) Execute(db *data.DB, args []string, writer *resp.Writer) error {
	key, value := args[0], args[1]

	existing, ok := db.Get(key)
	var newValue string
	if ok {
		str, ok := existing.(*data.String)
		if !ok {
			return writer.WriteError("WRONGTYPE Operation against a key holding the wrong kind of value")
		}
		newValue = str.Value + value
	} else {
		newValue = value
	}

	db.Set(key, &data.String{Value: newValue})
	return writer.WriteInteger(int64(len(newValue)))
}

type StrLenCommand struct{}

func (c *StrLenCommand) Name() string  { return "STRLEN" }
func (c *StrLenCommand) ArgCount() int { return 1 }
func (c *StrLenCommand) Execute(db *data.DB, args []string, writer *resp.Writer) error {
	val, ok := db.Get(args[0])
	if !ok {
		return writer.WriteInteger(0)
	}
	str, ok := val.(*data.String)
	if !ok {
		return writer.WriteError("WRONGTYPE Operation against a key holding the wrong kind of value")
	}
	return writer.WriteInteger(int64(len(str.Value)))
}

type IncrCommand struct{}

func (c *IncrCommand) Name() string  { return "INCR" }
func (c *IncrCommand) ArgCount() int { return 1 }
func (c *IncrCommand) Execute(db *data.DB, args []string, writer *resp.Writer) error {
	return incrBy(db, args[0], 1, writer)
}

type DecrCommand struct{}

func (c *DecrCommand) Name() string  { return "DECR" }
func (c *DecrCommand) ArgCount() int { return 1 }
func (c *DecrCommand) Execute(db *data.DB, args []string, writer *resp.Writer) error {
	return incrBy(db, args[0], -1, writer)
}

type IncrByCommand struct{}

func (c *IncrByCommand) Name() string  { return "INCRBY" }
func (c *IncrByCommand) ArgCount() int { return 2 }
func (c *IncrByCommand) Execute(db *data.DB, args []string, writer *resp.Writer) error {
	increment, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil {
		return writer.WriteError("ERR value is not an integer or out of range")
	}
	return incrBy(db, args[0], increment, writer)
}

type DecrByCommand struct{}

func (c *DecrByCommand) Name() string  { return "DECRBY" }
func (c *DecrByCommand) ArgCount() int { return 2 }
func (c *DecrByCommand) Execute(db *data.DB, args []string, writer *resp.Writer) error {
	decrement, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil {
		return writer.WriteError("ERR value is not an integer or out of range")
	}
	return incrBy(db, args[0], -decrement, writer)
}

func incrBy(db *data.DB, key string, increment int64, writer *resp.Writer) error {
	var current int64 = 0

	existing, ok := db.Get(key)
	if ok {
		str, ok := existing.(*data.String)
		if !ok {
			return writer.WriteError("WRONGTYPE Operation against a key holding the wrong kind of value")
		}
		var err error
		current, err = strconv.ParseInt(str.Value, 10, 64)
		if err != nil {
			return writer.WriteError("ERR value is not an integer or out of range")
		}
	}

	current += increment
	db.Set(key, &data.String{Value: strconv.FormatInt(current, 10)})
	return writer.WriteInteger(current)
}

type GetSetCommand struct{}

func (c *GetSetCommand) Name() string  { return "GETSET" }
func (c *GetSetCommand) ArgCount() int { return 2 }
func (c *GetSetCommand) Execute(db *data.DB, args []string, writer *resp.Writer) error {
	key, value := args[0], args[1]

	oldVal, ok := db.Get(key)
	db.Set(key, &data.String{Value: value})

	if !ok {
		return writer.WriteNullBulkString()
	}
	str, ok := oldVal.(*data.String)
	if !ok {
		return writer.WriteNullBulkString()
	}
	return writer.WriteBulkString(str.Value)
}

type MGetCommand struct{}

func (c *MGetCommand) Name() string  { return "MGET" }
func (c *MGetCommand) ArgCount() int { return 1 }
func (c *MGetCommand) Execute(db *data.DB, args []string, writer *resp.Writer) error {
	values := make([]resp.Value, len(args))
	for i, key := range args {
		val, ok := db.Get(key)
		if !ok {
			values[i] = &resp.BulkString{Content: nil}
		} else {
			str, ok := val.(*data.String)
			if ok {
				values[i] = &resp.BulkString{Content: []byte(str.Value)}
			} else {
				values[i] = &resp.BulkString{Content: nil}
			}
		}
	}
	return writer.Write(&resp.Array{Items: values})
}

type MSetCommand struct{}

func (c *MSetCommand) Name() string  { return "MSET" }
func (c *MSetCommand) ArgCount() int { return 2 }
func (c *MSetCommand) Execute(db *data.DB, args []string, writer *resp.Writer) error {
	if len(args)%2 != 0 {
		return writer.WriteError("ERR wrong number of arguments for 'mset' command")
	}

	for i := 0; i < len(args); i += 2 {
		db.Set(args[i], &data.String{Value: args[i+1]})
	}
	return writer.WriteOK()
}
