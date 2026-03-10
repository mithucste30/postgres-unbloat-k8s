package main

import (
	"context"
	"log"

	"github.com/mithucste30/postgres-unbloat-k8s/pkg/vacuum"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// This example demonstrates how to use the JobExecutor to create vacuum jobs
func main() {
	// Create Kubernetes client from kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", "/Users/kahf/.kube/config")
	if err != nil {
		log.Fatalf("Failed to create Kubernetes config: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Failed to create clientset: %v", err)
	}

	// Create JobExecutor
	executor := vacuum.NewJobExecutor(
		clientset,
		false, // dry-run=false to actually create jobs
		"default", // namespace where jobs will be created
	)

	// Define the target database
	db := &vacuum.Database{
		Host:     "postgres-postgresql.default.svc.cluster.local",
		Port:     5432,
		Database: "mydb",
		Username: "postgres",
		Password: "secret",
		SSLMode:  "disable",
		Namespace: "default",
		PodName:   "postgres-0",
	}

	// Define an alert that triggers the vacuum
	alert := &vacuum.Alert{
		Name:     "PostgreSQLTableHighBloat",
		Severity: "warning",
		Labels: map[string]string{
			"namespace":  "default",
			"pod":        "postgres-0",
			"schemaname": "public",
			"table":      "users",
		},
		Annotations: map[string]string{
			"description": "Table public.users has 15% dead tuples",
		},
		Value: 15.0,
	}

	// Create the vacuum job
	ctx := context.Background()
	job, err := executor.ExecuteVacuumJob(ctx, db, alert)
	if err != nil {
		log.Fatalf("Failed to execute vacuum job: %v", err)
	}

	log.Printf("✅ Successfully created vacuum job: %s", job.Name)
	log.Printf("   Namespace: %s", job.Namespace)
	log.Printf("   Labels: %v", job.Labels)
	log.Printf("   SQL Query: %s", job.Annotations["sql-query"])
}
