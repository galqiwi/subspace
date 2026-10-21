package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/galqiwi/subspace/internal/client"
	"github.com/spf13/cobra"
)

var options client.Options

var ClientCmd = &cobra.Command{
	Use:   "client",
	Short: "expose local port through subspace server",
	RunE: func(cmd *cobra.Command, args []string) error {
		return clientCmd()
	},
}

func init() {
	ClientCmd.Flags().StringVar(
		&options.ServerURL,
		"server",
		"",
		"subspace server url, e.g. http://example.com:8080",
	)
	ClientCmd.Flags().StringVar(
		&options.Token,
		"token",
		"",
		"client token from server config",
	)
	ClientCmd.Flags().StringVar(
		&options.Id,
		"id",
		"",
		"unique id, requesters pass it as bearer token",
	)
	ClientCmd.Flags().IntVar(
		&options.Port,
		"port",
		0,
		"local port to expose",
	)
	ClientCmd.Flags().StringVar(
		&options.UpstreamAuthorization,
		"upstream-authorization",
		"",
		"Authorization header value sent to local port, e.g. \"Bearer <token>\" (stripped if not set)",
	)
	for _, name := range []string{"server", "token", "id", "port"} {
		_ = ClientCmd.MarkFlagRequired(name)
	}
}

func clientCmd() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	return client.Run(ctx, options)
}
