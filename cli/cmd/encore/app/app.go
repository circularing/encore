package app

import (
	"github.com/spf13/cobra"

	"github.com/circularing/encore/cli/cmd/encore/root"
)

// These can be overwritten using
// `go build -ldflags "-X github.com/circularing/encore/cli/cmd/encore/app.defaultGitRemoteName=encore"`.
var (
	defaultGitRemoteName = "encore"
	defaultGitRemoteURL  = "encore://"
)

var appCmd = &cobra.Command{
	Use:   "app",
	Short: "Commands to create and link Encore apps",
}

func init() {
	root.Cmd.AddCommand(appCmd)
}
