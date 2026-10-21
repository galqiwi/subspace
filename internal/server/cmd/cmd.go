package cmd

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/galqiwi/subspace/internal/server"
	"github.com/spf13/cobra"
)

var configPath string

var ServerCmd = &cobra.Command{
	Use:   "server",
	Short: "run public server that routes requests to connected clients",
	RunE: func(cmd *cobra.Command, args []string) error {
		return serverCmd()
	},
}

func init() {
	ServerCmd.Flags().StringVar(
		&configPath,
		"config",
		"",
		"path to server yaml config",
	)
	_ = ServerCmd.MarkFlagRequired("config")
}

func serverCmd() error {
	config, err := server.LoadConfig(configPath)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	httpServer := &http.Server{
		Addr:              config.Listen,
		Handler:           server.New(config),
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		<-ctx.Done()
		_ = httpServer.Close()
	}()

	log.Printf("listening on %v", config.Listen)
	err = httpServer.ListenAndServe()
	if ctx.Err() != nil {
		return nil
	}
	return err
}
