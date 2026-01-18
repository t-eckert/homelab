# Spark Namespace

This namespace hosts ephemeral development environments created by the `spark` CLI tool.

## Setup

### 1. Create the Namespace

```bash
kubectl apply -f namespace.yaml
```

### 2. Configure the Secret

Copy the template and fill in your credentials:

```bash
cp secret.yaml.template secret.yaml
```

Edit `secret.yaml` with your actual values:

- **ANTHROPIC_API_KEY**: Your Anthropic API key (starts with `sk-ant-`)
- **POSTGRES_PASSWORD**: Password for the `spark` PostgreSQL user
- **GITHUB_TOKEN**: GitHub personal access token (optional, for private repos)

### 3. Apply the Secret

```bash
kubectl apply -f secret.yaml
```

**Note**: `secret.yaml` is gitignored and will not be committed to the repository.

### 4. Verify

```bash
kubectl get secret spark-cli-config -n spark
```

## Using the Spark CLI

The `spark` CLI tool (located in `/spark`) uses these secrets when creating new dev environments. The secrets are not directly mounted into spark containers - instead, they're read by the CLI and propagated into per-spark secrets.

### Required PostgreSQL User

You need to create a `spark` user in your PostgreSQL instance with database creation privileges:

```sql
CREATE USER spark WITH PASSWORD 'your-password';
ALTER USER spark CREATEDB;
```

### Environment Variables for CLI

When running the `spark` CLI, you'll need these environment variables set:

```bash
export ANTHROPIC_API_KEY="sk-ant-..."
export POSTGRES_PASSWORD="your-postgres-password"
export GITHUB_TOKEN="ghp_..."  # Optional
```

Or source them from the Kubernetes secret:

```bash
export ANTHROPIC_API_KEY=$(kubectl get secret spark-cli-config -n spark -o jsonpath='{.data.ANTHROPIC_API_KEY}' | base64 -d)
export POSTGRES_PASSWORD=$(kubectl get secret spark-cli-config -n spark -o jsonpath='{.data.POSTGRES_PASSWORD}' | base64 -d)
export GITHUB_TOKEN=$(kubectl get secret spark-cli-config -n spark -o jsonpath='{.data.GITHUB_TOKEN}' | base64 -d)
```

## Resources Created by Sparks

Each spark creates these resources in the `spark` namespace:

- **Deployment**: `{spark-name}` (e.g., `brave-dolphin`)
- **Service**: `{spark-name}-ssh` (LoadBalancer with Tailscale)
- **PVC**: `{spark-name}-storage` (10GB persistent storage)
- **ConfigMap**: `{spark-name}-config` (SSH keys, git repo URL)
- **Secret**: `{spark-name}-secret` (DATABASE_URL, ANTHROPIC_API_KEY, GITHUB_TOKEN)

## Cleanup

To remove all sparks:

```bash
# List all sparks first
spark list

# Delete individual sparks
spark delete {spark-name}

# Or delete all resources in the namespace
kubectl delete all --all -n spark
kubectl delete pvc --all -n spark
kubectl delete configmap --all -n spark
kubectl delete secret --all -n spark
```
