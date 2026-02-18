package db

import (
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	d, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { d.Close() })
	return d
}

func makeEntry(id, path, binName string) Entry {
	return Entry{
		ID:           id,
		OriginalPath: path,
		BinName:      binName,
		TossedAt:     time.Now().Truncate(time.Second),
		IsDir:        false,
		SizeBytes:    42,
	}
}

func TestNewID_Format(t *testing.T) {
	id := NewID()
	parts := strings.Split(id, "-")
	if len(parts) != 5 {
		t.Fatalf("expected 5 parts, got %d: %q", len(parts), id)
	}
	lengths := []int{8, 4, 4, 4, 12}
	for i, want := range lengths {
		if got := len(parts[i]); got != want {
			t.Errorf("part %d: want len %d, got %d", i, want, got)
		}
	}
	// version nibble: first hex digit of part[2] should be '4'
	if parts[2][0] != '4' {
		t.Errorf("version nibble: want '4', got %c", parts[2][0])
	}
}

func TestNewID_Unique(t *testing.T) {
	seen := make(map[string]struct{}, 1000)
	for i := 0; i < 1000; i++ {
		id := NewID()
		if _, dup := seen[id]; dup {
			t.Fatalf("duplicate ID at iteration %d: %s", i, id)
		}
		seen[id] = struct{}{}
	}
}

func TestOpen_CreatesSchema(t *testing.T) {
	d := openTestDB(t)
	var count int
	if err := d.QueryRow(`SELECT COUNT(*) FROM entries`).Scan(&count); err != nil {
		t.Fatalf("SELECT COUNT(*): %v", err)
	}
}

func TestOpen_Idempotent(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	d1, err := Open(dbPath)
	if err != nil {
		t.Fatalf("first Open: %v", err)
	}
	d1.Close()
	d2, err := Open(dbPath)
	if err != nil {
		t.Fatalf("second Open: %v", err)
	}
	d2.Close()
}

func TestAppendAndAll_Roundtrip(t *testing.T) {
	d := openTestDB(t)
	now := time.Now().UTC().Truncate(time.Second)
	e := Entry{
		ID:           NewID(),
		OriginalPath: "/home/user/document.pdf",
		BinName:      "abc123-document.pdf",
		TossedAt:     now,
		IsDir:        false,
		SizeBytes:    12345,
	}
	if err := Append(d, e); err != nil {
		t.Fatalf("Append: %v", err)
	}
	entries, err := All(d)
	if err != nil {
		t.Fatalf("All: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("want 1 entry, got %d", len(entries))
	}
	got := entries[0]
	if got.ID != e.ID {
		t.Errorf("ID: want %q, got %q", e.ID, got.ID)
	}
	if got.OriginalPath != e.OriginalPath {
		t.Errorf("OriginalPath: want %q, got %q", e.OriginalPath, got.OriginalPath)
	}
	if got.BinName != e.BinName {
		t.Errorf("BinName: want %q, got %q", e.BinName, got.BinName)
	}
	if !got.TossedAt.UTC().Equal(now) {
		t.Errorf("TossedAt: want %v, got %v", now, got.TossedAt.UTC())
	}
	if got.IsDir != e.IsDir {
		t.Errorf("IsDir: want %v, got %v", e.IsDir, got.IsDir)
	}
	if got.SizeBytes != e.SizeBytes {
		t.Errorf("SizeBytes: want %d, got %d", e.SizeBytes, got.SizeBytes)
	}
}

func TestAppend_Dir(t *testing.T) {
	d := openTestDB(t)
	e := Entry{
		ID:           NewID(),
		OriginalPath: "/home/user/mydir",
		BinName:      "abc-mydir",
		TossedAt:     time.Now(),
		IsDir:        true,
		SizeBytes:    0,
	}
	if err := Append(d, e); err != nil {
		t.Fatalf("Append: %v", err)
	}
	entries, err := All(d)
	if err != nil {
		t.Fatalf("All: %v", err)
	}
	if len(entries) != 1 || !entries[0].IsDir {
		t.Errorf("IsDir: want true, got %v", entries[0].IsDir)
	}
}

func TestRemove_DeletesEntry(t *testing.T) {
	d := openTestDB(t)
	e := makeEntry(NewID(), "/some/file.txt", "id-file.txt")
	if err := Append(d, e); err != nil {
		t.Fatalf("Append: %v", err)
	}
	if err := Remove(d, e.ID); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	entries, err := All(d)
	if err != nil {
		t.Fatalf("All: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("want 0 entries after Remove, got %d", len(entries))
	}
}

func TestRemove_LeavesOtherEntries(t *testing.T) {
	d := openTestDB(t)
	e1 := makeEntry(NewID(), "/a.txt", "id1-a.txt")
	e2 := makeEntry(NewID(), "/b.txt", "id2-b.txt")
	for _, e := range []Entry{e1, e2} {
		if err := Append(d, e); err != nil {
			t.Fatalf("Append: %v", err)
		}
	}
	if err := Remove(d, e1.ID); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	entries, err := All(d)
	if err != nil {
		t.Fatalf("All: %v", err)
	}
	if len(entries) != 1 || entries[0].ID != e2.ID {
		t.Errorf("want only e2 remaining, got %+v", entries)
	}
}

func TestAll_OrderedByTossedAt(t *testing.T) {
	d := openTestDB(t)
	base := time.Now().UTC().Truncate(time.Second)
	e1 := Entry{ID: NewID(), OriginalPath: "/first.txt", BinName: "id1-first.txt", TossedAt: base, SizeBytes: 1}
	e2 := Entry{ID: NewID(), OriginalPath: "/second.txt", BinName: "id2-second.txt", TossedAt: base.Add(2 * time.Second), SizeBytes: 1}
	e3 := Entry{ID: NewID(), OriginalPath: "/third.txt", BinName: "id3-third.txt", TossedAt: base.Add(time.Second), SizeBytes: 1}
	// Insert out of order: e1, e3, e2
	for _, e := range []Entry{e1, e3, e2} {
		if err := Append(d, e); err != nil {
			t.Fatalf("Append: %v", err)
		}
	}
	entries, err := All(d)
	if err != nil {
		t.Fatalf("All: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("want 3 entries, got %d", len(entries))
	}
	// Should be ordered: e1 (base), e3 (base+1s), e2 (base+2s)
	want := []string{e1.ID, e3.ID, e2.ID}
	for i, w := range want {
		if entries[i].ID != w {
			t.Errorf("position %d: want ID %s, got %s", i, w, entries[i].ID)
		}
	}
}

func TestFindByQuery_MatchesFilename(t *testing.T) {
	d := openTestDB(t)
	e := makeEntry(NewID(), "/home/user/notes.txt", "id-notes.txt")
	if err := Append(d, e); err != nil {
		t.Fatalf("Append: %v", err)
	}
	results, err := FindByQuery(d, "notes")
	if err != nil {
		t.Fatalf("FindByQuery: %v", err)
	}
	if len(results) != 1 || results[0].ID != e.ID {
		t.Errorf("want 1 match for 'notes', got %d", len(results))
	}
}

func TestFindByQuery_MatchesPath(t *testing.T) {
	d := openTestDB(t)
	e := makeEntry(NewID(), "/home/user/projects/myapp/config.json", "id-config.json")
	if err := Append(d, e); err != nil {
		t.Fatalf("Append: %v", err)
	}
	results, err := FindByQuery(d, "projects")
	if err != nil {
		t.Fatalf("FindByQuery: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("want 1 match for 'projects', got %d", len(results))
	}
}

func TestFindByQuery_CaseInsensitive(t *testing.T) {
	d := openTestDB(t)
	e := makeEntry(NewID(), "/home/user/README.md", "id-README.md")
	if err := Append(d, e); err != nil {
		t.Fatalf("Append: %v", err)
	}
	for _, query := range []string{"readme", "README", "ReadMe"} {
		results, err := FindByQuery(d, query)
		if err != nil {
			t.Fatalf("FindByQuery(%q): %v", query, err)
		}
		if len(results) != 1 {
			t.Errorf("query %q: want 1 result, got %d", query, len(results))
		}
	}
}

func TestFindByQuery_NoMatch(t *testing.T) {
	d := openTestDB(t)
	e := makeEntry(NewID(), "/home/user/file.txt", "id-file.txt")
	if err := Append(d, e); err != nil {
		t.Fatalf("Append: %v", err)
	}
	results, err := FindByQuery(d, "zzznomatch")
	if err != nil {
		t.Fatalf("FindByQuery: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("want 0 results, got %d", len(results))
	}
}

func TestFindByQuery_MultipleMatches(t *testing.T) {
	d := openTestDB(t)
	e1 := makeEntry(NewID(), "/home/user/report.txt", "id1-report.txt")
	e2 := makeEntry(NewID(), "/home/user/report_final.txt", "id2-report_final.txt")
	e3 := makeEntry(NewID(), "/home/user/other.txt", "id3-other.txt")
	for _, e := range []Entry{e1, e2, e3} {
		if err := Append(d, e); err != nil {
			t.Fatalf("Append: %v", err)
		}
	}
	results, err := FindByQuery(d, "report")
	if err != nil {
		t.Fatalf("FindByQuery: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("want 2 results for 'report', got %d", len(results))
	}
}
