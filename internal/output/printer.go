package output

import (
	"github.com/xunull/imfd/internal/config"
	"github.com/xunull/imfd/internal/stats"
)

// Printer 输出接口
type Printer interface {
	Print(report stats.StatsReport) error
}

// PrintReport 根据配置选择输出方式打印报告
func PrintReport(cfg *config.Config, report stats.StatsReport) error {
	switch cfg.OutputFormat {
	case config.FormatJSON:
		return NewJSONPrinter().Print(report)
	case config.FormatBoth:
		if err := NewTablePrinter().Print(report); err != nil {
			return err
		}
		return NewJSONPrinter().Print(report)
	default:
		return NewTablePrinter().Print(report)
	}
}
