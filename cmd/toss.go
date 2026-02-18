package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/roman91DE/toss/internal/bin"
	"github.com/roman91DE/toss/internal/db"
	"github.com/spf13/cobra"
)

func runToss(cmd *cobra.Command, args []string) error {
	binDir, dbPath, err := bin.Paths()
	if err != nil {
		return err
	}

	home, _ := os.UserHomeDir()
	tossDirAbs, _ := filepath.Abs(filepath.Join(home, ".toss"))

	database, err := db.Open(dbPath)
	if err != nil {
		return err
	}
	defer database.Close()

	var hadError bool
	for _, arg := range args {
		abs, err := filepath.Abs(arg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "toss: %s: %v\n", arg, err)
			hadError = true
			continue
		}

		// Refuse to toss the bin itself
		if abs == tossDirAbs {
			fmt.Fprintf(os.Stderr, "toss: refusing to toss the bin directory itself\n")
			hadError = true
			continue
		}

		entry, err := bin.Move(arg, binDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "toss: %v\n", err)
			hadError = true
			continue
		}

		if err := db.Append(database, entry); err != nil {
			fmt.Fprintf(os.Stderr, "toss: recording %s: %v\n", arg, err)
			hadError = true
			continue
		}

		fmt.Printf("tossed: %s\n", abs)
	}

	if hadError {
		os.Exit(1)
	}
	return nil
}
