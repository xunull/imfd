package dimensions

import (
	"slices"
	"strings"
	"testing"

	"github.com/xunull/imfd/internal/media"
	"github.com/xunull/imfd/internal/stats"
)

// dimensionNames 取出 registry 当前的所有维度名（按注册顺序）
func dimensionNames(reg *stats.Registry) []string {
	report := reg.Report()
	names := make([]string, 0, len(report.Dimensions))
	for _, d := range report.Dimensions {
		names = append(names, d.DimensionName)
	}
	return names
}

func TestRegisterDefaults_NilActiveTypes_RegistersAll(t *testing.T) {
	reg := stats.NewRegistry()
	RegisterDefaults(reg, nil)
	names := dimensionNames(reg)

	// 至少 16 个图像/通用维度 + 5 个音频维度
	if len(names) < 21 {
		t.Errorf("nil activeTypes: want >=21 dimensions registered, got %d (%v)", len(names), names)
	}

	// 必须包含一个 image-only 维度
	if !slices.Contains(names, "相机型号") {
		t.Error("nil activeTypes should register 相机型号")
	}
	// 必须包含 audio 维度
	if !slices.Contains(names, "音频编解码器") {
		t.Error("nil activeTypes should register 音频编解码器")
	}
}

func TestRegisterDefaults_AudioOnly_OnlyAudioAndUntaggedDimensions(t *testing.T) {
	reg := stats.NewRegistry()
	RegisterDefaults(reg, []media.MediaType{media.TypeAudio})
	names := dimensionNames(reg)

	// 音频维度必须注册
	for _, want := range []string{"音频编解码器", "音频比特率", "音频采样率", "音频声道", "音频时长"} {
		if !slices.Contains(names, want) {
			t.Errorf("audio-only: missing %q (got %v)", want, names)
		}
	}

	// image-only 维度必须被过滤掉（DIM-1 后老维度都标了 AppliesTo=[TypeImage]）
	for _, unwanted := range []string{"相机型号", "镜头型号", "ISO感光度", "光圈", "快门速度", "焦距", "曝光模式", "白平衡", "省份", "城市", "省/市"} {
		if slices.Contains(names, unwanted) {
			t.Errorf("audio-only: should NOT register image-only dimension %q", unwanted)
		}
	}

	// 「拍摄时间段」AppliesTo=[image,video]，audio 下也不该出现
	if slices.Contains(names, "拍摄时间段") {
		t.Error("audio-only: 拍摄时间段 (image+video only) should not register")
	}

	// 「媒体类型」AppliesTo=nil（跨类型有意义），audio 下保留
	if !slices.Contains(names, "媒体类型") {
		t.Error("audio-only: 媒体类型 (cross-type) should remain registered")
	}
}

func TestRegisterDefaults_ImageOnly_SkipsAudioDimensions(t *testing.T) {
	reg := stats.NewRegistry()
	RegisterDefaults(reg, []media.MediaType{media.TypeImage})
	names := dimensionNames(reg)

	// 所有「音频...」维度都该被跳过（它们标了 AppliesTo=[TypeAudio]，和 [TypeImage] 无交集）
	for _, want := range []string{"音频编解码器", "音频比特率", "音频采样率", "音频声道", "音频时长"} {
		if slices.Contains(names, want) {
			t.Errorf("image-only: should NOT register %q", want)
		}
	}

	// 未标 AppliesTo 的维度全部保留
	for _, want := range []string{"相机型号", "镜头型号", "ISO感光度", "光圈"} {
		if !slices.Contains(names, want) {
			t.Errorf("image-only: missing image-applicable %q", want)
		}
	}
}

func TestRegisterDefaults_VideoOnly_SkipsAudioDimensions(t *testing.T) {
	reg := stats.NewRegistry()
	RegisterDefaults(reg, []media.MediaType{media.TypeVideo})
	names := dimensionNames(reg)

	for _, name := range names {
		if strings.HasPrefix(name, "音频") {
			t.Errorf("video-only: should NOT register audio dimension %q", name)
		}
	}
}

func TestRegisterDefaults_MixedTypes_RegistersUnionByAppliesTo(t *testing.T) {
	// scan audio + image 同时启用（理论场景；当前 CLI 不暴露）
	reg := stats.NewRegistry()
	RegisterDefaults(reg, []media.MediaType{media.TypeAudio, media.TypeImage})
	names := dimensionNames(reg)

	if !slices.Contains(names, "音频编解码器") {
		t.Error("audio+image: should register audio dimensions (intersect non-empty)")
	}
	if !slices.Contains(names, "相机型号") {
		t.Error("audio+image: should register untagged dimensions")
	}
}

func TestShouldRegister(t *testing.T) {
	cases := []struct {
		name         string
		dimAppliesTo []media.MediaType
		activeTypes  []media.MediaType
		want         bool
	}{
		{"nil dim, nil active", nil, nil, true},
		{"nil dim, audio active", nil, []media.MediaType{media.TypeAudio}, true},
		{"audio dim, nil active", []media.MediaType{media.TypeAudio}, nil, true},
		{"audio dim, audio active", []media.MediaType{media.TypeAudio}, []media.MediaType{media.TypeAudio}, true},
		{"audio dim, image active", []media.MediaType{media.TypeAudio}, []media.MediaType{media.TypeImage}, false},
		{"audio dim, video active", []media.MediaType{media.TypeAudio}, []media.MediaType{media.TypeVideo}, false},
		{"audio dim, audio+image active", []media.MediaType{media.TypeAudio}, []media.MediaType{media.TypeAudio, media.TypeImage}, true},
		{"audio+image dim, image active", []media.MediaType{media.TypeAudio, media.TypeImage}, []media.MediaType{media.TypeImage}, true},
		{"audio+image dim, video active", []media.MediaType{media.TypeAudio, media.TypeImage}, []media.MediaType{media.TypeVideo}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := shouldRegister(c.dimAppliesTo, c.activeTypes); got != c.want {
				t.Errorf("shouldRegister(%v, %v) = %v, want %v", c.dimAppliesTo, c.activeTypes, got, c.want)
			}
		})
	}
}
