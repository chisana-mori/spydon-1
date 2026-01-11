#!/bin/bash
set -e

CLUSTER_NAME="calico-dev"
CONFIG_FILE="$(dirname "$0")/calico-config.yaml"

echo "Creating Kind cluster '${CLUSTER_NAME}'..."
if kind get clusters | grep -q "^${CLUSTER_NAME}$"; then
  echo "Cluster '${CLUSTER_NAME}' already exists. Skipping creation."
else
  kind create cluster --name "${CLUSTER_NAME}" --config "${CONFIG_FILE}"
fi

echo "Installing Calico..."
# Install Calico Tigera Operator
kubectl create -f https://raw.githubusercontent.com/projectcalico/calico/v3.27.0/manifests/tigera-operator.yaml || echo "Operator manifest likely already applied"

echo "Waiting for CRDs to be established..."
kubectl wait --for condition=established --timeout=60s crd/installations.operator.tigera.io
kubectl wait --for condition=established --timeout=60s crd/bgpconfigurations.crd.projectcalico.org

# Install Calico CustomResources
# We configure it to match the podSubnet in our Kind config
cat <<EOF | kubectl apply -f -
apiVersion: operator.tigera.io/v1
kind: Installation
metadata:
  name: default
spec:
  # Configures Calico to use the CNI plugin
  cni:
    type: Calico
  calicoNetwork:
    # Note: The ipPools section cannot be modified post-install. This configures the default IP Pool.
    ipPools:
    - blockSize: 26
      cidr: 192.168.0.0/16
      encapsulation: VXLANCrossSubnet
      natOutgoing: Enabled
      nodeSelector: all()

---
apiVersion: operator.tigera.io/v1
kind: APIServer
metadata:
  name: default
spec: {}
EOF

echo "Waiting for Calico resources to be created..."
echo "Waiting for Calico resources to be created..."
# Wait for the namespace to exist (the operator creates it)
MAX_RETRIES=30
count=0
while ! kubectl get ns | grep -q calico-system; do
  sleep 2
  count=$((count+1))
  if [ $count -ge $MAX_RETRIES ]; then
    echo "Timed out waiting for calico-system namespace"
    break
  fi
done

# Wait for at least one pod to be created in calico-system
count=0
while ! kubectl -n calico-system get pods 2>/dev/null | grep -q "Running\|Pending\|ContainerCreating"; do
  sleep 2
  count=$((count+1))
  if [ $count -ge $MAX_RETRIES ]; then
    echo "Timed out waiting for pods in calico-system"
    break
  fi
done

echo "Waiting for Calico to be ready..."
# Now we can wait for them to be ready
kubectl wait --for=condition=Ready pods --all -n calico-system --timeout=300s || echo "Warning: Some pods might not be ready yet"

echo "Applying BGP and IPPool configurations..."

# Example BGP Configuration
cat <<EOF | kubectl apply -f -
apiVersion: crd.projectcalico.org/v1
kind: BGPConfiguration
metadata:
  name: default
spec:
  logSeverityScreen: Info
  nodeToNodeMeshEnabled: true
  asNumber: 63400
EOF

# Example additional IPPool
cat <<EOF | kubectl apply -f -
apiVersion: crd.projectcalico.org/v1
kind: IPPool
metadata:
  name: extra-pool
spec:
  cidr: 10.20.0.0/24
  ipipMode: Always
  natOutgoing: true
EOF

echo "Cluster setup complete. You can verify with:"
echo "kubectl get nodes -o wide"
echo "kubectl get ippools.crd.projectcalico.org"
