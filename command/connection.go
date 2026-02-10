package command

import (
	"go-redis/data"
	"go-redis/resp"
	"strconv"
	"strings"
)

// PingCommand PING 命令
type PingCommand struct{}

func (c *PingCommand) Name() string  { return "PING" }
func (c *PingCommand) ArgCount() int { return 0 }
func (c *PingCommand) Execute(db *data.DB, args []string, writer *resp.Writer) error {
	if len(args) > 0 {
		return writer.WriteBulkString(args[0])
	}
	return writer.WritePONG()
}

// EchoCommand ECHO 命令
type EchoCommand struct{}

func (c *EchoCommand) Name() string  { return "ECHO" }
func (c *EchoCommand) ArgCount() int { return 1 }
func (c *EchoCommand) Execute(db *data.DB, args []string, writer *resp.Writer) error {
	return writer.WriteBulkString(args[0])
}

// SelectCommand SELECT 命令
type SelectCommand struct{}

func (c *SelectCommand) Name() string  { return "SELECT" }
func (c *SelectCommand) ArgCount() int { return 1 }
func (c *SelectCommand) Execute(db *data.DB, args []string, writer *resp.Writer) error {
	// SELECT 命令在 handler 中处理，这里只是占位
	return writer.WriteOK()
}

// QuitCommand QUIT 命令
type QuitCommand struct{}

func (c *QuitCommand) Name() string  { return "QUIT" }
func (c *QuitCommand) ArgCount() int { return 0 }
func (c *QuitCommand) Execute(db *data.DB, args []string, writer *resp.Writer) error {
	return writer.WriteOK()
}

// ClientCommand CLIENT 命令
type ClientCommand struct{}

func (c *ClientCommand) Name() string  { return "CLIENT" }
func (c *ClientCommand) ArgCount() int { return 1 }
func (c *ClientCommand) Execute(db *data.DB, args []string, writer *resp.Writer) error {
	if len(args) == 0 {
		return writer.WriteError("ERR wrong number of arguments for 'client' command")
	}

	subCmd := strings.ToUpper(args[0])
	switch subCmd {
	case "LIST":
		// 返回空列表
		return writer.WriteBulkString("")
	case "SETNAME":
		if len(args) < 2 {
			return writer.WriteError("ERR wrong number of arguments for 'client|setname' command")
		}
		return writer.WriteOK()
	case "GETNAME":
		return writer.WriteNullBulkString()
	case "KILL":
		return writer.WriteError("ERR No such client")
	case "PAUSE":
		if len(args) < 2 {
			return writer.WriteError("ERR wrong number of arguments for 'client|pause' command")
		}
		_, err := strconv.Atoi(args[1])
		if err != nil {
			return writer.WriteError("ERR timeout is not an integer or out of range")
		}
		return writer.WriteOK()
	default:
		return writer.WriteError("ERR unknown subcommand '" + subCmd + "'")
	}
}
