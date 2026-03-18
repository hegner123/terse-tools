// Package openai adapts terse-tools for the OpenAI Chat Completions API.
//
// It converts tool definitions to OpenAI SDK types, parses tool call responses,
// and executes tools through a ToolRunner middleware chain.
package openai

import (
	"context"
	"encoding/json"

	tools "github.com/hegner123/terse-tools"

	"github.com/openai/openai-go"
)

// Adapter bridges terse-tools with the OpenAI Go SDK.
type Adapter struct {
	runner *tools.ToolRunner
}

// NewAdapter creates an Adapter that executes tools through the given ToolRunner.
func NewAdapter(runner *tools.ToolRunner) *Adapter {
	return &Adapter{runner: runner}
}

// Tools converts all registered tool definitions in the registry to
// OpenAI ChatCompletionToolParam values suitable for ChatCompletionNewParams.Tools.
func (a *Adapter) Tools(registry *tools.Registry) []openai.ChatCompletionToolParam {
	defs := registry.Defs()
	result := make([]openai.ChatCompletionToolParam, len(defs))
	for i, def := range defs {
		result[i] = ConvertToolDef(def)
	}
	return result
}

// Run executes a single tool call and returns the result as a
// ChatCompletionMessageParamUnion ready to append to messages.
// OpenAI has no IsError field, so errors are prefixed with "[ERROR] " in content.
func (a *Adapter) Run(ctx context.Context, tc openai.ChatCompletionMessageToolCall) openai.ChatCompletionMessageParamUnion {
	call := ParseToolCall(tc)
	result := a.runner.Run(ctx, call)

	content := result.Content
	if result.IsError {
		content = "[ERROR] " + content
	}

	return openai.ToolMessage(content, result.ID)
}

// RunAll executes multiple tool calls and returns all results.
func (a *Adapter) RunAll(ctx context.Context, tcs []openai.ChatCompletionMessageToolCall) []openai.ChatCompletionMessageParamUnion {
	results := make([]openai.ChatCompletionMessageParamUnion, len(tcs))
	for i, tc := range tcs {
		results[i] = a.Run(ctx, tc)
	}
	return results
}

// ConvertToolDef converts a terse-tools ToolDef to an OpenAI ChatCompletionToolParam.
// The InputSchema map is passed directly as FunctionParameters since they share
// the same underlying type (map[string]any).
func ConvertToolDef(def tools.ToolDef) openai.ChatCompletionToolParam {
	return openai.ChatCompletionToolParam{
		Function: openai.FunctionDefinitionParam{
			Name:        def.Name,
			Description: openai.String(def.Description),
			Parameters:  openai.FunctionParameters(def.InputSchema),
		},
	}
}

// ParseToolCall converts an OpenAI tool call response to a provider-agnostic ToolCall.
func ParseToolCall(tc openai.ChatCompletionMessageToolCall) tools.ToolCall {
	return tools.ToolCall{
		ID:    tc.ID,
		Name:  tc.Function.Name,
		Input: json.RawMessage(tc.Function.Arguments),
	}
}
