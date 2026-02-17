package geo

import (
	"fmt"
	"math"
	"sync"

	"github.com/xunull/imfd/internal/media"
)

// GeoResolver 地理反查接口
type GeoResolver interface {
	Resolve(lat, lon float64) (*media.GeoLocation, error)
}

// ChinaCity 中国城市信息
type ChinaCity struct {
	Name      string
	Province  string
	Latitude  float64
	Longitude float64
}

// OfflineResolver 离线地理反查器
// 基于预内置的城市坐标数据，通过最近距离匹配
type OfflineResolver struct {
	cities []ChinaCity
	cache  sync.Map // geohash -> *media.GeoLocation
}

// NewOfflineResolver 创建离线地理反查器
func NewOfflineResolver() *OfflineResolver {
	return &OfflineResolver{
		cities: defaultChinaCities,
	}
}

// Resolve 反查经纬度到省/市
func (r *OfflineResolver) Resolve(lat, lon float64) (*media.GeoLocation, error) {
	key := geohashKey(lat, lon)
	if cached, ok := r.cache.Load(key); ok {
		return cached.(*media.GeoLocation), nil
	}

	if len(r.cities) == 0 {
		return nil, fmt.Errorf("城市数据库为空")
	}

	minDist := math.MaxFloat64
	var nearest ChinaCity

	for _, city := range r.cities {
		dist := haversineDistance(lat, lon, city.Latitude, city.Longitude)
		if dist < minDist {
			minDist = dist
			nearest = city
		}
	}

	// 如果最近城市距离超过 200km，认为不在中国境内
	if minDist > 200 {
		loc := &media.GeoLocation{
			Country:  "海外",
			Province: "海外",
			City:     "海外",
		}
		r.cache.Store(key, loc)
		return loc, nil
	}

	loc := &media.GeoLocation{
		Country:  "中国",
		Province: nearest.Province,
		City:     nearest.Name,
	}
	r.cache.Store(key, loc)
	return loc, nil
}

// geohashKey 将经纬度量化为缓存键（精度约 1km）
func geohashKey(lat, lon float64) string {
	latQ := math.Round(lat*100) / 100
	lonQ := math.Round(lon*100) / 100
	return fmt.Sprintf("%.2f,%.2f", latQ, lonQ)
}

// haversineDistance 计算两点间距离（km）
func haversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadius = 6371.0

	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	deltaLat := (lat2 - lat1) * math.Pi / 180
	deltaLon := (lon2 - lon1) * math.Pi / 180

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(deltaLon/2)*math.Sin(deltaLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadius * c
}
