package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"go.xrstf.de/pjutil/pkg/prow"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
)

func LogsCommand(logger logrus.FieldLogger, rootFlags *RootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "logs ( PROWJOB_ID | PROWJOB_POD_NAME )",
		Short:        "Stream the logs of the test container of a Prow job Pod",
		SilenceUsage: true,
		RunE: func(c *cobra.Command, args []string) error {
			return logsAction(c.Context(), logger.WithField("namespace", rootFlags.Namespace), rootFlags, args)
		},
	}

	return cmd
}

func logsAction(ctx context.Context, logger logrus.FieldLogger, rootFlags *RootFlags, args []string) error {
	if len(args) == 0 {
		return errors.New("no job ID or Pod name given")
	}

	ident, err := prow.ParsePodIdentifier(args[0])
	if err != nil {
		return err
	}

	// watch pods until we see the test container running
	logger.WithFields(ident.Fields()).Info("Waiting for logs to be available…")

	pod, err := ident.WaitForPod(ctx, rootFlags.ClientSet, rootFlags.Namespace, logsAvailable, nil)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return nil
		}

		return fmt.Errorf("failed to watch Pods: %w", err)
	}

	logger = logger.WithField("pod", pod.Name)
	logger.Info("Starting to stream logs")

	request := rootFlags.ClientSet.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &corev1.PodLogOptions{
		Container: prow.TestContainerName,
		Follow:    true,
	})

	stream, err := request.Stream(ctx)
	if err != nil {
		return fmt.Errorf("failed to stream logs: %w", err)
	}
	defer stream.Close()

	if _, err := io.Copy(os.Stdout, stream); err != nil {
		if errors.Is(err, context.Canceled) {
			return nil
		}

		return fmt.Errorf("failed to write to stdout: %w", err)
	}

	return nil
}

func logsAvailable(pod *corev1.Pod) bool {
	for _, status := range pod.Status.ContainerStatuses {
		if status.Name == prow.TestContainerName {
			return status.State.Running != nil || status.State.Terminated != nil
		}
	}

	// container has no status yet
	return false
}
