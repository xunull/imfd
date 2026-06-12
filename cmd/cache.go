package cmd

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/xunull/imfd/internal/cache"
)

var flagCacheCleanOlderThan string

var cacheCmd = &cobra.Command{
	Use:           "cache",
	Short:         "管理 imfd 元数据 cache",
	SilenceUsage:  true,
	SilenceErrors: true,
}

var cacheStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "显示 cache 数据库统计信息",
	RunE:  runCacheStats,
}

var cacheCleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "删除超过指定时间的 cache 条目",
	Long: `删除 cached_at 超过指定时间的旧条目。

支持格式：Nd (天), Nh (小时), 例: --older-than 90d`,
	RunE: runCacheClean,
}

var cacheClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "清空所有 cache 条目",
	RunE:  runCacheClear,
}

func init() {
	cacheCleanCmd.Flags().StringVar(&flagCacheCleanOlderThan, "older-than", "90d",
		"删除早于此时长的条目（格式: Nd / Nh，例 90d）")
	cacheCmd.AddCommand(cacheStatsCmd, cacheCleanCmd, cacheClearCmd)
}

func runCacheStats(cmd *cobra.Command, _ []string) error {
	c, err := openCacheOrFail()
	if err != nil {
		return err
	}
	defer c.Close()

	s, err := c.GetStats()
	if err != nil {
		return fmt.Errorf("读取 cache 统计失败: %w", err)
	}

	fmt.Printf("Cache DB:  %s\n", s.Path)
	fmt.Printf("Entries:   %s\n", fmtCount(s.Entries))
	fmt.Printf("Size:      %s\n", fmtBytes(s.SizeBytes))
	if s.Entries > 0 && !s.OldestAt.IsZero() {
		age := int(time.Since(s.OldestAt).Hours() / 24)
		fmt.Printf("Oldest:    %s (%d days ago)\n", s.OldestAt.Format("2006-01-02"), age)
	}
	return nil
}

func runCacheClean(cmd *cobra.Command, _ []string) error {
	dur, err := parseCacheDuration(flagCacheCleanOlderThan)
	if err != nil {
		return err
	}

	c, err := openCacheOrFail()
	if err != nil {
		return err
	}
	defer c.Close()

	n, err := c.Clean(dur)
	if err != nil {
		return fmt.Errorf("清理 cache 失败: %w", err)
	}
	fmt.Printf("已删除 %s 条记录\n", fmtCount(n))
	return nil
}

func runCacheClear(cmd *cobra.Command, _ []string) error {
	c, err := openCacheOrFail()
	if err != nil {
		return err
	}
	defer c.Close()

	n, err := c.Clear()
	if err != nil {
		return fmt.Errorf("清空 cache 失败: %w", err)
	}
	fmt.Printf("已清空 %s 条记录\n", fmtCount(n))
	return nil
}

func openCacheOrFail() (*cache.Cache, error) {
	c, err := cache.Open(cache.DefaultDir())
	if err != nil {
		return nil, fmt.Errorf("打开 cache 失败: %w", err)
	}
	return c, nil
}

// parseCacheDuration parses "Nd" (days) or falls back to time.ParseDuration.
func parseCacheDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if strings.HasSuffix(s, "d") {
		n, err := strconv.Atoi(strings.TrimSuffix(s, "d"))
		if err != nil || n <= 0 {
			return 0, fmt.Errorf("无效的时间格式 %q：支持 Nd（天）或 time.ParseDuration 格式（如 24h）", s)
		}
		return time.Duration(n) * 24 * time.Hour, nil
	}
	return time.ParseDuration(s)
}

func fmtBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func fmtCount(n int64) string {
	s := strconv.FormatInt(n, 10)
	if len(s) <= 3 {
		return s
	}
	var b strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(c)
	}
	return b.String()
}
