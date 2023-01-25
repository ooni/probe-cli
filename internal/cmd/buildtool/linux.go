package main

//
// Linux builds entry point
//

import "github.com/spf13/cobra"

// linuxSubcommand returns the linux [cobra.Command].
func linuxSubcommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "linux",
		Short: "Builds ooniprobe and miniooni for linux",
	}
	cmd.AddCommand(linuxCdepsSubcommand())
	cmd.AddCommand(linuxDockerSubcommand())
	cmd.AddCommand(linuxStaticSubcommand())
	return cmd
}
