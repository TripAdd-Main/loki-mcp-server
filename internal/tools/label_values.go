package tools

import (
	"context"

	"github.com/incu6us/loki-mcp-server/internal/loki"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func NewLabelValuesTool(client loki.Client) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("label_values",
		mcp.WithDescription(`List the values a single label takes in Loki over a time window.

Use it after labels to fill in a selector: labels says "app" exists, label_values with label="app" says which apps are actually logging, and one of those goes into {app="..."} for query_range. Only values seen inside the window are returned, so widening start/end surfaces more.

Returns the raw Loki JSON response: {"status","data":["value","list"]}. An unknown label name is not an error; it comes back as an empty list. Read-only: it never writes to or mutates Loki.`),
		mcp.WithString("label", mcp.Required(), mcp.Pattern(`^[a-zA-Z_][a-zA-Z0-9_]*$`), mcp.Description(`Label name to look up, as returned by the labels tool (e.g. app, namespace, level). Must match ^[a-zA-Z_][a-zA-Z0-9_]*$.`)),
		mcp.WithString("start", mcp.Description("Start of the time range, RFC3339 (2026-03-25T10:00:00Z) or Unix nanoseconds. Defaults to 6 hours ago.")),
		mcp.WithString("end", mcp.Description("End of the time range, RFC3339 or Unix nanoseconds. Defaults to now.")),
		readOnlyTool("List Loki label values"),
	)

	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		label := extractString(req, "label")
		if label == "" {
			return mcp.NewToolResultError("label is required"), nil
		}
		if !validLabelName.MatchString(label) {
			return mcp.NewToolResultError("label must match [a-zA-Z_][a-zA-Z0-9_]*"), nil
		}

		values, err := client.LabelValues(ctx, label, loki.LabelsParams{
			Start: extractString(req, "start"),
			End:   extractString(req, "end"),
		})
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return jsonResult(values)
	}

	return tool, handler
}
