package secrets

import (
	"github.com/spf13/cobra"

	"github.com/circularing/encore/cli/cmd/encore/root"
)

var secretCmd = &cobra.Command{
	Use:     "secret",
	Short:   "Secret management commands",
	Aliases: []string{"secrets"},
}

func init() {
	root.Cmd.AddCommand(secretCmd)
}
