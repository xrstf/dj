package main

import (
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
	)

	if err := rootCmd.Execute(); err != nil {
		logger.Fatalf("Failed: %v", err)
	}
}
