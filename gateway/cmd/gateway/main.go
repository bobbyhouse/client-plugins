package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/bobbyhouse/plugins/gateway/internal/proxy"
)

var loadSchema = json.RawMessage(`{
	"type": "object",
	"required": ["profile"],
	"properties": {
		"profile": {
			"type": "string",
			"description": "OCI reference for the profile image"
		},
		"config": {
			"type": "object",
			"description": "Key/value pairs that satisfy ${KEY} placeholders in the profile's config values",
			"additionalProperties": {"type": "string"}
		}
	}
}`)

func main() {
	cache := proxy.NewCache()

	s := mcp.NewServer(&mcp.Implementation{Name: "profile-gateway", Version: "1.0.0"}, nil)
	s.AddTool(&mcp.Tool{
		Name:        "load",
		Description: "Load a profile and register its MCP server tools into this session.",
		InputSchema: loadSchema,
	}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var params struct {
			Profile string            `json:"profile"`
			Config  map[string]string `json:"config"`
		}
		if err := json.Unmarshal(req.Params.Arguments, &params); err != nil {
			return errorResult(fmt.Errorf("invalid arguments: %w", err)), nil
		}

		session, err := cache.GetOrLoad(ctx, params.Profile, params.Config)
		if err != nil {
			return errorResult(err), nil
		}

		for serverName, cs := range session.Clients() {
			result, err := cs.ListTools(ctx, nil)
			if err != nil {
				return errorResult(fmt.Errorf("server %s: list tools: %w", serverName, err)), nil
			}
			for _, t := range result.Tools {
				inputSchema := t.InputSchema
				if inputSchema == nil {
					inputSchema = json.RawMessage(`{"type":"object"}`)
				}
				tool := &mcp.Tool{
					Name:        serverName + "__" + t.Name,
					Description: t.Description,
					InputSchema: inputSchema,
				}
				cs := cs
				bareName := t.Name
				s.AddTool(tool, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
					return cs.CallTool(ctx, &mcp.CallToolParams{
						Name:      bareName,
						Arguments: req.Params.Arguments,
					})
				})
			}
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: "profile loaded"}},
		}, nil
	})

	if err := s.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatal(err)
	}
}

func errorResult(err error) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
	}
}
