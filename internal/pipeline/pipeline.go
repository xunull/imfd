package pipeline

import (
	"fmt"
	"sync"

	"github.com/panjf2000/ants/v2"
	"github.com/xunull/imfd/internal/config"
	"github.com/xunull/imfd/internal/extract"
	"github.com/xunull/imfd/internal/geo"
	"github.com/xunull/imfd/internal/media"
	"github.com/xunull/imfd/internal/output"
	"github.com/xunull/imfd/internal/stats"
	"github.com/xunull/imfd/internal/stats/dimensions"
	"github.com/xunull/imfd/internal/walker"
)

// Run 执行完整的扫描-提取-统计-输出流水线
func Run(cfg *config.Config) error {
	// 创建通道
	fileCh := make(chan string, cfg.ChannelSize)
	recordCh := make(chan *media.MediaRecord, cfg.ChannelSize)

	// 创建地理反查器
	resolver := geo.NewOfflineResolver()

	// 创建统计注册中心并注册默认维度
	registry := stats.NewRegistry()
	dimensions.RegisterDefaults(registry)

	// 阶段 1: 启动并行目录遍历
	walkerDone := make(chan error, 1)
	go func() {
		w, err := walker.NewParallelWalker(cfg.Workers, fileCh)
		if err != nil {
			walkerDone <- fmt.Errorf("创建遍历器失败: %w", err)
			close(fileCh)
			return
		}
		err = w.Walk(cfg.Dir)
		close(fileCh)
		walkerDone <- err
	}()

	// 阶段 2: 启动并行媒体提取
	extractDone := make(chan struct{})
	go func() {
		var extractWg sync.WaitGroup

		extractPool, err := ants.NewPool(cfg.Extractors, ants.WithPreAlloc(false))
		if err != nil {
			fmt.Printf("警告: 创建提取池失败，将使用串行提取: %v\n", err)
			for filePath := range fileCh {
				record := extractAndResolve(filePath, resolver)
				recordCh <- record
			}
			close(recordCh)
			close(extractDone)
			return
		}
		defer extractPool.Release()

		for filePath := range fileCh {
			extractWg.Add(1)
			fp := filePath
			err := extractPool.Submit(func() {
				defer extractWg.Done()
				record := extractAndResolve(fp, resolver)
				recordCh <- record
			})
			if err != nil {
				extractWg.Done()
				// 池满时同步执行
				record := extractAndResolve(fp, resolver)
				recordCh <- record
			}
		}

		extractWg.Wait()
		close(recordCh)
		close(extractDone)
	}()

	// 阶段 3: 单点聚合 goroutine
	aggregateDone := make(chan struct{})
	go func() {
		for record := range recordCh {
			registry.Consume(record)
		}
		close(aggregateDone)
	}()

	// 等待所有阶段完成
	if err := <-walkerDone; err != nil {
		fmt.Printf("警告: 目录遍历出现错误: %v\n", err)
	}
	<-extractDone
	<-aggregateDone

	// 阶段 4: 输出统计结果
	report := registry.Report()
	return output.PrintReport(cfg, report)
}

// extractAndResolve 提取媒体信息并做地理反查
func extractAndResolve(filePath string, resolver *geo.OfflineResolver) *media.MediaRecord {
	record := extract.Extract(filePath)

	// 对有 GPS 信息的记录做地理反查
	if record.HasGPS() {
		loc, err := resolver.Resolve(record.Exif.GPS.Latitude, record.Exif.GPS.Longitude)
		if err == nil {
			record.Location = loc
		}
	}

	return record
}
