package main

import (
	"context"
	"os"
	"os/signal"
	"time"

	"go.xrstf.de/pjutil/pkg/cmd"

	"github.com/sirupsen/logrus"
)

func main() {
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: time.RFC1123,
	})

	rootCmd, rootFlags := cmd.RootCommand(logger)
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
