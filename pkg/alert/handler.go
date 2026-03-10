package alert

import (
	"context"
	"fmt"
	"log"

	"github.com/mithucste30/postgres-unbloat-k8s/pkg/discoverer"
	"github.com/mithucste30/postgres-unbloat-k8s/pkg/vacuum"
)

type AlertHandler struct {
	discoverer discoverer.Discoverer
	executor   vacuum.Executor
	dryRun     bool
}

func NewHandler(disc discoverer.Discoverer, exec vacuum.Executor, dryRun bool) *AlertHandler {
	return &AlertHandler{
		discoverer: disc,
		executor:   exec,
		dryRun:     dryRun,
	}
}

func (h *AlertHandler) Handle(ctx context.Context, alert *Alert) error {
	log.Printf("[Handler] Handling alert: %s", alert.Name)

	namespace := alert.GetLabel("namespace")
	podName := alert.GetLabel("pod")

	if namespace == "" || podName == "" {
		return fmt.Errorf("missing namespace or pod")
	}

	instance, err := h.discoverer.FindByAlert(ctx, namespace, podName)
	if err != nil {
		return err
	}

	creds, err := h.discoverer.GetCredentials(ctx, instance)
	if err != nil {
		return err
	}

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

	table := alert.GetLabel("schemaname") + "." + alert.GetLabel("table")
	opts := vacuum.Options{DryRun: h.dryRun, Analyze: true}

	return h.executor.VacuumAnalyze(ctx, db, table, opts)
}
