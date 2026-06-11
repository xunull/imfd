package pipeline

import (
	"github.com/xunull/imfd/internal/media"
)

// RecordHandler 是 pipeline 阶段 3 的可替换处理器。
//
// scan 命令传 RegistryHandler 走 dimensions aggregation；
// list 命令传 FilterPrintHandler 走 filter + stdout print。
//
// 同 pipeline 同时只用一个 handler 实例。handler 在单点 goroutine 串行被调用
// （不需要并发安全；stdout 写入也天然顺序）。
type RecordHandler interface {
	// Handle 接收一条 record。返回 error 不中断 pipeline；caller 决定如何累计。
	Handle(*media.MediaRecord) error
}

// HandlerFunc 让普通函数也能当 RecordHandler 用。
type HandlerFunc func(*media.MediaRecord) error

func (f HandlerFunc) Handle(r *media.MediaRecord) error { return f(r) }
