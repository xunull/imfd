package pipeline

import (
	"fmt"
	"slices"
	"sync"
	"time"

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
	start := time.Now()

	// 启动 stderr spinner（无 TTY / NO_COLOR / IMFD_NO_SPINNER 时是 no-op）
	spinner := output.NewSpinner(nil)
	spinner.Start()
	defer spinner.Stop()

	// 创建通道
	fileCh := make(chan string, cfg.ChannelSize)
	recordCh := make(chan *media.MediaRecord, cfg.ChannelSize)

	// 只有扫描 image 时才需要 GPS 反查（GPS 来自 EXIF，仅图像有）。
	// scan audio / scan video 完全跳过 resolver 初始化。
	// 注意：故意不再 println "使用 GPS 反查方式" —— dashboard 输出由
	// printer 完全控制，pipeline 不应往 stdout 印散落的状态行。
	var resolver geo.GeoResolver
	if needsGeoResolver(cfg.MediaTypes) {
		geoProvider, err := geo.ParseGeoProvider(cfg.GeoProvider)
		if err != nil {
			return err
		}
		resolver, err = geo.NewResolver(geoProvider)
		if err != nil {
			return fmt.Errorf("创建地理反查器失败: %w", err)
		}
	}

	// 创建统计注册中心并注册默认维度
	registry := stats.NewRegistry()
	dimensions.RegisterDefaults(registry, cfg.MediaTypes)

	// 阶段 1: 启动并行目录遍历
	walkerDone := make(chan error, 1)
	go func() {
		w, err := walker.NewParallelWalker(cfg.Workers, fileCh, cfg.MediaTypes)
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
			spinner.IncFiles()
			extractWg.Add(1)
			fp := filePath
			err := extractPool.Submit(func() {
				defer extractWg.Done()
				record := extractAndResolve(fp, resolver)
				spinner.IncExtracted()
				recordCh <- record
			})
			if err != nil {
				extractWg.Done()
				// 池满时同步执行
				record := extractAndResolve(fp, resolver)
				spinner.IncExtracted()
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
	// 先 Stop spinner 清行，再让 dashboard 从干净的 stdout 行开始
	spinner.Stop()
	report := registry.Report()
	return output.PrintReport(cfg, report, time.Since(start))
}

// extractAndResolve 提取媒体信息并做地理反查
// resolver 可能为 nil（scan audio/video 时不初始化）；此时跳过 GPS 反查
func extractAndResolve(filePath string, resolver geo.GeoResolver) *media.MediaRecord {
	record := extract.Extract(filePath)

	// 对有 GPS 信息的记录做地理反查
	if resolver != nil && record.HasGPS() {
		loc, err := resolver.Resolve(record.Exif.GPS.Latitude, record.Exif.GPS.Longitude)
		if err == nil {
			record.Location = loc
		}
	}

	return record
}

// needsGeoResolver 判断当前 scan 类型是否需要初始化 GPS 反查器
// GPS 来自 EXIF，只对 image 有意义；mediaTypes=nil 表示全扫，需要。
func needsGeoResolver(mediaTypes []media.MediaType) bool {
	if mediaTypes == nil {
		return true
	}
	return slices.Contains(mediaTypes, media.TypeImage)
}
