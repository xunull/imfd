package geo

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/xunull/imfd/internal/media"
)

// nominatimResponse Nominatim 反查响应结构
type nominatimResponse struct {
	Address nominatimAddress `json:"address"`
}

type nominatimAddress struct {
	Country     string `json:"country"`
	State       string `json:"state"`
	Province    string `json:"province"`
	City        string `json:"city"`
	Town        string `json:"town"`
	County      string `json:"county"`
	Village     string `json:"village"`
	Suburb      string `json:"suburb"`
	CountryCode string `json:"country_code"`
}

// NominatimResolver 基于 OpenStreetMap Nominatim 的在线反查器
// Nominatim 使用政策: 最多 1 请求/秒，需设置 User-Agent
// https://operations.osmfoundation.org/policies/nominatim/
type NominatimResolver struct {
	client  *http.Client
	cache   sync.Map
	limiter chan struct{} // 速率限制
	baseURL string
}

// NewNominatimResolver 创建 Nominatim 反查器
func NewNominatimResolver() *NominatimResolver {
	return &NominatimResolver{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		limiter: make(chan struct{}, 1),
		baseURL: "https://nominatim.openstreetmap.org",
	}
}

// Name 返回提供者名称
func (r *NominatimResolver) Name() string {
	return "nominatim"
}

// Resolve 通过 Nominatim API 反查经纬度到省/市
func (r *NominatimResolver) Resolve(lat, lon float64) (*media.GeoLocation, error) {
	key := geohashKey(lat, lon)
	if cached, ok := r.cache.Load(key); ok {
		return cached.(*media.GeoLocation), nil
	}

	// 速率限制: Nominatim 要求最多 1 请求/秒
	r.limiter <- struct{}{}
	defer func() {
		go func() {
			time.Sleep(1100 * time.Millisecond)
			<-r.limiter
		}()
	}()

	url := fmt.Sprintf(
		"%s/reverse?format=json&lat=%.6f&lon=%.6f&zoom=10&accept-language=zh",
		r.baseURL, lat, lon,
	)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("User-Agent", "imfd/1.0 (media file detective)")

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Nominatim 请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Nominatim 返回状态码 %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	var result nominatimResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析 Nominatim 响应失败: %w", err)
	}

	loc := r.parseAddress(&result.Address)
	r.cache.Store(key, loc)
	return loc, nil
}

// parseAddress 从 Nominatim 地址信息中提取省/市
func (r *NominatimResolver) parseAddress(addr *nominatimAddress) *media.GeoLocation {
	loc := &media.GeoLocation{}

	// 国家
	if addr.Country != "" {
		loc.Country = addr.Country
	} else {
		loc.Country = "Unknown"
	}

	// 省份: Nominatim 中国地址的 state 通常为省
	if addr.State != "" {
		loc.Province = addr.State
	} else if addr.Province != "" {
		loc.Province = addr.Province
	} else {
		loc.Province = "Unknown"
	}

	// 城市: 优先 city，然后 town、county、village
	switch {
	case addr.City != "":
		loc.City = addr.City
	case addr.Town != "":
		loc.City = addr.Town
	case addr.County != "":
		loc.City = addr.County
	case addr.Village != "":
		loc.City = addr.Village
	default:
		loc.City = "Unknown"
	}

	return loc
}
