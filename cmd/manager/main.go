package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/mithucste30/postgres-unbloat-k8s/pkg/config"
	"github.com/mithucste30/postgres-unbloat-k8s/pkg/discoverer"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("Starting PostgreSQL Unbloat for Kubernetes (Job-based)...")

	cfg := config.DefaultConfig()

	mode := flag.String("mode", cfg.Server.Mode, "Execution mode")
	dryRun := flag.Bool("dry-run", cfg.Server.DryRun, "Dry-run mode")
	namespace := flag.String("namespace", cfg.Kubernetes.Namespace, "Namespace for jobs")
	kubeconfig := flag.String("kubeconfig", cfg.Kubernetes.Kubeconfig, "Path to kubeconfig")
	logLevel := flag.String("log-level", cfg.Logging.Level, "Log level")

	flag.Parse()

	cfg.Server.Mode = *mode
	cfg.Server.DryRun = *dryRun
	cfg.Kubernetes.Namespace = *namespace
	cfg.Kubernetes.Kubeconfig = *kubeconfig
	cfg.Logging.Level = *logLevel

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	k8sClient, err := createKubernetesClient(cfg)
	if err != nil {
		log.Fatalf("Failed to create Kubernetes client: %v", err)
	}

	disc := discoverer.NewKubectlDiscoverer(
		cfg.Kubernetes.Kubeconfig,
		cfg.Kubernetes.Context,
		cfg.Discovery.Namespaces,
		cfg.Discovery.LabelSelectors,
	)

	// TODO: Implement webhook handler
	_ = k8sClient // Will be used by webhook handler

	instances, err := disc.DiscoverPostgreSQL(ctx)
	if err != nil {
		log.Printf("Warning: Discovery failed: %v", err)
	} else {
		log.Printf("Discovered %d PostgreSQL instances", len(instances))
		for _, inst := range instances {
			log.Printf("  - %s/%s at %s:%d", inst.Namespace, inst.PodName, inst.Host, inst.Port)
		}
	}

	log.Println("System ready. Press Ctrl+C to exit.")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down...")
}

func createKubernetesClient(cfg *config.Config) (*kubernetes.Clientset, error) {
	if cfg.Server.Mode == "in-cluster" {
		config, err := rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
		return kubernetes.NewForConfig(config)
	}

	kubeconfig := cfg.Kubernetes.Kubeconfig
	if kubeconfig == "" {
		kubeconfig = os.Getenv("KUBECONFIG")
	}
	if kubeconfig == "" {
		home, _ := os.UserHomeDir()
		kubeconfig = home + "/.kube/config"
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(config)
}
