package main

import (
	"fmt"
	"io"
	"net"
)

func main() {
	// 1. 监听本地 6379 端口
	listener, err := net.Listen("tcp", ":6379")
	if err != nil {
		fmt.Println("启动监听失败:", err)
		return
	}
	defer listener.Close()

	fmt.Println("Mini-Redis 正在运行在 :6379 ...")

	for {
		// 2. 等待客户端连接 (比如 redis-cli)
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("接收连接失败:", err)
			continue
		}

		// 3. 为每个连接启动一个协程
		go handleClient(conn)
	}
}

func handleClient(conn net.Conn) {
	defer conn.Close()
	fmt.Printf("客户端已连接: %s\n", conn.RemoteAddr())

	// 初始化我们写好的解析器
	reader := NewRespReader(conn)

	for {
		// 4. 循环读取客户端发送的命令
		value, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				fmt.Printf("客户端 %s 断开连接\n", conn.RemoteAddr())
			} else {
				fmt.Println("读取命令错误:", err)
			}
			return
		}

		// 5. 打印解析出来的结果
		fmt.Printf("收到原始数据: %v\n", value)

		// 暂时先简单地给客户端回一个 "OK"
		conn.Write([]byte("+OK\r\n"))
	}
}
