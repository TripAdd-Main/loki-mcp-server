package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mark3labs/mcp-go/server"
)

// TestHealthz covers the monitoring contract: a probe gets 200 and a JSON body
// naming the running version, without any Loki call.
func TestHealthz(t *testing.T) {
	mux := newHTTPHandler(http.NotFoundHandler())

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, healthzPath, nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type %q, want application/json", got)
	}

	var body healthResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decoding body: %v", err)
	}
	if body.Status != "ok" {
		t.Errorf("status %q, want ok", body.Status)
	}
	if body.Version != version {
		t.Errorf("version %q, want %q", body.Version, version)
	}
}

// TestHealthzMethods: probes that use HEAD must work too, and anything that
// writes is rejected rather than silently treated as a probe.
func TestHealthzMethods(t *testing.T) {
	mux := newHTTPHandler(http.NotFoundHandler())

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

// TestHTTPHandlerRouting guards the wiring main() builds: /healthz must not
// shadow the MCP endpoint, and unknown paths stay 404.
func TestHTTPHandlerRouting(t *testing.T) {
	ts := httptest.NewServer(newHTTPHandler(server.NewStreamableHTTPServer(
		newMCPServer(stubClient{}),
		server.WithStateLess(true),
	)))
	defer ts.Close()

	get := func(path string) (int, []byte) {
		t.Helper()
		resp, err := http.Get(ts.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = resp.Body.Close() }()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		return resp.StatusCode, body
	}

	if code, _ := get(healthzPath); code != http.StatusOK {
		t.Errorf("%s: status %d, want %d", healthzPath, code, http.StatusOK)
	}
	if code, _ := get("/nope"); code != http.StatusNotFound {
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
			Tools []any `json:"tools"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if got := len(out.Result.Tools); got != 5 {
		t.Errorf("got %d tools, want 5", got)
	}
}
