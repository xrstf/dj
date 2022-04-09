package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"

	"go.xrstf.de/pjutil/pkg/prow"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/kubectl/pkg/scheme"
	"k8s.io/kubectl/pkg/util/term"
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

	logger = logger.WithField("pod", pod.Name)
	logger.WithField("cmd", args[1:]).Info("Running command")

	terminal := term.TTY{
		In:  os.Stdin,
		Out: os.Stdout,
		Raw: true, // we want stdin attached and a TTY
	}

	var sizeQueue remotecommand.TerminalSizeQueue
	if terminal.Raw {
		// this call spawns a goroutine to monitor/update the terminal size
		sizeQueue = terminal.MonitorSize(terminal.GetSize())
	}

	return terminal.Safe(func() error {
		request := rootFlags.ClientSet.CoreV1().RESTClient().
			Post().
			Resource("pods").
			Name(pod.Name).
			Namespace(rootFlags.Namespace).
			SubResource("exec")

		option := &corev1.PodExecOptions{
			Container: prow.TestContainerName,
			Command:   args[1:],
			Stdin:     true,
			Stdout:    true,
			Stderr:    true,
			TTY:       true,
		}

		request.VersionedParams(option, scheme.ParameterCodec)
		exec, err := remotecommand.NewSPDYExecutor(rootFlags.RESTConfig, "POST", request.URL())
		if err != nil {
			return err
		}

		return exec.Stream(remotecommand.StreamOptions{
			Stdin:             os.Stdin,
			Stdout:            os.Stdout,
			Stderr:            os.Stderr,
			Tty:               true,
			TerminalSizeQueue: sizeQueue,
		})
	})
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
