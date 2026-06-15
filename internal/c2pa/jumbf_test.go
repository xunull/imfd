package c2pa

import (
	"encoding/binary"
	"hash/crc32"
	"testing"

	"github.com/fxamacker/cbor/v2"
)

// ───────────────── test byte builders ─────────────────
// 这些 helper 编码了对 JUMBF / App11 / PNG 格式的理解，既造 fixture 又是
// 格式文档。parser 能 round-trip 这些 helper 的输出 = 解析逻辑正确。

// jumbfBox 构造一个 JUMBF box: LBox(4) + TBox(4) + payload。
func mkBox(typ string, payload []byte) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint32(b[:4], uint32(8+len(payload)))
	copy(b[4:8], typ)
	return append(b, payload...)
}

// jumdBox 构造 description box，label 决定 superbox 用途（"c2pa" 是 C2PA 顶层）。
func jumdBox(label string) []byte {
	payload := make([]byte, 16) // UUID（内容无所谓）
	payload = append(payload, 0x03)
	payload = append(payload, []byte(label)...)
	payload = append(payload, 0x00)
	return mkBox("jumd", payload)
}

// cborBox 构造 content box，payload 是 CBOR 编码的 map。
func cborBox(t *testing.T, m map[string]any) []byte {
	t.Helper()
	data, err := cbor.Marshal(m)
	if err != nil {
		t.Fatalf("cbor.Marshal: %v", err)
	}
	return mkBox("cbor", data)
}

// c2paManifestBox 构造完整 C2PA superbox：jumb { jumd("c2pa") + claim }。
// claim 是嵌套的 jumb { jumd("c2pa.claim") + cbor(claimMap) }。
func c2paManifestBox(t *testing.T, claimMap map[string]any) []byte {
	t.Helper()
	claim := mkBox("jumb", append(jumdBox("c2pa.claim"), cborBox(t, claimMap)...))
	return mkBox("jumb", append(jumdBox("c2pa"), claim...))
}

// app11JUMBF 构造一个 App11 segment 携带（部分）JUMBF 数据。
func app11JUMBF(boxInstance uint16, packetSeq uint32, jumbfData []byte) []byte {
	payload := make([]byte, 8)
	payload[0], payload[1] = 0x4A, 0x50 // "JP"
	binary.BigEndian.PutUint16(payload[2:4], boxInstance)
	binary.BigEndian.PutUint32(payload[4:8], packetSeq)
	payload = append(payload, jumbfData...)
	segLen := 2 + len(payload)
	seg := []byte{0xFF, 0xEB, byte(segLen >> 8), byte(segLen)}
	return append(seg, payload...)
}

// minimalJPEG 拼一个最小合法 JPEG：SOI + segments + EOI。
func minimalJPEG(segments ...[]byte) []byte {
	out := []byte{0xFF, 0xD8}
	for _, s := range segments {
		out = append(out, s...)
	}
	return append(out, 0xFF, 0xD9)
}

// pngChunk 构造一个 PNG chunk: length(4) + type(4) + data + CRC(4)。
func pngChunk(ctype string, data []byte) []byte {
	out := make([]byte, 4)
	binary.BigEndian.PutUint32(out, uint32(len(data)))
	out = append(out, []byte(ctype)...)
	out = append(out, data...)
	crc := crc32.ChecksumIEEE(append([]byte(ctype), data...))
	c := make([]byte, 4)
	binary.BigEndian.PutUint32(c, crc)
	return append(out, c...)
}

func minimalPNG(chunks ...[]byte) []byte {
	out := append([]byte{}, pngMagic...)
	for _, c := range chunks {
		out = append(out, c...)
	}
	return append(out, pngChunk("IEND", nil)...)
}

// ───────────────── JUMBF / CBOR tests ─────────────────

func TestGeneratorFromCBOR_V1ClaimGenerator(t *testing.T) {
	box := cborBox(t, map[string]any{"claim_generator": "DALL·E 3.0"})
	// box 是 [header][cbor]，generatorFromCBOR 接受裸 cbor，所以剥头
	got := generatorFromCBOR(box[8:])
	if got != "DALL·E 3.0" {
		t.Errorf("got %q, want %q", got, "DALL·E 3.0")
	}
}

func TestGeneratorFromCBOR_V2GeneratorInfo(t *testing.T) {
	box := cborBox(t, map[string]any{
		"claim_generator_info": []any{
			map[string]any{"name": "Adobe Firefly", "version": "2.0"},
		},
	})
	got := generatorFromCBOR(box[8:])
	if got != "Adobe Firefly 2.0" {
		t.Errorf("got %q, want %q", got, "Adobe Firefly 2.0")
	}
}

func TestGeneratorFromCBOR_GeneratorInfoNoVersion(t *testing.T) {
	box := cborBox(t, map[string]any{
		"claim_generator_info": []any{
			map[string]any{"name": "Midjourney"},
		},
	})
	if got := generatorFromCBOR(box[8:]); got != "Midjourney" {
		t.Errorf("got %q, want %q", got, "Midjourney")
	}
}

// TestGeneratorFromCBOR_GeneratorInfoMapForm — c2patool 0.26 实测把
// claim_generator_info 写成单个 map（非 spec 的 array）。真实文件验证发现的形态。
func TestGeneratorFromCBOR_GeneratorInfoMapForm(t *testing.T) {
	box := cborBox(t, map[string]any{
		"instanceID":           "xmp:iid:abc",
		"claim_generator_info": map[string]any{"name": "imfd-test-generator", "version": "1.0"},
	})
	if got := generatorFromCBOR(box[8:]); got != "imfd-test-generator 1.0" {
		t.Errorf("got %q, want %q", got, "imfd-test-generator 1.0")
	}
}

func TestGeneratorFromCBOR_NotCBOR(t *testing.T) {
	if got := generatorFromCBOR([]byte("not cbor at all")); got != "" {
		t.Errorf("expected empty for non-CBOR, got %q", got)
	}
}

func TestExtractGenerator_NestedManifest(t *testing.T) {
	box := c2paManifestBox(t, map[string]any{"claim_generator": "c2pa-test/1.0"})
	got := extractGenerator(box)
	if got != "c2pa-test/1.0" {
		t.Errorf("got %q, want %q", got, "c2pa-test/1.0")
	}
}

func TestExtractGenerator_NoClaimGenerator(t *testing.T) {
	// manifest 存在但 claim 里没 claim_generator 字段
	box := c2paManifestBox(t, map[string]any{"dc:title": "untitled"})
	if got := extractGenerator(box); got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestWalkJUMBF_TruncatedBox(t *testing.T) {
	// LBox 声称 1000 字节但实际只有 20 → 容错返回空，不 panic
	b := make([]byte, 20)
	binary.BigEndian.PutUint32(b[:4], 1000)
	copy(b[4:8], "jumb")
	boxes := walkJUMBF(b)
	if len(boxes) != 0 {
		t.Errorf("expected 0 boxes for truncated, got %d", len(boxes))
	}
}
