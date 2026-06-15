package media

import "testing"

func aiRecord(software string, c2pa *C2PAInfo, png []PNGTextEntry) *MediaRecord {
	return &MediaRecord{
		FilePath: "/test.jpg",
		Type:     TypeImage,
		Exif: &ExifInfo{
			Software: software,
			C2PA:     c2pa,
			PNGText:  png,
		},
	}
}

func TestIsAIGenerated_C2PAPresent(t *testing.T) {
	r := aiRecord("", &C2PAInfo{Present: true, Generator: "DALL·E 3.0"}, nil)
	if !IsAIGenerated(r) {
		t.Error("C2PA present should be AI (strong S1)")
	}
	if v := AIVerdict(r); v != AIVerdictGenerated {
		t.Errorf("verdict: got %q, want %q", v, AIVerdictGenerated)
	}
}

func TestIsAIGenerated_EXIFSoftwareKeyword(t *testing.T) {
	r := aiRecord("Midjourney v6", nil, nil)
	if !IsAIGenerated(r) {
		t.Error("EXIF Software 'Midjourney' should be AI (strong S2)")
	}
}

func TestIsAIGenerated_PNGSoftwareKeyword(t *testing.T) {
	r := aiRecord("", nil, []PNGTextEntry{{Key: "Software", Value: "NovelAI"}})
	if !IsAIGenerated(r) {
		t.Error("PNG Software 'NovelAI' should be AI (strong S3)")
	}
}

func TestIsAIGenerated_SDParametersSignature(t *testing.T) {
	// Stable Diffusion WebUI 单 parameters chunk — 强信号 S4
	params := "masterpiece, 1girl\nSteps: 28, Sampler: DPM++ 2M Karras, CFG scale: 7, Seed: 12345"
	r := aiRecord("", nil, []PNGTextEntry{{Key: "parameters", Value: params}})
	if !IsAIGenerated(r) {
		t.Error("SD parameters signature should be AI (strong S4)")
	}
}

func TestIsAIGenerated_ComfyUITwoWeakKeys(t *testing.T) {
	// ComfyUI 写 prompt + workflow 两个 key → 2 weak → AI
	r := aiRecord("", nil, []PNGTextEntry{
		{Key: "prompt", Value: `{"1":{"class_type":"KSampler"}}`},
		{Key: "workflow", Value: `{"nodes":[]}`},
	})
	if !IsAIGenerated(r) {
		t.Error("prompt + workflow (2 weak keys) should be AI")
	}
}

func TestIsAIGenerated_SingleWeakKeyNotEnough(t *testing.T) {
	// 只有一个弱信号 key 且无 SD 签名 → 不判 AI
	r := aiRecord("", nil, []PNGTextEntry{
		{Key: "parameters", Value: "just a short note"},
	})
	if IsAIGenerated(r) {
		t.Error("single weak key without SD signature should NOT be AI")
	}
}

func TestIsAIGenerated_FalsePositiveGuard(t *testing.T) {
	// 普通照片 Description 含 "midjourney" 文字 → 不在锚定字段 → 不判 AI
	r := aiRecord("Adobe Lightroom", nil, []PNGTextEntry{
		{Key: "Description", Value: "shot in a midjourney inspired style"},
		{Key: "Comment", Value: "stable diffusion of light through trees"},
	})
	if IsAIGenerated(r) {
		t.Error("AI keywords in arbitrary text fields should NOT trigger (FP guard)")
	}
	if v := AIVerdict(r); v != AIVerdictNotAI {
		t.Errorf("verdict: got %q, want %q", v, AIVerdictNotAI)
	}
}

func TestIsAIGenerated_NilSafe(t *testing.T) {
	if IsAIGenerated(nil) {
		t.Error("nil record should not be AI")
	}
	if IsAIGenerated(&MediaRecord{}) {
		t.Error("nil Exif should not be AI")
	}
	if v := AIVerdict(nil); v != AIVerdictUnknown {
		t.Errorf("nil verdict: got %q, want %q", v, AIVerdictUnknown)
	}
}

func TestAIVerdict_NotAIWithMetadata(t *testing.T) {
	// 有 EXIF（Sony 直出）但无 AI 信号 → not-ai
	r := &MediaRecord{
		Type: TypeImage,
		Exif: &ExifInfo{Software: "", HasDateTime: true},
	}
	if v := AIVerdict(r); v != AIVerdictNotAI {
		t.Errorf("verdict: got %q, want %q", v, AIVerdictNotAI)
	}
}

func TestAIVerdict_UnknownNoMetadata(t *testing.T) {
	// 无任何元数据（PNG screenshot 无 EXIF/C2PA/text）→ unknown
	r := &MediaRecord{Type: TypeImage, Exif: &ExifInfo{}}
	if v := AIVerdict(r); v != AIVerdictUnknown {
		t.Errorf("verdict: got %q, want %q", v, AIVerdictUnknown)
	}
}

func TestClassifySoftware_AIPriority(t *testing.T) {
	// "Adobe Firefly" 含 "firefly" AI 关键字 → 归 AI，不归 editor
	if cls := classifySoftware("Adobe Firefly"); cls != softwareAI {
		t.Errorf("'Adobe Firefly': got %v, want softwareAI", cls)
	}
	// DALL·E → AI
	if cls := classifySoftware("DALL·E 3.0"); cls != softwareAI {
		t.Errorf("'DALL·E 3.0': got %v, want softwareAI", cls)
	}
	// Lightroom 仍归 editor（不含 AI 关键字）
	if cls := classifySoftware("Adobe Lightroom Classic"); cls != softwareEditor {
		t.Errorf("'Adobe Lightroom': got %v, want softwareEditor", cls)
	}
}

func TestAISignals_C2PAAndSoftware(t *testing.T) {
	r := aiRecord("DALL·E 3.0", &C2PAInfo{Present: true, Generator: "DALL·E 3.0"}, nil)
	sigs := AISignals(r)
	if len(sigs) < 2 {
		t.Fatalf("expected ≥2 signals, got %v", sigs)
	}
	foundC2PA, foundSoftware := false, false
	for _, s := range sigs {
		if contains(s, "C2PA") {
			foundC2PA = true
		}
		if contains(s, "Software") {
			foundSoftware = true
		}
	}
	if !foundC2PA || !foundSoftware {
		t.Errorf("missing signals in %v", sigs)
	}
}

func TestAISignals_NoSignal(t *testing.T) {
	r := &MediaRecord{Type: TypeImage, Exif: &ExifInfo{HasDateTime: true}}
	sigs := AISignals(r)
	if len(sigs) != 1 || !contains(sigs[0], "未检测到") {
		t.Errorf("expected single 'no signal' line, got %v", sigs)
	}
}
