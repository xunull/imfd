package dimensions

import (
	"slices"

	"github.com/xunull/imfd/internal/media"
	"github.com/xunull/imfd/internal/stats"
)

// RegisterDefaults 注册所有默认统计维度。
//
// activeTypes 控制按 scan 类型过滤：
//   - nil（默认 scan / scan all）：全部维度都注册
//   - 非 nil（scan audio/image/video）：跳过 AppliesTo 与 activeTypes 无交集的维度
//
// 维度的 AppliesTo 为空意味着「适用所有类型」——这些维度永远注册。
// 这是向后兼容设计：现有 17 个图像/通用维度都不带 AppliesTo，默认行为不变。
func RegisterDefaults(registry *stats.Registry, activeTypes []media.MediaType) {
	candidates := []stats.DimensionCounter{
		NewMediaTypeDimension(),
		NewCameraModelDimension(),
		NewLensModelDimension(),
		NewTimeBucketDimension(),
		NewProvinceDimension(),
		NewCityDimension(),
		NewProvinceCityDimension(),
		NewISODimension(),
		NewApertureDimension(),
		NewShutterSpeedDimension(),
		NewFocalLengthDimension(),
		NewExposureModeDimension(),
		NewExposureProgramDimension(),
		NewWhiteBalanceDimension(),
		NewMeteringModeDimension(),
		NewFlashDimension(),
		// 音频维度（均标 AppliesTo=[TypeAudio]）
		NewAudioCodecDimension(),
		NewAudioBitrateDimension(),
		NewAudioSampleRateDimension(),
		NewAudioChannelsDimension(),
		NewAudioDurationBucketDimension(),
	}

	for _, dim := range candidates {
		if shouldRegister(dim.Result().Meta.AppliesTo, activeTypes) {
			registry.Register(dim)
		}
	}
}

// shouldRegister 判断一个维度是否应该在当前 scan 类型下注册。
//
//	dimAppliesTo=nil  | activeTypes=任意 | 注册   （维度声称适用所有类型）
//	dimAppliesTo=非空 | activeTypes=nil  | 注册   （未限定 scan 类型，全开）
//	dimAppliesTo=非空 | activeTypes=非空 | 仅当两者有交集时注册
func shouldRegister(dimAppliesTo, activeTypes []media.MediaType) bool {
	if dimAppliesTo == nil {
		return true
	}
	if activeTypes == nil {
		return true
	}
	for _, t := range dimAppliesTo {
		if slices.Contains(activeTypes, t) {
			return true
		}
	}
	return false
}
