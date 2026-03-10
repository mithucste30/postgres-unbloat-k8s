package main

import (
	"context"
	"fmt"
	"log"

	"github.com/mithucste30/postgres-unbloat-k8s/pkg/alert"
	"github.com/mithucste30/postgres-unbloat-k8s/pkg/discoverer"
	"github.com/mithucste30/postgres-unbloat-k8s/pkg/vacuum"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// This example demonstrates how to create a custom alert handler
// that responds to different alert types
func main() {
	ctx := context.Background()

	// Setup Kubernetes client
	config, err := clientcmd.BuildConfigFromFlags("", "/Users/kahf/.kube/config")
	if err != nil {
		log.Fatalf("Failed to create config: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Failed to create clientset: %v", err)
	}

	// Create components
	disc := discoverer.NewKubectlDiscoverer(
		"/Users/kahf/.kube/config",
		"",
		[]string{"default"},
		map[string]string{"app": "postgresql"},
	)

	jobExecutor := vacuum.NewJobExecutor(clientset, false, "default")

	// Create custom handler
	handler := &CustomVacuumHandler{
		discoverer:  disc,
		jobExecutor: jobExecutor,
	}

	// Simulate receiving alerts from Prometheus
	alerts := createTestAlerts()

	log.Printf("Processing %d test alerts...\n\n", len(alerts))

	for i, alert := range alerts {
		log.Printf("Alert %d: %s", i+1, alert.Name)
		log.Printf("  Table: %s.%s", alert.Labels["schemaname"], alert.Labels["table"])
		log.Printf("  Severity: %s", alert.Severity)

		if err := handler.Handle(ctx, alert); err != nil {
			log.Printf("  ❌ Failed: %v\n", err)
		} else {
			log.Printf("  ✅ Success\n")
		}
	}
}

// CustomVacuumHandler handles alerts by creating vacuum jobs
type CustomVacuumHandler struct {
	discoverer  discoverer.Discoverer
	jobExecutor *vacuum.JobExecutor
}

func (h *CustomVacuumHandler) Handle(ctx context.Context, alert *alert.Alert) error {
	// Extract alert labels
	namespace := alert.GetLabel("namespace")
	podName := alert.GetLabel("pod")

	if namespace == "" || podName == "" {
		return fmt.Errorf("missing namespace or pod in alert")
	}

	// Find the PostgreSQL instance
	instance, err := h.discoverer.FindByAlert(ctx, namespace, podName)
	if err != nil {
		return fmt.Errorf("failed to find instance: %w", err)
	}

	// Get credentials
	creds, err := h.discoverer.GetCredentials(ctx, instance)
	if err != nil {
		return fmt.Errorf("failed to get credentials: %w", err)
	}

	// Build database connection info
	db := &vacuum.Database{
		Namespace: instance.Namespace,
		PodName:   instance.PodName,
		Host:      creds.Host,
		Port:      creds.Port,
		Database:  creds.Database,
		Username:  creds.Username,
		Password:  creds.Password,
		SSLMode:   "disable",
	}

	// Create vacuum job
	vacuumAlert := &vacuum.Alert{
		Name:     alert.Name,
		Severity: alert.Severity,
		Labels:   alert.Labels,
	}

	job, err := h.jobExecutor.ExecuteVacuumJob(ctx, db, vacuumAlert)
	if err != nil {
		return fmt.Errorf("failed to create job: %w", err)
	}

	log.Printf("  Created job: %s", job.Name)
	return nil
}

// createTestAlerts creates sample alerts for testing
func createTestAlerts() []*alert.Alert {
	return []*alert.Alert{
		{
			Name:     "PostgreSQLTableHighBloat",
			Severity: "warning",
			Status:   "firing",
			Labels: map[string]string{
				"namespace":  "default",
				"pod":        "postgres-0",
				"schemaname": "public",
				"table":      "users",
			},
			Annotations: map[string]string{
				"description": "Table has 15% dead tuples",
			},
			Value: 15.0,
		},
		{
			Name:     "PostgreSQLTableCriticalBloat",
			Severity: "critical",
			Status:   "firing",
			Labels: map[string]string{
				"namespace":  "default",
				"pod":        "postgres-0",
				"schemaname": "public",
				"table":      "orders",
			},
			Annotations: map[string]string{
				"description": "Table has 35% dead tuples (CRITICAL)",
			},
			Value: 35.0,
		},
		{
			Name:     "PostgreSQLTableNotVacuumed",
			Severity: "warning",
			Status:   "firing",
			Labels: map[string]string{
				"namespace":  "default",
				"pod":        "postgres-0",
				"schemaname": "public",
				"table":      "products",
			},
			Annotations: map[string]string{
				"description": "Table hasn't been vacuumed in 24 hours",
			},
			Value: 24.0,
		},
	}
}
