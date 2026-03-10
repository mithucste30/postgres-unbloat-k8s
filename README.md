# PostgreSQL Unbloat for Kubernetes

[![CI](https://github.com/mithucste30/postgres-unbloat-k8s/workflows/CI/badge.svg)](https://github.com/mithucste30/postgres-unbloat-k8s/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/mithucste30/postgres-unbloat-k8s)](https://goreportcard.com/report/github.com/mithucste30/postgres-unbloat-k8s)
[![License: Apache 2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

**PostgreSQL Unbloat for Kubernetes** is an automated PostgreSQL bloat remediation system that uses **Kubernetes Jobs** for vacuum operations.

## 🎯 Key Features

- 🚀 **Job-based Architecture** - No direct DB connections, uses Kubernetes Jobs
- 🔔 **Alert-Driven** - Responds to Prometheus alerts automatically
- 🔍 **Auto-Discovery** - Finds PostgreSQL instances and credentials
- 🔒 **Safe by Default** - Dry-run mode for testing
- 📦 **Zero Configuration** - Just install and it works
- 🧩 **Reusable Library** - Use as Go package in your apps

## 🏗️ Architecture

### Job-Based Design

Instead of connecting directly to PostgreSQL, the application creates **Kubernetes Jobs** that run vacuum operations:

```
┌─────────────────┐
│  Prometheus     │
│  Alertmanager   │
└────────┬────────┘
         │ Webhook
         ▼
┌─────────────────────────────────┐
│  postgres-unbloat-k8s           │
│  (Controller)                    │
│                                 │
│  Receives alert                  │
│      ↓                           │
│  Discovers PostgreSQL            │
│      ↓                           │
│  Creates Kubernetes Job          │
└────────┬────────────────────────┘
         │ Creates
         ▼
┌─────────────────────────────────┐
│  Kubernetes Job                  │
│  - Image: postgres:17-alpine     │
│  - Command: psql -c "VACUUM..."  │
│  - Runs in-cluster               │
│  - Auto-deleted after 1 hour     │
└────────┬────────────────────────┘
         │ Connects
         ▼
┌─────────────────┐
│  PostgreSQL     │
│  Database       │
└─────────────────┘
```

### Benefits of Job-Based Architecture

✅ **No Port-Forwarding** - Jobs run in-cluster with full network access
✅ **Better Testing** - Run locally, Jobs execute in-cluster
✅ **Cleaner Separation** - App is a controller, not a DB client
✅ **Standard Tools** - Uses official PostgreSQL client images
✅ **Scalability** - Multiple vacuums run in parallel via Jobs
✅ **Observability** - Job status, logs, and history visible in Kubernetes

## 🚀 Quick Start

### Prerequisites

- Kubernetes cluster (v1.19+)
- kubectl configured
- Prometheus with PostgreSQL metrics exporter

### Installation

```bash
# Clone the repository
git clone https://github.com/mithucste30/postgres-unbloat-k8s.git
cd postgres-unbloat-k8s

# Build the binary
go build -o bin/postgres-unbloat-k8s ./cmd/manager

# Run locally (discovery + dry-run mode)
./bin/postgres-unbloat-k8s \
  --mode=local \
  --dry-run=true \
  --log-level=info
```

### Kubernetes Deployment

```bash
# Update Helm values
cat > values.yaml <<EOF
config:
  mode: in-cluster
  dryRun: false  # Set to true for testing
  discovery:
    namespaces:
      - default
      - database
rbac:
  create: true
EOF

# Install via Helm
helm install postgres-unbload deploy/helm -f values.yaml
```

## 📋 Alert Types

The system responds to these Prometheus alerts:

| Alert | Condition | Action | Job Type |
|-------|-----------|--------|----------|
| `PostgreSQLTableHighBloat` | >10% bloat | `VACUUM ANALYZE` | Standard |
| `PostgreSQLTableCriticalBloat` | >30% bloat | `VACUUM FULL ANALYZE` | Extended |
| `PostgreSQLTableNotVacuumed` | Not vacuumed in 24h | `VACUUM ANALYZE` | Standard |
| `PostgreSQLAutovacuumNotRunning` | No workers | Manual vacuum | Sequential |
| `PostgreSQLLargeTableBloat` | Large table + bloat | Extended timeout | Long-running |
| `PostgreSQLMultipleTablesNeedVacuum` | >5 tables | Sequential vacuums | Batch |
| `PostgreSQLTableNeverVacuumed` | Never vacuumed | Initial vacuum | Standard |

## 🔧 Configuration

### Command-Line Flags

```bash
./bin/postgres-unbloat-k8s \
  --mode=local              # Execution mode: local or in-cluster
  --dry-run=true            # Dry-run mode (log only, no jobs created)
  --namespace=default        # Namespace for Jobs
  --kubeconfig=~/.kube/config  # Path to kubeconfig
  --log-level=info          # Log level: debug, info, warn, error
  --discovery-enabled=true  # Enable PostgreSQL discovery
```

### Environment Variables

```bash
export SERVER_MODE=in-cluster
export DRY_RUN=false
export NAMESPACE=postgres-unbloat
export LOG_LEVEL=debug
```

## 📦 Kubernetes Job Specification

When a vacuum is needed, the system creates a Job like this:

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: vacuum-public-users-1234567890
  labels:
    app: postgres-unbloat
    component: vacuum-job
    alert-name: PostgreSQLTableHighBloat
    table: public.users
    target-ns: database
    target-pod: postgres-0
spec:
  ttlSecondsAfterFinished: 3600  # Auto-cleanup after 1 hour
  backoffLimit: 0                 # No retries
  activeDeadlineSeconds: 7200     # Max 2 hours
  template:
    spec:
      restartPolicy: Never
      containers:
      - name: vacuum
        image: postgres:17-alpine   # Official PostgreSQL client
        command: ["psql"]
        args:
          - "-h"
          - "postgres.database.svc"
          - "-p"
          - "5432"
          - "-U"
          - "postgres"
          - "-d"
          - "mydb"
          - "-c"
          - "VACUUM ANALYZE public.users"
        env:
          - name: PGPASSWORD
            value: "secret"
```

## 🧪 Testing

### Local Testing (Dry-Run)

```bash
# Run locally in dry-run mode
./bin/postgres-unbloat-k8s \
  --mode=local \
  --dry-run=true \
  --log-level=debug

# This will:
# 1. Discover PostgreSQL instances
# 2. Log what Jobs WOULD be created
# 3. Not actually create any Jobs
```

### In-Cluster Testing

```bash
# Deploy with dry-run=true
helm install postgres-unbloat deploy/helm \
  --set config.dryRun=true \
  --namespace postgres-unbload \
  --create-namespace

# Check logs
kubectl logs -n postgres-unbload deployment/postgres-unbloat-k8s -f

# View Jobs (should show none in dry-run)
kubectl get jobs -n postgres-unbload
```

## 📚 Usage as a Library

```go
package main

import (
    "context"
    "github.com/mithucste30/postgres-unbloat-k8s/pkg/vacuum"
    "github.com/mithucste30/postgres-unbloat-k8s/pkg/discoverer"
    "k8s.io/client-go/kubernetes"
)

func main() {
    // Create Kubernetes client
    clientset, _ := kubernetes.NewForConfig(config)

    // Create job executor
    executor := vacuum.NewJobExecutor(clientset, false, "default")

    // Discover PostgreSQL
    discoverer := discoverer.NewKubectlDiscoverer(...)
    instance, _ := discoverer.FindByAlert(ctx, "default", "postgres-0")
    creds, _ := discoverer.GetCredentials(ctx, instance)

    // Create vacuum job
    db := &vacuum.Database{
        Host: creds.Host,
        Port: creds.Port,
        Database: creds.Database,
        Username: creds.Username,
        Password: creds.Password,
    }

    alert := &vacuum.Alert{
        Name: "PostgreSQLTableHighBloat",
        Labels: map[string]string{
            "schemaname": "public",
            "table": "users",
        },
    }

    job, _ := executor.ExecuteVacuumJob(ctx, db, alert)
    println("Created job:", job.Name)
}
```

## 🔐 RBAC Requirements

The application needs these permissions:

```yaml
rules:
  # For discovery
  - apiGroups: [""]
    resources: ["pods", "secrets", "configmaps"]
    verbs: ["get", "list", "watch"]

  # For Job management
  - apiGroups: ["batch"]
    resources: ["jobs"]
    verbs: ["get", "list", "watch", "create", "delete"]
```

## 📊 Monitoring

### Job Status

```bash
# List all vacuum jobs
kubectl get jobs -l app=postgres-unbloat

# Watch job completion
kubectl wait --for=condition=complete job -l app=postgres-unbloat

# View job logs
kubectl logs job/vacuum-public-users-1234567890
```

### Metrics

The application exposes metrics on port 9090:

```bash
kubectl port-forward svc/postgres-unbloat-k8s 9090:9090
curl http://localhost:9090/metrics
```

## 🤝 Contributing

Contributions welcome! Please read [CONTRIBUTING.md](CONTRIBUTING.md).

## 📄 License

Apache License 2.0 - see [LICENSE](LICENSE) for details.

## 🙏 Acknowledgments

- Built with [client-go](https://github.com/kubernetes/client-go)
- Uses [postgres:17-alpine](https://hub.docker.com/_/postgres) official image
- Inspired by PostgreSQL vacuum best practices
