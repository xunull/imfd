package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xunull/imfd/internal/media"
)

// resetViewFlags resets all view flag vars between tests.
// Also forces currentOS to "darwin" so happy-path tests run on Linux CI.
// Tests that want to test the non-mac rejection path must override
// currentOS themselves after calling this (see TestRunView_NonMacReturnsError).
func resetViewFlags(t *testing.T) {
	t.Helper()
	flagViewType = "all"
	flagViewCameraMakes = nil
	flagViewCameraModels = nil
	flagViewLensModels = nil
	flagViewDevice = ""
	flagViewCodecs = nil
	flagViewAudioCodecs = nil
	flagViewVideoCodecs = nil
	flagViewProvinces = nil
	flagViewCities = nil
	flagViewScene = ""
	flagViewISO = ""
	flagViewYear = ""
	flagViewFilter = ""
	flagViewRename = ""
	flagViewNoOpen = true // always skip Finder in tests
	flagViewNoCache = true
	flagViewWorkers = 2
	flagViewExtractors = 2
	flagViewChannelSize = 16
	flagViewGeoProvider = "offline"

	// Force darwin so the platform guard in runView lets the happy path
	// execute on Linux CI runners. Auto-restore at end of test.
	origOS := currentOS
	currentOS = "darwin"
	t.Cleanup(func() { currentOS = origOS })
}

// --- viewDirPath ---

func TestViewDirPath_SameArgsSameDir(t *testing.T) {
	paths := []string{"/Users/q/Photos"}
	d1 := viewDirPath("true", paths)
	d2 := viewDirPath("true", paths)
	if d1 != d2 {
		t.Errorf("same args should produce same dir: %s vs %s", d1, d2)
	}
}

func TestViewDirPath_DifferentFilterDifferentDir(t *testing.T) {
	paths := []string{"/Users/q/Photos"}
	d1 := viewDirPath("true", paths)
	d2 := viewDirPath("(province == \"云南\")", paths)
	if d1 == d2 {
		t.Error("different filter should produce different dir")
	}
}

func TestViewDirPath_DifferentPathsDifferentDir(t *testing.T) {
	d1 := viewDirPath("true", []string{"/a"})
	d2 := viewDirPath("true", []string{"/b"})
	if d1 == d2 {
		t.Error("different paths should produce different dir")
	}
}

func TestViewDirPath_PathOrderIndependent(t *testing.T) {
	d1 := viewDirPath("true", []string{"/a", "/b"})
	d2 := viewDirPath("true", []string{"/b", "/a"})
	if d1 != d2 {
		t.Error("path order should not affect hash")
	}
}

func TestViewDirPath_UnderTmpDir(t *testing.T) {
	d := viewDirPath("true", []string{"/a"})
	if !strings.HasPrefix(d, os.TempDir()) {
		t.Errorf("expected prefix %s, got %s", os.TempDir(), d)
	}
	if !strings.Contains(filepath.Base(d), "imfd-view-") {
		t.Errorf("expected imfd-view- prefix in base, got %s", filepath.Base(d))
	}
}

// --- uniqueSymlinkPath ---

func TestUniqueSymlinkPath_NoConflict(t *testing.T) {
	dir := t.TempDir()
	got := uniqueSymlinkPath(dir, "photo.jpg")
	want := filepath.Join(dir, "photo.jpg")
	if got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestUniqueSymlinkPath_OneConflict(t *testing.T) {
	dir := t.TempDir()
	// Create a real file to force collision
	if err := os.WriteFile(filepath.Join(dir, "photo.jpg"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	got := uniqueSymlinkPath(dir, "photo.jpg")
	want := filepath.Join(dir, "photo_1.jpg")
	if got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestUniqueSymlinkPath_MultipleConflicts(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"photo.jpg", "photo_1.jpg", "photo_2.jpg"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	got := uniqueSymlinkPath(dir, "photo.jpg")
	want := filepath.Join(dir, "photo_3.jpg")
	if got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

// --- cleanOldSymlinks ---

func TestCleanOldSymlinks_RemovesSymlinks(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "real.jpg")
	if err := os.WriteFile(target, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link.jpg")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}

	cleanOldSymlinks(dir)

	if _, err := os.Lstat(link); !os.IsNotExist(err) {
		t.Error("symlink should be removed")
	}
}

func TestCleanOldSymlinks_PreservesRegularFiles(t *testing.T) {
	dir := t.TempDir()
	real := filepath.Join(dir, "keep.txt")
	if err := os.WriteFile(real, []byte("keep me"), 0644); err != nil {
		t.Fatal(err)
	}

	cleanOldSymlinks(dir)

	if _, err := os.Stat(real); err != nil {
		t.Errorf("regular file should survive cleanOldSymlinks: %v", err)
	}
}

func TestCleanOldSymlinks_NoErrorOnMissingDir(t *testing.T) {
	// Should not panic or return error on non-existent dir
	cleanOldSymlinks("/tmp/imfd-nonexistent-xyz-123456")
}

// --- applyViewTemplate ---

func TestApplyViewTemplate_AllVariables(t *testing.T) {
	r := &media.MediaRecord{
		FilePath:       "/Photos/IMG_001.JPG",
		Type:           media.TypeImage,
		HasCaptureTime: true,
		CaptureTime:    time.Date(2024, 3, 5, 0, 0, 0, 0, time.UTC),
		Exif: &media.ExifInfo{
			CameraMake:  "Sony",
			CameraModel: "A7IV",
			ISO:         "800",
		},
		Location: &media.GeoLocation{
			City:     "昆明",
			Province: "云南省",
		},
	}

	tmpl := "{year}-{month}-{day}_{city}_{camera_make}_{iso}.{ext}"
	got := applyViewTemplate(tmpl, r)
	want := "2024-03-05_昆明_Sony_800.jpg"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestApplyViewTemplate_DateShortcut(t *testing.T) {
	r := &media.MediaRecord{
		FilePath:       "/a/b.mp4",
		Type:           media.TypeVideo,
		HasCaptureTime: true,
		CaptureTime:    time.Date(2024, 1, 9, 0, 0, 0, 0, time.UTC),
	}
	got := applyViewTemplate("{date}.{ext}", r)
	want := "2024-01-09.mp4"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestApplyViewTemplate_FallbackToMtime(t *testing.T) {
	r := &media.MediaRecord{
		FilePath:       "/a/photo.jpg",
		Type:           media.TypeImage,
		HasCaptureTime: false,
		ModTime:        time.Date(2023, 6, 1, 0, 0, 0, 0, time.UTC),
	}
	got := applyViewTemplate("{year}", r)
	if got != "2023" {
		t.Errorf("expected mtime year 2023, got %q", got)
	}
}

func TestApplyViewTemplate_MissingExif(t *testing.T) {
	r := &media.MediaRecord{
		FilePath: "/a/photo.jpg",
		Type:     media.TypeImage,
		ModTime:  time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		// No Exif, no Location
	}
	got := applyViewTemplate("{camera_make}_{city}", r)
	if got != "Unknown_Unknown" {
		t.Errorf("got %q, want %q", got, "Unknown_Unknown")
	}
}

func TestApplyViewTemplate_FilenameWithoutExt(t *testing.T) {
	r := &media.MediaRecord{
		FilePath: "/Photos/IMG_042.JPEG",
		ModTime:  time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	got := applyViewTemplate("{filename}.{ext}", r)
	if got != "IMG_042.jpeg" {
		t.Errorf("got %q", got)
	}
}

// --- sanitizeFilename ---

func TestSanitizeFilename_SlashReplaced(t *testing.T) {
	got := sanitizeFilename("Sony/Canon")
	if strings.Contains(got, "/") {
		t.Errorf("slash should be replaced: %q", got)
	}
}

// --- runView: platform guard ---

func TestRunView_NonMacReturnsError(t *testing.T) {
	resetViewFlags(t)
	origOS := currentOS
	currentOS = "linux"
	defer func() { currentOS = origOS }()

	var stdout, stderr bytes.Buffer
	err := runView([]string{t.TempDir()}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected ErrMacOSOnly")
	}
	if !strings.Contains(err.Error(), "macOS") {
		t.Errorf("error should mention macOS: %v", err)
	}
	if !strings.Contains(stderr.String(), "error:") {
		t.Errorf("stderr should contain 'error:': %q", stderr.String())
	}
}

// --- runView: path validation ---

func TestRunView_PathNotFound(t *testing.T) {
	resetViewFlags(t)
	var stdout, stderr bytes.Buffer
	err := runView([]string{"/nonexistent/path/zzz"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for missing path")
	}
}

func TestRunView_PathIsFile(t *testing.T) {
	resetViewFlags(t)
	dir := t.TempDir()
	f := filepath.Join(dir, "file.jpg")
	if err := os.WriteFile(f, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	err := runView([]string{f}, &stdout, &stderr)
	if err == nil {
		t.Error("expected error when path is a file")
	}
}

// --- runView: symlink creation ---

func TestRunView_EmptyDirProducesZeroMatches(t *testing.T) {
	resetViewFlags(t)
	dir := t.TempDir()
	var stdout, stderr bytes.Buffer
	err := runView([]string{dir}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("empty dir should not error: %v", err)
	}
	if !strings.Contains(stderr.String(), "0 files matched") {
		t.Errorf("expected '0 files matched' in stderr: %q", stderr.String())
	}
	// stdout should be empty (no view dir printed)
	if stdout.Len() != 0 {
		t.Errorf("stdout should be empty on 0 matches, got %q", stdout.String())
	}
}

func TestRunView_CreatesSymlinkInViewDir(t *testing.T) {
	resetViewFlags(t)
	dir := t.TempDir()
	// Create a fake jpg file to trigger the walker
	fakeJpg := filepath.Join(dir, "photo.jpg")
	if err := os.WriteFile(fakeJpg, []byte("not-real-jpg"), 0644); err != nil {
		t.Fatal(err)
	}

	// Intercept openDir to prevent real Finder call
	called := false
	origOpen := openDir
	openDir = func(d string) error { called = true; return nil }
	defer func() { openDir = origOpen }()

	flagViewNoOpen = false // allow openDir to be called (we've replaced it)

	var stdout, stderr bytes.Buffer
	err := runView([]string{dir}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("runView: %v\nstderr: %s", err, stderr.String())
	}

	// stdout contains the view dir path
	vDir := strings.TrimSpace(stdout.String())
	if vDir == "" {
		t.Fatal("expected view dir path on stdout")
	}

	// View dir should contain exactly 1 symlink pointing to fakeJpg
	entries, err := os.ReadDir(vDir)
	if err != nil {
		t.Fatalf("ReadDir %s: %v", vDir, err)
	}
	var links []string
	for _, e := range entries {
		if e.Type()&os.ModeSymlink != 0 {
			links = append(links, e.Name())
		}
	}
	if len(links) != 1 {
		t.Errorf("expected 1 symlink, got %d: %v", len(links), links)
	}

	// Verify symlink target
	target, err := os.Readlink(filepath.Join(vDir, links[0]))
	if err != nil {
		t.Fatalf("Readlink: %v", err)
	}
	if target != fakeJpg {
		t.Errorf("symlink target: got %s, want %s", target, fakeJpg)
	}

	if !called {
		t.Error("openDir should have been called")
	}
}

func TestRunView_SymlinkTargetIsAbsolute(t *testing.T) {
	// Regression: symlink target must be absolute so Finder can open the file
	// even though the symlink lives in /tmp/imfd-view-xxx/ far from cwd.
	resetViewFlags(t)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "photo.jpg"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	origOpen := openDir
	openDir = func(d string) error { return nil }
	defer func() { openDir = origOpen }()
	flagViewNoOpen = false

	var stdout bytes.Buffer
	if err := runView([]string{dir}, &stdout, &bytes.Buffer{}); err != nil {
		t.Fatalf("runView: %v", err)
	}

	vDir := strings.TrimSpace(stdout.String())
	entries, _ := os.ReadDir(vDir)
	for _, e := range entries {
		if e.Type()&os.ModeSymlink != 0 {
			target, err := os.Readlink(filepath.Join(vDir, e.Name()))
			if err != nil {
				t.Fatalf("Readlink: %v", err)
			}
			if !filepath.IsAbs(target) {
				t.Errorf("symlink target must be absolute, got %q", target)
			}
		}
	}
}

func TestRunView_NoOpenFlag(t *testing.T) {
	resetViewFlags(t) // flagViewNoOpen = true already
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "photo.jpg"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	called := false
	origOpen := openDir
	openDir = func(d string) error { called = true; return nil }
	defer func() { openDir = origOpen }()

	var stdout, stderr bytes.Buffer
	if err := runView([]string{dir}, &stdout, &stderr); err != nil {
		t.Fatalf("runView: %v", err)
	}
	if called {
		t.Error("openDir should NOT be called when --no-open is set")
	}
}

func TestRunView_RenameTemplate(t *testing.T) {
	resetViewFlags(t)
	flagViewRename = "{type}_{filename}.{ext}"
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "IMG_001.jpg"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	origOpen := openDir
	openDir = func(d string) error { return nil }
	defer func() { openDir = origOpen }()
	flagViewNoOpen = false

	var stdout, stderr bytes.Buffer
	if err := runView([]string{dir}, &stdout, &stderr); err != nil {
		t.Fatalf("runView: %v\nstderr: %s", err, stderr.String())
	}

	vDir := strings.TrimSpace(stdout.String())
	entries, _ := os.ReadDir(vDir)
	var linkNames []string
	for _, e := range entries {
		if e.Type()&os.ModeSymlink != 0 {
			linkNames = append(linkNames, e.Name())
		}
	}
	if len(linkNames) != 1 {
		t.Fatalf("expected 1 symlink, got %v", linkNames)
	}
	// Name should start with "image_" or "unknown_" (fake jpg content won't parse)
	if !strings.HasSuffix(linkNames[0], ".jpg") {
		t.Errorf("symlink name should end with .jpg: %s", linkNames[0])
	}
}

func TestRunView_SameQuerySameViewDir(t *testing.T) {
	resetViewFlags(t)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.jpg"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	origOpen := openDir
	openDir = func(d string) error { return nil }
	defer func() { openDir = origOpen }()
	flagViewNoOpen = false

	var out1, out2 bytes.Buffer
	if err := runView([]string{dir}, &out1, &bytes.Buffer{}); err != nil {
		t.Fatalf("first run: %v", err)
	}
	if err := runView([]string{dir}, &out2, &bytes.Buffer{}); err != nil {
		t.Fatalf("second run: %v", err)
	}

	d1 := strings.TrimSpace(out1.String())
	d2 := strings.TrimSpace(out2.String())
	if d1 != d2 {
		t.Errorf("same query should produce same view dir:\n  run1=%s\n  run2=%s", d1, d2)
	}
}
