// SPDX-FileCopyrightText: 2024 Christoph Mewes
// SPDX-License-Identifier: MIT

package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"go.xrstf.de/dj/pkg/prow"
	"go.xrstf.de/dj/pkg/util"
)

func ProxyCommand(logger logrus.FieldLogger, rootFlags *RootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "kind-proxy [ PROWJOB_ID | PROWJOB_POD_NAME ]",
		Short:        "Tunnel through to a kind cluster running inside a Prow job pod, making it available on localhost:8080",
		SilenceUsage: true,
		RunE: func(c *cobra.Command, args []string) error {
			return proxyAction(c.Context(), logger, rootFlags, args)
		},
	}

	return cmd
}

func proxyAction(ctx context.Context, logger logrus.FieldLogger, rootFlags *RootFlags, args []string) error {
	if len(args) == 0 {
		return errors.New("no job ID or Pod name given")
	}

	ident, err := prow.ParsePodIdentifier(args[0])
	if err != nil {
		return err
	}

	// watch pods until we see the test container running
	logger = logger.WithFields(ident.Fields())
	logger.Info("Waiting for Pod to be running…")

	pod, err := ident.WaitForPod(ctx, rootFlags.ClientSet, rootFlags.Namespace, podIsRunninng, podIsTerminated)
	if err != nil {
		return fmt.Errorf("failed to watch Pods: %w", err)
	}
	if pod == nil {
		return errors.New("Pod is terminated, cannot create proxy")
	}

	logger = logger.WithField("pod", pod.Name)
	logger.Info("Retrieving cluster name…")

	logger.Info("Waiting for Kind cluster to be available…")

	script := strings.TrimSpace(util.KindClusterIsReadyScript)
	if _, err := util.RunCommand(ctx, rootFlags.ClientSet, rootFlags.RESTConfig, pod, prow.TestContainerName, []string{"bash", "-c", script}, nil); err != nil {
		return err
	}

	logger.Info("Kind cluster is ready.")

	kubectlCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	cmd := exec.CommandContext(kubectlCtx, "kubectl", "--kubeconfig", rootFlags.Kubeconfig, "--namespace", pod.Namespace, "port-forward", pod.Name, "8080:27251")
	if err := cmd.Start(); err != nil {
		logger.WithError(err).Error("Failed to start kubectl.")
	}
	time.Sleep(2 * time.Second)

	// The following is the sane solution, but it has the weird drawback that if someone tries to use
	// the port-forwarding before the kubectl-proxy command is started, the forwarding simply ends with
	// no error being returned. The errors printed on stderr are identical to the errors that kubectl
	// would produce, but for some reason kubectl's port-forward doesn't end, it just retries. I checked
	// the source code but cannot find where it does that.
	// So ...... with a heavy heart, the above code snippet is the bruteforce stupid solution. Please
	// submit a patch to make this work. Please, someone...

	// request := rootFlags.ClientSet.CoreV1().RESTClient().
	// 	Post().
	// 	Resource("pods").
	// 	Name(pod.Name).
	// 	Namespace(rootFlags.Namespace).
	// 	SubResource("portforward")

	// transport, upgrader, err := spdy.RoundTripperFor(rootFlags.RESTConfig)
	// if err != nil {
	// 	return err
	// }

	// dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, "POST", request.URL())
	// logger = logger.WithField("localport", 8080).WithField("remoteport", 27251)

	// readyChan := make(chan struct{})
	// fwCtx, cancel := context.WithCancel(ctx)
	// defer cancel()

	// fw, err := portforward.New(dialer, []string{"8080:27251"}, fwCtx.Done(), readyChan, io.Discard, io.Discard)
	// if err != nil {
	// 	return err
	// }

	// fwEnded := make(chan struct{})

	// go func() {
	// 	logger.Info("Forwarding kubectl-proxy to localhost…")

	// 	// for {
	// 	// 	select {
	// 	// 	case <-fwCtx.Done():
	// 	// 		logger.Info("Forwarding ended.")
	// 	// 		close(fwEnded)
	// 	// 		return

	// 	// 	default:
	// 	// 		if err := fw.ForwardPorts(); err != nil {
	// 	// 			logger.WithError(err).Error("Port-forwarding failed.")
	// 	// 		}
	// 	// 	}
	// 	// }

	// 	if err := fw.ForwardPorts(); err != nil {
	// 		logger.WithError(err).Error("Port-forwarding failed.")
	// 	}
	// 	logger.Info("Forwarding ended.")
	// 	close(fwEnded)
	// }()

	// <-readyChan
	logger.Info("Port-forwarding is ready.")

	// establish a kubectl proxy inside the test container, which makes the kube API
	// available without authentication on a local port inside the test container
	// (the one we just created a port-forwarding to); this command will block until
	// the user Ctrl-C's out.
	// Note that it's not possible to interrupt a SPDY connection, so it is effectively
	// impossible to cancel a running command. For this reason we cannot have this
	// command run in the background (without a TTY attached), because then we can never
	// kill it and even when dj ends, the bash will continue to run, and this will
	// block the temporary port and could lead to trouble when a second proxy is attempted.
	// Due to this, we instead make the kubectl-proxy command interactive and attach the TTY,
	// so that any Ctrl-C will be caught by the bash in the container and the kubectl-proxy
	// can be stopped as intended.
	logger.Info("Proxying Kind cluster to localhost…")

	script = strings.TrimSpace(util.CreateKindClusterProxyScript)
	err = util.RunCommandWithTTY(ctx, rootFlags.ClientSet, rootFlags.RESTConfig, pod, prow.TestContainerName, []string{"bash", "-c", script}, os.Stdin, io.Discard, io.Discard)
	if err != nil {
		return fmt.Errorf("failed to run proxy: %w", err)
	}

	// Now the kubectl-proxy has ended, we can stop the port forwarding
	logger.Info("Stopping port-forwarding…")
	cancel()
	// <-fwEnded

	return cmd.Wait()
}
