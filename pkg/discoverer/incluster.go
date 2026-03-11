package discoverer

import (
	"context"
	"fmt"
	"log"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type InClusterDiscoverer struct {
	clientset      *kubernetes.Clientset
	namespaces     []string
	labelSelectors []string // Multiple selectors for OR logic
}

func NewInClusterDiscoverer(clientset *kubernetes.Clientset, namespaces []string, labelSelectors map[string]string) *InClusterDiscoverer {
	// Convert each label selector into its own selector string
	// This allows OR logic - we'll try each selector separately
	var selectors []string
	for key, value := range labelSelectors {
		// Empty values mean "presence-only" selector (label key exists regardless of value)
		// This is needed for CNPG labels like cnpg.io/cluster
		if value == "" {
			selectors = append(selectors, key) // Presence-based selector
		} else {
			selectors = append(selectors, fmt.Sprintf("%s=%s", key, value)) // Exact match
		}
	}

	return &InClusterDiscoverer{
		clientset:      clientset,
		namespaces:     namespaces,
		labelSelectors: selectors,
	}
}

func (d *InClusterDiscoverer) DiscoverPostgreSQL(ctx context.Context) ([]*PostgreSQLInstance, error) {
	log.Printf("[InClusterDiscoverer] Discovering PostgreSQL instances...")
	log.Printf("[InClusterDiscoverer] Label selectors: %v", d.labelSelectors)

	var instances []*PostgreSQLInstance
	seen := make(map[string]bool) // Track discovered instances to avoid duplicates

	// Determine which namespaces to scan
	namespaces := d.namespaces
	if len(namespaces) == 1 && namespaces[0] == "*" {
		// Scan all namespaces
		nsList, err := d.clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to list namespaces: %w", err)
		}
		namespaces = make([]string, 0, len(nsList.Items))
		for _, ns := range nsList.Items {
			namespaces = append(namespaces, ns.Name)
		}
		log.Printf("[InClusterDiscoverer] Scanning %d namespaces", len(namespaces))
	} else {
		log.Printf("[InClusterDiscoverer] Scanning %d specific namespaces", len(namespaces))
	}

	// Try each label selector (OR logic)
	for _, selector := range d.labelSelectors {
		log.Printf("[InClusterDiscoverer] Trying selector: %s", selector)

		// Scan each namespace for PostgreSQL services
		for _, ns := range namespaces {
			services, err := d.clientset.CoreV1().Services(ns).List(ctx, metav1.ListOptions{
				LabelSelector: selector,
			})
			if err != nil {
				log.Printf("[InClusterDiscoverer] Failed to list services in namespace %s: %v", ns, err)
				continue
			}

			log.Printf("[InClusterDiscoverer] Found %d services with selector %s in namespace %s", len(services.Items), selector, ns)

			for _, svc := range services.Items {
				// Skip if already discovered
				key := fmt.Sprintf("%s/%s", ns, svc.Name)
				if seen[key] {
					continue
				}
				seen[key] = true

				if svc.Spec.Type != corev1.ServiceTypeClusterIP {
					log.Printf("[InClusterDiscoverer] Skipping service %s/%s (not ClusterIP)", ns, svc.Name)
					continue
				}

				// Find PostgreSQL port
				var pgPort int32 = 5432
				for _, port := range svc.Spec.Ports {
					if port.Name == "postgresql" || port.Port == 5432 {
						pgPort = port.Port
						break
					}
				}

				instance := &PostgreSQLInstance{
					Namespace: ns,
					PodName:   svc.Name,
					Host:      fmt.Sprintf("%s.%s.svc.cluster.local", svc.Name, ns),
					Port:      int(pgPort),
				}

				log.Printf("[InClusterDiscoverer] Discovered: %s/%s at %s:%d", ns, svc.Name, instance.Host, instance.Port)
				instances = append(instances, instance)
			}
		}
	}

	log.Printf("[InClusterDiscoverer] Total discovered instances: %d", len(instances))
	return instances, nil
}

func (d *InClusterDiscoverer) GetCredentials(ctx context.Context, instance *PostgreSQLInstance) (*Credentials, error) {
	// Get the service to find matching secrets
	svc, err := d.clientset.CoreV1().Services(instance.Namespace).Get(ctx, instance.PodName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get service: %w", err)
	}

	// Convert selector map to selector string
	selectorStr := ""
	for key, value := range svc.Spec.Selector {
		if selectorStr != "" {
			selectorStr += ","
		}
		selectorStr += fmt.Sprintf("%s=%s", key, value)
	}

	// Try to get credentials from secrets that match the service's pod selector
	if selectorStr != "" {
		secrets, err := d.clientset.CoreV1().Secrets(instance.Namespace).List(ctx, metav1.ListOptions{
			LabelSelector: selectorStr,
		})
		if err != nil {
			log.Printf("[InClusterDiscoverer] Failed to list secrets by selector: %v", err)
		} else {
			for _, secret := range secrets.Items {
				// Try common secret keys
				if username, ok := secret.Data["username"]; ok {
					if password, ok := secret.Data["password"]; ok {
						return &Credentials{
							Username: string(username),
							Password: string(password),
							Database: string(secret.Data["database"]),
							Host:     instance.Host,
							Port:     instance.Port,
						}, nil
					}
				}
				if postgresPassword, ok := secret.Data["postgres-password"]; ok {
					return &Credentials{
						Username: "postgres",
						Password: string(postgresPassword),
						Database: "postgres",
						Host:     instance.Host,
						Port:     instance.Port,
					}, nil
				}
				if password, ok := secret.Data["password"]; ok {
					return &Credentials{
						Username: "postgres",
						Password: string(password),
						Database: "postgres",
						Host:     instance.Host,
						Port:     instance.Port,
					}, nil
				}
			}
		}
	}

	// Fallback: Try secrets with the service's labels
	serviceLabels := svc.Labels
	for key, value := range serviceLabels {
		secrets, err := d.clientset.CoreV1().Secrets(instance.Namespace).List(ctx, metav1.ListOptions{
			LabelSelector: fmt.Sprintf("%s=%s", key, value),
		})
		if err != nil {
			continue
		}

		for _, secret := range secrets.Items {
			if postgresPassword, ok := secret.Data["postgres-password"]; ok {
				return &Credentials{
					Username: "postgres",
					Password: string(postgresPassword),
					Database: "postgres",
					Host:     instance.Host,
					Port:     instance.Port,
				}, nil
			}
			if username, ok := secret.Data["username"]; ok {
				if password, ok := secret.Data["password"]; ok {
					return &Credentials{
						Username: string(username),
						Password: string(password),
						Database: "postgres",
						Host:     instance.Host,
						Port:     instance.Port,
					}, nil
				}
			}
		}
	}

	// CNPG fallback: Try ${cluster-name}-super-user secret
	if cluster, ok := serviceLabels["cnpg.io/cluster"]; ok {
		secretName := fmt.Sprintf("%s-super-user", cluster)
		secret, err := d.clientset.CoreV1().Secrets(instance.Namespace).Get(ctx, secretName, metav1.GetOptions{})
		if err == nil {
			if username, ok := secret.Data["username"]; ok {
				if password, ok := secret.Data["password"]; ok {
					log.Printf("[InClusterDiscoverer] Found CNPG superuser secret: %s", secretName)
					return &Credentials{
						Username: string(username),
						Password: string(password),
						Database: "postgres",
						Host:     instance.Host,
						Port:     instance.Port,
					}, nil
				}
			}
		}
	}

	// Bitnami fallback: Try ${name}-postgresql secret
	if bitnamiInstance, ok := serviceLabels["app.kubernetes.io/instance"]; ok {
		secretName := fmt.Sprintf("%s-postgresql", bitnamiInstance)
		secret, err := d.clientset.CoreV1().Secrets(instance.Namespace).Get(ctx, secretName, metav1.GetOptions{})
		if err == nil {
			if password, ok := secret.Data["password"]; ok {
				log.Printf("[InClusterDiscoverer] Found Bitnami secret: %s", secretName)
				return &Credentials{
					Username: "postgres",
					Password: string(password),
					Database: "postgres",
					Host:     instance.Host,
					Port:     instance.Port,
				}, nil
			}
		}
	}

	// Return default credentials if no secret found
	return &Credentials{
		Username: "postgres",
		Password: "postgres",
		Database: "postgres",
		Host:     instance.Host,
		Port:     instance.Port,
	}, nil
}

func (d *InClusterDiscoverer) FindByAlert(ctx context.Context, namespace, podName string) (*PostgreSQLInstance, error) {
	// If pod name is provided, use the original logic
	if podName != "" {
		// Get the pod to verify it exists and get its labels
		pod, err := d.clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to get pod: %w", err)
		}

		// Find the service that selects this pod
		services, err := d.clientset.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to list services: %w", err)
		}

		var matchingService *corev1.Service
		for _, svc := range services.Items {
			// Check if this service's selector matches the pod's labels
			if svc.Spec.Selector != nil {
				matches := true
				for key, value := range svc.Spec.Selector {
					podLabelValue, ok := pod.Labels[key]
					if !ok || podLabelValue != value {
						matches = false
						break
					}
				}
				if matches && len(svc.Spec.Selector) > 0 {
					matchingService = &svc
					break
				}
			}
		}

		// If no matching service found, try to derive service name from pod name
		// Handles StatefulSet pods like postgres-postgresql-0 -> postgres-postgresql
		if matchingService == nil {
			// Try removing the suffix (e.g., "-0")
			serviceName := podName
			for i := len(podName) - 1; i >= 0; i-- {
				if podName[i] == '-' {
					serviceName = podName[:i]
					break
				}
			}
			_, err := d.clientset.CoreV1().Services(namespace).Get(ctx, serviceName, metav1.GetOptions{})
			if err == nil {
				matchingService = &corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name: serviceName,
					},
				}
			}
		}

		if matchingService == nil {
			return nil, fmt.Errorf("no service found matching pod %s", podName)
		}

		return d.createInstanceFromService(namespace, matchingService)
	}

	// If no pod name provided, try to find PostgreSQL service using label selectors
	log.Printf("[InClusterDiscoverer] No pod provided, searching for PostgreSQL service in namespace %s", namespace)

	// Try each label selector to find a PostgreSQL service
	for _, selector := range d.labelSelectors {
		services, err := d.clientset.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{
			LabelSelector: selector,
		})
		if err != nil {
			continue
		}

		// Return the first matching service
		if len(services.Items) > 0 {
			svc := &services.Items[0]
			log.Printf("[InClusterDiscoverer] Found PostgreSQL service: %s", svc.Name)
			return d.createInstanceFromService(namespace, svc)
		}
	}

	return nil, fmt.Errorf("no PostgreSQL service found in namespace %s", namespace)
}

func (d *InClusterDiscoverer) createInstanceFromService(namespace string, svc *corev1.Service) (*PostgreSQLInstance, error) {
	// Find PostgreSQL port
	var pgPort int32 = 5432
	if svc.Spec.Ports != nil {
		for _, port := range svc.Spec.Ports {
			if port.Name == "postgresql" || port.Port == 5432 {
				pgPort = port.Port
				break
			}
		}
	}

	return &PostgreSQLInstance{
		Namespace: namespace,
		PodName:   svc.Name,
		Host:      fmt.Sprintf("%s.%s.svc.cluster.local", svc.Name, namespace),
		Port:      int(pgPort),
	}, nil
}
