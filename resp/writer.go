package resp

import (
	"io"
)

// Writer RESP 协议写入器
type Writer struct {
	writer io.Writer
}

// NewWriter 创建新的 RESP 写入器
func NewWriter(w io.Writer) *Writer {
	return &Writer{writer: w}
}

// Write 写入 RESP 值
func (w *Writer) Write(v Value) error {
	_, err := w.writer.Write(v.ToBytes())
	return err
}

// WriteSimpleString 写入简单字符串
func (w *Writer) WriteSimpleString(s string) error {
	return w.Write(&SimpleString{Content: s})
}

// WriteError 写入错误
func (w *Writer) WriteError(msg string) error {
	return w.Write(&Error{Content: "ERR " + msg})
}

// WriteInteger 写入整数
func (w *Writer) WriteInteger(n int64) error {
	return w.Write(&Integer{Value: n})
}

// WriteBulkString 写入批量字符串
func (w *Writer) WriteBulkString(s string) error {
	return w.Write(&BulkString{Content: []byte(s)})
}

// WriteNullBulkString 写入空批量字符串
func (w *Writer) WriteNullBulkString() error {
	return w.Write(&BulkString{Content: nil})
}

// WriteArray 写入数组
func (w *Writer) WriteArray(items []Value) error {
	return w.Write(&Array{Items: items})
}

// WriteStringArray 写入字符串数组
func (w *Writer) WriteStringArray(items []string) error {
	values := make([]Value, len(items))
	for i, item := range items {
		values[i] = &BulkString{Content: []byte(item)}
	}
	return w.Write(&Array{Items: values})
}

// WriteBulkStringArray 写入批量字符串数组
func (w *Writer) WriteBulkStringArray(items [][]byte) error {
	values := make([]Value, len(items))
	for i, item := range items {
		values[i] = &BulkString{Content: item}
	}
	return w.Write(&Array{Items: values})
}

// WriteOK 写入 OK 响应
func (w *Writer) WriteOK() error {
	return w.WriteSimpleString("OK")
}

// WritePONG 写入 PONG 响应
func (w *Writer) WritePONG() error {
	return w.WriteSimpleString("PONG")
}

// WriteIntOrNull 根据条件写入整数或空值
func (w *Writer) WriteIntOrNull(n int64, exists bool) error {
	if exists {
		return w.WriteInteger(n)
	}
	return w.WriteNullBulkString()
}
