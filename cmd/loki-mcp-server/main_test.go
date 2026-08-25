package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/incu6us/loki-mcp-server/internal/loki"
	"github.com/mark3labs/mcp-go/server"
)

type stubClient struct{}

func (stubClient) QueryRange(context.Context, loki.QueryRangeParams) (*loki.QueryResponse, error) {
	return &loki.QueryResponse{}, nil
}
func (stubClient) Query(context.Context, loki.QueryParams) (*loki.QueryResponse, error) {
	return &loki.QueryResponse{}, nil
}
func (stubClient) Labels(context.Context, loki.LabelsParams) ([]string, error) { return nil, nil }
func (stubClient) LabelValues(context.Context, string, loki.LabelsParams) ([]string, error) {
	return nil, nil
}
func (stubClient) Series(context.Context, loki.SeriesParams) ([]map[string]string, error) {
	return nil, nil
}

// TestStreamableHTTP covers the MCP_HTTP_ADDR transport: a gateway must be able
// to initialize and list tools over plain HTTP POST.
func TestStreamableHTTP(t *testing.T) {
	ts := server.NewTestStreamableHTTPServer(newMCPServer(stubClient{}), server.WithStateLess(true))
	defer ts.Close()

	post := func(body string) map[string]any {
		t.Helper()
		req, err := http.NewRequest(http.MethodPost, ts.URL, bytes.NewBufferString(body))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/event-stream")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status %d", resp.StatusCode)
		}

		var out map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			t.Fatal(err)
		}
		return out
	}

	post(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"t","version":"1"}}}`)

	out := post(`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`)
	result, ok := out["result"].(map[string]any)
	if !ok {
		t.Fatalf("no result: %v", out)
	}
	if got := len(result["tools"].([]any)); got != 5 {
		t.Errorf("got %d tools, want 5", got)
	}
}
