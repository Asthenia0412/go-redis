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
	writer := NewRespWriter(conn) // 初始化 Writer

	for {
		value, err := reader.Read()
		if err != nil {
			if err != io.EOF {
				fmt.Println("读取错误:", err)
			}
			break
		}

		args, ok := value.([]interface{})
		if !ok || len(args) == 0 {
			continue
		}

		cmd := strings.ToUpper(args[0].(string))

		switch cmd {
		case "SET":
			if len(args) < 3 {
				writer.WriteError("wrong number of arguments for 'set' command")
				continue
			}
			db.Set(args[1].(string), args[2].(string))
			writer.WriteSimpleString("OK")

		case "GET":
			val, ok := db.Get(args[1].(string))
			if !ok {
				writer.WriteBulk("") // 返回空值
			} else {
				writer.WriteBulk(val)
			}

		case "DEL":
			// 练习：实现 DEL 命令，返回删除的数量
			key := args[1].(string)
			// 需要在 db.go 增加 Delete 方法
			count := db.Delete(key)
			writer.WriteInteger(count)

		case "PING":
			writer.WriteSimpleString("PONG")

		default:
			writer.WriteError("unknown command '" + cmd + "'")
		}
	}
}
