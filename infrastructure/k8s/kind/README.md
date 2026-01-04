This folder contains helper manifests and an installer script for running AWX on a local kind cluster.

Files:
- `kind-awx-config.yaml` - Kind cluster config (3 nodes). Maps host `19999` -> node `30080` on the control-plane.
- `awx-deployment.yaml` - AWX custom resource (applies after AWX operator is installed).
- `install-awx.sh` - Script that creates the kind cluster, installs the AWX operator, and applies the AWX CR.

Usage:
1. Ensure you have `kind`, `kubectl`, and `docker` installed on your machine.
2. Run the installer (may require sudo depending on your environment):

```bash
bash k8s/install-awx.sh
```

3. Watch pods:

```bash
kubectl -n awx get pods -w
```

4. After AWX is ready, check services and access:

```bash
kubectl -n awx get svc
# Option A: If a NodePort is created and mapped to port 30080, access at http://localhost:19999
# Option B: Use port-forward to map the AWX service to localhost:19999:
kubectl -n awx port-forward svc/<svc-name> 19999:80
```
