package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"go.xrstf.de/dj/pkg/prow"
	"go.xrstf.de/dj/pkg/util"
)

func KKPUserClusterCommand(logger logrus.FieldLogger, rootFlags *RootFlags) *cobra.Command {
	var writeToFile = false

	cmd := &cobra.Command{
		Use:          "kkp-usercluster ( PROWJOB_ID | PROWJOB_POD_NAME )",
		Short:        "Retrieves the kubeconfig for accessing the KKP user cluster in an e2e job",
		SilenceUsage: true,
		RunE: func(c *cobra.Command, args []string) error {
			return kkpUserClusterAction(c.Context(), logger.WithField("namespace", rootFlags.Namespace), rootFlags, writeToFile, args)
		},
	}

	pFlags := cmd.PersistentFlags()
	pFlags.BoolVarP(&writeToFile, "write", "w", writeToFile, "write the kubeconfig to a <clusterid>.kubeconfig file instead of outputting it on stdout")

	return cmd
}

func kkpUserClusterAction(ctx context.Context, logger logrus.FieldLogger, rootFlags *RootFlags, writeToFile bool, args []string) error {
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

	logger.Info("Waiting for Kind cluster to be available…")

	script := strings.TrimSpace(util.KindClusterIsReadyScript)
	if _, err := util.RunCommand(ctx, rootFlags.ClientSet, rootFlags.RESTConfig, pod, prow.TestContainerName, []string{"bash", "-c", script}, nil); err != nil {
		return err
	}

	logger.Info("Kind cluster is ready.")

	clusterName := ""
	if writeToFile {
		logger.Info("Retrieving cluster name…")

		command := []string{"bash", "-c", util.OutputKKPUserClusterName}
		clusterName, err = util.RunCommand(ctx, rootFlags.ClientSet, rootFlags.RESTConfig, pod, prow.TestContainerName, command, nil)
		if err != nil {
			return fmt.Errorf("failed to get cluster name: %w", err)
		}

		logger = logger.WithField("cluster", clusterName)
		logger.Info("Cluster found.")
	}

	logger.Info("Retrieving kubeconfig…")

	command := []string{"bash", "-c", util.OutputKKPUserClusterKubeconfig}
	kubeconfig, err := util.RunCommand(ctx, rootFlags.ClientSet, rootFlags.RESTConfig, pod, prow.TestContainerName, command, nil)
	if err != nil {
		return fmt.Errorf("failed to get kubeconfig: %w", err)
	}

	if writeToFile {
		filename := fmt.Sprintf("%s.kubeconfig", clusterName)
		logger.Infof("Writing kubeconfig to %s…", filename)

		// use pretty strict permissions, because tools like Helm like to complain about it
		return os.WriteFile(filename, []byte(kubeconfig), 0600)
	}

	fmt.Println(kubeconfig)

	return nil
}
