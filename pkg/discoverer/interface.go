package discoverer

import "context"

type Discoverer interface {
	DiscoverPostgreSQL(ctx context.Context) ([]*PostgreSQLInstance, error)
	GetCredentials(ctx context.Context, instance *PostgreSQLInstance) (*Credentials, error)
	FindByAlert(ctx context.Context, namespace, podName string) (*PostgreSQLInstance, error)
}

type PostgreSQLInstance struct {
	Namespace  string
	PodName    string
	PodLabels  map[string]string
	Host       string
	Port       int
	SecretName string
}

type Credentials struct {
	Username string
	Password string
	Database string
	Host     string
	Port     int
}
