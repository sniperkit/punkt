package cmd

import (
	"strings"

	"github.com/mbark/punkt/mgr/homebrew"
	"github.com/mbark/punkt/symlink"

	"github.com/spf13/cobra"
)

var (
	directories []string
	depth       int
)

var message = strings.TrimSpace(`
dump the current working environment to your dotfile configuration files. Doing
this will search for symlinks and add your currently installed packages to their
corresponding package-manager's config files.`)

// ensureCmd represents the ensure command
var dumpCmd = &cobra.Command{
	Use:   "dump",
	Short: "dump your current environment to config files",
	Long:  message,
	Run: func(cmd *cobra.Command, args []string) {
		dump(cmd, args)
	},
}

func init() {
	dumpCmd.Flags().StringArrayVar(&directories, "symlink-directories", []string{"~"}, `Search the given directories for symlinks to add`)
	dumpCmd.Flags().IntVar(&depth, "depth", 2, `The depth to stop recursively searching for symlinks`)

	RootCmd.AddCommand(dumpCmd)
}

func dump(cmd *cobra.Command, args []string) {
	symlink.Dump(directories, depth, punktHome, dotfiles)
	homebrew.Dump(punktHome)
}
