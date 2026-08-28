package main

import (
	"encoding/json"
	"net/http"
)

const (
	mcpPath     = "/mcp"
	healthzPath = "/healthz"
)

type healthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

// newHTTPHandler routes the MCP transport and the health endpoint onto the single
// listener MCP_HTTP_ADDR opens.
func newHTTPHandler(mcpHandler http.Handler) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle(mcpPath, mcpHandler)
	// ponytail: liveness only, deliberately no Loki round-trip. A probe that went
	// red whenever Loki was unreachable would have the orchestrator restart a
	// perfectly healthy process over someone else's outage, and every monitoring
	// system polling it would put load on Loki. Whether Loki answers is what the
	// tools report; whether this process is up is what /healthz reports.
	mux.HandleFunc("GET "+healthzPath, handleHealthz)
	return mux
}

// handleHealthz answers monitoring probes. The GET pattern above also serves HEAD,
// and leaves any other method to the mux's 405.
func handleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(healthResponse{Status: "ok", Version: version})
}
