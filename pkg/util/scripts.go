package util

var (
	lib = `
get_cluster_name() {
  kubectl get clusters --output jsonpath='{.items[].metadata.name}' 2>/dev/null
}

get_cluster_namespace() {
  kubectl get clusters --output jsonpath='{.items[].status.namespaceName}' 2>/dev/null
}

get_cluster_kubeconfig() {
  encoded="$(kubectl --namespace "$1" get secret admin-kubeconfig --ignore-not-found --output jsonpath='{.data.kubeconfig}')"
  if [ -z "$encoded" ]; then
    return
  fi

  echo "$encoded" | base64 -d
}`

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

	OutputKKPUserClusterName = lib + `
export KUBECONFIG=$(mktemp)

clusterName="$(kind get clusters | head -n1)"
kind get kubeconfig --name "$clusterName" > $KUBECONFIG

while [ -z "$(get_cluster_namespace)" ]; do
  slee 1
done

kubectl get clusters --output jsonpath='{.items[].metadata.name}' 2>/dev/null
`

	OutputKKPUserClusterKubeconfig = lib + `
export KUBECONFIG=$(mktemp)

clusterName="$(kind get clusters | head -n1)"
kind get kubeconfig --name "$clusterName" > $KUBECONFIG

while [ -z "$(get_cluster_namespace)" ]; do
  slee 1
done

clusterNamespace="$(get_cluster_namespace)"

while [ -z "$(get_cluster_kubeconfig "$clusterNamespace")" ]; do
  slee 1
done

get_cluster_kubeconfig "$clusterNamespace"
`
)
