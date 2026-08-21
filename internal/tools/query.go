package tools

import (
	"context"
	"fmt"

	"github.com/incu6us/loki-mcp-server/internal/loki"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func NewQueryTool(client loki.Client) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("query",
		mcp.WithDescription(`Run a LogQL instant query against Loki, evaluating the expression at a single point in time.

Use this for metric expressions (rate, count_over_time, sum by) when one value per series is enough, or for a quick "what is happening right now" check. To read log lines across a time window, use query_range instead. To discover which labels exist before writing a selector, use labels and label_values.

Returns the raw Loki JSON response: {"status","data":{"resultType","result"}}, where resultType is "vector" for metric expressions and "streams" for log selectors. Read-only: it never writes to or mutates Loki.`),
		mcp.WithString("query", mcp.Required(), mcp.Description(`LogQL expression. Metric example: sum(rate({app="nginx"} |= "error" [5m])). Log example: {app="nginx"} |= "error".`)),
		mcp.WithNumber("limit", mcp.Description("Maximum log entries to return. Applies to log selectors only; metric expressions ignore it. Defaults to 100, must not exceed 5000.")),
		mcp.WithString("time", mcp.Description("Evaluation timestamp, RFC3339 (2026-03-25T10:00:00Z) or Unix nanoseconds. Defaults to now.")),
		mcp.WithString("direction", mcp.Enum("forward", "backward"), mcp.Description("Order of returned log entries: backward (newest first, the default) or forward (oldest first).")),
		readOnlyTool("Loki instant query"),
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

		resp, err := client.Query(ctx, loki.QueryParams{
			Query:     query,
			Limit:     limit,
			Time:      extractString(req, "time"),
			Direction: direction,
		})
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return jsonResult(resp)
	}

	return tool, handler
}
