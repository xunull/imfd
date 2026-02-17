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
	pool    *ants.Pool
	fileCh  chan<- string
	wg      sync.WaitGroup
	errOnce sync.Once
	err     error
}

// NewParallelWalker 创建并行遍历器
func NewParallelWalker(poolSize int, fileCh chan<- string) (*ParallelWalker, error) {
	w := &ParallelWalker{
		fileCh: fileCh,
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

		// 仅发送媒体文件
		if media.IsMediaFile(entry.Name()) {
			w.fileCh <- fullPath
		}
	}
}
