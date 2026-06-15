//go:build integration

// Integration tests for imfd verify against real exiftool-generated fixtures.
//
// 跑法:
//   go test -tags=integration ./cmd/
//
// 依赖 internal/extract/testdata/gen.sh 生成的图像 fixture：
//   - image_original_sony.jpg          → verdict=original
//   - image_edited_lightroom.jpg       → verdict=edited
//   - image_camera_rendered_sony.jpg   → verdict=camera-rendered
//   - image_no_exif.png                → verdict=unknown

package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const verifyFixtureDir = "../internal/extract/testdata"

func verifyFixturePath(t *testing.T, name string) string {
	t.Helper()
	p := filepath.Join(verifyFixtureDir, name)
	if _, err := os.Stat(p); err != nil {
		t.Skipf("fixture %s missing (run testdata/gen.sh): %v", name, err)
	}
	return p
}

func TestIntegration_VerifyOriginalSony(t *testing.T) {
	resetVerifyFlags(t)
	path := verifyFixturePath(t, "image_original_sony.jpg")

	var stdout, stderr bytes.Buffer
	if err := runVerify([]string{path}, "json", false, &stdout, &stderr); err != nil {
		t.Fatalf("runVerify: %v\nstderr: %s", err, stderr.String())
	}

	var r struct {
		Verdict  string `json:"verdict"`
		IsEdited bool   `json:"is_edited"`
		Software string `json:"software"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &r); err != nil {
		t.Fatalf("invalid JSON: %v\nbytes: %s", err, stdout.String())
	}
	if r.Verdict != "original" {
		t.Errorf("verdict: got %q, want 'original'", r.Verdict)
	}
	if r.IsEdited {
		t.Error("is_edited should be false for Sony OOC")
	}
	if r.Software != "" {
		t.Errorf("software should be empty for OOC, got %q", r.Software)
	}
}

func TestIntegration_VerifyLightroomEdited(t *testing.T) {
	resetVerifyFlags(t)
	path := verifyFixturePath(t, "image_edited_lightroom.jpg")

	var stdout, stderr bytes.Buffer
	if err := runVerify([]string{path}, "json", false, &stdout, &stderr); err != nil {
		t.Fatalf("runVerify: %v\nstderr: %s", err, stderr.String())
	}

	var r struct {
		Verdict  string   `json:"verdict"`
		IsEdited bool     `json:"is_edited"`
		Software string   `json:"software"`
		Signals  []string `json:"signals"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &r); err != nil {
		t.Fatalf("invalid JSON: %v\nbytes: %s", err, stdout.String())
	}
	if r.Verdict != "edited" {
		t.Errorf("verdict: got %q, want 'edited'", r.Verdict)
	}
	if !r.IsEdited {
		t.Error("is_edited should be true for Lightroom export")
	}
	if !strings.Contains(strings.ToLower(r.Software), "lightroom") {
		t.Errorf("software should contain 'lightroom', got %q", r.Software)
	}
	// 至少一条 signal 提到 lightroom
	foundSoftwareSignal := false
	for _, s := range r.Signals {
		if strings.Contains(strings.ToLower(s), "lightroom") {
			foundSoftwareSignal = true
			break
		}
	}
	if !foundSoftwareSignal {
		t.Errorf("expected signal mentioning lightroom, got: %v", r.Signals)
	}
}

func TestIntegration_VerifyCameraRendered(t *testing.T) {
	resetVerifyFlags(t)
	path := verifyFixturePath(t, "image_camera_rendered_sony.jpg")

	var stdout, stderr bytes.Buffer
	if err := runVerify([]string{path}, "json", false, &stdout, &stderr); err != nil {
		t.Fatalf("runVerify: %v\nstderr: %s", err, stderr.String())
	}

	var r struct {
		Verdict  string `json:"verdict"`
		IsEdited bool   `json:"is_edited"`
		Software string `json:"software"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &r); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if r.Verdict != "camera-rendered" {
		t.Errorf("verdict: got %q, want 'camera-rendered'", r.Verdict)
	}
	if r.IsEdited {
		t.Error("Imaging Edge should NOT be flagged as edited")
	}
}

func TestIntegration_VerifyNoExifPNG(t *testing.T) {
	resetVerifyFlags(t)
	path := verifyFixturePath(t, "image_no_exif.png")

	var stdout, stderr bytes.Buffer
	if err := runVerify([]string{path}, "json", false, &stdout, &stderr); err != nil {
		t.Fatalf("runVerify: %v", err)
	}

	var r struct {
		Verdict  string `json:"verdict"`
		IsEdited bool   `json:"is_edited"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &r); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if r.Verdict != "unknown" {
		t.Errorf("verdict: got %q, want 'unknown' for PNG with no EXIF", r.Verdict)
	}
	if r.IsEdited {
		t.Error("PNG with no EXIF should not be edited")
	}
}

// TestIntegration_VerifyC2PAManifest 用真实 c2patool 签名的 JPEG 验证
// JUMBF/CBOR 解析在野外有效（不是 self-fulfilling synthetic oracle）。
// fixture 由 testdata/gen.sh 经 c2patool 生成（仓库已 checked-in 一份）。
func TestIntegration_VerifyC2PAManifest(t *testing.T) {
	resetVerifyFlags(t)
	path := verifyFixturePath(t, "image_ai_c2pa.jpg")

	var stdout, stderr bytes.Buffer
	if err := runVerify([]string{path}, "json", true, &stdout, &stderr); err != nil {
		t.Fatalf("runVerify: %v\nstderr: %s", err, stderr.String())
	}

	var r struct {
		AIVerdict     string `json:"ai_verdict"`
		IsAIGenerated bool   `json:"is_ai_generated"`
		C2PA          *struct {
			Present   bool   `json:"present"`
			Generator string `json:"generator"`
			Trust     string `json:"trust"`
		} `json:"c2pa"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &r); err != nil {
		t.Fatalf("invalid JSON: %v\nbytes: %s", err, stdout.String())
	}

	if r.AIVerdict != "ai-generated" {
		t.Errorf("ai_verdict: got %q, want 'ai-generated'", r.AIVerdict)
	}
	if !r.IsAIGenerated {
		t.Error("is_ai_generated should be true for C2PA-signed file")
	}
	if r.C2PA == nil || !r.C2PA.Present {
		t.Fatalf("c2pa should be present, got %+v", r.C2PA)
	}
	// 关键断言：真实文件的 claim_generator_info 被正确提取
	if r.C2PA.Generator == "" {
		t.Error("generator should be extracted from real C2PA manifest (regression: map-form claim_generator_info)")
	}
	if r.C2PA.Trust != "detection-only" {
		t.Errorf("trust: got %q, want 'detection-only'", r.C2PA.Trust)
	}
}
