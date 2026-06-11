package output

import (
	"fmt"
	"io"
	"os"
)

// ListPrinter 是 imfd list 命令的 stdout 写入层。
//
// 两种模式：
//   - plain: 每行一个路径，LF 分隔（用户读 / 简单 pipe）
//   - NUL:   每个路径后 \0 分隔（xargs -0 友好；filenames 含 \n 也安全）
//
// stdout 顺序由 pipeline 的 stage 3 单 goroutine 保证；ListPrinter 本身不加锁。
type ListPrinter struct {
	out  io.Writer
	null bool // -0 / --print0 时 true
}

// NewListPrinter 构造 printer。out=nil 用 os.Stdout。
func NewListPrinter(out io.Writer, useNull bool) *ListPrinter {
	if out == nil {
		out = os.Stdout
	}
	return &ListPrinter{out: out, null: useNull}
}

// Print writes one path with the configured separator.
func (p *ListPrinter) Print(path string) error {
	sep := "\n"
	if p.null {
		sep = "\x00"
	}
	_, err := fmt.Fprint(p.out, path, sep)
	return err
}
