package output

import (
	"fmt"
	"os"
	"sort"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/xunull/imfd/internal/stats"
)

// TablePrinter 终端表格输出
type TablePrinter struct{}

// NewTablePrinter 创建表格输出器
func NewTablePrinter() *TablePrinter {
	return &TablePrinter{}
}

func (p *TablePrinter) Print(report stats.StatsReport) error {
	p.printTotals(report.Totals)
	for _, dim := range report.Dimensions {
		p.printDimension(dim)
	}
	return nil
}

func (p *TablePrinter) printTotals(totals stats.Totals) {
	fmt.Println()
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetTitle("总量统计")
	t.AppendHeader(table.Row{"类型", "数量"})
	t.AppendRows([]table.Row{
		{"图像", totals.ImageCount},
		{"视频", totals.VideoCount},
		{"总计", totals.TotalCount},
		{"错误", totals.ErrorCount},
	})
	t.SetStyle(table.StyleRounded)
	t.Render()
	fmt.Println()
}

func (p *TablePrinter) printDimension(dim stats.DimensionResult) {
	if len(dim.Buckets) == 0 {
		return
	}

	buckets := make([]stats.Bucket, len(dim.Buckets))
	copy(buckets, dim.Buckets)

	sortBuckets(buckets, dim.Meta)

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetTitle(dim.DimensionName)
	t.AppendHeader(table.Row{dim.DimensionName, "数量"})
	for _, b := range buckets {
		t.AppendRow(table.Row{b.Key, b.Count})
	}
	t.SetStyle(table.StyleRounded)
	t.Render()
	fmt.Println()
}

func sortBuckets(buckets []stats.Bucket, meta stats.DimensionMeta) {
	sortBy := meta.SortBy
	sortOrder := meta.SortOrder
	if sortBy == "" {
		sortBy = "count"
	}
	if sortOrder == "" {
		sortOrder = "desc"
	}

	sort.Slice(buckets, func(i, j int) bool {
		if sortBy == "key" {
			if sortOrder == "asc" {
				return buckets[i].Key < buckets[j].Key
			}
			return buckets[i].Key > buckets[j].Key
		}
		if sortOrder == "asc" {
			return buckets[i].Count < buckets[j].Count
		}
		return buckets[i].Count > buckets[j].Count
	})
}
