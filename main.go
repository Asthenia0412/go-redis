package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"strings"
)

// 全局数据库实例
var db = NewDB()

const (
	SET  = "SET"
	GET  = "GET"
	DEL  = "DEL"
	PING = "PING"
)

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
		log.Println("检测到新的链接请求，地址来自" + conn.RemoteAddr().String())
		if err != nil {
			continue
		}
		go handleClient(conn)
	}
}

func handleClient(conn net.Conn) {
	defer conn.Close()
	reader := NewRespReader(conn)
	writer := NewRespWriter(conn)

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
		case SET:
			if len(args) < 3 {
				writer.WriteError("wrong number of arguments for 'set' command")
				continue
			}
			db.Set(args[1].(string), args[2].(string))
			writer.WriteSimpleString("OK")
			log.Println(conn.RemoteAddr().String() + ":执行了Set操作:" + args[1].(string) + ":" + args[2].(string))

		case GET:
			val, ok := db.Get(args[1].(string))
			if !ok {
				writer.WriteBulk("") // 返回空值
			} else {
				writer.WriteBulk(val)
			}
			log.Println(conn.RemoteAddr().String() + ":执行了Get操作:" + args[1].(string) + ":" + val)

		case DEL:
			key := args[1].(string)
			count := db.Delete(key)
			writer.WriteInteger(count)
			log.Println(conn.RemoteAddr().String() + ":执行了DEL操作:" + args[1].(string) + ":")

		case PING:
			writer.WriteSimpleString("PONG")

		default:
			writer.WriteError("unknown command '" + cmd + "'")
		}
	}
}
