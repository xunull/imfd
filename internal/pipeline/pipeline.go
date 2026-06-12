package pipeline

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"github.com/panjf2000/ants/v2"
	"github.com/xunull/imfd/internal/cache"
	"github.com/xunull/imfd/internal/config"
	"github.com/xunull/imfd/internal/extract"
	"github.com/xunull/imfd/internal/geo"
	"github.com/xunull/imfd/internal/media"
	"github.com/xunull/imfd/internal/output"
	"github.com/xunull/imfd/internal/stats"
	"github.com/xunull/imfd/internal/stats/dimensions"
	"github.com/xunull/imfd/internal/walker"
)

// Run 执行完整的 scan 流水线（向后兼容包装；新 caller 用 RunWithHandler）
func Run(cfg *config.Config) error {
	return RunWithHandler(cfg, nil)
}

// RunWithHandler 是 scan / list 共享的流水线。
// handler=nil 时走 scan aggregate 路径（保留向后兼容；现 scan 走这条）。
// handler 非 nil 时跳过 dimensions registry，stage 3 调 handler.Handle。
//
// stage 3 单点串行 goroutine（stdout 顺序写入天然安全）。
func RunWithHandler(cfg *config.Config, handler RecordHandler) error {
	start := time.Now()

	// 打开 cache（cfg.NoCache=true 时跳过；打开失败时降级为无 cache 模式）
	var c *cache.Cache
	if !cfg.NoCache {
		dir := cfg.CacheDir
		if dir == "" {
			dir = cache.DefaultDir()
		}
		var err error
		c, err = cache.Open(dir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: cache 不可用，降级为无 cache 模式: %v\n", err)
		}
		if c != nil {
			defer c.Close()
		}
	}

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

	// 创建统计注册中心（仅 scan 模式需要；list 模式 handler 非 nil 跳过）
	var registry *stats.Registry
	if handler == nil {
		registry = stats.NewRegistry()
		dimensions.RegisterDefaults(registry, cfg.MediaTypes)
	}

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
				record := extractAndResolve(filePath, resolver, c)
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
				spinner.SetCurrent(filepath.Base(fp))
				record := extractAndResolve(fp, resolver, c)
				spinner.SetCurrent("")
				spinner.IncExtracted()
				recordCh <- record
			})
			if err != nil {
				extractWg.Done()
				// 池满时同步执行
				spinner.SetCurrent(filepath.Base(fp))
				record := extractAndResolve(fp, resolver, c)
				spinner.SetCurrent("")
				spinner.IncExtracted()
				recordCh <- record
			}
		}

		extractWg.Wait()
		close(recordCh)
		close(extractDone)
	}()

	// 阶段 3: 单点处理 goroutine（scan: registry consume / list: handler.Handle）
	aggregateDone := make(chan struct{})
	go func() {
		for record := range recordCh {
			if handler != nil {
				_ = handler.Handle(record) // 错误不中断 pipeline；handler 自己累计
			} else {
				registry.Consume(record)
			}
		}
		close(aggregateDone)
	}()

	// 等待所有阶段完成
	if err := <-walkerDone; err != nil {
		fmt.Printf("警告: 目录遍历出现错误: %v\n", err)
	}
	<-extractDone
	<-aggregateDone

	// 阶段 4: 输出（scan 模式才印 dashboard；list 模式 handler 已在 stage 3 直接写 stdout）
	spinner.Stop()
	if handler != nil {
		return nil
	}
	report := registry.Report()
	return output.PrintReport(cfg, report, time.Since(start))
}

// extractAndResolve 提取媒体信息并做地理反查。
// c 为 nil 时跳过 cache（--no-cache 或 cache 打开失败时）。
// resolver 可能为 nil（scan audio/video 时不初始化）；此时跳过 GPS 反查。
func extractAndResolve(filePath string, resolver geo.GeoResolver, c *cache.Cache) *media.MediaRecord {
	// Cache 查找：用 os.Stat 取 mtime_ns 作为 key（与 extract 内部 stat 是两次调用，
	// stat ~1μs，相比提取 ~100ms 可忽略不计）
	if c != nil {
		if fi, err := os.Stat(filePath); err == nil {
			mtimeNs := fi.ModTime().UnixNano()
			if record, ok := c.Get(filePath, mtimeNs); ok {
				resolveGPS(record, resolver)
				return record
			}
		}
	}

	record := extract.Extract(filePath)
	resolveGPS(record, resolver)

	// 只缓存成功记录；用 record.ModTime（extract 已 stat）作为 key，避免额外 stat
	if c != nil && record.Error == nil && !record.ModTime.IsZero() {
		_ = c.Set(filePath, record.ModTime.UnixNano(), record)
	}

	return record
}

// resolveGPS 对有 GPS 坐标的图像做地理反查（offline 内存查表 ~0ms）
func resolveGPS(record *media.MediaRecord, resolver geo.GeoResolver) {
	if resolver != nil && record.HasGPS() {
		loc, err := resolver.Resolve(record.Exif.GPS.Latitude, record.Exif.GPS.Longitude)
		if err == nil {
			record.Location = loc
		}
	}
}

// needsGeoResolver 判断当前 scan 类型是否需要初始化 GPS 反查器
// GPS 来自 EXIF，只对 image 有意义；mediaTypes=nil 表示全扫，需要。
func needsGeoResolver(mediaTypes []media.MediaType) bool {
	if mediaTypes == nil {
		return true
	}
	return slices.Contains(mediaTypes, media.TypeImage)
}
