package config

import "time"

type Config struct {
	Server     ServerConfig     `json:"server"`
	Discovery  DiscoveryConfig  `json:"discovery"`
	Vacuum     VacuumConfig     `json:"vacuum"`
	Webhook    WebhookConfig    `json:"webhook"`
	Logging    LoggingConfig    `json:"logging"`
	Kubernetes KubernetesConfig `json:"kubernetes"`
}

type ServerConfig struct {
	Mode        string `json:"mode"`
	Address     string `json:"address"`
	Port        int    `json:"port"`
	MetricsPort int    `json:"metricsPort"`
	DryRun      bool   `json:"dryRun"`
}

type DiscoveryConfig struct {
	Enabled        bool              `json:"enabled"`
	Namespaces     []string          `json:"namespaces"`
	LabelSelectors map[string]string `json:"labelSelectors"`
}

type VacuumConfig struct {
	DefaultTimeout       time.Duration `json:"defaultTimeout"`
	AnalyzeAfterVacuum   bool          `json:"analyzeAfterVacuum"`
	MaxConcurrentVacuums int           `json:"maxConcurrentVacuums"`
}

type WebhookConfig struct {
	Enabled bool   `json:"enabled"`
	Port    int    `json:"port"`
	Path    string `json:"path"`
	Secret  string `json:"secret"`
}

type LoggingConfig struct {
	Level  string `json:"level"`
	Format string `json:"format"`
}

type KubernetesConfig struct {
	Kubeconfig         string `json:"kubeconfig"`
	Context            string `json:"context"`
	Namespace          string `json:"namespace"`
	InClusterNamespace string `json:"inClusterNamespace"`
	InCluster          bool   `json:"inCluster"`
}

func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Mode:        "local",
			Address:     "0.0.0.0",
			Port:        8080,
			MetricsPort: 9090,
			DryRun:      true,
		},
		Discovery: DiscoveryConfig{
			Enabled:    true,
			Namespaces: []string{"*"}, // Scan all namespaces by default
			LabelSelectors: map[string]string{
				// Standard PostgreSQL labels
				"app.kubernetes.io/name":     "postgresql",
				"app":                        "postgresql",
				"app.kubernetes.io/instance": "postgres",
				"role":                       "database",
				// CloudNativePG labels (presence-based - empty values match any)
				"cnpg.io/cluster":      "",
				"cnpg.io/instanceName": "",
				// Generic database label
				"app.kubernetes.io/component": "database",
			},
		},
		Vacuum: VacuumConfig{
			DefaultTimeout:       30 * time.Minute,
			AnalyzeAfterVacuum:   true,
			MaxConcurrentVacuums: 3,
		},
		Webhook: WebhookConfig{
			Enabled: true,
			Port:    8080,
			Path:    "/webhook",
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
		},
		Kubernetes: KubernetesConfig{
			Namespace:          "default",
			InClusterNamespace: "default",
			InCluster:          false,
		},
	}
}
