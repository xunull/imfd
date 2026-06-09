package stats

import (
	"github.com/xunull/imfd/internal/media"
)

// Bucket 统计桶：一个分组键及其计数
type Bucket struct {
	Key   string `json:"key"`
	Count int    `json:"count"`
}

// DimensionMeta 维度元信息
type DimensionMeta struct {
	Unit      string `json:"unit,omitempty"`
	SortBy    string `json:"sort_by,omitempty"` // "count" or "key"
	SortOrder string `json:"sort_order,omitempty"` // "asc" or "desc"
	Desc      string `json:"desc,omitempty"`

	// AppliesTo 维度适用于哪些媒体类型；nil 表示适用所有类型（默认）。
	// 用法：scan audio 时 RegisterDefaults 会跳过 AppliesTo 与 [TypeAudio] 无交集的维度。
	// 例如音频专属维度（采样率/比特率等）填 []MediaType{TypeAudio}。
	AppliesTo []media.MediaType `json:"applies_to,omitempty"`
}

// DimensionResult 一个维度的统计结果
type DimensionResult struct {
	DimensionName string        `json:"dimension_name"`
	Buckets       []Bucket      `json:"buckets"`
	Meta          DimensionMeta `json:"meta,omitempty"`
}

// Totals 总量统计
//
// 契约：TotalCount = ImageCount + VideoCount + AudioCount + ...
// 未来新增媒体类型会继续加 *Count 字段。下游 JSON 消费者请用 TotalCount，
// 不要用 ImageCount + VideoCount 推算总数。
type Totals struct {
	ImageCount int `json:"image_count"`
	VideoCount int `json:"video_count"`
	AudioCount int `json:"audio_count"`
	TotalCount int `json:"total_count"`
	ErrorCount int `json:"error_count"`
}

// StatsReport 完整统计报告
type StatsReport struct {
	Totals     Totals            `json:"totals"`
	Dimensions []DimensionResult `json:"dimensions"`
}

// DimensionCounter 维度统计器接口
// 每个统计维度实现该接口，通过注册中心自动装配
type DimensionCounter interface {
	// Name 维度名称
	Name() string
	// Consume 消费一条媒体记录
	Consume(record *media.MediaRecord)
	// Result 返回该维度的统计结果
	Result() DimensionResult
	// Reset 重置统计（用于测试）
	Reset()
}

// KeyExtractor 键提取函数类型
// 从 MediaRecord 中提取一个或多个分组键
type KeyExtractor func(record *media.MediaRecord) []string

// GroupCounter 通用分组计数器
// 通过 KeyExtractor 实现低代码新增统计维度
type GroupCounter struct {
	name      string
	extractor KeyExtractor
	counts    map[string]int
	meta      DimensionMeta
}

// NewGroupCounter 创建通用分组计数器
func NewGroupCounter(name string, extractor KeyExtractor, meta DimensionMeta) *GroupCounter {
	return &GroupCounter{
		name:      name,
		extractor: extractor,
		counts:    make(map[string]int),
		meta:      meta,
	}
}

func (g *GroupCounter) Name() string {
	return g.name
}

func (g *GroupCounter) Consume(record *media.MediaRecord) {
	keys := g.extractor(record)
	for _, k := range keys {
		g.counts[k]++
	}
}

func (g *GroupCounter) Result() DimensionResult {
	buckets := make([]Bucket, 0, len(g.counts))
	for k, v := range g.counts {
		buckets = append(buckets, Bucket{Key: k, Count: v})
	}
	return DimensionResult{
		DimensionName: g.name,
		Buckets:       buckets,
		Meta:          g.meta,
	}
}

func (g *GroupCounter) Reset() {
	g.counts = make(map[string]int)
}

// Registry 统计维度注册中心
type Registry struct {
	counters []DimensionCounter
	totals   Totals
}

// NewRegistry 创建注册中心
func NewRegistry() *Registry {
	return &Registry{
		counters: make([]DimensionCounter, 0),
	}
}

// Register 注册一个统计维度
func (r *Registry) Register(counter DimensionCounter) {
	r.counters = append(r.counters, counter)
}

// Consume 消费一条媒体记录（分发给所有维度统计器）
func (r *Registry) Consume(record *media.MediaRecord) {
	if record.Error != nil {
		r.totals.ErrorCount++
		return
	}

	r.totals.TotalCount++
	switch record.Type {
	case media.TypeImage:
		r.totals.ImageCount++
	case media.TypeVideo:
		r.totals.VideoCount++
	case media.TypeAudio:
		r.totals.AudioCount++
	}

	for _, c := range r.counters {
		c.Consume(record)
	}
}

// Report 生成完整统计报告
func (r *Registry) Report() StatsReport {
	dims := make([]DimensionResult, 0, len(r.counters))
	for _, c := range r.counters {
		dims = append(dims, c.Result())
	}
	return StatsReport{
		Totals:     r.totals,
		Dimensions: dims,
	}
}
