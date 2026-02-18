package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "toss <file...>",
	Short: "A safer rm — moves files to ~/.toss/ instead of deleting them",
	Long: `toss moves files and directories to ~/.toss/files/ instead of permanently
deleting them. Files can be restored to their original location with 'toss restore'.`,
	SilenceUsage: true,
	Args:         cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}
		return runToss(cmd, args)
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(restoreCmd)
	rootCmd.AddCommand(emptyCmd)
}
