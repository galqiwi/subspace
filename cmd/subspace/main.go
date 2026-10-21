package main

import (
	"fmt"
	"os"

	"github.com/galqiwi/subspace/internal/hello/cmd"

	"github.com/spf13/cobra"
)

var SubspaceCmd = &cobra.Command{
	Use:   "subspace",
	Short: "subspace utils",
}

func init() {
	SubspaceCmd.AddCommand(cmd.HelloCmd)
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
