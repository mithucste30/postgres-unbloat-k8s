# Examples

This directory contains examples demonstrating how to use `postgres-unbloat-k8s` as a reusable Go library.

## Examples

### 1. job_executor

Demonstrates how to use the `JobExecutor` to create Kubernetes Jobs for vacuum operations.

**Run:**
```bash
cd examples/job_executor
go run main.go
```

**What it does:**
- Creates a Kubernetes client
- Initializes a JobExecutor
- Creates a vacuum job for a specific table
- Shows job details and SQL query

**Use case:** When you need programmatic control over vacuum job creation.

---

### 2. discovery

Shows how to discover PostgreSQL instances running in Kubernetes.

**Run:**
```bash
cd examples/discovery
go run main.go
```

**What it does:**
- Scans Kubernetes for PostgreSQL pods
- Extracts connection information
- Retrieves credentials from secrets/configmaps
- Displays all discovered instances

**Use case:** When you need to find and catalog PostgreSQL instances in your cluster.

---

### 3. alert_handler

Demonstrates a complete alert handling workflow.

**Run:**
```bash
cd examples/alert_handler
go run main.go
```

**What it does:**
- Receives simulated Prometheus alerts
- Finds the target PostgreSQL instance
- Retrieves credentials
- Creates appropriate vacuum jobs
- Handles different alert types (high bloat, critical, not vacuumed)

**Use case:** When building a custom alert handler or integrating with your monitoring system.

---

### 4. library_usage

Shows how to integrate postgres-unbloat-k8s into your own application.

**Run:**
```bash
cd examples/library_usage
go run main.go
```

**What it does:**
- Simulates custom bloat detection logic
- Uses the JobExecutor as a library
- Creates vacuum jobs based on custom criteria
- Demonstrates dry-run mode for safe testing

**Use case:** When you want to add vacuum automation to an existing Go application.

---

## Running the Examples

### Prerequisites

1. Go 1.25 or later installed
2. Kubernetes cluster with kubectl configured
3. PostgreSQL running in the cluster

### Setup

1. Update kubeconfig paths in examples to point to your kubeconfig:
   ```bash
   export KUBECONFIG=/Users/kahf/.kube/config
   ```

2. Update database connection details if different from defaults:
   - Host: `postgres-postgresql.default.svc.cluster.local`
   - Port: `5432`
   - Username: `postgres`
   - Password: `secret`

3. Run examples in dry-run mode first to see what would happen:
   ```go
   executor := vacuum.NewJobExecutor(clientset, true, "default")
   //                                               ^^^
   //                                          dry-run=true
   ```

### Testing with Real PostgreSQL

If you want to test with a real PostgreSQL instance:

1. Deploy a test PostgreSQL:
   ```bash
   kubectl create namespace postgres-test
   helm install postgres bitnami/postgresql \
     --namespace postgres-test \
     --set auth.password=secret
   ```

2. Update connection details in the example

3. Run the example:
   ```bash
   cd examples/job_executor
   go run main.go
   ```

4. Check the created job:
   ```bash
   kubectl get jobs -n default
   kubectl logs job/vacuum-xxx -n default
   ```

---

## Common Patterns

### Pattern 1: Create Vacuum Jobs on Schedule

```go
// Run every hour
ticker := time.NewTicker(1 * time.Hour)
for range ticker.C {
    tables := detectBloatedTables()
    for _, table := range tables {
        executor.ExecuteVacuumJob(ctx, db, alertForTable(table))
    }
}
```

### Pattern 2: Respond to Custom Metrics

```go
// Query your metrics system
if metricValue > threshold {
    alert := createAlertFromMetric(metric)
    executor.ExecuteVacuumJob(ctx, db, alert)
}
```

### Pattern 3: Conditional Vacuum Based on Table Size

```go
if table.sizeBytes > 100*1024*1024 { // >100MB
    // Use extended timeout for large tables
    alert.Labels["use_extended_timeout"] = "true"
}
executor.ExecuteVacuumJob(ctx, db, alert)
```

---

## Tips

1. **Always start with dry-run mode** to verify what will be created
2. **Check job status** after creation to ensure completion
3. **Monitor job logs** to see vacuum output
4. **Set appropriate timeouts** for large tables
5. **Use labels** to track which alert triggered which job
6. **Clean up old jobs** - they auto-delete after 1 hour by default

## Troubleshooting

**Job not created:**
- Check RBAC permissions (need `create jobs`)
- Verify kubeconfig is correct
- Check namespace exists

**Job failed:**
- Check job logs: `kubectl logs job/<job-name>`
- Verify PostgreSQL credentials are correct
- Ensure PostgreSQL is reachable from the cluster

**Discovery fails:**
- Verify pod labels match your selectors
- Check secrets exist and contain required fields
- Ensure kubectl can access the cluster
