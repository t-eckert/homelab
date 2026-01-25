# Spark Setup Guide

## 1. Create the Spark Namespace

```bash
kubectl apply -f cluster/spark/namespace.yaml
```

## 2. Create PostgreSQL User

The spark CLI needs a PostgreSQL user with database creation privileges. Connect to your PostgreSQL instance and run:

```bash
# Connect to postgres pod
kubectl exec -it -n postgres postgres-0 -- psql -U postgres -d homelab

# In the psql prompt:
CREATE USER spark WITH PASSWORD 'wie22A7.!bHqb2cqQxARQ_v!Eq';
ALTER USER spark CREATEDB;
\q
```

Verify the user was created:

```bash
kubectl exec -it -n postgres postgres-0 -- psql -U postgres -d homelab -c "\du spark"
```

## 3. Apply the Secret

The `secret.yaml` file has already been configured with your credentials:

```bash
kubectl apply -f cluster/spark/secret.yaml
```

Verify the secret was created:

```bash
kubectl get secret spark-cli-config -n spark
kubectl describe secret spark-cli-config -n spark
```

## 4. Create the Tools PVC

The shared tools PVC provides development tools and configurations to all sparks. Create it with:

```bash
kubectl apply -f cluster/spark/tools-pvc.yaml
```

Then populate it with the latest tools:

```bash
kubectl apply -f cluster/spark/tools-job.yaml
```

Wait for the job to complete:

```bash
kubectl get job -n spark
kubectl logs -n spark -l app=spark-tools -f
```

This will populate the PVC with tools like `kubectl`, `nvim`, `gh`, `node`, `python`, and more.

To refresh tools to the latest container image in the future, re-run:

```bash
kubectl delete job spark-tools-populator -n spark
kubectl apply -f cluster/spark/tools-job.yaml
```

## 5. Build and Install the Spark CLI

```bash
cd spark/
go build -o spark
sudo mv spark /usr/local/bin/
```

## 6. Configure Environment Variables

Option A - Load from Kubernetes secret (recommended):

```bash
source cluster/spark/load-env.sh
```

Option B - Export manually:

```bash
export ANTHROPIC_API_KEY="sk-ant-..."
export POSTGRES_PASSWORD="wie22A7.!bHqb2cqQxARQ_v!Eq"
export GITHUB_TOKEN="ghp_..."  # Optional
```

## 7. Create Your First Spark

```bash
spark create
```

This will:
1. Generate a random name (e.g., `brave-dolphin`)
2. Create a PostgreSQL database
3. Deploy a Kubernetes pod in the `spark` namespace
4. Wait for the pod to be ready
5. Automatically SSH into the container

## 8. Verify Everything Works

Inside the spark container, check that everything is configured:

```bash
# Check environment variables
echo $DATABASE_URL
echo $ANTHROPIC_API_KEY
echo $SPARK_NAME

# Test database connection
psql $DATABASE_URL -c "SELECT version();"

# Verify Claude Code is installed
claude --version

# Check dotfiles were installed
ls -la ~/.dotfiles

# Verify tools are available
which kubectl nvim gh node python
ls ~/.local/bin/ | head -20
cat ~/.local/manifest.txt
```

## Troubleshooting

### "failed to connect to postgres"

Check that:
- PostgreSQL is running: `kubectl get pods -n postgres`
- The spark user exists: `kubectl exec -it -n postgres postgres-0 -- psql -U postgres -d homelab -c "\du spark"`
- The password in `secret.yaml` matches the PostgreSQL user password

### "failed to create k8s client"

Ensure your kubeconfig is set up correctly:

```bash
kubectl config current-context
kubectl get nodes
```

### "no running pod found for spark"

The pod might still be initializing. Check the status:

```bash
kubectl get pods -n spark
kubectl describe pod -n spark -l spark-name=<your-spark-name>
kubectl logs -n spark -l spark-name=<your-spark-name>
```

### SSH connection fails

Verify Tailscale is working:

```bash
kubectl get svc -n spark
kubectl get pods -n tailscale
tailscale status | grep spark
```

The Tailscale operator should have created a proxy pod for each spark's SSH service.

## Cleanup

To remove a spark:

```bash
spark delete <spark-name>
```

To remove all sparks:

```bash
kubectl delete deployments,services,pvc,configmaps,secrets -n spark -l app=spark
```

To completely remove the spark system:

```bash
kubectl delete namespace spark
```
