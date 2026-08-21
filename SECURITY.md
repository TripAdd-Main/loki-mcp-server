# Security Policy

## Supported Versions

Only the latest release is supported. Fixes land on `master` and ship in the next tag.

## Reporting a Vulnerability

Report privately via [GitHub Security Advisories](https://github.com/incu6us/loki-mcp-server/security/advisories/new).
Please do not open a public issue for security problems.

Expect an initial response within 7 days.

## Scope

This server only reads from the Loki API endpoint you configure (`LOKI_URL`) and speaks MCP over stdio.
Credentials are read from the environment and are never logged.
