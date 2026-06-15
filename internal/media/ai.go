package media

import (
	"fmt"
	"strings"
)

// AI 生成检测（detection-only）。
//
// 多信号「强/弱二值 OR」判定（plan-eng-review CQ1）：
//   - STRONG 信号任一命中 → ai-generated
//   - WEAK 信号需 ≥2 个不同 key 才判 ai-generated
//
// STRONG:
//   S1  C2PA manifest 存在（DALL·E 3 / Firefly / ChatGPT 等默认嵌入）
//   S2  EXIF Software 字段含 AI 工具关键字
//   S3  PNG "Software" 文本字段含 AI 工具关键字
//   S4  PNG "parameters" 字段含 Stable Diffusion 生成签名（Steps:/Sampler:/CFG scale:）
//
// WEAK（需 ≥2 不同 key）:
//   W   PNG 文本里出现生成工具 key：prompt / workflow / comfyui / parameters
//       （ComfyUI 写 prompt+workflow 两个 → 命中）
//
// 关键防假阳（plan-eng-review accepted concern #2）：AI 关键字只在 Software /
// C2PA generator / 指定 PNG key 上匹配，绝不在任意文本里 substring 匹配——
// 否则 Description="midjourney inspired" 的普通照片会被误判。

const (
	AIVerdictGenerated = "ai-generated"
	AIVerdictNotAI     = "not-ai"
	AIVerdictUnknown   = "unknown"
)

// aiSoftwareKeywords：出现在 Software / C2PA generator 字段即判 AI 的关键字。
// 调用方先 ToLower。
var aiSoftwareKeywords = []string{
	"dall·e", "dall-e", "dalle",
	"midjourney",
	"stable diffusion", "stable-diffusion", "stablediffusion",
	"sdxl", "automatic1111", "a1111", "comfyui", "invokeai", "fooocus",
	"adobe firefly", "firefly",
	"bing image creator", "image creator",
	"imagen",
	"flux",
	"novelai",
	"leonardo.ai", "ideogram", "recraft", "dreamstudio",
	"gpt-image", "gpt image",
}

// aiGenerationKeys：PNG tEXt/iTXt 里代表 AI 生成工具的 key（弱信号）。
var aiGenerationKeys = map[string]bool{
	"prompt":     true,
	"workflow":   true,
	"comfyui":    true,
	"parameters": true,
}

func containsAIKeyword(low string) bool {
	for _, k := range aiSoftwareKeywords {
		if strings.Contains(low, k) {
			return true
		}
	}
	return false
}

// looksLikeSDParameters：Stable Diffusion WebUI 在 "parameters" 写的签名结构。
// 普通照片不会有 "Steps: N, Sampler: X" 这种文本 → 强信号。
func looksLikeSDParameters(value string) bool {
	return strings.Contains(value, "Steps:") &&
		(strings.Contains(value, "Sampler:") || strings.Contains(value, "CFG scale:"))
}

// aiSignalCounts 返回 (是否有强信号, 不同弱信号 key 数量)。
func aiSignalCounts(r *MediaRecord) (strong bool, weakKeys int) {
	if r == nil || r.Exif == nil {
		return false, 0
	}
	e := r.Exif

	// S1: C2PA manifest
	if e.C2PA != nil && e.C2PA.Present {
		strong = true
	}
	// S2: EXIF Software
	if containsAIKeyword(strings.ToLower(e.Software)) {
		strong = true
	}

	// PNG 文本信号
	seen := map[string]bool{}
	for _, t := range e.PNGText {
		keyLow := strings.ToLower(t.Key)
		// S3: PNG Software 关键字
		if keyLow == "software" && containsAIKeyword(strings.ToLower(t.Value)) {
			strong = true
		}
		// S4: SD parameters 签名
		if keyLow == "parameters" && looksLikeSDParameters(t.Value) {
			strong = true
		}
		// 弱信号：生成工具 key
		if aiGenerationKeys[keyLow] {
			seen[keyLow] = true
		}
	}
	return strong, len(seen)
}

// IsAIGenerated：strong 一个即判；weak ≥2 个不同 key 才判。nil-safe。
func IsAIGenerated(r *MediaRecord) bool {
	strong, weak := aiSignalCounts(r)
	return strong || weak >= 2
}

// AIVerdict：3 级判定（verify --c2pa 展示用）。
//   - ai-generated：strong 命中或 weak ≥2
//   - not-ai：有可分析元数据但无 AI 信号
//   - unknown：元数据不足（无 EXIF/C2PA/PNG 文本）
func AIVerdict(r *MediaRecord) string {
	if r == nil || r.Exif == nil {
		return AIVerdictUnknown
	}
	if IsAIGenerated(r) {
		return AIVerdictGenerated
	}
	e := r.Exif
	if e.Software != "" || e.C2PA != nil || len(e.PNGText) > 0 || e.HasDateTime {
		return AIVerdictNotAI
	}
	return AIVerdictUnknown
}

// AISignals 返回人类可读的判定依据（verify --c2pa 展示）。
func AISignals(r *MediaRecord) []string {
	if r == nil || r.Exif == nil {
		return []string{"✗ 无可分析的图像元数据"}
	}
	e := r.Exif
	var sig []string

	if e.C2PA != nil && e.C2PA.Present {
		if e.C2PA.Generator != "" {
			sig = append(sig, "✓ C2PA manifest 存在（生成器: "+e.C2PA.Generator+"）")
		} else {
			sig = append(sig, "✓ C2PA manifest 存在（生成器未知）")
		}
	}
	if containsAIKeyword(strings.ToLower(e.Software)) {
		sig = append(sig, "✓ EXIF Software 含 AI 工具关键字: "+e.Software)
	}

	var weakKeys []string
	for _, t := range e.PNGText {
		keyLow := strings.ToLower(t.Key)
		if keyLow == "software" && containsAIKeyword(strings.ToLower(t.Value)) {
			sig = append(sig, "✓ PNG Software 含 AI 工具关键字: "+t.Value)
		}
		if keyLow == "parameters" && looksLikeSDParameters(t.Value) {
			sig = append(sig, "✓ PNG parameters 含 Stable Diffusion 生成签名")
		}
		if aiGenerationKeys[keyLow] {
			weakKeys = append(weakKeys, keyLow)
		}
	}
	if len(weakKeys) > 0 {
		mark := "✗"
		if len(weakKeys) >= 2 {
			mark = "✓"
		}
		sig = append(sig, fmt.Sprintf("%s PNG 生成工具字段 [%s]（弱信号，需≥2）",
			mark, strings.Join(weakKeys, ", ")))
	}

	if len(sig) == 0 {
		sig = append(sig, "✗ 未检测到任何 AI 生成信号")
	}
	return sig
}
