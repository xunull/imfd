package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/xunull/imfd/internal/config"
	"github.com/xunull/imfd/internal/pipeline"
)

var (
	flagDir          string
	flagWorkers      int
	flagExtractors   int
	flagOutputFormat string
	flagChannelSize  int
	flagGeoProvider  string
)

var scanCmd = &cobra.Command{
	Use:   "scan [directory]",
	Short: "扫描目录中的图像和视频文件并输出统计结果",
	Long:  "递归扫描指定目录，提取图像 EXIF 信息和视频元数据，按多个维度进行统计并输出结果。",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runScan,
}

func init() {
	scanCmd.Flags().StringVarP(&flagDir, "dir", "d", ".", "要扫描的目录路径")
	scanCmd.Flags().IntVarP(&flagWorkers, "workers", "w", 8, "目录遍历并发数")
	scanCmd.Flags().IntVarP(&flagExtractors, "extractors", "e", 16, "媒体提取并发数")
	scanCmd.Flags().StringVarP(&flagOutputFormat, "format", "f", "table", "输出格式: table, json, both")
	scanCmd.Flags().IntVar(&flagChannelSize, "channel-size", 1024, "内部通道缓冲大小")
	scanCmd.Flags().StringVarP(&flagGeoProvider, "geo-provider", "g", "offline", "GPS 反查方式: offline(离线), nominatim(OpenStreetMap在线)")
}

func runScan(cmd *cobra.Command, args []string) error {
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
	}

	if err := cfg.Validate(); err != nil {
		return err
	}

	return pipeline.Run(cfg)
}
