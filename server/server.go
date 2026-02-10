package server

import (
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"go-redis/command"
	"go-redis/data"
	"go-redis/persist"
	"go-redis/resp"
)

// Server Redis 服务器结构体
type Server struct {
	addr      string
	redis     *data.Redis
	registry  *command.CommandRegistry
	aof       *persist.AOF
	listener  net.Listener
	clients   map[net.Conn]*Client
	clientsMu sync.RWMutex
	stopCh    chan struct{}
	wg        sync.WaitGroup
}

// Client 客户端连接
type Client struct {
	conn   net.Conn
	db     *data.DB
	server *Server
}

// NewServer 创建新的 Redis 服务器
func NewServer(addr string, dbCount int, aofFile string) (*Server, error) {
	redis := data.NewRedis(dbCount)
	registry := command.NewCommandRegistry()

	var aof *persist.AOF
	var err error
	if aofFile != "" {
		aof, err = persist.NewAOF(aofFile)
		if err != nil {
			return nil, fmt.Errorf("failed to open AOF file: %v", err)
		}
		// 加载 AOF 文件
		if err := aof.Load(redis, registry); err != nil {
			log.Printf("Warning: failed to load AOF file: %v", err)
		}
	}

	return &Server{
		addr:     addr,
		redis:    redis,
		registry: registry,
		aof:      aof,
		clients:  make(map[net.Conn]*Client),
		stopCh:   make(chan struct{}),
	}, nil
}

// Start 启动服务器
func (s *Server) Start() error {
	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}
	s.listener = listener

	log.Printf("Redis server listening on %s", s.addr)

	// 启动过期键清理协程
	s.wg.Add(1)
	go s.expireLoop()

	for {
		select {
		case <-s.stopCh:
			return nil
		default:
		}

		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-s.stopCh:
				return nil
			default:
				log.Printf("Accept error: %v", err)
				continue
			}
		}

		s.wg.Add(1)
		go s.handleClient(conn)
	}
}

// Stop 停止服务器
func (s *Server) Stop() error {
	close(s.stopCh)

	if s.listener != nil {
		s.listener.Close()
	}

	// 关闭所有客户端连接
	s.clientsMu.Lock()
	for conn := range s.clients {
		conn.Close()
	}
	s.clients = make(map[net.Conn]*Client)
	s.clientsMu.Unlock()

	// 等待所有协程结束
	s.wg.Wait()

	// 关闭 AOF
	if s.aof != nil {
		s.aof.Close()
	}

	return nil
}

// handleClient 处理客户端连接
func (s *Server) handleClient(conn net.Conn) {
	defer s.wg.Done()
	defer conn.Close()

	client := &Client{
		conn:   conn,
		db:     s.redis.GetCurrentDB(),
		server: s,
	}

	s.clientsMu.Lock()
	s.clients[conn] = client
	s.clientsMu.Unlock()

	defer func() {
		s.clientsMu.Lock()
		delete(s.clients, conn)
		s.clientsMu.Unlock()
	}()

	log.Printf("New connection from %s", conn.RemoteAddr())

	reader := resp.NewReader(conn)
	writer := resp.NewWriter(conn)

	for {
		select {
		case <-s.stopCh:
			return
		default:
		}

		// 设置读取超时
		conn.SetReadDeadline(time.Now().Add(5 * time.Minute))

		value, err := reader.Read()
		if err != nil {
			if err != io.EOF {
				log.Printf("Read error from %s: %v", conn.RemoteAddr(), err)
			}
			return
		}

		// 处理命令
		if err := s.processCommand(client, value, writer); err != nil {
			log.Printf("Process command error from %s: %v", conn.RemoteAddr(), err)
			return
		}
	}
}

// processCommand 处理单个命令
func (s *Server) processCommand(client *Client, value resp.Value, writer *resp.Writer) error {
	// 检查是否是数组类型（命令参数）
	array, ok := value.(*resp.Array)
	if !ok || array.Items == nil || len(array.Items) == 0 {
		return writer.WriteError("ERR invalid command format")
	}

	// 提取命令名和参数
	args := make([]string, len(array.Items))
	for i, item := range array.Items {
		switch v := item.(type) {
		case *resp.BulkString:
			if v.Content == nil {
				args[i] = ""
			} else {
				args[i] = string(v.Content)
			}
		case *resp.SimpleString:
			args[i] = v.Content
		default:
			return writer.WriteError("ERR invalid argument type")
		}
	}

	cmdName := strings.ToUpper(args[0])
	cmdArgs := args[1:]

	// 特殊处理 SELECT 命令
	if cmdName == "SELECT" {
		return s.handleSelect(client, cmdArgs, writer)
	}

	// 特殊处理 QUIT 命令
	if cmdName == "QUIT" {
		writer.WriteOK()
		return io.EOF
	}

	// 查找命令
	cmd, ok := s.registry.Get(cmdName)
	if !ok {
		return writer.WriteError("ERR unknown command '" + cmdName + "'")
	}

	// 检查参数数量
	if len(cmdArgs) < cmd.ArgCount() {
		return writer.WriteError("ERR wrong number of arguments for '" + strings.ToLower(cmdName) + "' command")
	}

	// 执行命令
	if err := cmd.Execute(client.db, cmdArgs, writer); err != nil {
		return err
	}

	// 写入 AOF（如果是写命令）
	if s.aof != nil && s.isWriteCommand(cmdName) {
		s.aof.Write(args)
	}

	return nil
}

// handleSelect 处理 SELECT 命令
func (s *Server) handleSelect(client *Client, args []string, writer *resp.Writer) error {
	if len(args) != 1 {
		return writer.WriteError("ERR wrong number of arguments for 'select' command")
	}

	index, err := strconv.Atoi(args[0])
	if err != nil {
		return writer.WriteError("ERR invalid DB index")
	}

	db, ok := s.redis.SelectDB(index)
	if !ok {
		return writer.WriteError("ERR DB index is out of range")
	}

	client.db = db
	return writer.WriteOK()
}

// isWriteCommand 判断是否是写命令
func (s *Server) isWriteCommand(cmdName string) bool {
	writeCommands := map[string]bool{
		"SET": true, "SETEX": true, "SETNX": true, "GETSET": true,
		"MSET": true, "APPEND": true, "INCR": true, "DECR": true,
		"INCRBY": true, "DECRBY": true,
		"DEL": true, "EXPIRE": true, "PERSIST": true, "RENAME": true,
		"FLUSHDB": true, "FLUSHALL": true,
		"LPUSH": true, "RPUSH": true, "LPOP": true, "RPOP": true,
		"LREM": true, "LSET": true,
		"HSET": true, "HDEL": true, "HMSET": true, "HINCRBY": true,
		"SADD": true, "SREM": true, "SPOP": true,
		"ZADD": true, "ZREM": true, "ZINCRBY": true,
		"ZREMRANGEBYRANK": true, "ZREMRANGEBYSCORE": true,
	}
	return writeCommands[cmdName]
}

// expireLoop 定期清理过期键
func (s *Server) expireLoop() {
	defer s.wg.Done()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.cleanExpiredKeys()
		}
	}
}

// cleanExpiredKeys 清理所有数据库中的过期键
func (s *Server) cleanExpiredKeys() {
	for i := 0; i < s.redis.GetDBCount(); i++ {
		db, ok := s.redis.GetDB(i)
		if !ok {
			continue
		}

		expiredKeys := db.GetExpiredKeys()
		for _, key := range expiredKeys {
			db.Delete(key)
			// 写入 AOF 删除命令
			if s.aof != nil {
				s.aof.Write([]string{"DEL", key})
			}
		}
	}
}

// GetRedis 获取 Redis 实例
func (s *Server) GetRedis() *data.Redis {
	return s.redis
}

// GetAOF 获取 AOF 实例
func (s *Server) GetAOF() *persist.AOF {
	return s.aof
}
