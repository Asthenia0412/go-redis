package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"go-redis/server"
)

func main() {
	var (
		addr    = flag.String("addr", ":6379", "Server address")
		dbCount = flag.Int("databases", 16, "Number of databases")
		aofFile = flag.String("aof", "appendonly.aof", "AOF file path (empty to disable)")
	)
	flag.Parse()

	// 如果 AOF 文件路径为空，禁用 AOF
	aof := *aofFile
	if aof == "" {
		aof = ""
	}

	// 创建服务器
	srv, err := server.NewServer(*addr, *dbCount, aof)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// 处理信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		fmt.Println("\nShutting down...")
		srv.Stop()
	}()

	fmt.Printf("Go-Redis server starting on %s\n", *addr)
	fmt.Printf("Databases: %d\n", *dbCount)
	if aof != "" {
		fmt.Printf("AOF file: %s\n", aof)
	}
	fmt.Println("Press Ctrl+C to stop")

	// 启动服务器
	if err := srv.Start(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
