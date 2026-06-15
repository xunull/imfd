package c2pa

import (
	"bytes"
	"encoding/binary"
)

var pngMagic = []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}

func isPNG(data []byte) bool {
	return len(data) >= 8 && bytes.Equal(data[:8], pngMagic)
}

// detectPNG 走 PNG chunk 流，提取 tEXt / iTXt(未压缩) 文本条目，
// 并检测是否有内嵌 C2PA JUMBF（C2PA 把 manifest 放进自定义 chunk，
// 我们对任何含 jumb+c2pa 的 chunk 都尝试解析）。
//
// PNG chunk 布局：
//
//	┌──────────┬──────────┬──────────────┬──────────┐
//	│ length(4)│ type(4)  │ data(length) │ CRC(4)   │
//	└──────────┴──────────┴──────────────┴──────────┘
//
// tEXt:  keyword \0 text                       (Latin-1，未压缩)
// iTXt:  keyword \0 cflag(1) cmethod(1) lang \0 transkw \0 text   (UTF-8)
//        cflag==0 表示 text 未压缩；==1 是 zlib 压缩（v1 跳过）。
func detectPNG(data []byte) (*Manifest, []TextEntry) {
	if !isPNG(data) {
		return nil, nil
	}

	var entries []TextEntry
	var manifest *Manifest

	pos := 8 // 跳过 PNG magic
	for pos+8 <= len(data) {
		length := int(binary.BigEndian.Uint32(data[pos : pos+4]))
		ctype := string(data[pos+4 : pos+8])
		dataStart := pos + 8
		dataEnd := dataStart + length

		if length < 0 || dataEnd > len(data) {
			// head read 截断了这个 chunk（或长度损坏）——停
			break
		}
		chunk := data[dataStart:dataEnd]

		switch ctype {
		case "tEXt":
			if kv := parseTEXt(chunk); kv != nil {
				entries = append(entries, *kv)
			}
		case "iTXt":
			if kv := parseITXt(chunk); kv != nil {
				entries = append(entries, *kv)
			}
		case "IEND":
			return manifest, entries
		default:
			// C2PA-in-PNG：宽松检测——任何 chunk 含 jumb+c2pa 就尝试解析。
			// 标准是 caBX chunk，但不同实现命名不一，宽松更稳。
			if manifest == nil && bytes.Contains(chunk, []byte("jumb")) && bytes.Contains(chunk, []byte("c2pa")) {
				manifest = &Manifest{Present: true, Generator: extractGenerator(chunk)}
			}
		}

		pos = dataEnd + 4 // +4 跳过 CRC
	}
	return manifest, entries
}

// parseTEXt 解析 tEXt chunk: keyword \0 text。
func parseTEXt(chunk []byte) *TextEntry {
	i := bytes.IndexByte(chunk, 0)
	if i < 0 {
		return nil
	}
	return &TextEntry{
		Key:   string(chunk[:i]),
		Value: string(chunk[i+1:]),
	}
}

// parseITXt 解析 iTXt chunk:
//
//	keyword \0 compression_flag(1) compression_method(1) language_tag \0 translated_keyword \0 text
//
// 只处理 compression_flag==0（未压缩）；压缩的跳过（v1 不解 zlib）。
func parseITXt(chunk []byte) *TextEntry {
	i := bytes.IndexByte(chunk, 0)
	if i < 0 || i+2 >= len(chunk) {
		return nil
	}
	keyword := string(chunk[:i])
	compressionFlag := chunk[i+1]
	// chunk[i+2] = compression method（忽略）
	rest := chunk[i+3:]

	if compressionFlag != 0 {
		// 压缩的 iTXt：v1 不解，但仍记录 key（让 weak-signal 检测能看到 key 名）
		return &TextEntry{Key: keyword, Value: ""}
	}

	// language_tag \0
	j := bytes.IndexByte(rest, 0)
	if j < 0 {
		return nil
	}
	rest = rest[j+1:]
	// translated_keyword \0
	k := bytes.IndexByte(rest, 0)
	if k < 0 {
		return nil
	}
	text := rest[k+1:]
	return &TextEntry{Key: keyword, Value: string(text)}
}
