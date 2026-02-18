package cmd

import (
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/roman91DE/toss/internal/bin"
	"github.com/roman91DE/toss/internal/ui"
	"github.com/spf13/cobra"
)

var memCmd = &cobra.Command{
	Use:          "mem",
	Short:        "Show disk space used by the toss bin",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		binDir, _, err := bin.Paths()
		if err != nil {
			return err
		}

		var total int64
		err = filepath.WalkDir(binDir, func(_ string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if !d.IsDir() {
				info, err := d.Info()
				if err == nil {
					total += info.Size()
				}
			}
			return nil
		})
		if err != nil {
			return err
		}

		fmt.Printf("bin usage: %s\n", ui.FormatSize(total))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(memCmd)
}
