package output

import (
	"time"

	"github.com/xunull/imfd/internal/config"
	"github.com/xunull/imfd/internal/stats"
)

// Printer 输出接口
type Printer interface {
	Print(report stats.StatsReport) error
}

// PrintReport 根据配置选择输出方式打印报告。
//
// 路由：
//   - FormatJSON → JSON only
//   - FormatBoth → dashboard (or legacy table) + JSON
//   - 默认（table）→ dashboard，除非 LegacyTable=true 则走 go-pretty 老表
func PrintReport(cfg *config.Config, report stats.StatsReport, duration time.Duration) error {
	switch cfg.OutputFormat {
	case config.FormatJSON:
		return NewJSONPrinter().Print(report)

	case config.FormatBoth:
		if err := printTextReport(cfg, report, duration); err != nil {
			return err
		}
		return NewJSONPrinter().Print(report)

	default:
		return printTextReport(cfg, report, duration)
	}
}

// printTextReport 是 table/dashboard 二选一
func printTextReport(cfg *config.Config, report stats.StatsReport, duration time.Duration) error {
	if cfg.LegacyTable {
		return NewTablePrinter().Print(report)
	}
	return NewDashboardPrinter(nil, cfg.Verbose, cfg.Dir, duration).Print(report)
}
