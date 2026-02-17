package dimensions

import (
	"github.com/xunull/imfd/internal/media"
	"github.com/xunull/imfd/internal/stats"
)

// NewCameraModelDimension 按相机型号分组统计
func NewCameraModelDimension() stats.DimensionCounter {
	return stats.NewGroupCounter(
		"相机型号",
		func(record *media.MediaRecord) []string {
			return []string{record.GetCameraModel()}
		},
		stats.DimensionMeta{
			SortBy:    "count",
			SortOrder: "desc",
			Desc:      "按相机型号分组统计",
		},
	)
}
