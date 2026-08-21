package tools

import (
	"context"

	"github.com/incu6us/loki-mcp-server/internal/loki"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func NewSeriesTool(client loki.Client) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("series",
		mcp.WithDescription(`List the log streams matching a selector, as complete label sets.

Use it to see which concrete streams a selector actually covers and what other labels they carry: {app="nginx"} may expand to one series per pod, region or level. That answers "does this selector match anything" and "how many streams am I about to read" without fetching log lines. For the lines themselves, use query_range; for label names and values in isolation, use labels and label_values.

Returns the raw Loki JSON response: {"status","data":[{"label":"value"}]}, one object per stream. A selector matching nothing comes back as an empty list. Read-only: it never writes to or mutates Loki.`),
		mcp.WithString("match", mcp.Required(), mcp.Description(`Stream selector in LogQL brace syntax, e.g. {app="nginx"} or {namespace="prod",level=~"error|warn"}. Line filters (|=) are not accepted here.`)),
		mcp.WithString("start", mcp.Description("Start of the time range, RFC3339 (2026-03-25T10:00:00Z) or Unix nanoseconds. Defaults to 6 hours ago.")),
		mcp.WithString("end", mcp.Description("End of the time range, RFC3339 or Unix nanoseconds. Defaults to now.")),
		readOnlyTool("List Loki streams"),
	)

	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		match := extractString(req, "match")
		if match == "" {
			return mcp.NewToolResultError("match is required"), nil
		}

		series, err := client.Series(ctx, loki.SeriesParams{
			Match: []string{match},
			Start: extractString(req, "start"),
			End:   extractString(req, "end"),
		})
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return jsonResult(series)
	}

	return tool, handler
}
