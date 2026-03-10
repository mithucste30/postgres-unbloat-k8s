package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/mithucste30/postgres-unbloat-k8s/pkg/alert"
)

// Server represents the webhook HTTP server
type Server struct {
	handler    *alert.AlertHandler
	port       int
	path       string
	secret     string
	httpServer *http.Server
}

// NewServer creates a new webhook server
func NewServer(handler *alert.AlertHandler, port int, path, secret string) *Server {
	return &Server{
		handler: handler,
		port:    port,
		path:    path,
		secret:  secret,
	}
}

// Start starts the webhook server
func (s *Server) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc(s.path, s.handleWebhook)

	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.port),
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("[Webhook] Starting server on port %d", s.port)
	log.Printf("[Webhook] Webhook path: %s", s.path)

	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("webhook server error: %w", err)
	}

	return nil
}

// Stop stops the webhook server gracefully
func (s *Server) Stop(ctx context.Context) error {
	log.Printf("[Webhook] Shutting down server...")
	return s.httpServer.Shutdown(ctx)
}

// PrometheusAlert represents a Prometheus alert webhook payload
type PrometheusAlert struct {
	Status       string            `json:"status"`
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	StartsAt     time.Time         `json:"startsAt"`
	EndsAt       time.Time         `json:"endsAt"`
	GeneratorURL string            `json:"generatorURL"`
	Fingerprint  string            `json:"fingerprint"`
}

// PrometheusWebhookPayload represents the full webhook payload from Alertmanager
type PrometheusWebhookPayload struct {
	Receiver          string             `json:"receiver"`
	Status            string             `json:"status"`
	Alerts            []PrometheusAlert  `json:"alerts"`
	GroupLabels       map[string]string `json:"groupLabels"`
	CommonLabels      map[string]string `json:"commonLabels"`
	CommonAnnotations map[string]string `json:"commonAnnotations"`
	ExternalURL       string             `json:"externalURL"`
	Version           string             `json:"version"`
	GroupKey          string             `json:"groupKey"`
	TruncatedAlerts   int                `json:"truncatedAlerts"`
}

func (s *Server) handleWebhook(w http.ResponseWriter, r *http.Request) {
	// Verify method
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Verify secret if configured
	if s.secret != "" {
		providedSecret := r.Header.Get("X-Webhook-Secret")
		if providedSecret != s.secret {
			log.Printf("[Webhook] Unauthorized request from %s", r.RemoteAddr)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
	}

	// Read body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("[Webhook] Failed to read request body: %v", err)
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Parse payload
	var payload PrometheusWebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		log.Printf("[Webhook] Failed to parse payload: %v", err)
		http.Error(w, "Failed to parse payload", http.StatusBadRequest)
		return
	}

	log.Printf("[Webhook] Received %d alerts, status: %s", len(payload.Alerts), payload.Status)

	// Process only firing alerts
	if payload.Status != "firing" {
		log.Printf("[Webhook] Skipping non-firing alerts (status: %s)", payload.Status)
		w.WriteHeader(http.StatusOK)
		return
	}

	// Process each alert
	for _, promAlert := range payload.Alerts {
		if err := s.processAlert(r.Context(), promAlert); err != nil {
			log.Printf("[Webhook] Failed to process alert: %v", err)
			// Continue processing other alerts
			continue
		}
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status": "success", "processed": %d}`, len(payload.Alerts))
}

func (s *Server) processAlert(ctx context.Context, promAlert PrometheusAlert) error {
	// Convert Prometheus alert to internal Alert format
	internalAlert := &alert.Alert{
		Name:        getAlertName(promAlert.Labels),
		Severity:    getSeverity(promAlert.Labels),
		Labels:      promAlert.Labels,
		Annotations: promAlert.Annotations,
		Fingerprint: promAlert.Fingerprint,
	}

	// Extract required labels
	namespace := promAlert.Labels["namespace"]
	podName := promAlert.Labels["pod"]
	schemaname := promAlert.Labels["schemaname"]
	tableName := promAlert.Labels["table"]

	if namespace == "" || podName == "" || schemaname == "" || tableName == "" {
		return fmt.Errorf("missing required labels (namespace, pod, schemaname, table)")
	}

	log.Printf("[Webhook] Processing alert: %s for %s.%s", internalAlert.Name, schemaname, tableName)
	log.Printf("[Webhook]   Namespace: %s, Pod: %s", namespace, podName)
	log.Printf("[Webhook]   Severity: %s", internalAlert.Severity)

	// Handle the alert
	return s.handler.Handle(ctx, internalAlert)
}

// getAlertName extracts alert name from labels or annotations
func getAlertName(labels map[string]string) string {
	for _, key := range []string{"alertname", "alert_name", "alert"} {
		if name, ok := labels[key]; ok {
			return name
		}
	}
	return "UnknownAlert"
}

// getSeverity extracts severity from labels
func getSeverity(labels map[string]string) string {
	for _, key := range []string{"severity", "alert_severity", "level"} {
		if severity, ok := labels[key]; ok {
			return severity
		}
	}
	return "warning"
}
