package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/xunull/imfd/internal/config"
	"github.com/xunull/imfd/internal/media"
	"github.com/xunull/imfd/internal/pipeline"
)

var (
	flagDir          string
	flagWorkers      int
	flagExtractors   int
	flagOutputFormat string
	flagChannelSize  int
	flagGeoProvider  string
	flagVerbose      bool
	flagLegacyTable  bool
)

// scanRunner 是 pipeline.Run 的接缝，便于在测试里注入 fake runner
// 验证子命令路由是否传出正确的 cfg.MediaTypes。
var scanRunner = pipeline.Run

// scanCmd 父命令；不带子命令时（imfd scan /dir）扫所有媒体类型，向后兼容
var scanCmd = &cobra.Command{
	Use:   "scan [directory]",
	Short: "扫描目录中的图像、视频和音频文件并输出统计结果",
	Long: `递归扫描指定目录，提取图像 EXIF、视频元数据和音频元数据
（编解码器/采样率/比特率/声道/时长等），按多个维度进行统计并输出结果。

不带子命令时扫描全部媒体类型（等同于 ` + "`scan all`" + `）；
可用 ` + "`scan audio` / `scan image` / `scan video`" + ` 仅扫指定类型。`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runScanWithTypes(args, nil)
	},
}

// scanAllCmd 显式 "all" 子命令；等价于裸 scan
var scanAllCmd = &cobra.Command{
	Use:   "all [directory]",
	Short: "扫描全部媒体类型（图像 + 视频 + 音频）",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runScanWithTypes(args, nil)
	},
}

var scanAudioCmd = &cobra.Command{
	Use:   "audio [directory]",
	Short: "仅扫描音频文件",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runScanWithTypes(args, []media.MediaType{media.TypeAudio})
	},
}

var scanImageCmd = &cobra.Command{
	Use:   "image [directory]",
	Short: "仅扫描图像文件",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runScanWithTypes(args, []media.MediaType{media.TypeImage})
	},
}

var scanVideoCmd = &cobra.Command{
	Use:   "video [directory]",
	Short: "仅扫描视频文件",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runScanWithTypes(args, []media.MediaType{media.TypeVideo})
	},
}

func init() {
	// PersistentFlags 让 -d/-w/-e/-f/--channel-size/-g 自动继承到所有子命令
	scanCmd.PersistentFlags().StringVarP(&flagDir, "dir", "d", ".", "要扫描的目录路径")
	scanCmd.PersistentFlags().IntVarP(&flagWorkers, "workers", "w", 8, "目录遍历并发数")
	scanCmd.PersistentFlags().IntVarP(&flagExtractors, "extractors", "e", 0, "媒体提取并发数 (默认: CPU核心数x5)")
	scanCmd.PersistentFlags().StringVarP(&flagOutputFormat, "format", "f", "table", "输出格式: table, json, both")
	scanCmd.PersistentFlags().IntVar(&flagChannelSize, "channel-size", 1024, "内部通道缓冲大小")
	scanCmd.PersistentFlags().StringVarP(&flagGeoProvider, "geo-provider", "g", "offline", "GPS 反查方式: offline(离线), nominatim(OpenStreetMap在线)")
	scanCmd.PersistentFlags().BoolVarP(&flagVerbose, "verbose", "v", false, "展开全 Unknown 的维度（默认折叠）")
	scanCmd.PersistentFlags().BoolVar(&flagLegacyTable, "legacy-table", false, "使用旧版 go-pretty 表格输出（默认: dashboard）")

	scanCmd.AddCommand(scanAllCmd, scanAudioCmd, scanImageCmd, scanVideoCmd)
}

// runScanWithTypes 是所有 scan 变体的公共执行体；mediaTypes=nil 表示全扫
func runScanWithTypes(args []string, mediaTypes []media.MediaType) error {
	dir := flagDir
	if len(args) > 0 {
		dir = args[0]
	}

	info, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("无法访问目录 %s: %w", dir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s 不是一个目录", dir)
	}

	cfg := &config.Config{
		Dir:          dir,
		Workers:      flagWorkers,
		Extractors:   flagExtractors,
		OutputFormat: config.ParseOutputFormat(flagOutputFormat),
		ChannelSize:  flagChannelSize,
		GeoProvider:  flagGeoProvider,
		MediaTypes:   mediaTypes,
		Verbose:      flagVerbose,
		LegacyTable:  flagLegacyTable,
	}

	if err := cfg.Validate(); err != nil {
		return err
	}

	return scanRunner(cfg)
}
