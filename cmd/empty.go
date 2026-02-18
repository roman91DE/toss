package cmd

import (
	"fmt"

	"github.com/roman91DE/toss/internal/bin"
	"github.com/roman91DE/toss/internal/db"
	"github.com/roman91DE/toss/internal/ui"
	"github.com/spf13/cobra"
)

var emptyCmd = &cobra.Command{
	Use:          "empty",
	Short:        "Permanently delete all tossed items",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		force, _ := cmd.Flags().GetBool("force")

		binDir, dbPath, err := bin.Paths()
		if err != nil {
			return err
		}

		database, err := db.Open(dbPath)
		if err != nil {
			return err
		}
		defer database.Close()

		entries, err := db.All(database)
		if err != nil {
			return err
		}

		if len(entries) == 0 {
			fmt.Println("bin is already empty")
			return nil
		}

		if !force {
			ok, err := ui.Confirm(fmt.Sprintf("Permanently delete %d item(s)?", len(entries)))
			if err != nil {
				return err
			}
			if !ok {
				fmt.Println("aborted")
				return nil
			}
		}

		if err := bin.Empty(binDir); err != nil {
			return err
		}

		if _, err := database.Exec(`DELETE FROM entries`); err != nil {
			return fmt.Errorf("clearing db: %w", err)
		}

		fmt.Printf("emptied bin (%d item(s) permanently deleted)\n", len(entries))
		return nil
	},
}

func init() {
	emptyCmd.Flags().BoolP("force", "f", false, "skip confirmation prompt")
}
