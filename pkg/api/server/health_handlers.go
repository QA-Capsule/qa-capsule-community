package server

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/QA-Capsule/qa-capsule-community/pkg/core"
)

func registerHealthRoutes(config *core.Config) {
	http.HandleFunc("/healthz", handleHealthz)
	http.HandleFunc("/readyz", handleReadyz(config))
	http.HandleFunc("/metrics", handleMetrics)
}

func handleHealthz(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if r.Method != http.MethodHead {
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}
}

func handleReadyz(config *core.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		checks := map[string]string{}
		ready := true

		if core.DB == nil {
			checks["database"] = "not_initialized"
			ready = false
		} else if err := core.DB.Ping(); err != nil {
			checks["database"] = err.Error()
			ready = false
		} else {
			checks["database"] = "ok"
		}

		storagePath := ""
		if config != nil {
			storagePath = config.Storage.LocalPath
		}
		if err := core.StorageReady(storagePath); err != nil {
			checks["storage"] = err.Error()
			ready = false
		} else {
			checks["storage"] = "ok"
		}

		w.Header().Set("Content-Type", "application/json")
		if !ready {
			w.WriteHeader(http.StatusServiceUnavailable)
			if r.Method != http.MethodHead {
				fmt.Fprintf(w, `{"status":"not_ready","checks":%s}`, jsonChecks(checks))
			}
			return
		}
		w.WriteHeader(http.StatusOK)
		if r.Method != http.MethodHead {
			fmt.Fprintf(w, `{"status":"ready","checks":%s}`, jsonChecks(checks))
		}
	}
}

func handleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	uptime := time.Since(processStartedAt).Seconds()
	lines := []string{
		"# HELP qacapsule_up QA Capsule process is running.",
		"# TYPE qacapsule_up gauge",
		"qacapsule_up 1",
		"# HELP qacapsule_uptime_seconds Process uptime in seconds.",
		"# TYPE qacapsule_uptime_seconds gauge",
		fmt.Sprintf("qacapsule_uptime_seconds %.0f", uptime),
		"# HELP qacapsule_ingest_queue_depth Ingest jobs waiting in buffer.",
		"# TYPE qacapsule_ingest_queue_depth gauge",
		fmt.Sprintf("qacapsule_ingest_queue_depth %d", core.IngestQueueDepth()),
		"# HELP qacapsule_ingest_processed_total Ingest batches processed by workers.",
		"# TYPE qacapsule_ingest_processed_total counter",
		fmt.Sprintf("qacapsule_ingest_processed_total %d", core.IngestProcessedTotal()),
		"# HELP qacapsule_ingest_dropped_total Ingest batches rejected (queue full).",
		"# TYPE qacapsule_ingest_dropped_total counter",
		fmt.Sprintf("qacapsule_ingest_dropped_total %d", core.IngestDroppedTotal()),
	}
	_, _ = w.Write([]byte(strings.Join(lines, "\n") + "\n"))
}

func jsonChecks(m map[string]string) string {
	var b strings.Builder
	b.WriteString("{")
	first := true
	for k, v := range m {
		if !first {
			b.WriteString(",")
		}
		first = false
		b.WriteString(fmt.Sprintf("%q:%q", k, v))
	}
	b.WriteString("}")
	return b.String()
}
