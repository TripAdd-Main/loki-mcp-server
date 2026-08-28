package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// TestHealthz covers the monitoring contract: a probe gets 200 and a JSON body
// saying so, without any Loki call.
func TestHealthz(t *testing.T) {
	mux := newHandler(http.NotFoundHandler())

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, healthzPath, nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type %q, want application/json", got)
	}

	var body struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decoding body: %v", err)
	}
	if body.Status != "ok" {
		t.Errorf("status %q, want ok", body.Status)
	}
}

// TestHealthzMethods: probes that use HEAD must work too, and anything that
// writes is rejected rather than silently treated as a probe.
func TestHealthzMethods(t *testing.T) {
	mux := newHandler(http.NotFoundHandler())

	for _, tc := range []struct {
		method string
		want   int
	}{
		{http.MethodHead, http.StatusOK},
		{http.MethodPost, http.StatusMethodNotAllowed},
		{http.MethodDelete, http.StatusMethodNotAllowed},
	} {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(tc.method, healthzPath, nil))
		if rec.Code != tc.want {
			t.Errorf("%s: status %d, want %d", tc.method, rec.Code, tc.want)
		}
	}
}

// TestStartHTTPListenError: a listen address that cannot be bound has to surface
// as an error rather than a silent no-op, since main() exits on it.
func TestStartHTTPListenError(t *testing.T) {
	occupied, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = occupied.Close() }()

	if err := StartHTTP(server.NewMCPServer("test", "1.2.3"), occupied.Addr().String()); err == nil {
		t.Error("StartHTTP on an occupied port returned nil, want an error")
	}
}

// TestHandlerRouting guards the wiring StartHTTP builds: /healthz must not
// shadow the MCP endpoint, and unknown paths stay 404.
func TestHandlerRouting(t *testing.T) {
	mcpServer := server.NewMCPServer("test", "1.2.3")
	mcpServer.AddTool(
		mcp.NewTool("ping"),
		func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return mcp.NewToolResultText("pong"), nil
		},
	)

	ts := httptest.NewServer(newHandler(
		server.NewStreamableHTTPServer(mcpServer, server.WithStateLess(true)),
	))
	defer ts.Close()

	get := func(path string) int {
		t.Helper()
		resp, err := http.Get(ts.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = resp.Body.Close() }()
		if _, err := io.Copy(io.Discard, resp.Body); err != nil {
			t.Fatal(err)
		}
		return resp.StatusCode
	}

	if code := get(healthzPath); code != http.StatusOK {
		t.Errorf("%s: status %d, want %d", healthzPath, code, http.StatusOK)
	}
	if code := get("/nope"); code != http.StatusNotFound {
		t.Errorf("/nope: status %d, want %d", code, http.StatusNotFound)
	}

	req, err := http.NewRequest(http.MethodPost, ts.URL+mcpPath, bytes.NewBufferString(
		`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("%s: status %d, want %d", mcpPath, resp.StatusCode, http.StatusOK)
	}

	var out struct {
		Result struct {
			Tools []struct {
				Name string `json:"name"`
			} `json:"tools"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if len(out.Result.Tools) != 1 || out.Result.Tools[0].Name != "ping" {
		t.Errorf("%s: tools %+v, want [ping]", mcpPath, out.Result.Tools)
	}
}
