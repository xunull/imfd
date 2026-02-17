package main

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "imfd",
	Short: "Image & Media File Detective - 媒体文件统计工具",
	Long:  "imfd 是一个高性能的媒体文件统计工具，用于扫描目录中的图像和视频文件，提取元数据与 EXIF 信息，并进行多维统计分析。",
}

func init() {
	rootCmd.AddCommand(scanCmd)
}
