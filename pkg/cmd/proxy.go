package cmd

import (
	"context"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func ProxyCommand(logger logrus.FieldLogger, rootFlags *RootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "proxy [ PROWJOB_ID | PROWJOB_POD_NAME ]",
		Short: "Tunnel through to a kind cluster running inside a Prow job pod, making it avialable on localhost:8080",
		RunE: func(c *cobra.Command, args []string) error {
			return proxyAction(c.Context(), logger, rootFlags, args)
		},
	}

	return cmd
}

func proxyAction(ctx context.Context, logger logrus.FieldLogger, rootFlags *RootFlags, args []string) error {
	return nil
}
