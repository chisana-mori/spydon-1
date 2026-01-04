#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(cd "$(dirname "$0")" && pwd)
KIND_CONFIG="$ROOT_DIR/kind-awx-config.yaml"

# --- 检查依赖 ---
if ! command -v kubectl >/dev/null 2>&1; then
    echo "ERROR: 'kubectl' is required."
    exit 1
fi

echo "== Create kind cluster 'kind-awx' (3 nodes) =="
if ! command -v kind >/dev/null 2>&1; then
  echo "ERROR: 'kind' not found. Install kind first: https://kind.sigs.k8s.io"
  exit 1
fi

if kind get clusters | grep -q "^kind-awx$"; then
    echo "kind cluster 'kind-awx' already exists, skipping create"
else
    # 如果本地没有 kind-awx-config.yaml，创建一个简单的默认配置
    if [ ! -f "$KIND_CONFIG" ]; then
        echo "Creating default kind config..."
        cat <<EOF > "$KIND_CONFIG"
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
- role: worker
- role: worker
EOF
    fi
    kind create cluster --name kind-awx --config "$KIND_CONFIG"
fi

echo "== Install AWX Operator =="
# create namespace
kubectl create namespace awx --dry-run=client -o yaml | kubectl apply -f -

# 定义临时文件路径
TMP_MANIFEST="$(mktemp -t awx-operator.XXXXXX).yaml"

echo "[install] Fetching latest version tag from GitHub API..."
# 获取最新版本号 (例如 2.19.1)
LATEST_TAG=$(curl -s https://api.github.com/repos/ansible/awx-operator/releases/latest | grep "tag_name" | cut -d '"' -f 4)

if [ -z "$LATEST_TAG" ]; then
    echo "ERROR: Failed to fetch latest tag. Using fallback version 2.19.1"
    LATEST_TAG="2.19.1"
fi

echo "[install] Generating operator manifest for version: $LATEST_TAG"

# === 关键修改 ===
# 官方不再提供直接下载的 yaml。
# 使用 kubectl kustomize 从远程 git 仓库动态生成 yaml 文件。
if kubectl kustomize "github.com/ansible/awx-operator/config/default?ref=${LATEST_TAG}" > "$TMP_MANIFEST"; then
    echo "[install] Manifest generated successfully."
else
    echo "ERROR: failed to generate manifest via kustomize."
    rm -f "$TMP_MANIFEST"
    exit 1
fi

# 应用生成的 manifest
kubectl apply -f "$TMP_MANIFEST"
rm -f "$TMP_MANIFEST"

echo "== Wait for operator deployment to be ready (120s) =="
# 注意：Operator 的名称可能会随版本变化，通常是 awx-operator-controller-manager
kubectl -n awx rollout status deployment/awx-operator-controller-manager --timeout=120s || true

echo "== Create AWX CR =="

# 检查是否存在 awx-deployment.yaml，如果不存在则生成一个最简版
AWX_CR_FILE="$ROOT_DIR/awx-deployment.yaml"
if [ ! -f "$AWX_CR_FILE" ]; then
    echo "Creating default AWX Custom Resource file..."
    cat <<EOF > "$AWX_CR_FILE"
apiVersion: awx.ansible.com/v1beta1
kind: AWX
metadata:
  name: awx-demo
  namespace: awx
spec:
  service_type: nodeport
  # 适合测试环境的资源限制
  web_resource_requirements:
    requests:
      cpu: "500m"
      memory: "1Gi"
  task_resource_requirements:
    requests:
      cpu: "500m"
      memory: "1Gi"
EOF
fi

kubectl apply -f "$AWX_CR_FILE"

cat <<'EOF'

Next steps:
- Watch operator and AWX pods (Wait for awx-demo-web and awx-demo-task):
    kubectl -n awx get pods -w

- After pods are running, find the AWX service port:
    kubectl -n awx get svc awx-demo-service

- Access AWX:
    You need the admin password to login.
    Get the password:
    kubectl -n awx get secret awx-demo-admin-password -o jsonpath="{.data.password}" | base64 --decode; echo

- Forward port (if nodeport is hard to reach):
    kubectl -n awx port-forward svc/awx-demo-service 19999:80
    Then visit: http://localhost:19999

- print password
    kubectl -n awx get secret awx-admin-password -o jsonpath="{.data.password}" | base64 --decode; echo


- get token

  curl -X POST \c
  -u "admin:EiG8FpCjjw1mcPXPfjjroFEyCCsBqKPY" \
  -H "Content-Type: application/json" \
  -d '{"description":"My CLI Token", "scope":"write"}' \
  http://localhost:19999/api/v2/users//personal_tokens/

EOF