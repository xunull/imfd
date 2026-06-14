package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:     "imfd",
	Short:   "Image & Media File Detective - 媒体文件统计工具",
	Long:    "imfd 是一个高性能的媒体文件统计工具，用于扫描目录中的图像、视频和音频文件，提取元数据与 EXIF 信息，并进行多维统计分析。",
	Version: "dev", // 实际值由 main.go 通过 SetVersionInfo 注入
}

func init() {
	rootCmd.AddCommand(scanCmd, infoCmd, listCmd, cacheCmd, viewCmd, verifyCmd)
}

// SetVersionInfo 由 main.go 在 init 后调用，把 goreleaser 注入的 ldflags 版本信息写到 rootCmd.Version。
// 显示格式：v1.2.3 (commit abc1234, built 2026-06-13)。
// 本地 go build 时 main.go 默认值是 dev/unknown/unknown。
func SetVersionInfo(version, commit, date string) {
	rootCmd.Version = fmt.Sprintf("%s (commit %s, built %s)", version, commit, date)
}

// Execute 是包对外的入口；由 main.go 调用
func Execute() error {
	return rootCmd.Execute()
}
