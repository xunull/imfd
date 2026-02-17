package geo

import (
	"fmt"

	"github.com/xunull/imfd/internal/media"
)

// GeoProvider 地理反查提供者类型
type GeoProvider string

const (
	ProviderOffline   GeoProvider = "offline"
	ProviderNominatim GeoProvider = "nominatim"
)

// ParseGeoProvider 解析地理反查提供者字符串
func ParseGeoProvider(s string) (GeoProvider, error) {
	switch s {
	case "offline":
		return ProviderOffline, nil
	case "nominatim":
		return ProviderNominatim, nil
	default:
		return "", fmt.Errorf("不支持的地理反查提供者: %q（可选: offline, nominatim）", s)
	}
}

// GeoResolver 地理反查接口
// 所有 GPS 反查实现必须实现该接口
type GeoResolver interface {
	// Resolve 将经纬度反查为地理位置
	Resolve(lat, lon float64) (*media.GeoLocation, error)
	// Name 返回提供者名称
	Name() string
}

// NewResolver 根据提供者类型创建对应的 GeoResolver
func NewResolver(provider GeoProvider) (GeoResolver, error) {
	switch provider {
	case ProviderOffline:
		return NewOfflineResolver(), nil
	case ProviderNominatim:
		return NewNominatimResolver(), nil
	default:
		return nil, fmt.Errorf("不支持的地理反查提供者: %q", provider)
	}
}
