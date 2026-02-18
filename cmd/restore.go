package cmd

import (
	"fmt"
	"os"

	"github.com/roman91DE/toss/internal/bin"
	"github.com/roman91DE/toss/internal/db"
	"github.com/roman91DE/toss/internal/ui"
	"github.com/spf13/cobra"
)

var restoreCmd = &cobra.Command{
	Use:          "restore [query]",
	Short:        "Restore a tossed item to its original location",
	Args:         cobra.MaximumNArgs(1),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		binDir, dbPath, err := bin.Paths()
		if err != nil {
			return err
		}

		database, err := db.Open(dbPath)
		if err != nil {
			return err
		}
		defer database.Close()

		var entries []db.Entry
		if len(args) == 0 {
			entries, err = db.All(database)
		} else {
			entries, err = db.FindByQuery(database, args[0])
		}
		if err != nil {
			return err
		}

		if len(entries) == 0 {
			return fmt.Errorf("no matching items found")
		}

		var entry db.Entry
		switch len(entries) {
		case 1:
			entry = entries[0]
		default:
			entry, err = ui.PickEntry(entries)
			if err != nil {
				return err
			}
		}

		// Check if destination already exists
		if _, err := os.Lstat(entry.OriginalPath); err == nil {
			ok, err := ui.Confirm(fmt.Sprintf("%s already exists. Overwrite?", entry.OriginalPath))
			if err != nil {
				return err
			}
			if !ok {
				fmt.Println("aborted")
				return nil
			}
			if err := os.RemoveAll(entry.OriginalPath); err != nil {
				return fmt.Errorf("removing existing file: %w", err)
			}
		}

		if err := bin.Restore(entry, binDir); err != nil {
			return err
		}

		if err := db.Remove(database, entry.ID); err != nil {
			return fmt.Errorf("updating db: %w", err)
		}

		fmt.Printf("restored: %s\n", entry.OriginalPath)
		return nil
	},
}
