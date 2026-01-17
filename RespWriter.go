package main

import (
	"fmt"
	"io"
)

type RespWriter struct {
	writer io.Writer
}

func NewRespWriter(w io.Writer) *RespWriter {
	return &RespWriter{writer: w}
}

// WriteSimpleString 回复 +OK\r\n
func (w *RespWriter) WriteSimpleString(s string) error {
	_, err := w.writer.Write([]byte("+" + s + "\r\n"))
	return err
}

// WriteError 回复 -ERR message\r\n
func (w *RespWriter) WriteError(msg string) error {
	_, err := w.writer.Write([]byte("-ERR " + msg + "\r\n"))
	return err
}

// WriteBulk 回复 $5\r\nhello\r\n 或 $-1\r\n
func (w *RespWriter) WriteBulk(s string) error {
	if s == "" {
		_, err := w.writer.Write([]byte("$-1\r\n"))
		return err
	}
	resp := fmt.Sprintf("$%d\r\n%s\r\n", len(s), s)
	_, err := w.writer.Write([]byte(resp))
	return err
}

// WriteInteger 回复 :10\r\n
func (w *RespWriter) WriteInteger(n int64) error {
	_, err := w.writer.Write([]byte(fmt.Sprintf(":%d\r\n", n)))
	return err
}
