package util

var (
	KindClusterIsReadyScript = `
set -e

# wait for the first kindcluster, we want its name
while [ -z "$(kind get clusters 2>/dev/null)" ]; do
  sleep 1
done

clusterName="$(kind get clusters | head -n1)"

# wait until a kubeconfig is available
while [ -z "$(kind get kubeconfig --name "$clusterName" 2>/dev/null)" ]; do
  sleep 1
done

export KUBECONFIG=$(mktemp)
kind get kubeconfig --name "$clusterName" > $KUBECONFIG

# wait until the cluster is ready
until kubectl get ns >/dev/null 2>&1; do
  sleep 1
done
`

	CreateKindClusterProxyScript = `
# enable job control
set -m

export KUBECONFIG=$(mktemp)

pidFile=/tmp/kubectl-proxy-27251.pid
if [ -f $pidFile ]; then
  pkill -F $pidFile
  rm $pidFile
fi

clusterName="$(kind get clusters | head -n1)"
kind get kubeconfig --name "$clusterName" > $KUBECONFIG

kubectl proxy --port=27251 >/dev/null &
echo $! > $pidFile
fg
`
)
