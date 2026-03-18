// Package anthropic adapts terse-tools for the Anthropic Messages API.
//
// It converts tool definitions to Anthropic SDK types, parses tool_use blocks,
// and executes tools through a ToolRunner middleware chain.
package anthropic

import (
	"context"
	"encoding/json"

	tools "github.com/hegner123/terse-tools"

	"github.com/anthropics/anthropic-sdk-go"
)

// Adapter bridges terse-tools with the Anthropic Go SDK.
type Adapter struct {
	runner *tools.ToolRunner
}

// NewAdapter creates an Adapter that executes tools through the given ToolRunner.
func NewAdapter(runner *tools.ToolRunner) *Adapter {
	return &Adapter{runner: runner}
}

// Tools converts all registered tool definitions in the registry to
// Anthropic ToolUnionParam values suitable for MessageNewParams.Tools.
func (a *Adapter) Tools(registry *tools.Registry) []anthropic.ToolUnionParam {
	defs := registry.Defs()
	result := make([]anthropic.ToolUnionParam, len(defs))
	for i, def := range defs {
		result[i] = ConvertToolDef(def)
	}
	return result
}

// Run executes a single tool_use block and returns the result as a
// ContentBlockParamUnion ready to include in a user message.
func (a *Adapter) Run(ctx context.Context, block anthropic.ToolUseBlock) anthropic.ContentBlockParamUnion {
	call := ParseToolUse(block)
	result := a.runner.Run(ctx, call)
	return anthropic.NewToolResultBlock(result.ID, result.Content, result.IsError)
}

// RunAll executes multiple tool_use blocks and returns all results.
func (a *Adapter) RunAll(ctx context.Context, blocks []anthropic.ToolUseBlock) []anthropic.ContentBlockParamUnion {
	results := make([]anthropic.ContentBlockParamUnion, len(blocks))
	for i, block := range blocks {
		results[i] = a.Run(ctx, block)
	}
	return results
}

// ConvertToolDef converts a terse-tools ToolDef to an Anthropic ToolUnionParam.
func ConvertToolDef(def tools.ToolDef) anthropic.ToolUnionParam {
	var properties any
	if p, ok := def.InputSchema["properties"]; ok {
		properties = p
	}

	required := extractRequired(def.InputSchema["required"])

	return anthropic.ToolUnionParam{
		OfTool: &anthropic.ToolParam{
			Name:        def.Name,
			Description: anthropic.String(def.Description),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: properties,
				Required:   required,
			},
		},
	}
}

// ParseToolUse converts an Anthropic ToolUseBlock to a provider-agnostic ToolCall.
func ParseToolUse(block anthropic.ToolUseBlock) tools.ToolCall {
	return tools.ToolCall{
		ID:    block.ID,
		Name:  block.Name,
		Input: json.RawMessage(block.Input),
	}
}

// extractRequired handles both []string and []any for the "required" schema field.
func extractRequired(v any) []string {
	switch req := v.(type) {
	case []string:
		return req
	case []any:
		result := make([]string, 0, len(req))
		for _, item := range req {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	default:
		return nil
	}
}
