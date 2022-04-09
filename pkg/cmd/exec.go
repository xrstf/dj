package cmd

import (
	"context"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func ExecCommand(logger logrus.FieldLogger, rootFlags *RootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "exec [ PROWJOB_ID | PROWJOB_POD_NAME ] COMMAND",
		Short: "Execute a command in a Prow job Pod",
		RunE: func(c *cobra.Command, args []string) error {
			return execAction(c.Context(), logger, rootFlags, args)
		},
	}

	return cmd
}

func execAction(ctx context.Context, logger logrus.FieldLogger, rootFlags *RootFlags, args []string) error {
	return nil
}
