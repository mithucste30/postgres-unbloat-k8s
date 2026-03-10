package vacuum

import (
	"context"
	"time"
)

// Database represents a PostgreSQL database connection.
type Database struct {
	Host      string
	Port      int
	Database  string
	Username  string
	Password  string
	SSLMode   string
	Namespace string
	PodName   string
}

// Options configure vacuum operations.
type Options struct {
	DryRun              bool
	Full                bool
	Analyze             bool
	VacuumIndex         bool
	Timeout             time.Duration
	MinDeadTuples       int
	MaxDeadTuplePercent int
	TableSizeThreshold  int64
}

// Executor defines the interface for executing VACUUM operations.
type Executor interface {
	Vacuum(ctx context.Context, db *Database, table string, opts Options) error
	Analyze(ctx context.Context, db *Database, table string) error
	VacuumAnalyze(ctx context.Context, db *Database, table string, opts Options) error
}

// Strategy defines a vacuum strategy for a specific alert type.
type Strategy interface {
	ShouldExecute(alert *Alert) bool
	GetOptions(alert *Alert) Options
	Execute(ctx context.Context, executor Executor, db *Database, alert *Alert) error
}

// Alert represents a Prometheus alert with relevant metadata.
type Alert struct {
	Name        string
	Severity    string
	Labels      map[string]string
	Annotations map[string]string
	Value       float64
}
