package dimensions

import (
	"github.com/xunull/imfd/internal/media"
	"github.com/xunull/imfd/internal/stats"
	"github.com/xunull/imfd/internal/timebucket"
)

// NewTimeBucketDimension 按时间段分组统计
func NewTimeBucketDimension() stats.DimensionCounter {
	return stats.NewGroupCounter(
		"拍摄时间段",
		func(record *media.MediaRecord) []string {
			if !record.HasCaptureTime {
				return []string{"Unknown"}
			}
			return []string{timebucket.Classify(record.CaptureTime)}
		},
		stats.DimensionMeta{
			SortBy:    "key",
			SortOrder: "asc",
			Desc:      "按拍摄时间段(凌晨/上午/中午/下午/晚上/半夜)分组统计",
		},
	)
}
