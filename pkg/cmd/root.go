package cmd

import (
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type RootFlags struct {
	Kubeconfig string
	Namespace  string
	RESTConfig *rest.Config
	ClientSet  *kubernetes.Clientset
	Verbose    bool
}

func RootCommand(logger *logrus.Logger) (*cobra.Command, *RootFlags) {
	opt := RootFlags{
		Namespace: metav1.NamespaceDefault,
	}

	cmd := &cobra.Command{
		Use:           "pjutil",
		Short:         "Makes working with tests in Prowjobs easier",
		Version:       "v0.1.0",
		SilenceErrors: true,
		PersistentPreRunE: func(_ *cobra.Command, _ []string) (err error) {
			if opt.Kubeconfig == "" {
				opt.Kubeconfig = os.Getenv("KUBECONFIG")
			}

			if opt.Verbose {
				logger.SetLevel(logrus.DebugLevel)
			}

			opt.RESTConfig, err = clientcmd.BuildConfigFromFlags("", opt.Kubeconfig)
			if err != nil {
				logger.Fatalf("Failed to create Kubernetes client: %v", err)
			}

			opt.ClientSet, err = kubernetes.NewForConfig(opt.RESTConfig)
			if err != nil {
				logger.Fatalf("Failed to create Kubernetes clientset: %v", err)
			}

			return nil
		},
	}

	pFlags := cmd.PersistentFlags()
	pFlags.StringVar(&opt.Kubeconfig, "kubeconfig", opt.Kubeconfig, "kubeconfig file to use (uses $KUBECONFIG by default)")
	pFlags.StringVarP(&opt.Namespace, "namespace", "n", opt.Namespace, "Kubernetes namespace where Prow jobs are running in")
	pFlags.BoolVarP(&opt.Verbose, "verbose", "v", opt.Verbose, "Enable more verbose output")

	return cmd, &opt
}
