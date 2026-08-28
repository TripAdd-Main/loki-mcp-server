# Loki MCP Server

[![CI](https://github.com/incu6us/loki-mcp-server/actions/workflows/ci.yml/badge.svg)](https://github.com/incu6us/loki-mcp-server/actions/workflows/ci.yml)
[![Release](https://github.com/incu6us/loki-mcp-server/actions/workflows/release.yml/badge.svg)](https://github.com/incu6us/loki-mcp-server/actions/workflows/release.yml)
[![codecov](https://codecov.io/gh/incu6us/loki-mcp-server/branch/master/graph/badge.svg)](https://codecov.io/gh/incu6us/loki-mcp-server)
[![Go Report Card](https://goreportcard.com/badge/github.com/incu6us/loki-mcp-server)](https://goreportcard.com/report/github.com/incu6us/loki-mcp-server)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
![Stars](https://img.shields.io/github/stars/incu6us/loki-mcp-server)
![MCP](https://img.shields.io/badge/MCP-compatible-green)

[![loki-mcp-server MCP server](https://glama.ai/mcp/servers/incu6us/loki-mcp-server/badges/card.svg)](https://glama.ai/mcp/servers/incu6us/loki-mcp-server)

Query Grafana Loki logs directly from AI agents using the [Model Context Protocol](https://modelcontextprotocol.io/) (MCP).

Built in Go. Enables AI-powered log analysis using LogQL.

Supports integration with:
- Claude Desktop
- AI agent frameworks
- automation tools
- DevOps workflows

## Motivation

The official [grafana/loki-mcp](https://github.com/grafana/loki-mcp) exposes a single `loki_query` tool, which means the LLM must already know valid label names and values before it can build a query. This project takes a different approach by providing **5 granular tools** — `labels`, `label_values`, and `series` let the LLM discover what's available in Loki first, then construct precise `query_range` or `query` calls. The result is more accurate log retrieval with fewer wasted round-trips.

Additionally, this server enforces **strict input validation** (limit caps, direction validation, label name format checks, mutually exclusive auth) to surface errors early instead of forwarding bad requests to Loki.

## Features

- **query_range** — Execute LogQL range queries to fetch logs over a time window
- **query** — Execute LogQL instant queries for point-in-time evaluation
- **labels** — List all available label names
- **label_values** — List values for a specific label
- **series** — Find active log stream series matching a selector

## Installation

### Homebrew

```bash
brew install incu6us/tap/loki-mcp-server
```

### Go install

```bash
go install github.com/incu6us/loki-mcp-server/cmd/loki-mcp-server@latest
```

Or build from source:

```bash
git clone https://github.com/incu6us/loki-mcp-server.git
cd loki-mcp
go build -o loki-mcp-server ./cmd/loki-mcp-server
```

## Configuration

The server is configured entirely via environment variables, injected by the MCP client.

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `LOKI_URL` | yes | — | Base URL of the Loki instance |
| `LOKI_USERNAME` | no | — | Basic auth username |
| `LOKI_PASSWORD` | no | — | Basic auth password |
| `LOKI_BEARER_TOKEN` | no | — | Bearer token authentication |
| `LOKI_TLS_SKIP_VERIFY` | no | `false` | Skip TLS certificate verification |
| `LOKI_TENANT_ID` | no | — | `X-Scope-OrgID` header for multi-tenant deployments |
| `LOKI_HTTP_TIMEOUT` | no | `30s` | HTTP request timeout (Go duration, e.g. `10s`, `1m`) |
| `MCP_HTTP_ADDR` | no | — | Listen address for the streamable HTTP transport and `/healthz`, e.g. `:8080`. Unset means stdio |

> **Note:** Basic auth (`LOKI_USERNAME`/`LOKI_PASSWORD`) and bearer token (`LOKI_BEARER_TOKEN`) are mutually exclusive.

## Transports

By default the server speaks MCP over stdio, which is what Claude Code, Claude Desktop
and most local clients expect.

Set `MCP_HTTP_ADDR` to serve the streamable HTTP transport instead, for running the
server as a remote endpoint behind a proxy or gateway:

```bash
LOKI_URL=http://loki:3100 MCP_HTTP_ADDR=:8080 loki-mcp-server
# MCP endpoint:    http://localhost:8080/mcp
# Health endpoint: http://localhost:8080/healthz
```

The HTTP mode is stateless, so it can run behind a load balancer with several replicas.
It carries no authentication of its own — put it behind TLS and an authenticating proxy
before exposing it, and remember that whoever reaches the endpoint can read every log
line the configured `LOKI_URL` credentials can see.

### Health check

`GET /healthz` (also `HEAD`; other methods get `405`) is served on the same listener
whenever `MCP_HTTP_ADDR` is set, and answers `200` with:

```json
{"status":"ok","version":"1.2.3"}
```

This is a **liveness** check: it reports that the process is up and serving, and
deliberately does not query Loki. A probe that went red during a Loki outage would have
your orchestrator restart a healthy process over an upstream problem, and every monitor
polling it would add load to Loki. Whether Loki is reachable is what the tools report.

Like `/mcp`, it is unauthenticated and exposes the server version, so keep it behind the
same proxy.

Kubernetes:

```yaml
livenessProbe:
  httpGet:
    path: /healthz
    port: 8080
readinessProbe:
  httpGet:
    path: /healthz
    port: 8080
```

Docker Compose:

```yaml
healthcheck:
  test: ["CMD-SHELL", "wget -q --spider http://localhost:8080/healthz || exit 1"]
  interval: 10s
  timeout: 3s
  retries: 3
```

## Usage with Claude Code

Add to your Claude Code MCP configuration (`~/.claude.json`):

```json
{
  "mcpServers": {
    "loki-mcp-server": {
      "type": "stdio",
      "command": "/path/to/loki-mcp-server",
      "args": [],
      "env": {
        "LOKI_URL": "http://loki:3100",
        "LOKI_USERNAME": "admin",
        "LOKI_PASSWORD": "secret"
      }
    }
  }
}
```

## Usage with Claude Desktop

Add to your Claude Desktop config (`~/Library/Application Support/Claude/claude_desktop_config.json` on macOS):

```json
{
  "mcpServers": {
    "loki-mcp-server": {
      "type": "stdio",
      "command": "/path/to/loki-mcp-server",
      "args": [],
      "env": {
        "LOKI_URL": "http://loki:3100",
        "LOKI_USERNAME": "admin",
        "LOKI_PASSWORD": "secret"
      }
    }
  }
}
```

## Tools

### query_range

Execute a LogQL range query against Loki to fetch logs over a time window.

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `query` | string | yes | — | LogQL query expression |
| `start` | string | no | 1 hour ago | Start of time range (RFC3339 or Unix nano) |
| `end` | string | no | now | End of time range |
| `limit` | number | no | 100 | Max entries (max 5000) |
| `direction` | string | no | backward | `forward` or `backward` |

### query

Execute a LogQL instant query for point-in-time evaluation.

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `query` | string | yes | — | LogQL query expression |
| `limit` | number | no | 100 | Max entries (max 5000) |
| `time` | string | no | now | Evaluation timestamp |
| `direction` | string | no | backward | `forward` or `backward` |

### labels

List all available label names in Loki.

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `start` | string | no | 6 hours ago | Start of time range |
| `end` | string | no | now | End of time range |

### label_values

List values for a specific label.

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `label` | string | yes | — | Label name |
| `start` | string | no | 6 hours ago | Start of time range |
| `end` | string | no | now | End of time range |

### series

Find active log stream series matching a selector.

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `match` | string | yes | — | Stream selector (e.g. `{app="nginx"}`) |
| `start` | string | no | 6 hours ago | Start of time range |
| `end` | string | no | now | End of time range |

## Local Development Stack

A Docker Compose setup is included under `deploy/` to spin up a full Loki environment for testing:

- **Loki** — log storage at `http://localhost:3100`
- **Grafana** — UI at `http://localhost:3000` (anonymous admin, Loki pre-configured as datasource)
- **Promtail** — collects container logs and ships them to Loki
- **Log generator** — emits structured JSON logs with randomized apps (`nginx`, `api`, `gateway`, `auth`, `payments`), levels, and messages

```bash
# Start the stack
docker compose -f deploy/docker-compose.yml up -d

# Use loki-mcp-server against local Loki
LOKI_URL=http://localhost:3100 loki-mcp-server

# Stop the stack
docker compose -f deploy/docker-compose.yml down
```

## Development

```bash
# Run tests
go test ./...

# Build
go build -o loki-mcp-server ./cmd/loki-mcp-server

# Vet
go vet ./...
```

⭐  If this project is useful for you, please star the repository.
