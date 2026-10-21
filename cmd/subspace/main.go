package main

import (
	"fmt"
	"os"

	clientcmd "github.com/galqiwi/subspace/internal/client/cmd"
	servercmd "github.com/galqiwi/subspace/internal/server/cmd"

	"github.com/spf13/cobra"
)

var SubspaceCmd = &cobra.Command{
	Use:           "subspace",
	Short:         "expose local http services through a public server",
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	SubspaceCmd.AddCommand(servercmd.ServerCmd)
	SubspaceCmd.AddCommand(clientcmd.ClientCmd)
}

func main() {
	err := Main()
	if err != nil {
		if err.Error() != "" {
			fmt.Println(err)
		}
		os.Exit(1)
	}
}

func Main() error {
	return SubspaceCmd.Execute()
}
