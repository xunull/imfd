package dimensions

import "github.com/xunull/imfd/internal/stats"

// RegisterDefaults 注册所有默认统计维度
func RegisterDefaults(registry *stats.Registry) {
	registry.Register(NewMediaTypeDimension())
	registry.Register(NewCameraModelDimension())
	registry.Register(NewLensModelDimension())
	registry.Register(NewTimeBucketDimension())
	registry.Register(NewProvinceDimension())
	registry.Register(NewCityDimension())
	registry.Register(NewProvinceCityDimension())
	registry.Register(NewISODimension())
	registry.Register(NewApertureDimension())
	registry.Register(NewShutterSpeedDimension())
	registry.Register(NewFocalLengthDimension())
	registry.Register(NewExposureModeDimension())
	registry.Register(NewExposureProgramDimension())
	registry.Register(NewWhiteBalanceDimension())
	registry.Register(NewMeteringModeDimension())
	registry.Register(NewFlashDimension())
}
