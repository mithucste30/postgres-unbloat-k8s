package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/mithucste30/postgres-unbloat-k8s/pkg/vacuum"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// This example demonstrates how to integrate postgres-unbloat-k8s
// as a library into your own Go application
func main() {
	ctx := context.Background()

	// Initialize Kubernetes client
	config, err := clientcmd.BuildConfigFromFlags("", "/Users/kahf/.kube/config")
	if err != nil {
		log.Fatalf("Failed to create config: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Failed to create clientset: %v", err)
	}

	// Create the job executor
	executor := vacuum.NewJobExecutor(
		clientset,
		true, // dry-run mode for safety
		"default",
	)

	// Your application logic might detect bloat in a custom way
	// For example, querying PostgreSQL directly or using custom metrics
	tablesInNeedOfVacuum := detectBloatedTables()

	fmt.Printf("Found %d tables needing vacuum:\n\n", len(tablesInNeedOfVacuum))

	for _, table := range tablesInNeedOfVacuum {
		// Create a custom alert
		alert := &vacuum.Alert{
			Name:     "CustomBloatDetection",
			Severity: getSeverity(table.bloatPercent),
			Labels: map[string]string{
				"namespace":  "default",
				"pod":        "postgres-0",
				"schemaname": table.schema,
				"table":      table.name,
			},
			Value: table.bloatPercent,
		}

		// Database connection info
		db := &vacuum.Database{
			Host:      "postgres-postgresql.default.svc.cluster.local",
			Port:      5432,
			Database:  "myapp",
			Username:  "appuser",
			Password:  "secret",
			SSLMode:   "disable",
			Namespace: "default",
			PodName:   "postgres-0",
		}

		// Create vacuum job
		job, err := executor.ExecuteVacuumJob(ctx, db, alert)
		if err != nil {
			log.Printf("❌ Failed to vacuum %s.%s: %v\n", table.schema, table.name, err)
			continue
		}

		fmt.Printf("✅ Created job for %s.%s: %s\n", table.schema, table.name, job.Name)
		fmt.Printf("   SQL: %s\n\n", job.Annotations["sql-query"])

		// In production, you might want to wait for the job to complete
		// or just monitor it asynchronously
	}
}

// bloatedTable represents a table that needs vacuuming
type bloatedTable struct {
	schema       string
	name         string
	bloatPercent float64
	lastVacuumed time.Time
}

// detectBloatedTables simulates your custom bloat detection logic
// In a real application, this might query pg_stat_user_tables directly
func detectBloatedTables() []*bloatedTable {
	return []*bloatedTable{
		{
			schema:       "public",
			name:         "users",
			bloatPercent: 25.5,
			lastVacuumed: time.Now().Add(-48 * time.Hour),
		},
		{
			schema:       "public",
			name:         "orders",
			bloatPercent: 42.8,
			lastVacuumed: time.Now().Add(-72 * time.Hour),
		},
		{
			schema:       "public",
			name:         "audit_logs",
			bloatPercent: 18.2,
			lastVacuumed: time.Now().Add(-36 * time.Hour),
		},
	}
}

// getSeverity determines alert severity based on bloat percentage
func getSeverity(bloatPercent float64) string {
	if bloatPercent > 30 {
		return "critical"
	}
	if bloatPercent > 20 {
		return "warning"
	}
	return "info"
}
