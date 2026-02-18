package bin

import (
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

// Helpers

func writeFile(t *testing.T, path, content string, mode fs.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", path, err)
	}
	return string(b)
}

func checkPerm(t *testing.T, path string, want fs.FileMode) {
	t.Helper()
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatalf("Lstat(%q): %v", path, err)
	}
	got := info.Mode().Perm()
	if got != want {
		t.Errorf("perm(%q): want %04o, got %04o", path, want, got)
	}
}

func checkSymlink(t *testing.T, path, wantTarget string) {
	t.Helper()
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatalf("Lstat(%q): %v", path, err)
	}
	if info.Mode()&fs.ModeSymlink == 0 {
		t.Fatalf("%q is not a symlink (mode: %v)", path, info.Mode())
	}
	got, err := os.Readlink(path)
	if err != nil {
		t.Fatalf("Readlink(%q): %v", path, err)
	}
	if got != wantTarget {
		t.Errorf("symlink target: want %q, got %q", wantTarget, got)
	}
}

// zeroUmask sets umask to 0 and restores the old value on cleanup.
// Tests that use this must NOT call t.Parallel().
func zeroUmask(t *testing.T) {
	t.Helper()
	old := syscall.Umask(0)
	t.Cleanup(func() { syscall.Umask(old) })
}

// copyFile tests

func TestCopyFile_CopiesContent(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")
	writeFile(t, src, "hello world", 0644)
	if err := copyFile(src, dst, 0644); err != nil {
		t.Fatalf("copyFile: %v", err)
	}
	if got := readFile(t, dst); got != "hello world" {
		t.Errorf("content: want %q, got %q", "hello world", got)
	}
}

func TestCopyFile_PreservesPermissions(t *testing.T) {
	zeroUmask(t)
	dir := t.TempDir()
	src := filepath.Join(dir, "src.sh")
	dst := filepath.Join(dir, "dst.sh")
	writeFile(t, src, "#!/bin/sh", 0755)
	if err := copyFile(src, dst, 0755); err != nil {
		t.Fatalf("copyFile: %v", err)
	}
	checkPerm(t, dst, 0755)
}

// copyDir tests

func TestCopyDir_CopiesFiles(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	dst := filepath.Join(dir, "dst")
	writeFile(t, filepath.Join(src, "a.txt"), "aaa", 0644)
	writeFile(t, filepath.Join(src, "b.txt"), "bbb", 0644)
	if err := copyDir(src, dst); err != nil {
		t.Fatalf("copyDir: %v", err)
	}
	if got := readFile(t, filepath.Join(dst, "a.txt")); got != "aaa" {
		t.Errorf("a.txt: want 'aaa', got %q", got)
	}
	if got := readFile(t, filepath.Join(dst, "b.txt")); got != "bbb" {
		t.Errorf("b.txt: want 'bbb', got %q", got)
	}
}

func TestCopyDir_NestedStructure(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	dst := filepath.Join(dir, "dst")
	writeFile(t, filepath.Join(src, "sub", "file.txt"), "nested", 0644)
	if err := copyDir(src, dst); err != nil {
		t.Fatalf("copyDir: %v", err)
	}
	if got := readFile(t, filepath.Join(dst, "sub", "file.txt")); got != "nested" {
		t.Errorf("nested file: want 'nested', got %q", got)
	}
}

func TestCopyDir_PreservesFilePermissions(t *testing.T) {
	zeroUmask(t)
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	dst := filepath.Join(dir, "dst")
	writeFile(t, filepath.Join(src, "script.sh"), "#!/bin/sh", 0755)
	writeFile(t, filepath.Join(src, "data.bin"), "secret", 0600)
	if err := copyDir(src, dst); err != nil {
		t.Fatalf("copyDir: %v", err)
	}
	checkPerm(t, filepath.Join(dst, "script.sh"), 0755)
	checkPerm(t, filepath.Join(dst, "data.bin"), 0600)
}

func TestCopyDir_PreservesDirPermissions(t *testing.T) {
	zeroUmask(t)
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	dst := filepath.Join(dir, "dst")
	if err := os.MkdirAll(src, 0700); err != nil {
		t.Fatalf("MkdirAll src: %v", err)
	}
	subdir := filepath.Join(src, "sub")
	if err := os.Mkdir(subdir, 0750); err != nil {
		t.Fatalf("Mkdir sub: %v", err)
	}
	// Write file directly to avoid writeFile helper calling MkdirAll on subdir
	if err := os.WriteFile(filepath.Join(subdir, "f.txt"), []byte("x"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := copyDir(src, dst); err != nil {
		t.Fatalf("copyDir: %v", err)
	}
	checkPerm(t, dst, 0700)
	checkPerm(t, filepath.Join(dst, "sub"), 0750)
}

func TestCopyDir_RecreatesSymlinks(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	dst := filepath.Join(dir, "dst")
	if err := os.MkdirAll(src, 0755); err != nil {
		t.Fatalf("MkdirAll src: %v", err)
	}
	writeFile(t, filepath.Join(src, "real.txt"), "content", 0644)
	if err := os.Symlink("real.txt", filepath.Join(src, "link.txt")); err != nil {
		t.Fatalf("Symlink: %v", err)
	}
	if err := copyDir(src, dst); err != nil {
		t.Fatalf("copyDir: %v", err)
	}
	checkSymlink(t, filepath.Join(dst, "link.txt"), "real.txt")
}

func TestCopyDir_DanglingSymlink(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	dst := filepath.Join(dir, "dst")
	if err := os.MkdirAll(src, 0755); err != nil {
		t.Fatalf("MkdirAll src: %v", err)
	}
	if err := os.Symlink("/nonexistent/path", filepath.Join(src, "dangling.txt")); err != nil {
		t.Fatalf("Symlink: %v", err)
	}
	if err := copyDir(src, dst); err != nil {
		t.Fatalf("copyDir with dangling symlink: %v", err)
	}
	checkSymlink(t, filepath.Join(dst, "dangling.txt"), "/nonexistent/path")
}

func TestCopyDir_AbsoluteSymlink(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	dst := filepath.Join(dir, "dst")
	if err := os.MkdirAll(src, 0755); err != nil {
		t.Fatalf("MkdirAll src: %v", err)
	}
	absTarget := "/usr/bin/env"
	if err := os.Symlink(absTarget, filepath.Join(src, "env")); err != nil {
		t.Fatalf("Symlink: %v", err)
	}
	if err := copyDir(src, dst); err != nil {
		t.Fatalf("copyDir: %v", err)
	}
	checkSymlink(t, filepath.Join(dst, "env"), absTarget)
}

// copyThenDelete tests

func TestCopyThenDelete_File(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")
	writeFile(t, src, "hello", 0644)
	if err := copyThenDelete(src, dst); err != nil {
		t.Fatalf("copyThenDelete: %v", err)
	}
	if _, err := os.Lstat(src); !os.IsNotExist(err) {
		t.Errorf("src should be deleted after copyThenDelete")
	}
	if got := readFile(t, dst); got != "hello" {
		t.Errorf("dst content: want 'hello', got %q", got)
	}
}

func TestCopyThenDelete_Directory(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	dst := filepath.Join(dir, "dst")
	writeFile(t, filepath.Join(src, "sub", "file.txt"), "nested content", 0644)
	if err := copyThenDelete(src, dst); err != nil {
		t.Fatalf("copyThenDelete: %v", err)
	}
	if _, err := os.Lstat(src); !os.IsNotExist(err) {
		t.Errorf("src dir should be deleted")
	}
	if got := readFile(t, filepath.Join(dst, "sub", "file.txt")); got != "nested content" {
		t.Errorf("nested content: want 'nested content', got %q", got)
	}
}

func TestCopyThenDelete_TopLevelSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target.txt")
	writeFile(t, target, "target content", 0644)
	src := filepath.Join(dir, "link.txt")
	if err := os.Symlink(target, src); err != nil {
		t.Fatalf("Symlink: %v", err)
	}
	dst := filepath.Join(dir, "dst.txt")
	if err := copyThenDelete(src, dst); err != nil {
		t.Fatalf("copyThenDelete: %v", err)
	}
	if _, err := os.Lstat(src); !os.IsNotExist(err) {
		t.Errorf("src symlink should be deleted")
	}
	checkSymlink(t, dst, target)
}

func TestCopyThenDelete_DanglingSymlink(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "dangling")
	if err := os.Symlink("/nonexistent/path", src); err != nil {
		t.Fatalf("Symlink: %v", err)
	}
	dst := filepath.Join(dir, "dst")
	if err := copyThenDelete(src, dst); err != nil {
		t.Fatalf("copyThenDelete with dangling symlink: %v", err)
	}
	checkSymlink(t, dst, "/nonexistent/path")
}

// Move tests

func TestMove_RegularFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "hello.txt")
	binDir := filepath.Join(dir, "bin")
	writeFile(t, src, "content", 0644)
	entry, err := Move(src, binDir)
	if err != nil {
		t.Fatalf("Move: %v", err)
	}
	if _, err := os.Lstat(src); !os.IsNotExist(err) {
		t.Errorf("src should be gone after Move")
	}
	if entry.OriginalPath != src {
		t.Errorf("OriginalPath: want %q, got %q", src, entry.OriginalPath)
	}
	if entry.BinName == "" {
		t.Error("BinName should not be empty")
	}
	if entry.SizeBytes <= 0 {
		t.Errorf("SizeBytes should be > 0, got %d", entry.SizeBytes)
	}
	binPath := filepath.Join(binDir, entry.BinName)
	if _, err := os.Lstat(binPath); err != nil {
		t.Errorf("file should exist in bin at %q: %v", binPath, err)
	}
}

func TestMove_Directory(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "mydir")
	binDir := filepath.Join(dir, "bin")
	writeFile(t, filepath.Join(src, "file.txt"), "data", 0644)
	entry, err := Move(src, binDir)
	if err != nil {
		t.Fatalf("Move: %v", err)
	}
	if !entry.IsDir {
		t.Error("IsDir should be true for directory")
	}
	binPath := filepath.Join(binDir, entry.BinName, "file.txt")
	if _, err := os.Lstat(binPath); err != nil {
		t.Errorf("nested file should exist in bin: %v", err)
	}
}

func TestMove_Symlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target.txt")
	writeFile(t, target, "content", 0644)
	src := filepath.Join(dir, "link.txt")
	if err := os.Symlink(target, src); err != nil {
		t.Fatalf("Symlink: %v", err)
	}
	binDir := filepath.Join(dir, "bin")
	entry, err := Move(src, binDir)
	if err != nil {
		t.Fatalf("Move symlink: %v", err)
	}
	binPath := filepath.Join(binDir, entry.BinName)
	info, err := os.Lstat(binPath)
	if err != nil {
		t.Fatalf("Lstat bin item: %v", err)
	}
	if info.Mode()&fs.ModeSymlink == 0 {
		t.Error("item in bin should still be a symlink")
	}
}

func TestMove_Nonexistent(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "does_not_exist.txt")
	binDir := filepath.Join(dir, "bin")
	_, err := Move(src, binDir)
	if err == nil {
		t.Error("expected error for nonexistent src, got nil")
	}
}

func TestMove_UniqueNamesOnCollision(t *testing.T) {
	dir := t.TempDir()
	binDir := filepath.Join(dir, "bin")
	src1 := filepath.Join(dir, "file.txt")
	src2 := filepath.Join(dir, "sub", "file.txt")
	writeFile(t, src1, "first", 0644)
	writeFile(t, src2, "second", 0644)
	entry1, err := Move(src1, binDir)
	if err != nil {
		t.Fatalf("Move 1: %v", err)
	}
	entry2, err := Move(src2, binDir)
	if err != nil {
		t.Fatalf("Move 2: %v", err)
	}
	if entry1.BinName == entry2.BinName {
		t.Errorf("expected unique BinNames, both got %q", entry1.BinName)
	}
}

// Restore tests

func TestRestore_RegularFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "hello.txt")
	binDir := filepath.Join(dir, "bin")
	writeFile(t, src, "content", 0644)
	entry, err := Move(src, binDir)
	if err != nil {
		t.Fatalf("Move: %v", err)
	}
	if err := Restore(entry, binDir); err != nil {
		t.Fatalf("Restore: %v", err)
	}
	if got := readFile(t, src); got != "content" {
		t.Errorf("restored content: want 'content', got %q", got)
	}
	binPath := filepath.Join(binDir, entry.BinName)
	if _, err := os.Lstat(binPath); !os.IsNotExist(err) {
		t.Errorf("file should be gone from bin after Restore")
	}
}

func TestRestore_Directory(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "mydir")
	binDir := filepath.Join(dir, "bin")
	writeFile(t, filepath.Join(src, "sub", "file.txt"), "nested", 0644)
	entry, err := Move(src, binDir)
	if err != nil {
		t.Fatalf("Move: %v", err)
	}
	if err := Restore(entry, binDir); err != nil {
		t.Fatalf("Restore: %v", err)
	}
	if got := readFile(t, filepath.Join(src, "sub", "file.txt")); got != "nested" {
		t.Errorf("restored nested content: want 'nested', got %q", got)
	}
}

func TestRestore_RecreatesParentDirs(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "parent", "child", "file.txt")
	binDir := filepath.Join(dir, "bin")
	writeFile(t, src, "data", 0644)
	entry, err := Move(src, binDir)
	if err != nil {
		t.Fatalf("Move: %v", err)
	}
	if err := os.RemoveAll(filepath.Join(dir, "parent")); err != nil {
		t.Fatalf("RemoveAll: %v", err)
	}
	if err := Restore(entry, binDir); err != nil {
		t.Fatalf("Restore: %v", err)
	}
	if got := readFile(t, src); got != "data" {
		t.Errorf("restored content: want 'data', got %q", got)
	}
}

func TestRestore_RoundTrip_PreservesContent(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "binary.bin")
	binDir := filepath.Join(dir, "bin")
	content := []byte{0x00, 0xff, 0x7f, 0x80, 0x01}
	if err := os.WriteFile(src, content, 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	entry, err := Move(src, binDir)
	if err != nil {
		t.Fatalf("Move: %v", err)
	}
	if err := Restore(entry, binDir); err != nil {
		t.Fatalf("Restore: %v", err)
	}
	got, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("binary content not preserved: want %v, got %v", content, got)
	}
}

// Empty tests

func TestEmpty_RemovesFiles(t *testing.T) {
	dir := t.TempDir()
	binDir := filepath.Join(dir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	writeFile(t, filepath.Join(binDir, "file1.txt"), "a", 0644)
	writeFile(t, filepath.Join(binDir, "file2.txt"), "b", 0644)
	if err := Empty(binDir); err != nil {
		t.Fatalf("Empty: %v", err)
	}
	entries, err := os.ReadDir(binDir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("bin should be empty, got %d entries", len(entries))
	}
}

func TestEmpty_RecreatesBinDir(t *testing.T) {
	dir := t.TempDir()
	binDir := filepath.Join(dir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := Empty(binDir); err != nil {
		t.Fatalf("Empty: %v", err)
	}
	info, err := os.Lstat(binDir)
	if err != nil {
		t.Fatalf("Lstat after Empty: %v", err)
	}
	if !info.IsDir() {
		t.Error("binDir should still be a directory after Empty")
	}
}

func TestEmpty_OnAlreadyEmptyBin(t *testing.T) {
	dir := t.TempDir()
	binDir := filepath.Join(dir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := Empty(binDir); err != nil {
		t.Fatalf("first Empty: %v", err)
	}
	if err := Empty(binDir); err != nil {
		t.Fatalf("second Empty on already-empty bin: %v", err)
	}
}
