package main

import (
	"github.com/spf13/cobra"
)

func SetupCommands(a *App) *cobra.Command {
	// root command
	rootCmd := &cobra.Command{
		Use:   "timetick",
		Short: "A time tracking CLI application",
	}

	// command for creating new tracking sheet or changing the current tracking sheet to specified name
	sheetCmd := &cobra.Command{
		Use:   "sheet [name]",
		Short: "Create or change tracking sheet",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			sheets, err := a.repo.GetAllSheets()
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			return sheets, cobra.ShellCompDirectiveNoFileComp
		},
		Run: func(cmd *cobra.Command, args []string) {
			name := args[0]

			a.ChangeSheet(name)
		},
	}

	// command for start time tracking
	startCmd := &cobra.Command{
		Use:   "start [note]",
		Short: "Start tracking time",
		Run: func(cmd *cobra.Command, args []string) {
			var note string
			if len(args) > 0 {
				note = args[0]
			}

			a.StartTracking(note)
		},
	}

	// command for stop time tracking
	stopCmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop tracking time",
		Run: func(cmd *cobra.Command, args []string) {
			var note string
			if len(args) > 0 {
				note = args[0]
			}

			a.StopTracking(note)
		},
	}

	// add commands
	rootCmd.AddCommand(sheetCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
	// TODO: add display command

	return rootCmd
}
