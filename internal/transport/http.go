package transport

import (
	"encoding/json"
	"net/http"

	"github.com/mark3labs/mcp-go/server"
)

const (
	mcpPath     = "/mcp"
	healthzPath = "/healthz"
)

type healthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

// StartHTTP serves the MCP streamable HTTP transport on addr and blocks until the
// server stops.
func StartHTTP(mcpServer *server.MCPServer, addr, version string) error {
	// ponytail: the transport still owns the listener, but the mux is ours, so
	// /healthz shares the one port instead of needing a second one.
	httpServer := &http.Server{}
	streamable := server.NewStreamableHTTPServer(mcpServer,
		server.WithStateLess(true),
		server.WithStreamableHTTPServer(httpServer),
	)
	httpServer.Handler = newHandler(streamable, version)

	return streamable.Start(addr)
}

// newHandler routes the MCP transport and the health endpoint onto the single
// listener StartHTTP opens.
func newHandler(mcpHandler http.Handler, version string) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle(mcpPath, mcpHandler)
	// ponytail: liveness only, deliberately no Loki round-trip. A probe that went
	// red whenever Loki was unreachable would have the orchestrator restart a
	// perfectly healthy process over someone else's outage, and every monitoring
	// system polling it would put load on Loki. Whether Loki answers is what the
	// tools report; whether this process is up is what /healthz reports.
	mux.HandleFunc("GET "+healthzPath, healthzHandler(version))
	return mux
}

// healthzHandler answers monitoring probes. The GET pattern above also serves HEAD,
// and leaves any other method to the mux's 405.
func healthzHandler(version string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(healthResponse{Status: "ok", Version: version})
	}
}
