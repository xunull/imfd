package stats

import "github.com/xunull/imfd/internal/media"

// NewFieldDimension 通用「按某字段分组」维度的工厂函数。
//
// 复制 exif_fields.go 9 个函数里那段一样的 nil-check + 空串-check + Unknown
// 兜底逻辑——以工厂函数集中表达，使用方只需描述"维度名 + 描述 + 提取器"。
//
// 例：
//
//   stats.NewFieldDimension("音频编解码器", "按音频编解码器分组统计",
//       func(r *media.MediaRecord) string {
//           if r.Audio == nil { return "" }
//           return r.Audio.Codec
//       }, media.TypeAudio)
//
// 提取器返回 "" 时桶 key 自动落入 "Unknown"。
//
// appliesTo 可变参数声明维度适用于哪些媒体类型；不传等价于「适用所有类型」（向后兼容）。
//
// 默认排序：按 count 降序。
func NewFieldDimension(name, desc string, getter func(*media.MediaRecord) string, appliesTo ...media.MediaType) DimensionCounter {
	meta := DimensionMeta{SortBy: "count", SortOrder: "desc", Desc: desc}
	if len(appliesTo) > 0 {
		meta.AppliesTo = appliesTo
	}
	return NewGroupCounter(name,
		func(r *media.MediaRecord) []string {
			v := getter(r)
			if v == "" {
				return []string{"Unknown"}
			}
			return []string{v}
		},
		meta,
	)
}
