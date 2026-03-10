package vacuum

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type JobExecutor struct {
	clientset *kubernetes.Clientset
	dryRun    bool
	defaultNS string
}

func NewJobExecutor(clientset *kubernetes.Clientset, dryRun bool, defaultNS string) *JobExecutor {
	return &JobExecutor{
		clientset: clientset,
		dryRun:    dryRun,
		defaultNS: defaultNS,
	}
}

func (e *JobExecutor) ExecuteVacuumJob(ctx context.Context, db *Database, alert *Alert) (*batchv1.Job, error) {
	table := alert.Labels["schemaname"] + "." + alert.Labels["table"]
	jobName := fmt.Sprintf("vacuum-%s-%d", strings.ReplaceAll(table, ".", "-"), time.Now().Unix())
	jobName = sanitizeJobName(jobName)

	// Determine vacuum type based on alert
	sql := e.buildVacuumSQLFromAlert(table, alert)

	// Create Job
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: e.defaultNS,
			Labels: map[string]string{
				"app":            "postgres-unbloat",
				"component":      "vacuum-job",
				"alert-name":     alert.Name,
				"alert-severity": alert.Severity,
				"table":          table,
				"database":       db.Database,
				"target-ns":      db.Namespace,
				"target-pod":     db.PodName,
			},
			Annotations: map[string]string{
				"created-by": "postgres-unbloat-k8s",
				"sql-query":  sql,
			},
		},
		Spec: batchv1.JobSpec{
			TTLSecondsAfterFinished: func(i int32) *int32 { return &i }(3600), // Cleanup after 1 hour
			BackoffLimit:            func(i int32) *int32 { return &i }(0),    // No retries
			ActiveDeadlineSeconds:   func(i int64) *int64 { return &i }(7200), // Max 2 hours
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						{
							Name:    "vacuum",
							Image:   "postgres:17-alpine",
							Command: []string{"psql"},
							Args: []string{
								"-h", db.Host,
								"-p", fmt.Sprintf("%d", db.Port),
								"-U", db.Username,
								"-d", db.Database,
								"-c", sql,
							},
							Env: []corev1.EnvVar{
								{
									Name:  "PGPASSWORD",
									Value: db.Password,
								},
							},
						},
					},
				},
			},
		},
	}

	logPrefix := "[JOB]"
	if e.dryRun {
		logPrefix = "[DRY-RUN JOB]"
		log.Printf("%s Would create Job: %s", logPrefix, jobName)
		log.Printf("%s Target: %s@%s:%d/%s", logPrefix, db.Username, db.Host, db.Port, db.Database)
		log.Printf("%s SQL: %s", logPrefix, sql)
		return job, nil
	}

	log.Printf("[JOB] Creating vacuum job: %s", jobName)
	log.Printf("[JOB] Target: %s@%s:%d/%s", db.Username, db.Host, db.Port, db.Database)
	log.Printf("[JOB] SQL: %s", sql)

	created, err := e.clientset.BatchV1().Jobs(e.defaultNS).Create(ctx, job, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create job: %w", err)
	}

	log.Printf("[JOB] Job created successfully: %s", created.Name)
	return created, nil
}

func (e *JobExecutor) buildVacuumSQLFromAlert(table string, alert *Alert) string {
	sql := "VACUUM"

	// Adjust vacuum type based on alert
	switch alert.Name {
	case "PostgreSQLTableCriticalBloat":
		sql += " FULL"
	}

	sql += " ANALYZE " + table
	return sql
}

func sanitizeJobName(name string) string {
	sanitized := ""
	for _, c := range name {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' {
			sanitized += string(c)
		} else if c >= 'A' && c <= 'Z' {
			sanitized += string(c + 32)
		} else {
			sanitized += "-"
		}
	}
	if len(sanitized) > 63 {
		sanitized = sanitized[:63]
	}
	return sanitized
}
