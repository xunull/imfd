package dimensions

import (
	"github.com/xunull/imfd/internal/media"
	"github.com/xunull/imfd/internal/stats"
)

// NewLensModelDimension 按镜头型号分组统计
func NewLensModelDimension() stats.DimensionCounter {
	return stats.NewGroupCounter(
		"镜头型号",
		func(record *media.MediaRecord) []string {
			return []string{record.GetLensModel()}
		},
		stats.DimensionMeta{
			SortBy:    "count",
			SortOrder: "desc",
			Desc:      "按镜头型号分组统计",
		},
	)
}
