package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"go.xrstf.de/pjutil/pkg/prow"
	"go.xrstf.de/pjutil/pkg/util"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
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

	logger.Info("Waiting for Kind cluster to be available…")

	stdin := strings.NewReader(strings.TrimSpace(util.KindClusterIsReadyScript))
	if _, err := util.RunCommand(rootFlags.ClientSet, rootFlags.RESTConfig, pod, prow.TestContainerName, []string{"bash"}, stdin); err != nil {
		return err
	}

	logger.Info("Kind cluster is ready.")

	request := rootFlags.ClientSet.CoreV1().RESTClient().
		Post().
		Resource("pods").
		Name(pod.Name).
		Namespace(rootFlags.Namespace).
		SubResource("portforward")

	transport, upgrader, err := spdy.RoundTripperFor(rootFlags.RESTConfig)
	if err != nil {
		return err
	}

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, "POST", request.URL())
	logger = logger.WithField("localport", 8080).WithField("remoteport", 27251)

	stopChan := make(chan struct{})
	readyChan := make(chan struct{})

	// stop forwarding when user hits Ctrl-C
	go func() {
		<-ctx.Done()
		logger.Info("Stopping port-forwarding…")
		close(stopChan)
	}()

	fw, err := portforward.New(dialer, []string{"8080:27251"}, stopChan, readyChan, io.Discard, os.Stderr)
	if err != nil {
		return err
	}

	wg := sync.WaitGroup{}
	fwEnded := make(chan struct{})

	wg.Add(1)
	go func() {
		logger.Info("Forwarding kubectl-proxy to localhost…")
		if err := fw.ForwardPorts(); err != nil {
			logger.WithError(err).Error("Port-forwarding failed.")
		}
		wg.Done()
		close(fwEnded)
	}()

	<-readyChan
	logger.Info("Port-forwarding is ready.")

	// establish a kubectl proxy inside the test container, which makes the kube API
	// available without authentication on a local port inside the test container
	// (the one we just created a port-forwarding to)
	wg.Add(1)
	go func() {
		stdin = strings.NewReader(strings.TrimSpace(util.CreateKindClusterProxyScript))
		if _, err := util.RunCommand(rootFlags.ClientSet, rootFlags.RESTConfig, pod, prow.TestContainerName, []string{"bash"}, stdin); err != nil {
			if err.Error() != "" {
				logger.WithError(err).Error("Failed to create proxy.")
			}
		}
		wg.Done()
	}()

	// It's not possible to cancel a SPDY connection stream (https://github.com/kubernetes/client-go/issues/884),
	// so instead we cheat and exec even more!
	go func() {
		<-fwEnded // wait the port-forwarding ends
		stdin = strings.NewReader("pkill -P 0 kubectl")
		_, _ = util.RunCommand(rootFlags.ClientSet, rootFlags.RESTConfig, pod, prow.TestContainerName, []string{"bash"}, stdin)
	}()

	logger.Info("Kind cluster is now available locally.")
	wg.Wait()

	return nil
}
