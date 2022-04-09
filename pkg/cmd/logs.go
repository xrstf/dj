package cmd

import (
	"context"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func LogsCommand(logger logrus.FieldLogger, rootFlags *RootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logs",
		Short: "Stream the logs of the test container of a Prow job Pod",
		RunE: func(c *cobra.Command, args []string) error {
			return logsAction(c.Context(), logger, rootFlags, args)
		},
	}

	return cmd
}

func logsAction(ctx context.Context, logger logrus.FieldLogger, rootFlags *RootFlags, args []string) error {
	return nil
}
