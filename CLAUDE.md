# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
make build       # compile → ./toss
make install     # build + copy to ~/.local/bin/toss
make test        # go test ./...
make clean       # remove ./toss binary
go test ./internal/db/...   # run a single package's tests
```

## Architecture

**Module:** `github.com/roman91DE/toss`
**Dependencies:** `cobra` (CLI), `modernc.org/sqlite` (pure Go, no CGo)

### Data flow

`cmd/` commands open the SQLite DB via `internal/db`, call filesystem ops in `internal/bin`, and use `internal/ui` for output/prompts.

```
cmd/          → cobra subcommands (toss, list, restore, empty, mem)
internal/db/  → SQLite-backed store: Open, Append, Remove, All, FindByQuery
internal/bin/ → filesystem ops: Move, Restore, Empty, Paths, EnsureDirs
internal/ui/  → tabwriter table, confirmation prompt, interactive picker, FormatSize
main.go       → calls cmd.Execute()
```

### Storage layout (`~/.toss/`)

```
~/.toss/
├── toss.db        # SQLite database
└── files/
    └── <uuid>-<original-basename>
```

`db.Open()` creates `~/.toss/` if it doesn't exist. `bin.Move()` generates the UUID prefix via `db.NewID()` (crypto/rand, no external uuid package).

### Cross-device moves

`bin.moveItem` attempts `os.Rename`; on `EXDEV` error it falls back to `copyThenDelete`, which walks the tree for directories.

### Adding a new subcommand

1. Create `cmd/<name>.go` with a `var <name>Cmd = &cobra.Command{...}`
2. Register it in `cmd/root.go`'s `init()` via `rootCmd.AddCommand(<name>Cmd)`, or register inside the new file's own `init()` (see `cmd/mem.go` as the pattern to follow).
