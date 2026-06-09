package walker

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/panjf2000/ants/v2"
	"github.com/xunull/imfd/internal/media"
)

// ParallelWalker 并行目录遍历器
type ParallelWalker struct {
	pool         *ants.Pool
	fileCh       chan<- string
	allowedTypes []media.MediaType
	wg           sync.WaitGroup
	errOnce      sync.Once
	err          error
}

// NewParallelWalker 创建并行遍历器
//
// allowedTypes=nil 时遍历所有媒体类型（向后兼容）。
// 非 nil 时只发送匹配的文件到 fileCh，不匹配的在 walker 层就 skip，
// 不进入 extractor 阶段，避免对 scan audio 模式下视频文件的多余 ffprobe 调用。
func NewParallelWalker(poolSize int, fileCh chan<- string, allowedTypes []media.MediaType) (*ParallelWalker, error) {
	w := &ParallelWalker{
		fileCh:       fileCh,
		allowedTypes: allowedTypes,
	}

	pool, err := ants.NewPool(poolSize, ants.WithPreAlloc(false))
	if err != nil {
		return nil, err
	}
	w.pool = pool

	return w, nil
}

// Walk 开始遍历指定目录
func (w *ParallelWalker) Walk(root string) error {
	w.wg.Add(1)
	if err := w.pool.Submit(func() {
		w.walkDir(root)
	}); err != nil {
		w.wg.Done()
		return err
	}

	w.wg.Wait()
	w.pool.Release()

	return w.err
}

func (w *ParallelWalker) walkDir(dir string) {
	defer w.wg.Done()

	entries, err := os.ReadDir(dir)
	if err != nil {
		w.errOnce.Do(func() {
			w.err = err
		})
		return
	}

	for _, entry := range entries {
		fullPath := filepath.Join(dir, entry.Name())

		if entry.IsDir() {
			w.wg.Add(1)
			err := w.pool.Submit(func() {
				w.walkDir(fullPath)
			})
			if err != nil {
				w.wg.Done()
				// 池满则同步处理
				w.wg.Add(1)
				w.walkDir(fullPath)
			}
			continue
		}

		// 跳过符号链接
		if entry.Type()&os.ModeSymlink != 0 {
			continue
		}

		// 仅发送匹配当前 scan 类型的媒体文件
		if media.IsMatchedFile(entry.Name(), w.allowedTypes) {
			w.fileCh <- fullPath
		}
	}
}
