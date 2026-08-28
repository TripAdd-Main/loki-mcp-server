package transport

import (
	"net/http"

	"github.com/mark3labs/mcp-go/server"
)

const (
	mcpPath     = "/mcp"
	healthzPath = "/healthz"
)

// StartHTTP serves the MCP streamable HTTP transport and the health endpoint on
// addr, blocking until the server stops.
func StartHTTP(mcpServer *server.MCPServer, addr string) error {
	// The transport owns the listener; the mux is ours so /healthz can share the port.
	httpServer := &http.Server{}
	streamable := server.NewStreamableHTTPServer(mcpServer,
		server.WithStateLess(true),
		server.WithStreamableHTTPServer(httpServer),
	)
	httpServer.Handler = newHandler(streamable)

	return streamable.Start(addr)
}

func newHandler(mcpHandler http.Handler) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle(mcpPath, mcpHandler)
	// GET also serves HEAD; other methods get the mux's 405.
	mux.HandleFunc("GET "+healthzPath, handleHealthz)
	return mux
}

// handleHealthz reports that this process is up. It deliberately does not reach
// Loki: a probe that went red during a Loki outage would restart a healthy process.
func handleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}
