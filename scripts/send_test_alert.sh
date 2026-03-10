#!/bin/bash

# Test Alert Webhook Sender for postgres-unbloat-k8s
# This script sends a test alert to the postgres-unbloat-k8s webhook endpoint

set -e

# Configuration
WEBHOOK_URL="${WEBHOOK_URL:-http://postgres-unbloat-postgres-unbloat-k8s.postgres-test.svc.cluster.local:8080/webhook}"
NAMESPACE="${NAMESPACE:-postgres-test}"
POD_NAME="${POD_NAME:-postgres-postgresql-0}"
SCHEMA_NAME="${SCHEMA_NAME:-bloat_test}"
TABLE_NAME="${TABLE_NAME:-high_bloat_table}"

echo "======================================"
echo "PostgreSQL Unbloat - Test Alert Sender"
echo "======================================"
echo ""
echo "Webhook URL: $WEBHOOK_URL"
echo "Target Namespace: $NAMESPACE"
echo "Target Pod: $POD_NAME"
echo "Target Schema: $SCHEMA_NAME"
echo "Target Table: $TABLE_NAME"
echo ""

# Test Alert 1: High Bloat (>10%)
echo "Sending Test Alert 1: PostgreSQLTableHighBloat..."
curl -X POST "$WEBHOOK_URL" \
  -H "Content-Type: application/json" \
  -d '{
    "receiver": "postgres-unbloat",
    "status": "firing",
    "alerts": [
      {
        "status": "firing",
        "labels": {
          "alertname": "PostgreSQLTableHighBloat",
          "severity": "warning",
          "namespace": "'"$NAMESPACE"'",
          "pod": "'"$POD_NAME"'",
          "schemaname": "'"$SCHEMA_NAME"'",
          "table": "'"$TABLE_NAME"'"
        },
        "annotations": {
          "description": "Table '"$SCHEMA_NAME"."$TABLE_NAME"' has 15%% dead tuples",
          "summary": "High table bloat detected"
        },
        "startsAt": "'$(date -u +%Y-%m-%dT%H:%M:%SZ)'",
        "endsAt": "0001-01-01T00:00:00Z",
        "generatorURL": "http://prometheus:9090/graph",
        "fingerprint": "test_alert_1"
      }
    ],
    "groupLabels": {
      "alertname": "PostgreSQLTableHighBloat",
      "severity": "warning"
    },
    "commonLabels": {
      "namespace": "'"$NAMESPACE"'",
      "pod": "'"$POD_NAME"'"
    },
    "commonAnnotations": {
      "summary": "High bloat detected in PostgreSQL"
    },
    "externalURL": "http://alertmanager:9093",
    "version": "4",
    "groupKey": "{}:{alertname=\"PostgreSQLTableHighBloat\"}",
    "truncatedAlerts": 0
  }'

echo ""
echo "✅ Alert 1 sent!"
echo ""
echo "Expected behavior:"
echo "  - Kubernetes Job should be created"
echo "  - Job name pattern: vacuum-$SCHEMA_NAME-$TABLE_NAME-*"
echo "  - SQL: VACUUM ANALYZE $SCHEMA_NAME.$TABLE_NAME"
echo ""

sleep 2

# Test Alert 2: Critical Bloat (>30%)
echo "Sending Test Alert 2: PostgreSQLTableCriticalBloat..."
curl -X POST "$WEBHOOK_URL" \
  -H "Content-Type: application/json" \
  -d '{
    "receiver": "postgres-unbloat",
    "status": "firing",
    "alerts": [
      {
        "status": "firing",
        "labels": {
          "alertname": "PostgreSQLTableCriticalBloat",
          "severity": "critical",
          "namespace": "'"$NAMESPACE"'",
          "pod": "'"$POD_NAME"'",
          "schemaname": "'"$SCHEMA_NAME"'",
          "table": "critical_bloat_table"
        },
        "annotations": {
          "description": "Table '"$SCHEMA_NAME".critical_bloat_table' has 35%% dead tuples",
          "summary": "Critical table bloat detected - requires FULL vacuum"
        },
        "startsAt": "'$(date -u +%Y-%m-%dT%H:%M:%SZ)'",
        "endsAt": "0001-01-01T00:00:00Z",
        "generatorURL": "http://prometheus:9090/graph",
        "fingerprint": "test_alert_2"
      }
    ],
    "groupLabels": {
      "alertname": "PostgreSQLTableCriticalBloat",
      "severity": "critical"
    },
    "commonLabels": {
      "namespace": "'"$NAMESPACE"'",
      "pod": "'"$POD_NAME"'"
    },
    "commonAnnotations": {
      "summary": "Critical bloat detected in PostgreSQL"
    },
    "externalURL": "http://alertmanager:9093",
    "version": "4",
    "groupKey": "{}:{alertname=\"PostgreSQLTableCriticalBloat\"}",
    "truncatedAlerts": 0
  }'

echo ""
echo "✅ Alert 2 sent!"
echo ""
echo "Expected behavior:"
echo "  - Kubernetes Job should be created"
echo "  - SQL: VACUUM FULL ANALYZE $SCHEMA_NAME.critical_bloat_table"
echo ""

echo "======================================"
echo "Test alerts sent successfully!"
echo "======================================"
echo ""
echo "To check for created vacuum jobs, run:"
echo "  kubectl get jobs -n $NAMESPACE"
echo ""
echo "To view job logs:"
echo "  kubectl logs -n $NAMESPACE -l component=vacuum-job --tail=-1"
echo ""
echo "To check PostgreSQL bloat before/after:"
echo "  kubectl exec -n $NAMESPACE $POD_NAME -- psql -U postgres -d testdb -c \"SELECT schemaname, tablename, pg_stat_get_dead_tuples(c.oid) FROM pg_stat_user_tables s JOIN pg_class c ON c.relname = s.relname WHERE schemaname = '$SCHEMA_NAME';\""
echo ""
