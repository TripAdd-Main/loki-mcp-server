package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/incu6us/loki-mcp-server/internal/config"
	"github.com/incu6us/loki-mcp-server/internal/loki"
	"github.com/incu6us/loki-mcp-server/internal/tools"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

var version = "dev"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Println(version)
		return
	}

	logger := log.New(os.Stderr, "loki-mcp: ", log.LstdFlags)

	cfg, err := config.Load()
	if err != nil {
		logger.Fatalf("configuration error: %v", err)
	}

	s := newMCPServer(loki.NewClient(cfg))

	// ponytail: stdio stays the default. MCP_HTTP_ADDR opts into streamable HTTP so
	// the same binary can sit behind a gateway, which needs an HTTPS URL and this
	// transport. Stateless: no session state to lose across replicas.
	if addr := os.Getenv("MCP_HTTP_ADDR"); addr != "" {
		// ponytail: the transport still owns the listener, but the mux is ours, so
		// /healthz shares the one port MCP_HTTP_ADDR opens instead of needing a second.
		httpServer := &http.Server{}
		streamable := server.NewStreamableHTTPServer(s,
			server.WithStateLess(true),
			server.WithStreamableHTTPServer(httpServer),
		)
		httpServer.Handler = newHTTPHandler(streamable)

		logger.Printf("streamable http listening on %s (mcp: %s, health: %s)", addr, mcpPath, healthzPath)
		if err := streamable.Start(addr); err != nil {
			logger.Fatalf("server error: %v", err)
		}
		return
	}

	if err := server.ServeStdio(s); err != nil {
		logger.Fatalf("server error: %v", err)
	}
}

func newMCPServer(client loki.Client) *server.MCPServer {
	s := server.NewMCPServer(
		"loki-mcp-server",
		version,
		server.WithToolCapabilities(false),
	)

	for _, register := range []func(loki.Client) (mcp.Tool, server.ToolHandlerFunc){
		tools.NewQueryRangeTool,
		tools.NewQueryTool,
		tools.NewLabelsTool,
		tools.NewLabelValuesTool,
		tools.NewSeriesTool,
	} {
		tool, handler := register(client)
		s.AddTool(tool, handler)
	}

	return s
}
