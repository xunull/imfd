package dimensions

import (
	"github.com/xunull/imfd/internal/media"
	"github.com/xunull/imfd/internal/stats"
)

// NewISODimension 按 ISO 感光度分组统计
func NewISODimension() stats.DimensionCounter {
	return stats.NewGroupCounter(
		"ISO感光度",
		func(record *media.MediaRecord) []string {
			if record.Exif != nil && record.Exif.ISO != "" {
				return []string{record.Exif.ISO}
			}
			return []string{"Unknown"}
		},
		stats.DimensionMeta{SortBy: "count", SortOrder: "desc", Desc: "按ISO感光度分组统计", AppliesTo: []media.MediaType{media.TypeImage}},
	)
}

// NewApertureDimension 按光圈分组统计
func NewApertureDimension() stats.DimensionCounter {
	return stats.NewGroupCounter(
		"光圈",
		func(record *media.MediaRecord) []string {
			if record.Exif != nil && record.Exif.Aperture != "" {
				return []string{record.Exif.Aperture}
			}
			return []string{"Unknown"}
		},
		stats.DimensionMeta{SortBy: "count", SortOrder: "desc", Desc: "按光圈值分组统计", AppliesTo: []media.MediaType{media.TypeImage}},
	)
}

// NewShutterSpeedDimension 按快门速度分组统计
func NewShutterSpeedDimension() stats.DimensionCounter {
	return stats.NewGroupCounter(
		"快门速度",
		func(record *media.MediaRecord) []string {
			if record.Exif != nil && record.Exif.ShutterSpeed != "" {
				return []string{record.Exif.ShutterSpeed}
			}
			return []string{"Unknown"}
		},
		stats.DimensionMeta{SortBy: "count", SortOrder: "desc", Desc: "按快门速度分组统计", AppliesTo: []media.MediaType{media.TypeImage}},
	)
}

// NewFocalLengthDimension 按焦距分组统计
func NewFocalLengthDimension() stats.DimensionCounter {
	return stats.NewGroupCounter(
		"焦距",
		func(record *media.MediaRecord) []string {
			if record.Exif != nil && record.Exif.FocalLength != "" {
				return []string{record.Exif.FocalLength}
			}
			return []string{"Unknown"}
		},
		stats.DimensionMeta{SortBy: "count", SortOrder: "desc", Desc: "按焦距分组统计", AppliesTo: []media.MediaType{media.TypeImage}},
	)
}

// NewExposureModeDimension 按曝光模式分组统计
func NewExposureModeDimension() stats.DimensionCounter {
	return stats.NewGroupCounter(
		"曝光模式",
		func(record *media.MediaRecord) []string {
			if record.Exif != nil && record.Exif.ExposureMode != "" {
				return []string{record.Exif.ExposureMode}
			}
			return []string{"Unknown"}
		},
		stats.DimensionMeta{SortBy: "count", SortOrder: "desc", Desc: "按曝光模式分组统计", AppliesTo: []media.MediaType{media.TypeImage}},
	)
}

// NewExposureProgramDimension 按曝光程序分组统计
func NewExposureProgramDimension() stats.DimensionCounter {
	return stats.NewGroupCounter(
		"曝光程序",
		func(record *media.MediaRecord) []string {
			if record.Exif != nil && record.Exif.ExposureProgram != "" {
				return []string{record.Exif.ExposureProgram}
			}
			return []string{"Unknown"}
		},
		stats.DimensionMeta{SortBy: "count", SortOrder: "desc", Desc: "按曝光程序分组统计", AppliesTo: []media.MediaType{media.TypeImage}},
	)
}

// NewWhiteBalanceDimension 按白平衡分组统计
func NewWhiteBalanceDimension() stats.DimensionCounter {
	return stats.NewGroupCounter(
		"白平衡",
		func(record *media.MediaRecord) []string {
			if record.Exif != nil && record.Exif.WhiteBalance != "" {
				return []string{record.Exif.WhiteBalance}
			}
			return []string{"Unknown"}
		},
		stats.DimensionMeta{SortBy: "count", SortOrder: "desc", Desc: "按白平衡分组统计", AppliesTo: []media.MediaType{media.TypeImage}},
	)
}

// NewMeteringModeDimension 按测光模式分组统计
func NewMeteringModeDimension() stats.DimensionCounter {
	return stats.NewGroupCounter(
		"测光模式",
		func(record *media.MediaRecord) []string {
			if record.Exif != nil && record.Exif.MeteringMode != "" {
				return []string{record.Exif.MeteringMode}
			}
			return []string{"Unknown"}
		},
		stats.DimensionMeta{SortBy: "count", SortOrder: "desc", Desc: "按测光模式分组统计", AppliesTo: []media.MediaType{media.TypeImage}},
	)
}

// NewFlashDimension 按闪光灯状态分组统计
func NewFlashDimension() stats.DimensionCounter {
	return stats.NewGroupCounter(
		"闪光灯",
		func(record *media.MediaRecord) []string {
			if record.Exif != nil && record.Exif.Flash != "" {
				return []string{record.Exif.Flash}
			}
			return []string{"Unknown"}
		},
		stats.DimensionMeta{SortBy: "count", SortOrder: "desc", Desc: "按闪光灯状态分组统计", AppliesTo: []media.MediaType{media.TypeImage}},
	)
}
