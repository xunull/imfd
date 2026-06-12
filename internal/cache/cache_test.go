package cache

import (
	"bytes"
	"database/sql"
	"encoding/gob"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"github.com/xunull/imfd/internal/media"
)

func newTestCache(t *testing.T) *Cache {
	t.Helper()
	c, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}

func sampleRecord(path string) *media.MediaRecord {
	return &media.MediaRecord{
		FilePath: path,
		FileName: filepath.Base(path),
		FileSize: 1234567,
		ModTime:  time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		Type:     media.TypeImage,
	}
}

func TestCacheOpen_CreatesDir(t *testing.T) {
	dir := t.TempDir() + "/sub/imfd"
	c, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer c.Close()
	if c.path == "" {
		t.Error("path should be set")
	}
}

func TestCacheGetMiss(t *testing.T) {
	c := newTestCache(t)
	record, ok := c.Get("/non/existent/path.jpg", 12345)
	if ok || record != nil {
		t.Error("expected miss for non-existent entry")
	}
}

func TestCacheSetAndGet_HitMatch(t *testing.T) {
	c := newTestCache(t)
	path := "/test/photo.jpg"
	mtimeNs := int64(1704067200000000000)
	r := sampleRecord(path)

	if err := c.Set(path, mtimeNs, r); err != nil {
		t.Fatalf("Set: %v", err)
	}

	got, ok := c.Get(path, mtimeNs)
	if !ok {
		t.Fatal("expected cache hit")
	}
	if got.FilePath != r.FilePath || got.FileName != r.FileName || got.Type != r.Type {
		t.Errorf("got %+v, want %+v", got, r)
	}
}

func TestCacheGet_MtimeMismatch(t *testing.T) {
	c := newTestCache(t)
	path := "/test/photo.jpg"
	_ = c.Set(path, 111, sampleRecord(path))

	_, ok := c.Get(path, 222)
	if ok {
		t.Error("expected miss when mtime differs")
	}
}

func TestCacheSet_OverwritesPreviousEntry(t *testing.T) {
	c := newTestCache(t)
	path := "/test/photo.jpg"

	r1 := sampleRecord(path)
	r1.FileSize = 100
	_ = c.Set(path, 111, r1)

	r2 := sampleRecord(path)
	r2.FileSize = 200
	_ = c.Set(path, 222, r2)

	got, ok := c.Get(path, 222)
	if !ok {
		t.Fatal("expected cache hit for newer entry")
	}
	if got.FileSize != 200 {
		t.Errorf("expected FileSize 200, got %d", got.FileSize)
	}
	_, ok = c.Get(path, 111)
	if ok {
		t.Error("old mtime should miss after overwrite")
	}
}

func TestCacheStats(t *testing.T) {
	c := newTestCache(t)

	s, err := c.GetStats()
	if err != nil {
		t.Fatalf("GetStats: %v", err)
	}
	if s.Entries != 0 {
		t.Errorf("expected 0 entries, got %d", s.Entries)
	}

	_ = c.Set("/a.jpg", 1, sampleRecord("/a.jpg"))
	_ = c.Set("/b.jpg", 2, sampleRecord("/b.jpg"))

	s, err = c.GetStats()
	if err != nil {
		t.Fatalf("GetStats: %v", err)
	}
	if s.Entries != 2 {
		t.Errorf("expected 2 entries, got %d", s.Entries)
	}
	if s.SizeBytes <= 0 {
		t.Errorf("expected non-zero size")
	}
}

func TestCacheClean(t *testing.T) {
	c := newTestCache(t)

	// Insert an old entry by setting cached_at to 200 days ago via raw SQL.
	r := sampleRecord("/old.jpg")
	var buf bytes.Buffer
	_ = gob.NewEncoder(&buf).Encode(toCache(r))
	oldTime := time.Now().Add(-200 * 24 * time.Hour).Unix()
	_, err := c.db.Exec(
		`INSERT OR REPLACE INTO record_cache (abs_path, mtime_ns, cached_at, data) VALUES (?, ?, ?, ?)`,
		"/old.jpg", int64(1), oldTime, buf.Bytes(),
	)
	if err != nil {
		t.Fatalf("insert old entry: %v", err)
	}

	_ = c.Set("/new.jpg", 2, sampleRecord("/new.jpg"))

	n, err := c.Clean(90 * 24 * time.Hour)
	if err != nil {
		t.Fatalf("Clean: %v", err)
	}
	if n != 1 {
		t.Errorf("expected 1 deleted, got %d", n)
	}

	_, ok := c.Get("/new.jpg", 2)
	if !ok {
		t.Error("new entry should survive clean")
	}
	_, ok = c.Get("/old.jpg", 1)
	if ok {
		t.Error("old entry should be deleted")
	}
}

func TestCacheClear(t *testing.T) {
	c := newTestCache(t)
	_ = c.Set("/a.jpg", 1, sampleRecord("/a.jpg"))
	_ = c.Set("/b.jpg", 2, sampleRecord("/b.jpg"))

	n, err := c.Clear()
	if err != nil {
		t.Fatalf("Clear: %v", err)
	}
	if n != 2 {
		t.Errorf("expected 2 deleted, got %d", n)
	}

	s, _ := c.GetStats()
	if s.Entries != 0 {
		t.Errorf("expected 0 entries after clear, got %d", s.Entries)
	}
}

func TestCacheSchemaVersionMismatch(t *testing.T) {
	dir := t.TempDir()

	c1, err := Open(dir)
	if err != nil {
		t.Fatalf("first open: %v", err)
	}
	_ = c1.Set("/test.jpg", 123, sampleRecord("/test.jpg"))
	_ = c1.Close()

	// Simulate a future schema version by bumping the stored value.
	db, _ := sql.Open("sqlite", filepath.Join(dir, "cache.db"))
	_, _ = db.Exec(`UPDATE meta SET value = '999' WHERE key = 'schema_ver'`)
	_ = db.Close()

	c2, err := Open(dir)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer c2.Close()

	_, ok := c2.Get("/test.jpg", 123)
	if ok {
		t.Error("old entries should be gone after schema version mismatch")
	}
}
