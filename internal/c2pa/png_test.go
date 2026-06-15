package c2pa

import "testing"

func tEXtData(keyword, text string) []byte {
	d := append([]byte(keyword), 0)
	return append(d, []byte(text)...)
}

func iTXtData(keyword string, compressed bool, lang, transKw, text string) []byte {
	d := append([]byte(keyword), 0)
	if compressed {
		d = append(d, 1, 0) // compression_flag=1, method=0
	} else {
		d = append(d, 0, 0) // uncompressed
	}
	d = append(d, []byte(lang)...)
	d = append(d, 0)
	d = append(d, []byte(transKw)...)
	d = append(d, 0)
	d = append(d, []byte(text)...)
	return d
}

func TestDetectPNG_NotPNG(t *testing.T) {
	m, text := detectPNG([]byte("not a png"))
	if m != nil || text != nil {
		t.Errorf("non-PNG should give nil, got m=%v text=%v", m, text)
	}
}

func TestDetectPNG_TEXtParsing(t *testing.T) {
	png := minimalPNG(
		pngChunk("tEXt", tEXtData("Software", "Adobe Firefly")),
		pngChunk("tEXt", tEXtData("Comment", "hello world")),
	)
	_, text := detectPNG(png)
	if len(text) != 2 {
		t.Fatalf("expected 2 text entries, got %d: %+v", len(text), text)
	}
	if text[0].Key != "Software" || text[0].Value != "Adobe Firefly" {
		t.Errorf("entry 0: got %+v", text[0])
	}
	if text[1].Key != "Comment" || text[1].Value != "hello world" {
		t.Errorf("entry 1: got %+v", text[1])
	}
}

func TestDetectPNG_ITXtUncompressed(t *testing.T) {
	// ComfyUI 把 workflow 写进 iTXt（未压缩）
	png := minimalPNG(
		pngChunk("iTXt", iTXtData("workflow", false, "", "", `{"nodes":[...]}`)),
	)
	_, text := detectPNG(png)
	if len(text) != 1 {
		t.Fatalf("expected 1 iTXt entry, got %d", len(text))
	}
	if text[0].Key != "workflow" {
		t.Errorf("key: got %q", text[0].Key)
	}
	if text[0].Value != `{"nodes":[...]}` {
		t.Errorf("value: got %q", text[0].Value)
	}
}

func TestDetectPNG_ITXtCompressedKeepsKey(t *testing.T) {
	// 压缩 iTXt：v1 不解压，但保留 key 名（weak-signal 检测看 key）
	png := minimalPNG(
		pngChunk("iTXt", iTXtData("parameters", true, "", "", "compressed-garbage")),
	)
	_, text := detectPNG(png)
	if len(text) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(text))
	}
	if text[0].Key != "parameters" {
		t.Errorf("compressed iTXt should keep key, got %q", text[0].Key)
	}
	if text[0].Value != "" {
		t.Errorf("compressed iTXt value should be empty (not decompressed), got %q", text[0].Value)
	}
}

func TestDetectPNG_SDParametersChunk(t *testing.T) {
	// Stable Diffusion WebUI 经典：tEXt "parameters" 含长 prompt
	prompt := "masterpiece, best quality, 1girl, Steps: 20, Sampler: Euler a, CFG scale: 7"
	png := minimalPNG(pngChunk("tEXt", tEXtData("parameters", prompt)))
	_, text := detectPNG(png)
	if len(text) != 1 || text[0].Key != "parameters" {
		t.Fatalf("expected parameters entry, got %+v", text)
	}
	if text[0].Value != prompt {
		t.Errorf("value mismatch")
	}
}

func TestDetectPNG_StopsAtIEND(t *testing.T) {
	// IEND 后的 tEXt 不解析（IEND 是 PNG 终止 chunk）
	png := append([]byte{}, pngMagic...)
	png = append(png, pngChunk("tEXt", tEXtData("before", "seen"))...)
	png = append(png, pngChunk("IEND", nil)...)
	png = append(png, pngChunk("tEXt", tEXtData("after", "unseen"))...)
	_, text := detectPNG(png)
	if len(text) != 1 || text[0].Key != "before" {
		t.Errorf("only pre-IEND entries should be parsed, got %+v", text)
	}
}

func TestDetectPNG_NoTextChunks(t *testing.T) {
	png := minimalPNG() // 只有 magic + IEND
	m, text := detectPNG(png)
	if m != nil {
		t.Error("no C2PA → nil manifest")
	}
	if len(text) != 0 {
		t.Errorf("no text chunks → empty, got %+v", text)
	}
}
