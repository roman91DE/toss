package db

import (
	"crypto/rand"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type Entry struct {
	ID           string
	OriginalPath string
	BinName      string
	TossedAt     time.Time
	IsDir        bool
	SizeBytes    int64
}

const schema = `
CREATE TABLE IF NOT EXISTS entries (
	id           TEXT PRIMARY KEY,
	original_path TEXT NOT NULL,
	bin_name      TEXT NOT NULL,
	tossed_at     DATETIME NOT NULL,
	is_dir        INTEGER NOT NULL,
	size_bytes    INTEGER NOT NULL
);`

func Open(path string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, fmt.Errorf("creating db dir: %w", err)
	}
	d, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("opening db: %w", err)
	}
	if _, err := d.Exec(schema); err != nil {
		d.Close()
		return nil, fmt.Errorf("initializing schema: %w", err)
	}
	return d, nil
}

func NewID() string {
	b := make([]byte, 16)
	rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func Append(d *sql.DB, e Entry) error {
	_, err := d.Exec(
		`INSERT INTO entries (id, original_path, bin_name, tossed_at, is_dir, size_bytes)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		e.ID, e.OriginalPath, e.BinName, e.TossedAt.UTC().Format(time.RFC3339), boolToInt(e.IsDir), e.SizeBytes,
	)
	return err
}

func Remove(d *sql.DB, id string) error {
	_, err := d.Exec(`DELETE FROM entries WHERE id = ?`, id)
	return err
}

func All(d *sql.DB) ([]Entry, error) {
	rows, err := d.Query(`SELECT id, original_path, bin_name, tossed_at, is_dir, size_bytes FROM entries ORDER BY tossed_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanEntries(rows)
}

func FindByQuery(d *sql.DB, query string) ([]Entry, error) {
	lower := "%" + strings.ToLower(query) + "%"
	rows, err := d.Query(
		`SELECT id, original_path, bin_name, tossed_at, is_dir, size_bytes FROM entries
		 WHERE LOWER(original_path) LIKE ? OR LOWER(bin_name) LIKE ?
		 ORDER BY tossed_at`,
		lower, lower,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanEntries(rows)
}

func scanEntries(rows *sql.Rows) ([]Entry, error) {
	var entries []Entry
	for rows.Next() {
		var e Entry
		var tossedStr string
		var isDir int
		if err := rows.Scan(&e.ID, &e.OriginalPath, &e.BinName, &tossedStr, &isDir, &e.SizeBytes); err != nil {
			return nil, err
		}
		t, err := time.Parse(time.RFC3339, tossedStr)
		if err != nil {
			return nil, fmt.Errorf("parsing tossed_at: %w", err)
		}
		e.TossedAt = t.Local()
		e.IsDir = isDir != 0
		e.BinName = filepath.Base(e.BinName) // sanitize just in case
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
