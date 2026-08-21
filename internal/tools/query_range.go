package tools

import (
	"context"
	"fmt"

	"github.com/incu6us/loki-mcp-server/internal/loki"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func NewQueryRangeTool(client loki.Client) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("query_range",
		mcp.WithDescription(`Run a LogQL range query against Loki, returning results across a time window.

This is the tool to reach for when reading logs: searching for errors, tailing a service over the last hour, or graphing a metric expression over time. Use query instead when a single point-in-time value is enough. If the selector is unknown, call labels and label_values first.

Returns the raw Loki JSON response: {"status","data":{"resultType","result"}}, where resultType is "streams" for log selectors and "matrix" for metric expressions. Loki truncates at limit entries, so a full result set may mean logs were cut off; narrow start/end or tighten the selector rather than raising limit. Read-only: it never writes to or mutates Loki.`),
		mcp.WithString("query", mcp.Required(), mcp.Description(`LogQL expression. Log example: {app="nginx"} |= "error" | json. Metric example: sum by (app) (rate({app="nginx"}[5m])).`)),
		mcp.WithString("start", mcp.Description("Start of the time range, RFC3339 (2026-03-25T10:00:00Z) or Unix nanoseconds. Defaults to 1 hour ago.")),
		mcp.WithString("end", mcp.Description("End of the time range, RFC3339 or Unix nanoseconds. Defaults to now.")),
		mcp.WithNumber("limit", mcp.Description("Maximum log entries to return. Applies to log selectors only; metric expressions ignore it. Defaults to 100, must not exceed 5000.")),
		mcp.WithString("direction", mcp.Enum("forward", "backward"), mcp.Description("Order of returned log entries: backward (newest first, the default) or forward (oldest first). With backward and a hit limit, you keep the newest entries.")),
		readOnlyTool("Loki range query"),
	)

	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		query := extractString(req, "query")
		if query == "" {
			return mcp.NewToolResultError("query is required"), nil
		}

		direction := extractString(req, "direction")
		if err := validateDirection(direction); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		limit := extractLimit(req)
		if limit > MaxLimit {
			return mcp.NewToolResultError(fmt.Sprintf("limit must not exceed %d", MaxLimit)), nil
		}

		resp, err := client.QueryRange(ctx, loki.QueryRangeParams{
			Query:     query,
			Start:     extractString(req, "start"),
			End:       extractString(req, "end"),
			Limit:     limit,
			Direction: direction,
		})
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return jsonResult(resp)
	}

	return tool, handler
}
