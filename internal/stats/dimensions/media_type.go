package dimensions

import (
	"github.com/xunull/imfd/internal/media"
	"github.com/xunull/imfd/internal/stats"
)

// NewMediaTypeDimension 按媒体类型分组统计
func NewMediaTypeDimension() stats.DimensionCounter {
	return stats.NewGroupCounter(
		"媒体类型",
		func(record *media.MediaRecord) []string {
			return []string{record.Type.String()}
		},
		stats.DimensionMeta{
			SortBy:    "count",
			SortOrder: "desc",
			Desc:      "按媒体类型(图像/视频)分组统计",
		},
	)
}
