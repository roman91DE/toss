package bin

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/roman91DE/toss/internal/db"
)

func Paths() (binDir, dbPath string, err error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", "", fmt.Errorf("finding home dir: %w", err)
	}
	tossDir := filepath.Join(home, ".toss")
	binDir = filepath.Join(tossDir, "files")
	dbPath = filepath.Join(tossDir, "toss.db")
	return binDir, dbPath, nil
}

func EnsureDirs(binDir string) error {
	return os.MkdirAll(binDir, 0755)
}

func dirSize(path string) (int64, error) {
	var size int64
	err := filepath.WalkDir(path, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() {
			info, err := d.Info()
			if err == nil {
				size += info.Size()
			}
		}
		return nil
	})
	return size, err
}

func Move(src, binDir string) (db.Entry, error) {
	if err := EnsureDirs(binDir); err != nil {
		return db.Entry{}, fmt.Errorf("creating bin dir: %w", err)
	}

	abs, err := filepath.Abs(src)
	if err != nil {
		return db.Entry{}, fmt.Errorf("resolving path: %w", err)
	}

	info, err := os.Lstat(abs)
	if err != nil {
		return db.Entry{}, fmt.Errorf("%s: %w", src, err)
	}

	id := db.NewID()
	binName := id + "-" + filepath.Base(abs)
	dest := filepath.Join(binDir, binName)

	if err := moveItem(abs, dest); err != nil {
		return db.Entry{}, err
	}

	var size int64
	if info.IsDir() {
		size, _ = dirSize(dest)
	} else {
		size = info.Size()
	}

	return db.Entry{
		ID:           id,
		OriginalPath: abs,
		BinName:      binName,
		TossedAt:     time.Now(),
		IsDir:        info.IsDir(),
		SizeBytes:    size,
	}, nil
}

func Restore(entry db.Entry, binDir string) error {
	src := filepath.Join(binDir, entry.BinName)
	dest := entry.OriginalPath

	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return fmt.Errorf("recreating parent dirs: %w", err)
	}

	return moveItem(src, dest)
}

func Empty(binDir string) error {
	if err := os.RemoveAll(binDir); err != nil {
		return fmt.Errorf("removing bin contents: %w", err)
	}
	return os.MkdirAll(binDir, 0755)
}

func moveItem(src, dest string) error {
	err := os.Rename(src, dest)
	if err == nil {
		return nil
	}
	var linkErr *os.LinkError
	if errors.As(err, &linkErr) && errors.Is(linkErr.Err, syscall.EXDEV) {
		return copyThenDelete(src, dest)
	}
	return err
}

func copyThenDelete(src, dest string) error {
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}
	if info.IsDir() {
		if err := copyDir(src, dest); err != nil {
			return err
		}
	} else if info.Mode()&fs.ModeSymlink != 0 {
		linkTarget, err := os.Readlink(src)
		if err != nil {
			return err
		}
		if err := os.Symlink(linkTarget, dest); err != nil {
			return err
		}
	} else {
		if err := copyFile(src, dest, info.Mode()); err != nil {
			return err
		}
	}
	return os.RemoveAll(src)
}

func copyDir(src, dest string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dest, rel)
		if d.Type()&fs.ModeSymlink != 0 {
			linkTarget, err := os.Readlink(path)
			if err != nil {
				return err
			}
			return os.Symlink(linkTarget, target)
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if d.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}
		return copyFile(path, target, info.Mode())
	})
}

func copyFile(src, dest string, mode fs.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
