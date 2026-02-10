package resp

import (
	"bufio"
	"errors"
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

var (
	ErrInvalidSyntax = errors.New("invalid resp syntax")
	ErrEmptyBulk     = errors.New("empty bulk string")
)

// Value RESP 值接口
type Value interface {
	Type() byte
	ToBytes() []byte
}

// SimpleString 简单字符串
type SimpleString struct {
	Content string
}

func (s *SimpleString) Type() byte { return STRING }
func (s *SimpleString) ToBytes() []byte {
	return []byte("+" + s.Content + "\r\n")
}

// Error 错误类型
type Error struct {
	Content string
}

func (e *Error) Type() byte { return ERROR }
func (e *Error) ToBytes() []byte {
	return []byte("-" + e.Content + "\r\n")
}

// Integer 整数类型
type Integer struct {
	Value int64
}

func (i *Integer) Type() byte { return INTEGER }
func (i *Integer) ToBytes() []byte {
	return []byte(":" + strconv.FormatInt(i.Value, 10) + "\r\n")
}

// BulkString 批量字符串
type BulkString struct {
	Content []byte
}

func (b *BulkString) Type() byte { return BULK }
func (b *BulkString) ToBytes() []byte {
	if b.Content == nil {
		return []byte("$-1\r\n")
	}
	return []byte("$" + strconv.Itoa(len(b.Content)) + "\r\n" + string(b.Content) + "\r\n")
}

// Array 数组类型
type Array struct {
	Items []Value
}

func (a *Array) Type() byte { return ARRAY }
func (a *Array) ToBytes() []byte {
	if a.Items == nil {
		return []byte("*-1\r\n")
	}
	result := []byte("*" + strconv.Itoa(len(a.Items)) + "\r\n")
	for _, item := range a.Items {
		result = append(result, item.ToBytes()...)
	}
	return result
}

// Reader RESP 协议读取器
type Reader struct {
	reader *bufio.Reader
}

// NewReader 创建新的 RESP 读取器
func NewReader(rd io.Reader) *Reader {
	return &Reader{reader: bufio.NewReader(rd)}
}

// Read 读取一个 RESP 值
func (r *Reader) Read() (Value, error) {
	typeByte, err := r.reader.ReadByte()
	if err != nil {
		return nil, err
	}

	switch typeByte {
	case STRING:
		return r.readSimpleString()
	case ERROR:
		return r.readError()
	case INTEGER:
		return r.readInteger()
	case BULK:
		return r.readBulkString()
	case ARRAY:
		return r.readArray()
	default:
		return nil, ErrInvalidSyntax
	}
}

// readLine 读取一行，直到遇到 \r\n
func (r *Reader) readLine() ([]byte, error) {
	line, err := r.reader.ReadBytes('\n')
	if err != nil {
		return nil, err
	}
	if len(line) < 2 || line[len(line)-2] != '\r' {
		return nil, ErrInvalidSyntax
	}
	return line[:len(line)-2], nil
}

// readSimpleString 读取简单字符串
func (r *Reader) readSimpleString() (*SimpleString, error) {
	line, err := r.readLine()
	if err != nil {
		return nil, err
	}
	return &SimpleString{Content: string(line)}, nil
}

// readError 读取错误
func (r *Reader) readError() (*Error, error) {
	line, err := r.readLine()
	if err != nil {
		return nil, err
	}
	return &Error{Content: string(line)}, nil
}

// readInteger 读取整数
func (r *Reader) readInteger() (*Integer, error) {
	line, err := r.readLine()
	if err != nil {
		return nil, err
	}
	val, err := strconv.ParseInt(string(line), 10, 64)
	if err != nil {
		return nil, err
	}
	return &Integer{Value: val}, nil
}

// readBulkString 读取批量字符串
func (r *Reader) readBulkString() (*BulkString, error) {
	line, err := r.readLine()
	if err != nil {
		return nil, err
	}

	length, err := strconv.Atoi(string(line))
	if err != nil {
		return nil, err
	}

	if length == -1 {
		return &BulkString{Content: nil}, nil
	}

	if length < 0 {
		return nil, ErrInvalidSyntax
	}

	content := make([]byte, length)
	_, err = io.ReadFull(r.reader, content)
	if err != nil {
		return nil, err
	}

	// 读取结尾的 \r\n
	ending := make([]byte, 2)
	_, err = io.ReadFull(r.reader, ending)
	if err != nil || ending[0] != '\r' || ending[1] != '\n' {
		return nil, ErrInvalidSyntax
	}

	return &BulkString{Content: content}, nil
}

// readArray 读取数组
func (r *Reader) readArray() (*Array, error) {
	line, err := r.readLine()
	if err != nil {
		return nil, err
	}

	length, err := strconv.Atoi(string(line))
	if err != nil {
		return nil, err
	}

	if length == -1 {
		return &Array{Items: nil}, nil
	}

	if length < 0 {
		return nil, ErrInvalidSyntax
	}

	items := make([]Value, length)
	for i := 0; i < length; i++ {
		item, err := r.Read()
		if err != nil {
			return nil, err
		}
		items[i] = item
	}

	return &Array{Items: items}, nil
}
