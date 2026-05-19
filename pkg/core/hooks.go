package core

import "net/http"

var (
	IsEnterpriseActive = func() bool { return false }
	AdvancedFinOpsHandler http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusPaymentRequired) // Code 402
		w.Write([]byte(`{"error": "Enterprise license required"}`))
	}
)