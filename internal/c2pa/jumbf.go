package c2pa

import (
	"encoding/binary"
	"reflect"

	"github.com/fxamacker/cbor/v2"
)

// cborDecMode 让所有 CBOR map（含嵌套）都解成 map[string]any。
// fxamacker/cbor 默认把 interface{} 槽里的 map 解成 map[interface{}]interface{}，
// 导致 claim_generator_info[] 里的嵌套对象类型断言失败。DefaultMapType 修正这点。
var cborDecMode, _ = cbor.DecOptions{
	DefaultMapType: reflect.TypeOf(map[string]any(nil)),
}.DecMode()

// JUMBF (JPEG Universal Metadata Box Format, ISO/IEC 19566-5) box layout:
//
//	┌────────────┬────────────┬─────────────────────────────┐
//	│ LBox (4)   │ TBox (4)   │ payload                      │
//	│ box length │ box type   │ (nested boxes or raw data)   │
//	└────────────┴────────────┴─────────────────────────────┘
//
// C2PA 把 claim 存成嵌套结构：
//
//	jumb (superbox "c2pa")
//	 └─ jumb (claim "c2pa.claim")
//	     ├─ jumd (description: UUID + label)
//	     └─ cbor (content: CBOR map with "claim_generator")
//
// detection-only 只要拿到 claim_generator，所以策略是：
// 递归 walk 所有 box，对每个 "cbor" content box 尝试 CBOR 解码，
// 看 map 里有没有 claim_generator / claim_generator_info。
// 不需要严格按 label 路径下钻——任何含 claim_generator 的 CBOR map 都接受。

// jumbfBox 是解析出的一个 JUMBF box。
type jumbfBox struct {
	typ     string
	payload []byte
}

// walkJUMBF 解析一层 JUMBF box（不递归）。
// 容错：遇到非法长度就停（detection-only 不追求严格校验）。
func walkJUMBF(data []byte) []jumbfBox {
	var boxes []jumbfBox
	pos := 0
	for pos+8 <= len(data) {
		size := binary.BigEndian.Uint32(data[pos : pos+4])
		typ := string(data[pos+4 : pos+8])

		switch {
		case size == 0:
			// LBox==0：box 延伸到数据末尾
			boxes = append(boxes, jumbfBox{typ, data[pos+8:]})
			return boxes
		case size == 1:
			// LBox==1：紧跟 8 字节 XLBox (64-bit 真实长度)
			if pos+16 > len(data) {
				return boxes
			}
			xl := binary.BigEndian.Uint64(data[pos+8 : pos+16])
			end := pos + int(xl)
			if xl < 16 || end > len(data) || end < pos {
				return boxes
			}
			boxes = append(boxes, jumbfBox{typ, data[pos+16 : end]})
			pos = end
		default:
			end := pos + int(size)
			if size < 8 || end > len(data) || end < pos {
				return boxes
			}
			boxes = append(boxes, jumbfBox{typ, data[pos+8 : end]})
			pos = end
		}
	}
	return boxes
}

// extractGenerator 递归找第一个含 claim_generator 的 CBOR box，返回生成器名字。
// 找不到返回 ""（manifest 仍可能 Present=true）。
func extractGenerator(data []byte) string {
	for _, b := range walkJUMBF(data) {
		// content box "cbor"：payload 直接是 CBOR
		if b.typ == "cbor" {
			if gen := generatorFromCBOR(b.payload); gen != "" {
				return gen
			}
		}
		// superbox（jumb 等）：递归下钻
		if gen := extractGenerator(b.payload); gen != "" {
			return gen
		}
	}
	return ""
}

// generatorFromCBOR 把 CBOR bytes 解成 map，取 claim_generator 或
// claim_generator_info[].name。
//
// C2PA v1 用 claim_generator(text)；v2 推荐 claim_generator_info(array of
// {name, version}). 两个都试。
func generatorFromCBOR(data []byte) string {
	var m map[string]any
	if err := cborDecMode.Unmarshal(data, &m); err != nil {
		return ""
	}
	return generatorFromMap(m)
}

func generatorFromMap(m map[string]any) string {
	// v1: claim_generator 是 text string，形如
	// "Adobe_(Photoshop)/25.5 c2pa-rs/0.28.0"
	if g, ok := m["claim_generator"].(string); ok && g != "" {
		return g
	}
	// v2: claim_generator_info 可能是 [{name,version}]（spec array 形式）
	// 也可能是单个 {name,version} map（c2patool 0.26 实测输出）—— 两种都接受。
	switch info := m["claim_generator_info"].(type) {
	case map[string]any:
		return nameVersion(info)
	case []any:
		for _, item := range info {
			if im, ok := item.(map[string]any); ok {
				if s := nameVersion(im); s != "" {
					return s
				}
			}
		}
	}
	return ""
}

// nameVersion 从 {name, version} map 拼出 "name version"（version 缺则只 name）。
func nameVersion(m map[string]any) string {
	name, _ := m["name"].(string)
	if name == "" {
		return ""
	}
	if ver, ok := m["version"].(string); ok && ver != "" {
		return name + " " + ver
	}
	return name
}
