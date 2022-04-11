# pjutil

```
Makes working with tests in Prowjobs easier

Usage:
  pjutil [command]

Available Commands:
  completion      Generate the autocompletion script for the specified shell
  exec            Execute a command in a Prow job Pod
  help            Help about any command
  kind-proxy      Tunnel through to a kind cluster running inside a Prow job pod, making it available on localhost:8080
  kkp-usercluster Retrieves the kubeconfig for accessing the KKP user cluster in an e2e job
  logs            Stream the logs of the test container of a Prow job Pod

Flags:
  -h, --help                help for pjutil
      --kubeconfig string   kubeconfig file to use (uses $KUBECONFIG by default)
  -n, --namespace string    Kubernetes namespace where Prow jobs are running in (default "default")
  -v, --verbose             Enable more verbose output
      --version             version for pjutil

Use "pjutil [command] --help" for more information about a command.
```
