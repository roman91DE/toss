# toss

A safer `rm` — moves files and directories to `~/.toss/` instead of permanently deleting them.

## Install

```bash
make install   # builds and copies to ~/.local/bin/toss
```

Or with Go:

```bash
go install github.com/roman91DE/toss@latest
```

Make sure the install location is in your `PATH`:

```bash
export PATH="$HOME/.local/bin:$PATH"   # if using make install
export PATH="$HOME/go/bin:$PATH"       # if using go install
```

## Usage

```bash
toss file.txt dir/          # move to bin
toss list                   # show all tossed items
toss restore file.txt       # restore by name or path
toss empty                  # permanently delete all (prompts for confirmation)
toss empty -f               # skip confirmation
```

### `toss restore`

Matches case-insensitively against the filename or full original path. If multiple items match, an interactive picker is shown:

```
Multiple matches found:
  1) /home/user/notes.txt  (2026-02-18 10:00)
  2) /tmp/notes.txt        (2026-02-18 11:00)
Choose [1-2]:
```

If the original location already has a file, you'll be asked to confirm before overwriting. Parent directories are recreated automatically if they were deleted.

## How it works

Files are moved to `~/.toss/files/` with a UUID prefix to prevent name collisions:

```
~/.toss/
├── toss.db          # SQLite database: original paths + timestamps
└── files/
    ├── 3f2a...-notes.txt
    └── 7c1b...-src/
```

The SQLite database records each item's original path, toss time, size, and whether it's a directory — enough to restore it exactly.

Cross-filesystem moves (e.g. `/tmp` → home directory) fall back to copy + delete automatically.

## Build

```bash
make build   # → ./toss
make test    # go test ./...
make clean   # remove binary
```

**Dependencies:** [`cobra`](https://github.com/spf13/cobra), [`modernc.org/sqlite`](https://gitlab.com/cznic/sqlite) (pure Go, no CGo required)
