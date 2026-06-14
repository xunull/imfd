package media

import (
	"testing"
	"time"
)

// 测试矩阵（design doc Eng Review Section 3）：
//   - Lightroom Software           → edited
//   - Photoshop Software           → edited
//   - 相机内置 (Sony Imaging Edge)  → camera-rendered
//   - 完全无 Software 无 ModifyDate → original (有 DateTime)
//   - ModifyDate > DateTimeOriginal + 5d → edited（即使 Software 空）
//   - ModifyDate 与 DateTimeOriginal 差 30s → original (60s 窗口保护)
//   - nil-safe (nil record / nil Exif)
//   - Software 未归类关键字 → unknown

func recordWithExif(software string, dt, mod time.Time) *MediaRecord {
	exif := &ExifInfo{
		Software: software,
	}
	if !dt.IsZero() {
		exif.DateTimeOriginal = dt
		exif.HasDateTime = true
	}
	if !mod.IsZero() {
		exif.ModifyDate = mod
		exif.HasModifyDate = true
	}
	return &MediaRecord{
		FilePath: "/test.jpg",
		Type:     TypeImage,
		Exif:     exif,
	}
}

func TestIsEdited_LightroomSoftware(t *testing.T) {
	r := recordWithExif("Adobe Lightroom Classic 13.0", time.Time{}, time.Time{})
	if !IsEdited(r) {
		t.Error("Lightroom software should be detected as edited")
	}
	if v := Verdict(r); v != VerdictEdited {
		t.Errorf("Verdict: got %q, want %q", v, VerdictEdited)
	}
}

func TestIsEdited_PhotoshopSoftware(t *testing.T) {
	r := recordWithExif("Adobe Photoshop 25.0 (Macintosh)", time.Time{}, time.Time{})
	if !IsEdited(r) {
		t.Error("Photoshop software should be detected as edited")
	}
	if v := Verdict(r); v != VerdictEdited {
		t.Errorf("Verdict: got %q, want %q", v, VerdictEdited)
	}
}

func TestIsEdited_CameraSoftwareIsNotEdited(t *testing.T) {
	// Sony Imaging Edge 写 Software 字段，但不算 edited
	r := recordWithExif("Imaging Edge Desktop 1.2", time.Time{}, time.Time{})
	if IsEdited(r) {
		t.Error("Camera-rendered software should NOT be detected as edited")
	}
	if v := Verdict(r); v != VerdictCameraRendered {
		t.Errorf("Verdict: got %q, want %q", v, VerdictCameraRendered)
	}
}

func TestIsEdited_NoSoftwareNoModifyDate_HasDateTime(t *testing.T) {
	// 相机直出：Software 空、ModifyDate 缺、只有 DateTimeOriginal
	dt := time.Date(2024, 3, 15, 14, 23, 0, 0, time.UTC)
	r := recordWithExif("", dt, time.Time{})
	if IsEdited(r) {
		t.Error("OOC file should NOT be edited")
	}
	if v := Verdict(r); v != VerdictOriginal {
		t.Errorf("Verdict: got %q, want %q", v, VerdictOriginal)
	}
}

func TestIsEdited_ModifyDateLongAfterOriginal(t *testing.T) {
	// Software 空（被 metadata stripping 抹掉），但 ModifyDate 5 天后
	dt := time.Date(2024, 3, 15, 14, 23, 0, 0, time.UTC)
	mod := dt.Add(5 * 24 * time.Hour)
	r := recordWithExif("", dt, mod)
	if !IsEdited(r) {
		t.Error("ModifyDate 5 days after DateTimeOriginal should be edited")
	}
	if v := Verdict(r); v != VerdictEdited {
		t.Errorf("Verdict: got %q, want %q", v, VerdictEdited)
	}
}

func TestIsEdited_ModifyDateWithin60sWindow(t *testing.T) {
	// 相机内 RAW→JPEG 转换：ModifyDate 比 DateTimeOriginal 晚几十秒
	dt := time.Date(2024, 3, 15, 14, 23, 0, 0, time.UTC)
	mod := dt.Add(30 * time.Second) // 30s 内 → 不算 edited
	r := recordWithExif("", dt, mod)
	if IsEdited(r) {
		t.Error("ModifyDate within 60s window should NOT be edited (RAW conversion)")
	}
}

func TestIsEdited_NilRecord(t *testing.T) {
	if IsEdited(nil) {
		t.Error("nil record should return false (nil-safe)")
	}
	if v := Verdict(nil); v != VerdictUnknown {
		t.Errorf("Verdict(nil): got %q, want %q", v, VerdictUnknown)
	}
}

func TestIsEdited_NilExif(t *testing.T) {
	r := &MediaRecord{FilePath: "/test.mp4", Type: TypeVideo}
	if IsEdited(r) {
		t.Error("Record with nil Exif should return false")
	}
	if v := Verdict(r); v != VerdictUnknown {
		t.Errorf("Verdict(nil-exif): got %q, want %q", v, VerdictUnknown)
	}
}

func TestVerdict_UnknownSoftwareIsUnknown(t *testing.T) {
	// 写了不认识的 Software，没有 ModifyDate → unknown
	r := recordWithExif("SomeObscureChineseApp v3.2", time.Time{}, time.Time{})
	if v := Verdict(r); v != VerdictUnknown {
		t.Errorf("Verdict: got %q, want %q", v, VerdictUnknown)
	}
}

func TestVerdict_HasDateTime_NoModify_NoSoftware(t *testing.T) {
	// 标准的相机直出：DateTimeOriginal 有，其它都没
	dt := time.Date(2024, 3, 15, 14, 23, 0, 0, time.UTC)
	r := recordWithExif("", dt, time.Time{})
	if v := Verdict(r); v != VerdictOriginal {
		t.Errorf("Verdict: got %q, want %q", v, VerdictOriginal)
	}
}

func TestVerdict_CompletelyMissingMetadata(t *testing.T) {
	// PNG screenshot 没 EXIF / 没 DateTime / 没 ModifyDate
	r := recordWithExif("", time.Time{}, time.Time{})
	if v := Verdict(r); v != VerdictUnknown {
		t.Errorf("Verdict: got %q, want %q", v, VerdictUnknown)
	}
}

func TestClassifySoftware_CameraBeforeEditor(t *testing.T) {
	// 「Capture One」既含 "capture" 又含 "camera" 风险吗？我们的 camera keywords 没有 "capture"
	// 验证 Capture One 仍归 editor
	if cls := classifySoftware("Capture One 23 Pro"); cls != softwareEditor {
		t.Errorf("'Capture One 23 Pro': got %v, want softwareEditor", cls)
	}
}

func TestEditSignals_LightroomFile(t *testing.T) {
	dt := time.Date(2024, 3, 15, 14, 23, 0, 0, time.UTC)
	mod := dt.Add(5 * 24 * time.Hour)
	r := recordWithExif("Adobe Lightroom Classic 13.0", dt, mod)
	signals := EditSignals(r)
	if len(signals) < 2 {
		t.Fatalf("expected ≥2 signals, got %v", signals)
	}
	foundSoftware := false
	foundModify := false
	for _, s := range signals {
		if contains(s, "lightroom") || contains(s, "Lightroom") {
			foundSoftware = true
		}
		if contains(s, "ModifyDate") {
			foundModify = true
		}
	}
	if !foundSoftware {
		t.Errorf("missing Software signal in: %v", signals)
	}
	if !foundModify {
		t.Errorf("missing ModifyDate signal in: %v", signals)
	}
}

func TestEditSignals_NilRecord(t *testing.T) {
	signals := EditSignals(nil)
	if len(signals) != 1 {
		t.Errorf("expected 1 signal for nil record, got %v", signals)
	}
}

// contains is strings.Contains; copied to avoid import in test
func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
