package cache

import (
	"bytes"
	"database/sql"
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"

	"github.com/xunull/imfd/internal/media"
)

// schemaVersion must be incremented whenever cacheRecord fields change.
const schemaVersion = 1

// cacheRecord mirrors media.MediaRecord for gob serialization.
// Excludes the Error field (interface type; only successful records are cached).
type cacheRecord struct {
	FilePath       string
	FileName       string
	FileSize       int64
	ModTime        time.Time
	Type           media.MediaType
	Exif           *media.ExifInfo
	Video          *media.VideoInfo
	Audio          *media.AudioInfo
	Location       *media.GeoLocation
	CaptureTime    time.Time
	HasCaptureTime bool
	Attributes     map[string]string
}

// Stats holds cache database statistics.
type Stats struct {
	Path      string
	Entries   int64
	SizeBytes int64
	OldestAt  time.Time
}

// Cache wraps a SQLite database for MediaRecord caching keyed by (abs_path, mtime_ns).
type Cache struct {
	db   *sql.DB
	path string
}

// DefaultDir returns the default cache directory, respecting XDG_CACHE_HOME.
func DefaultDir() string {
	if xdg := os.Getenv("XDG_CACHE_HOME"); xdg != "" {
		return filepath.Join(xdg, "imfd")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cache", "imfd")
}

// Open opens (or creates) the cache database at dir/cache.db.
// On schema version mismatch the record table is dropped and recreated (full cold start).
// Returns a non-nil error only when the DB is structurally unusable.
func Open(dir string) (*Cache, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("创建 cache 目录失败: %w", err)
	}

	dbPath := filepath.Join(dir, "cache.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("打开 cache DB 失败: %w", err)
	}
	// Single connection avoids "database is locked" under concurrent goroutines.
	// WAL mode serialises writers at the SQLite level; reads remain concurrent.
	db.SetMaxOpenConns(1)

	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("设置 WAL 模式失败: %w", err)
	}
	if _, err := db.Exec("PRAGMA busy_timeout=5000"); err != nil {
		db.Close()
		return nil, fmt.Errorf("设置 busy_timeout 失败: %w", err)
	}

	c := &Cache{db: db, path: dbPath}
	if err := c.initSchema(); err != nil {
		db.Close()
		return nil, err
	}
	return c, nil
}

func (c *Cache) initSchema() error {
	if _, err := c.db.Exec(`
		CREATE TABLE IF NOT EXISTS meta (
			key   TEXT NOT NULL PRIMARY KEY,
			value TEXT NOT NULL
		)
	`); err != nil {
		return fmt.Errorf("创建 meta 表失败: %w", err)
	}

	var ver int
	err := c.db.QueryRow(`SELECT CAST(value AS INTEGER) FROM meta WHERE key = 'schema_ver'`).Scan(&ver)
	if err == sql.ErrNoRows {
		return c.createRecordTable()
	}
	if err != nil {
		return fmt.Errorf("读取 schema_ver 失败: %w", err)
	}

	if ver != schemaVersion {
		if _, err := c.db.Exec(`DROP TABLE IF EXISTS record_cache`); err != nil {
			return fmt.Errorf("清除旧 cache 表失败: %w", err)
		}
		return c.createRecordTable()
	}

	_, err = c.db.Exec(`
		CREATE TABLE IF NOT EXISTS record_cache (
			abs_path  TEXT    NOT NULL PRIMARY KEY,
			mtime_ns  INTEGER NOT NULL,
			cached_at INTEGER NOT NULL,
			data      BLOB    NOT NULL
		)
	`)
	return err
}

func (c *Cache) createRecordTable() error {
	if _, err := c.db.Exec(`
		CREATE TABLE IF NOT EXISTS record_cache (
			abs_path  TEXT    NOT NULL PRIMARY KEY,
			mtime_ns  INTEGER NOT NULL,
			cached_at INTEGER NOT NULL,
			data      BLOB    NOT NULL
		)
	`); err != nil {
		return fmt.Errorf("创建 record_cache 表失败: %w", err)
	}
	_, err := c.db.Exec(`INSERT OR REPLACE INTO meta (key, value) VALUES ('schema_ver', ?)`, schemaVersion)
	return err
}

// Close closes the underlying database.
func (c *Cache) Close() error {
	return c.db.Close()
}

// Get returns the cached record for absPath if mtime_ns matches exactly.
// Returns (nil, false) on any miss or decode error.
func (c *Cache) Get(absPath string, mtimeNs int64) (*media.MediaRecord, bool) {
	var storedMtime int64
	var data []byte
	err := c.db.QueryRow(
		`SELECT mtime_ns, data FROM record_cache WHERE abs_path = ?`, absPath,
	).Scan(&storedMtime, &data)
	if err != nil {
		return nil, false
	}
	if storedMtime != mtimeNs {
		return nil, false
	}

	var cr cacheRecord
	if err := gob.NewDecoder(bytes.NewReader(data)).Decode(&cr); err != nil {
		return nil, false
	}
	return fromCache(cr), true
}

// Set writes the record to cache. Only call for records with Error == nil.
// Write errors are silently ignored by the caller; cache is a performance layer only.
func (c *Cache) Set(absPath string, mtimeNs int64, r *media.MediaRecord) error {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(toCache(r)); err != nil {
		return fmt.Errorf("序列化 record 失败: %w", err)
	}
	_, err := c.db.Exec(
		`INSERT OR REPLACE INTO record_cache (abs_path, mtime_ns, cached_at, data) VALUES (?, ?, ?, ?)`,
		absPath, mtimeNs, time.Now().Unix(), buf.Bytes(),
	)
	return err
}

// GetStats returns aggregate statistics about the cache database.
func (c *Cache) GetStats() (Stats, error) {
	var s Stats
	s.Path = c.path

	if err := c.db.QueryRow(`SELECT COUNT(*) FROM record_cache`).Scan(&s.Entries); err != nil {
		return s, err
	}

	if fi, err := os.Stat(c.path); err == nil {
		s.SizeBytes = fi.Size()
	}

	var oldest sql.NullInt64
	if err := c.db.QueryRow(`SELECT MIN(cached_at) FROM record_cache`).Scan(&oldest); err == nil && oldest.Valid && oldest.Int64 > 0 {
		s.OldestAt = time.Unix(oldest.Int64, 0)
	}
	return s, nil
}

// Clean deletes entries whose cached_at is older than the given duration.
// Returns the number of deleted rows.
func (c *Cache) Clean(olderThan time.Duration) (int64, error) {
	cutoff := time.Now().Add(-olderThan).Unix()
	res, err := c.db.Exec(`DELETE FROM record_cache WHERE cached_at < ?`, cutoff)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// Clear deletes all cache entries and runs VACUUM to reclaim disk space.
// Returns the number of deleted rows.
func (c *Cache) Clear() (int64, error) {
	res, err := c.db.Exec(`DELETE FROM record_cache`)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	_, _ = c.db.Exec(`VACUUM`)
	return n, nil
}

func toCache(r *media.MediaRecord) cacheRecord {
	return cacheRecord{
		FilePath:       r.FilePath,
		FileName:       r.FileName,
		FileSize:       r.FileSize,
		ModTime:        r.ModTime,
		Type:           r.Type,
		Exif:           r.Exif,
		Video:          r.Video,
		Audio:          r.Audio,
		Location:       r.Location,
		CaptureTime:    r.CaptureTime,
		HasCaptureTime: r.HasCaptureTime,
		Attributes:     r.Attributes,
	}
}

func fromCache(cr cacheRecord) *media.MediaRecord {
	return &media.MediaRecord{
		FilePath:       cr.FilePath,
		FileName:       cr.FileName,
		FileSize:       cr.FileSize,
		ModTime:        cr.ModTime,
		Type:           cr.Type,
		Exif:           cr.Exif,
		Video:          cr.Video,
		Audio:          cr.Audio,
		Location:       cr.Location,
		CaptureTime:    cr.CaptureTime,
		HasCaptureTime: cr.HasCaptureTime,
		Attributes:     cr.Attributes,
	}
}
