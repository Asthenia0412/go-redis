package main

import (
	"fmt"
	"net"
	"os"
)

func main() {
	// 1. 监听端口6379
	listener, err := net.Listen("tcp", ":6379")
	if err != nil {
		fmt.Println("Error Listening", err)
		os.Exit(1)
	}
	defer listener.Close()
	fmt.Println("Redis Server已经启动成功了 端口是6379")
	// 2.接收客户端连接
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("接收错误！ ", err)
		}
		// 3.启动goroutine处理链接

		go handleClient(conn)
	}
}

func handleClient(conn net.Conn) {
	defer conn.Close()
	fmt.Println("新的链接建立了！", conn.RemoteAddr())
}
