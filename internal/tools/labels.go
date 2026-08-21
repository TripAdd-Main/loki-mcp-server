package tools

import (
	"context"

	"github.com/incu6us/loki-mcp-server/internal/loki"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func NewLabelsTool(client loki.Client) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("labels",
		mcp.WithDescription(`List the label names Loki knows about in a time window.

Start here when the log schema is unknown: labels gives the names (app, namespace, level), then label_values gives the values for one of them, and together they let you write a valid selector for query_range. Only labels present on streams that received data inside the window are returned, so widening start/end surfaces more.

Returns the raw Loki JSON response: {"status","data":["label","names"]}. Read-only: it never writes to or mutates Loki.`),
		mcp.WithString("start", mcp.Description("Start of the time range, RFC3339 (2026-03-25T10:00:00Z) or Unix nanoseconds. Defaults to 6 hours ago.")),
		mcp.WithString("end", mcp.Description("End of the time range, RFC3339 or Unix nanoseconds. Defaults to now.")),
		readOnlyTool("List Loki label names"),
	)

	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		labels, err := client.Labels(ctx, loki.LabelsParams{
			Start: extractString(req, "start"),
			End:   extractString(req, "end"),
		})
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return jsonResult(labels)
	}

	return tool, handler
}
