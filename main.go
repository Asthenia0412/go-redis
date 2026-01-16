package main

import (
	"fmt"
	"io"
	"net"
	"strings"
)

// 全局数据库实例
var db = NewDB()

func main() {
	listener, err := net.Listen("tcp", ":6379")
	if err != nil {
		fmt.Println("启动失败:", err)
		return
	}
	defer listener.Close()

	fmt.Println("Mini-Redis 运行中...")

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		go handleClient(conn)
	}
}

func handleClient(conn net.Conn) {
	defer conn.Close()
	reader := NewRespReader(conn)

	for {
		value, err := reader.Read()
		if err != nil {
			if err != io.EOF {
				fmt.Println("读取错误:", err)
			}
			break
		}

		// 1. 类型断言：确保收到的是一个命令数组 []interface{}
		args, ok := value.([]interface{})
		if !ok || len(args) == 0 {
			continue
		}

		// 2. 提取命令名称并转为大写 (不区分大小写，如 SET 或 set)
		cmd, _ := args[0].(string)
		cmd = strings.ToUpper(cmd)

		// 3. 处理业务逻辑
		switch cmd {
		case "SET":
			if len(args) != 3 {
				conn.Write([]byte("-ERR wrong number of arguments for 'set' command\r\n"))
				continue
			}
			key := args[1].(string)
			val := args[2].(string)
			db.Set(key, val)
			conn.Write([]byte("+OK\r\n")) // 回复简单字符串

		case "GET":
			if len(args) != 2 {
				conn.Write([]byte("-ERR wrong number of arguments for 'get' command\r\n"))
				continue
			}
			key := args[1].(string)
			val, ok := db.Get(key)
			if !ok {
				conn.Write([]byte("$-1\r\n")) // Redis 的空值 (Null Bulk String)
			} else {
				// 格式化为 Bulk String: $长度\r\n内容\r\n
				resp := fmt.Sprintf("$%d\r\n%s\r\n", len(val), val)
				conn.Write([]byte(resp))
			}

		case "PING":
			conn.Write([]byte("+PONG\r\n"))

		default:
			conn.Write([]byte("-ERR unknown command\r\n"))
		}
	}
}
