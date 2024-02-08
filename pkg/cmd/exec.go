package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"

	"go.xrstf.de/dj/pkg/prow"
	"go.xrstf.de/dj/pkg/util"
)

func ExecCommand(logger logrus.FieldLogger, rootFlags *RootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "exec ( PROWJOB_ID | PROWJOB_POD_NAME ) [ COMMAND = bash ]",
		Short:        "Execute a command in a Prow job Pod",
		SilenceUsage: true,
		RunE: func(c *cobra.Command, args []string) error {
			return execAction(c.Context(), logger.WithField("namespace", rootFlags.Namespace), rootFlags, args)
		},
	}

	return cmd
}

func execAction(ctx context.Context, logger logrus.FieldLogger, rootFlags *RootFlags, args []string) error {
	if len(args) < 1 {
		return errors.New("no job ID or Pod name given")
	}

	// default to running a shell
	if len(args) < 2 {
		args = append(args, "bash")
	}

	ident, err := prow.ParsePodIdentifier(args[0])
	if err != nil {
		return err
	}

	// watch pods until we see the test container running
	logger.WithFields(ident.Fields()).Info("Waiting for Pod to be running…")

	pod, err := ident.WaitForPod(ctx, rootFlags.ClientSet, rootFlags.Namespace, podIsRunninng, podIsTerminated)
	if err != nil {
		return fmt.Errorf("failed to watch Pods: %w", err)
	}
	if pod == nil {
		return errors.New("Pod is terminated, cannot execute commands")
	}

	command := args[1:]

	logger = logger.WithField("pod", pod.Name)
	logger.WithField("cmd", strings.Join(command, " ")).Info("Running command")

	return util.RunCommandWithTTY(rootFlags.ClientSet, rootFlags.RESTConfig, pod, prow.TestContainerName, command, os.Stdin, os.Stdout, os.Stderr)
}

func podIsRunninng(pod *corev1.Pod) bool {
	if pod == nil {
		return false
	}

	for _, status := range pod.Status.ContainerStatuses {
		if status.Name == prow.TestContainerName {
			return status.State.Running != nil
		}
	}

	// container has no status yet
	return false
}

func podIsTerminated(pod *corev1.Pod) bool {
	if pod == nil {
		return false
	}

	for _, status := range pod.Status.ContainerStatuses {
		if status.Name == prow.TestContainerName {
			return status.State.Terminated != nil
		}
	}

	// container has no status yet
	return false
}
