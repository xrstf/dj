package main

import (
	"context"
	"os"
	"os/signal"
	"time"

	"github.com/sirupsen/logrus"

	"go.xrstf.de/dj/pkg/cmd"
)

// These variables get set by ldflags during compilation.
var (
	BuildTag    string
	BuildCommit string
	BuildDate   string // RFC3339 format ("2006-01-02T15:04:05Z07:00")
)

func main() {
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: time.RFC1123,
	})

	rootCmd, rootFlags := cmd.RootCommand(logger, BuildTag)
	rootCmd.AddCommand(
		cmd.LogsCommand(logger, rootFlags),
		cmd.ExecCommand(logger, rootFlags),
		cmd.ProxyCommand(logger, rootFlags),
		cmd.KKPUserClusterCommand(logger, rootFlags),
	)

	ctx, cancel := context.WithCancel(context.Background())

	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		cancel()
		logger.Info("Shutting down…")
		<-c
		os.Exit(1) // second signal. Exit directly.
	}()

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		logger.Fatalf("Failed: %v", err)
	}
}
