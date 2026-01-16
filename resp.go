package main

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
)

// 定义 RESP 类型的常量
const (
	STRING  = '+'
	ERROR   = '-'
	INTEGER = ':'
	BULK    = '$'
	ARRAY   = '*'
)

type RespReader struct {
	reader *bufio.Reader
}

func NewRespReader(rd io.Reader) *RespReader {
	return &RespReader{reader: bufio.NewReader(rd)}
}

// readLine 读取一行，直到遇到 \r\n，并去掉它们
func (r *RespReader) readLine() (line []byte, err error) {
	line, err = r.reader.ReadBytes('\n')
	if err != nil {
		return nil, err
	}
	// 去掉末尾的 \r\n
	if len(line) >= 2 && line[len(line)-2] == '\r' {
		return line[:len(line)-2], nil
	}
	return line, nil
}

// readInteger 读取 $ 或 * 后面跟随的数字
func (r *RespReader) readInteger() (int, error) {
	line, err := r.readLine()
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(string(line))
}

// Read 解析 RESP 消息
func (r *RespReader) Read() (interface{}, error) {
	typeByte, err := r.reader.ReadByte()
	if err != nil {
		return nil, err
	}

	switch typeByte {
	case ARRAY:
		return r.readArray()
	case BULK:
		return r.readBulk()
	default:
		fmt.Printf("Unknown type: %v\n", typeByte)
		return nil, nil
	}
}

// readBulk 解析 $5\r\nhello\r\n
func (r *RespReader) readBulk() (string, error) {
	size, err := r.readInteger()
	if err != nil {
		return "", err
	}

	bulk := make([]byte, size)
	_, err = io.ReadFull(r.reader, bulk)
	if err != nil {
		return "", err
	}

	// 别忘了消耗掉结尾的 \r\n
	r.readLine()
	return string(bulk), nil
}

// readArray 解析 *2\r\n$3\r\nGET\r\n$1\r\na\r\n
func (r *RespReader) readArray() ([]interface{}, error) {
	len, err := r.readInteger()
	if err != nil {
		return nil, err
	}

	res := make([]interface{}, len)
	for i := 0; i < len; i++ {
		val, err := r.Read()
		if err != nil {
			return nil, err
		}
		res[i] = val
	}
	return res, nil
}
