package c2pa

import (
	"bytes"
	"encoding/binary"
	"sort"
)

// Detect sniffs the byte slice for JPEG or PNG and runs the matching detector.
// data 通常是文件头 64KB（见 internal/extract）。nil-safe：非图像 / 太短 → 空 Result。
func Detect(data []byte) Result {
	switch {
	case isJPEG(data):
		return Result{Manifest: detectJPEG(data)}
	case isPNG(data):
		m, text := detectPNG(data)
		return Result{Manifest: m, PNGText: text}
	default:
		return Result{}
	}
}

func isJPEG(data []byte) bool {
	return len(data) >= 3 && data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF
}

// detectJPEG 提取并重组 App11 里的 JUMBF box，检测是否含 c2pa manifest。
//
// JPEG App11 + JUMBF 布局（每个 0xFFEB segment）：
//
//	FF EB              marker
//	LL LL              segment length（含这两字节，big-endian）
//	4A 50              CI: common identifier "JP" (JUMBF in App11)
//	II II              En: box instance number
//	ZZ ZZ ZZ ZZ        Z: packet sequence number（同 box 内排序）
//	[JUMBF box bytes]   该 box 的全部或一个分片
//
// 大于 ~64KB 的 JUMBF box 会被切成多个 App11 segment（同 box instance、
// 递增 packet sequence）。重组 = 按 instance 分组、按 sequence 排序、拼接。
func detectJPEG(data []byte) *Manifest {
	if !isJPEG(data) {
		return nil
	}

	type fragment struct {
		seq  uint32
		data []byte
	}
	groups := map[uint16][]fragment{}

	pos := 2 // 跳过 SOI (FF D8)
	for pos+2 <= len(data) {
		if data[pos] != 0xFF {
			break // 不在 marker 边界，结构损坏
		}
		marker := data[pos+1]

		// 无 payload 的 standalone markers
		if marker == 0xD9 { // EOI
			break
		}
		if marker == 0x01 || (marker >= 0xD0 && marker <= 0xD7) { // TEM / RSTn
			pos += 2
			continue
		}
		// SOS：图像扫描数据开始，后面不再有元数据 marker
		if marker == 0xDA {
			break
		}

		if pos+4 > len(data) {
			break
		}
		segLen := int(data[pos+2])<<8 | int(data[pos+3])
		if segLen < 2 {
			break
		}
		segStart := pos + 4
		segEnd := pos + 2 + segLen
		truncated := false
		if segEnd > len(data) {
			// head read 把 segment 截断了——用拿到的部分（detection 仍可能命中）
			segEnd = len(data)
			truncated = true
		}

		if marker == 0xEB && segEnd-segStart >= 8 { // App11
			seg := data[segStart:segEnd]
			// CI == "JP" (0x4A50) 才是 JUMBF-in-App11
			if seg[0] == 0x4A && seg[1] == 0x50 {
				boxInstance := binary.BigEndian.Uint16(seg[2:4])
				packetSeq := binary.BigEndian.Uint32(seg[4:8])
				groups[boxInstance] = append(groups[boxInstance], fragment{packetSeq, seg[8:]})
			}
		}

		if truncated {
			break
		}
		pos = segEnd
	}

	if len(groups) == 0 {
		return nil
	}

	// 重组每个 box instance，找含 c2pa 标识的那个
	for _, frags := range groups {
		sort.Slice(frags, func(i, j int) bool { return frags[i].seq < frags[j].seq })
		var buf []byte
		for _, f := range frags {
			buf = append(buf, f.data...)
		}
		// "c2pa" 是 C2PA superbox 的 label；"jumb" 是 JUMBF box type。
		// 两者同时缺失说明这不是 C2PA（可能是别的 JUMBF 用途，忽略）。
		if !bytes.Contains(buf, []byte("c2pa")) {
			continue
		}
		m := &Manifest{Present: true}
		m.Generator = extractGenerator(buf)
		return m
	}
	return nil
}
