package main

import (
	"context"
	"fmt"
	"log"

	"github.com/mithucste30/postgres-unbloat-k8s/pkg/discoverer"
)

// This example demonstrates how to discover PostgreSQL instances in Kubernetes
func main() {
	ctx := context.Background()

	// Create a kubectl-based discoverer for local development
	disc := discoverer.NewKubectlDiscoverer(
		"/Users/kahf/.kube/config", // path to kubeconfig
		"",                          // use current context
		[]string{"default", "database"}, // namespaces to search
		map[string]string{           // label selectors
			"app": "postgresql",
		},
	)

	// Discover all PostgreSQL instances
	instances, err := disc.DiscoverPostgreSQL(ctx)
	if err != nil {
		log.Fatalf("Discovery failed: %v", err)
	}

	log.Printf("Found %d PostgreSQL instances:\n", len(instances))

	for i, instance := range instances {
		fmt.Printf("\n%d. %s/%s\n", i+1, instance.Namespace, instance.PodName)
		fmt.Printf("   Host: %s:%d\n", instance.Host, instance.Port)
		fmt.Printf("   Labels: %v\n", instance.PodLabels)

		// Try to get credentials for this instance
		creds, err := disc.GetCredentials(ctx, instance)
		if err != nil {
			fmt.Printf("   ⚠️  Could not get credentials: %v\n", err)
			continue
		}

		fmt.Printf("   ✅ Credentials:\n")
		fmt.Printf("      Username: %s\n", creds.Username)
		fmt.Printf("      Database: %s\n", creds.Database)
		fmt.Printf("      Host: %s\n", creds.Host)
		fmt.Printf("      Port: %d\n", creds.Port)
	}

	// Example: Find a specific instance by alert labels
	fmt.Println("\n--- Finding specific instance by alert ---")

	instance, err := disc.FindByAlert(ctx, "default", "postgres-0")
	if err != nil {
		log.Printf("Could not find instance: %v", err)
		return
	}

	fmt.Printf("Found instance: %s/%s\n", instance.Namespace, instance.PodName)
	fmt.Printf("Connection: %s:%d\n", instance.Host, instance.Port)
}
