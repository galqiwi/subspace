package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var HelloCmd = &cobra.Command{
	Use:   "hello",
	Short: "print hello world",
	RunE: func(cmd *cobra.Command, args []string) error {
		return helloCmd()
	},
}

func helloCmd() error {
	fmt.Println("Hello, world!")
	return nil
}
