package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "toss <file...>",
	Short: "A safer rm — moves files to ~/.toss/ instead of deleting them",
	Long: `toss moves files and directories to ~/.toss/files/ instead of permanently
deleting them. Files can be restored to their original location with 'toss restore'.`,
	Args:         cobra.MinimumNArgs(1),
	RunE:         runToss,
	SilenceUsage: true,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(restoreCmd)
	rootCmd.AddCommand(emptyCmd)
}
