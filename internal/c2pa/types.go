// Package c2pa 提供 C2PA (Content Provenance and Authenticity) 内容凭证的
// detection-only 解析——只判断「图像是否声称自己有 C2PA 来源信息 / 是谁生成的」，
// 不做密码学签名验证（cert chain / TSP timestamp / COSE signature）。
//
// 设计取舍（design doc + plan-eng-review）：
//   - 纯 Go，无 CGO，不调外部 c2patool
//   - JPEG: 解析 App11 (0xFFEB) marker 里的 JUMBF box，多 marker 自动重组
//   - PNG:  解析 tEXt / iTXt(未压缩) chunk，顺带检测内嵌 JUMBF
//   - JUMBF 内的 CBOR 用 github.com/fxamacker/cbor/v2 解析提取 claim_generator
//
// 「detection-only ≠ tamper-proof」：Software 字段和 C2PA manifest 都可被伪造。
// 不要把本包结果当作法律证据或内容审核唯一依据。
package c2pa

// Manifest 是 detection-only 的 C2PA 检测结果。
//
// Present=true 表示找到了 C2PA manifest（JUMBF + c2pa 标识）；
// Generator 是从 claim 的 CBOR 里提取的 claim_generator（best-effort，可能为空）。
type Manifest struct {
	Present   bool   `json:"present"`
	Generator string `json:"generator,omitempty"`
}

// TextEntry 是 PNG tEXt / iTXt chunk 的一个 key-value 对。
// 用于 AI 生成检测：Stable Diffusion / ComfyUI / NovelAI 把 prompt、参数、
// 生成器名字写进这些 chunk。
type TextEntry struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// Result 是 Detect 的统一返回：manifest（可能 nil）+ PNG 文本条目（仅 PNG 非空）。
type Result struct {
	Manifest *Manifest
	PNGText  []TextEntry
}
