package ui

import (
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/roman91DE/toss/internal/db"
)

// replaceStdin swaps os.Stdin with a pipe whose write end receives input.
// The swap and cleanup are sequential-test-safe (no t.Parallel in callers).
func replaceStdin(t *testing.T, input string) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	old := os.Stdin
	os.Stdin = r
	t.Cleanup(func() {
		os.Stdin = old
		r.Close()
	})
	go func() {
		defer w.Close()
		w.WriteString(input) //nolint:errcheck
	}()
}

// captureStdout runs fn and returns everything written to os.Stdout during fn.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	old := os.Stdout
	os.Stdout = w

	outCh := make(chan []byte, 1)
	go func() {
		b, _ := io.ReadAll(r)
		outCh <- b
	}()

	fn()
	w.Close()
	os.Stdout = old
	data := <-outCh // wait for goroutine to finish before closing read end
	r.Close()
	return string(data)
}

// FormatSize tests (pure, parallelisable)

func TestFormatSize_Bytes(t *testing.T) {
	t.Parallel()
	cases := []struct {
		input int64
		want  string
	}{
		{0, "0B"},
		{1, "1B"},
		{1023, "1023B"},
	}
	for _, c := range cases {
		got := FormatSize(c.input)
		if got != c.want {
			t.Errorf("FormatSize(%d): want %q, got %q", c.input, c.want, got)
		}
	}
}

func TestFormatSize_KB(t *testing.T) {
	t.Parallel()
	cases := []struct {
		input int64
		want  string
	}{
		{1024, "1.0KB"},
		{1536, "1.5KB"},
		{1023 * 1024, "1023.0KB"},
	}
	for _, c := range cases {
		got := FormatSize(c.input)
		if got != c.want {
			t.Errorf("FormatSize(%d): want %q, got %q", c.input, c.want, got)
		}
	}
}

func TestFormatSize_MB(t *testing.T) {
	t.Parallel()
	cases := []struct {
		input int64
		want  string
	}{
		{1024 * 1024, "1.0MB"},
		{5 * 1024 * 1024, "5.0MB"},
	}
	for _, c := range cases {
		got := FormatSize(c.input)
		if got != c.want {
			t.Errorf("FormatSize(%d): want %q, got %q", c.input, c.want, got)
		}
	}
}

func TestFormatSize_GB(t *testing.T) {
	t.Parallel()
	cases := []struct {
		input int64
		want  string
	}{
		{1024 * 1024 * 1024, "1.0GB"},
		{2 * 1024 * 1024 * 1024, "2.0GB"},
	}
	for _, c := range cases {
		got := FormatSize(c.input)
		if got != c.want {
			t.Errorf("FormatSize(%d): want %q, got %q", c.input, c.want, got)
		}
	}
}

// Confirm tests (stdin replacement — no t.Parallel)

func TestConfirm_Yes(t *testing.T) {
	replaceStdin(t, "y\n")
	got, err := Confirm("Delete?")
	if err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	if !got {
		t.Error("expected true for 'y'")
	}
}

func TestConfirm_YesFull(t *testing.T) {
	replaceStdin(t, "yes\n")
	got, err := Confirm("Delete?")
	if err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	if !got {
		t.Error("expected true for 'yes'")
	}
}

func TestConfirm_No(t *testing.T) {
	replaceStdin(t, "n\n")
	got, err := Confirm("Delete?")
	if err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	if got {
		t.Error("expected false for 'n'")
	}
}

func TestConfirm_Default(t *testing.T) {
	replaceStdin(t, "\n")
	got, err := Confirm("Delete?")
	if err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	if got {
		t.Error("expected false for empty input (default N)")
	}
}

func TestConfirm_CaseInsensitive(t *testing.T) {
	replaceStdin(t, "Y\n")
	got, err := Confirm("Delete?")
	if err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	if !got {
		t.Error("expected true for 'Y'")
	}
}

// PickEntry tests (stdin replacement — no t.Parallel)

func makeTestEntries(n int) []db.Entry {
	entries := make([]db.Entry, n)
	for i := range entries {
		entries[i] = db.Entry{
			ID:           "id",
			OriginalPath: "/path/to/file.txt",
			BinName:      "bin-file.txt",
			TossedAt:     time.Now(),
		}
	}
	return entries
}

func TestPickEntry_ValidSelection(t *testing.T) {
	entries := makeTestEntries(2)
	entries[0].OriginalPath = "/first.txt"
	entries[1].OriginalPath = "/second.txt"
	replaceStdin(t, "1\n")
	got, err := PickEntry(entries)
	if err != nil {
		t.Fatalf("PickEntry: %v", err)
	}
	if got.OriginalPath != "/first.txt" {
		t.Errorf("want first entry, got %q", got.OriginalPath)
	}
}

func TestPickEntry_InvalidIndex(t *testing.T) {
	entries := makeTestEntries(2)
	replaceStdin(t, "0\n")
	_, err := PickEntry(entries)
	if err == nil {
		t.Error("expected error for index 0")
	}
}

func TestPickEntry_OutOfRange(t *testing.T) {
	entries := makeTestEntries(2)
	replaceStdin(t, "5\n")
	_, err := PickEntry(entries)
	if err == nil {
		t.Error("expected error for index out of range")
	}
}

func TestPickEntry_NonNumeric(t *testing.T) {
	entries := makeTestEntries(2)
	replaceStdin(t, "abc\n")
	_, err := PickEntry(entries)
	if err == nil {
		t.Error("expected error for non-numeric input")
	}
}

// PrintTable tests (stdout capture)

func TestPrintTable_ContainsHeaders(t *testing.T) {
	output := captureStdout(t, func() {
		PrintTable(nil)
	})
	for _, header := range []string{"TOSSED AT", "SIZE", "PATH"} {
		if !strings.Contains(output, header) {
			t.Errorf("output missing header %q; got:\n%s", header, output)
		}
	}
}

func TestPrintTable_RowCount(t *testing.T) {
	entries := []db.Entry{
		{OriginalPath: "/a.txt", TossedAt: time.Now(), SizeBytes: 100},
		{OriginalPath: "/b.txt", TossedAt: time.Now(), SizeBytes: 200},
		{OriginalPath: "/c.txt", TossedAt: time.Now(), SizeBytes: 300},
	}
	output := captureStdout(t, func() {
		PrintTable(entries)
	})
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
	// lines[0] is the header row; the rest are data rows
	dataLines := lines[1:]
	if len(dataLines) != len(entries) {
		t.Errorf("want %d data rows, got %d; output:\n%s", len(entries), len(dataLines), output)
	}
}

func TestPrintTable_DirSuffix(t *testing.T) {
	entries := []db.Entry{
		{OriginalPath: "/mydir", IsDir: true, TossedAt: time.Now(), SizeBytes: 0},
	}
	output := captureStdout(t, func() {
		PrintTable(entries)
	})
	if !strings.Contains(output, "[dir]") {
		t.Errorf("expected '[dir]' in output for directory entry; got:\n%s", output)
	}
}
