package cmd

import (
	"fmt"

	"github.com/roman91DE/toss/internal/bin"
	"github.com/roman91DE/toss/internal/db"
	"github.com/roman91DE/toss/internal/ui"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:          "list",
	Short:        "List all tossed items",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		_, dbPath, err := bin.Paths()
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
			fmt.Println("bin is empty")
			return nil
		}

		ui.PrintTable(entries)
		return nil
	},
}
