package output

import (
	"encoding/json"
	"fmt"

	"github.com/xunull/imfd/internal/stats"
)

// JSONPrinter JSON 输出
type JSONPrinter struct{}

// NewJSONPrinter 创建 JSON 输出器
func NewJSONPrinter() *JSONPrinter {
	return &JSONPrinter{}
}

func (p *JSONPrinter) Print(report stats.StatsReport) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("JSON 序列化失败: %w", err)
	}
	fmt.Println(string(data))
	return nil
}
