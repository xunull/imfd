package c2pa

import "testing"

func TestDetect_NotImage(t *testing.T) {
	r := Detect([]byte("this is plain text, not an image"))
	if r.Manifest != nil || r.PNGText != nil {
		t.Errorf("expected empty result for non-image, got %+v", r)
	}
}

func TestDetect_EmptyInput(t *testing.T) {
	r := Detect(nil)
	if r.Manifest != nil {
		t.Error("nil input should give nil manifest")
	}
}

func TestDetectJPEG_NoApp11(t *testing.T) {
	// JPEG with only an APP0 (JFIF), no C2PA
	app0 := []byte{0xFF, 0xE0, 0x00, 0x06, 'J', 'F', 'I', 'F'}
	jpeg := minimalJPEG(app0)
	if m := detectJPEG(jpeg); m != nil {
		t.Errorf("expected nil manifest, got %+v", m)
	}
}

func TestDetectJPEG_SingleSegmentManifest(t *testing.T) {
	manifest := c2paManifestBox(t, map[string]any{"claim_generator": "DALL·E 3.0"})
	seg := app11JUMBF(1, 1, manifest)
	jpeg := minimalJPEG(seg)

	m := detectJPEG(jpeg)
	if m == nil {
		t.Fatal("expected manifest, got nil")
	}
	if !m.Present {
		t.Error("Present should be true")
	}
	if m.Generator != "DALL·E 3.0" {
		t.Errorf("Generator: got %q, want %q", m.Generator, "DALL·E 3.0")
	}
}

func TestDetectJPEG_MultiSegmentReassembly(t *testing.T) {
	// 把一个 manifest box 切成两段，乱序放（seq 2 在 seq 1 前），验证重组+排序
	manifest := c2paManifestBox(t, map[string]any{"claim_generator": "Adobe Firefly"})
	split := len(manifest) / 2
	frag1 := manifest[:split]
	frag2 := manifest[split:]

	// 文件里先放 seq=2，再放 seq=1 —— 测排序
	seg2 := app11JUMBF(7, 2, frag2)
	seg1 := app11JUMBF(7, 1, frag1)
	jpeg := minimalJPEG(seg2, seg1)

	m := detectJPEG(jpeg)
	if m == nil {
		t.Fatal("expected manifest from reassembled fragments")
	}
	if m.Generator != "Adobe Firefly" {
		t.Errorf("Generator: got %q, want %q", m.Generator, "Adobe Firefly")
	}
}

func TestDetectJPEG_App11WithoutJP(t *testing.T) {
	// App11 存在但不是 "JP" 标识（非 JUMBF）→ 不算 C2PA
	seg := []byte{0xFF, 0xEB, 0x00, 0x0A, 'X', 'X', 0, 1, 0, 0, 0, 1}
	jpeg := minimalJPEG(seg)
	if m := detectJPEG(jpeg); m != nil {
		t.Errorf("non-JP App11 should not be C2PA, got %+v", m)
	}
}

func TestDetectJPEG_JUMBFWithoutC2PA(t *testing.T) {
	// JUMBF box 存在但 label 不是 c2pa（别的 JUMBF 用途）→ 忽略
	otherBox := mkBox("jumb", append(jumdBox("other.metadata"), cborBox(t, map[string]any{"foo": "bar"})...))
	seg := app11JUMBF(1, 1, otherBox)
	jpeg := minimalJPEG(seg)
	if m := detectJPEG(jpeg); m != nil {
		t.Errorf("non-c2pa JUMBF should be ignored, got %+v", m)
	}
}

func TestDetectJPEG_PresentButNoGenerator(t *testing.T) {
	// c2pa manifest 存在但 claim 无 generator → Present=true, Generator=""
	manifest := c2paManifestBox(t, map[string]any{"dc:format": "image/jpeg"})
	seg := app11JUMBF(1, 1, manifest)
	jpeg := minimalJPEG(seg)

	m := detectJPEG(jpeg)
	if m == nil || !m.Present {
		t.Fatal("expected Present manifest")
	}
	if m.Generator != "" {
		t.Errorf("expected empty Generator, got %q", m.Generator)
	}
}

func TestDetectJPEG_StopsAtSOS(t *testing.T) {
	// SOS marker 后即使有看似 App11 的字节也不解析（那是图像数据）
	sos := []byte{0xFF, 0xDA, 0x00, 0x02} // SOS, minimal
	manifest := c2paManifestBox(t, map[string]any{"claim_generator": "ShouldNotSee"})
	segAfterSOS := app11JUMBF(1, 1, manifest)
	jpeg := minimalJPEG(sos, segAfterSOS)
	if m := detectJPEG(jpeg); m != nil {
		t.Errorf("App11 after SOS should be ignored, got %+v", m)
	}
}

func TestDetect_DispatchesJPEG(t *testing.T) {
	manifest := c2paManifestBox(t, map[string]any{"claim_generator": "test"})
	jpeg := minimalJPEG(app11JUMBF(1, 1, manifest))
	r := Detect(jpeg)
	if r.Manifest == nil || !r.Manifest.Present {
		t.Error("Detect should dispatch JPEG and find manifest")
	}
}
