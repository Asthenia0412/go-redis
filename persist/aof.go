package persist

import (
	"bufio"
	"fmt"
	"go-redis/command"
	"go-redis/data"
	"go-redis/resp"
	"log"
	"os"
	"strings"
	"sync"
)

// AOF AOF 持久化结构体
type AOF struct {
	file   *os.File
	writer *bufio.Writer
	mu     sync.Mutex
	path   string
}

// NewAOF 创建新的 AOF 实例
func NewAOF(path string) (*AOF, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}

	return &AOF{
		file:   file,
		writer: bufio.NewWriter(file),
		path:   path,
	}, nil
}

// Write 写入命令到 AOF
func (a *AOF) Write(args []string) error {
	if a == nil {
		return nil
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	// 构建 RESP 数组
	var respBuilder strings.Builder
	respBuilder.WriteString(fmt.Sprintf("*%d\r\n", len(args)))
	for _, arg := range args {
		respBuilder.WriteString(fmt.Sprintf("$%d\r\n%s\r\n", len(arg), arg))
	}

	_, err := a.writer.WriteString(respBuilder.String())
	if err != nil {
		return err
	}

	// 立即刷新到磁盘
	return a.writer.Flush()
}

// Load 从 AOF 文件加载数据
func (a *AOF) Load(redis *data.Redis, registry *command.CommandRegistry) error {
	if a == nil {
		return nil
	}

	// 获取文件信息
	fileInfo, err := a.file.Stat()
	if err != nil {
		return err
	}

	if fileInfo.Size() == 0 {
		return nil
	}

	log.Printf("Loading AOF file: %s", a.path)

	// 重新打开文件用于读取
	file, err := os.Open(a.path)
	if err != nil {
		return err
	}
	defer file.Close()

	reader := resp.NewReader(file)
	cmdCount := 0

	for {
		value, err := reader.Read()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			log.Printf("AOF load error: %v", err)
			break
		}

		array, ok := value.(*resp.Array)
		if !ok || array.Items == nil || len(array.Items) == 0 {
			continue
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
				continue
			}
		}

		cmdName := strings.ToUpper(args[0])
		cmdArgs := args[1:]

		// 特殊处理 SELECT 命令
		if cmdName == "SELECT" {
			if len(cmdArgs) > 0 {
				index := 0
				fmt.Sscanf(cmdArgs[0], "%d", &index)
				redis.SelectDB(index)
			}
			continue
		}

		// 查找并执行命令
		cmd, ok := registry.Get(cmdName)
		if !ok {
			log.Printf("Unknown command in AOF: %s", cmdName)
			continue
		}

		// 创建 dummy writer 来执行命令
		dummyWriter := resp.NewWriter(&nullWriter{})
		db := redis.GetCurrentDB()
		cmd.Execute(db, cmdArgs, dummyWriter)
		cmdCount++
	}

	log.Printf("AOF loaded %d commands", cmdCount)
	return nil
}

// Close 关闭 AOF 文件
func (a *AOF) Close() error {
	if a == nil {
		return nil
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	if err := a.writer.Flush(); err != nil {
		return err
	}

	return a.file.Close()
}

// nullWriter 空写入器
type nullWriter struct{}

func (w *nullWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}
