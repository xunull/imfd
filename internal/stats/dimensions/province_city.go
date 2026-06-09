package dimensions

import (
	"github.com/xunull/imfd/internal/media"
	"github.com/xunull/imfd/internal/stats"
)

// NewProvinceDimension 按省份分组统计
func NewProvinceDimension() stats.DimensionCounter {
	return stats.NewGroupCounter(
		"省份",
		func(record *media.MediaRecord) []string {
			return []string{record.GetProvince()}
		},
		stats.DimensionMeta{
			SortBy:    "count",
			SortOrder: "desc",
			Desc:      "按拍摄地省份分组统计",
			AppliesTo: []media.MediaType{media.TypeImage},
		},
	)
}

// NewCityDimension 按城市分组统计
func NewCityDimension() stats.DimensionCounter {
	return stats.NewGroupCounter(
		"城市",
		func(record *media.MediaRecord) []string {
			return []string{record.GetCity()}
		},
		stats.DimensionMeta{
			SortBy:    "count",
			SortOrder: "desc",
			Desc:      "按拍摄地城市分组统计",
			AppliesTo: []media.MediaType{media.TypeImage},
		},
	)
}

// NewProvinceCityDimension 按省市组合分组统计
func NewProvinceCityDimension() stats.DimensionCounter {
	return stats.NewGroupCounter(
		"省/市",
		func(record *media.MediaRecord) []string {
			return []string{record.GetProvinceCity()}
		},
		stats.DimensionMeta{
			SortBy:    "count",
			SortOrder: "desc",
			Desc:      "按拍摄地省/市组合分组统计",
			AppliesTo: []media.MediaType{media.TypeImage},
		},
	)
}
