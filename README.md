# dj

```
Makes working with KKP e2e tests in Prowjobs easier

Usage:
  dj [command]

Available Commands:
  completion      Generate the autocompletion script for the specified shell
  exec            Execute a command in a Prow job Pod
  help            Help about any command
  kind-proxy      Tunnel through to a kind cluster running inside a Prow job pod, making it available on localhost:8080
  kkp-usercluster Retrieves the kubeconfig for accessing the KKP user cluster in an e2e job
  logs            Stream the logs of the test container of a Prow job Pod

Flags:
  -h, --help                help for dj
      --kubeconfig string   kubeconfig file to use (uses $KUBECONFIG by default)
  -n, --namespace string    Kubernetes namespace where Prow jobs are running in (default "default")
  -v, --verbose             Enable more verbose output
      --version             version for dj

Use "dj [command] --help" for more information about a command.
```

## What does this what kubectl can't do?

You're asking the right questions, my friend!

Indeed, originally the code was just a bunch of shell scripts, but at some point it became too
difficult to maintain and the quoting issues of nesting bash in bash in kubectl exec in bash
was just no fun anymore. Also, sharing the code in form of scripts was hard.

What this offers over kubeconfig:

* `dj` will automatically wait for things to happen. You can run `kind-proxy` right when
  you Prowjob started and it will wait until kind is actually done creating the cluster.
* `dj` accepts both the build ID (64-bit integer, shown on Spyglass pages) and the job ID
  (UUID, equals the pod name). Again, you can write a label selector by hand, but `dj` is
  just more convenient.
