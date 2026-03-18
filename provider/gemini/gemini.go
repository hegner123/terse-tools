// Package gemini adapts terse-tools for the Google Gemini (Generative AI) API.
//
// It converts tool definitions to Gemini SDK types, parses function call responses,
// and executes tools through a ToolRunner middleware chain.
//
// Gemini has no call IDs — correlation is by function name.
// The schema conversion recursively walks map[string]any into *genai.Schema.
package gemini

import (
	"context"
	"encoding/json"
	"fmt"

	tools "github.com/hegner123/terse-tools"

	"github.com/google/generative-ai-go/genai"
)

// Adapter bridges terse-tools with the Google Generative AI Go SDK.
type Adapter struct {
	runner *tools.ToolRunner
}

// NewAdapter creates an Adapter that executes tools through the given ToolRunner.
func NewAdapter(runner *tools.ToolRunner) *Adapter {
	return &Adapter{runner: runner}
}

// Tools converts all registered tool definitions in the registry to a single
// genai.Tool containing all function declarations. Pass this to
// GenerativeModel.Tools.
func (a *Adapter) Tools(registry *tools.Registry) *genai.Tool {
	defs := registry.Defs()
	declarations := make([]*genai.FunctionDeclaration, len(defs))
	for i, def := range defs {
		declarations[i] = ConvertToolDef(def)
	}
	return &genai.Tool{FunctionDeclarations: declarations}
}

// Run executes a single function call and returns the result as a
// genai.FunctionResponse. Returns an error if the function call's Args
// cannot be marshaled to JSON.
func (a *Adapter) Run(ctx context.Context, fc genai.FunctionCall) (genai.FunctionResponse, error) {
	call, err := ParseFunctionCall(fc)
	if err != nil {
		return genai.FunctionResponse{}, err
	}
	result := a.runner.Run(ctx, call)
	return genai.FunctionResponse{
		Name: result.Name,
		Response: map[string]any{
			"content":  result.Content,
			"is_error": result.IsError,
		},
	}, nil
}

// RunAll executes multiple function calls and returns all results.
// Returns an error if any function call's Args cannot be marshaled.
func (a *Adapter) RunAll(ctx context.Context, calls []genai.FunctionCall) ([]genai.FunctionResponse, error) {
	results := make([]genai.FunctionResponse, len(calls))
	for i, fc := range calls {
		resp, err := a.Run(ctx, fc)
		if err != nil {
			return nil, fmt.Errorf("function call %d (%s): %w", i, fc.Name, err)
		}
		results[i] = resp
	}
	return results, nil
}

// ConvertToolDef converts a terse-tools ToolDef to a Gemini FunctionDeclaration.
// The InputSchema is recursively converted to *genai.Schema.
func ConvertToolDef(def tools.ToolDef) *genai.FunctionDeclaration {
	return &genai.FunctionDeclaration{
		Name:        def.Name,
		Description: def.Description,
		Parameters:  convertSchema(def.InputSchema),
	}
}

// ParseFunctionCall converts a Gemini FunctionCall to a provider-agnostic ToolCall.
// Returns an error if Args cannot be marshaled to JSON.
// The ToolCall.ID will be empty since Gemini uses name-based correlation.
func ParseFunctionCall(fc genai.FunctionCall) (tools.ToolCall, error) {
	input, err := json.Marshal(fc.Args)
	if err != nil {
		return tools.ToolCall{}, fmt.Errorf("marshal function call args: %w", err)
	}
	return tools.ToolCall{
		Name:  fc.Name,
		Input: json.RawMessage(input),
	}, nil
}

// convertSchema recursively converts a JSON Schema map[string]any to a *genai.Schema.
func convertSchema(raw map[string]any) *genai.Schema {
	if raw == nil {
		return nil
	}

	schema := &genai.Schema{}

	if t, ok := raw["type"].(string); ok {
		switch t {
		case "string":
			schema.Type = genai.TypeString
		case "integer":
			schema.Type = genai.TypeInteger
		case "number":
			schema.Type = genai.TypeNumber
		case "boolean":
			schema.Type = genai.TypeBoolean
		case "array":
			schema.Type = genai.TypeArray
		case "object":
			schema.Type = genai.TypeObject
		}
	}

	if d, ok := raw["description"].(string); ok {
		schema.Description = d
	}

	if f, ok := raw["format"].(string); ok {
		schema.Format = f
	}

	if e, ok := raw["enum"].([]any); ok {
		for _, v := range e {
			if s, ok := v.(string); ok {
				schema.Enum = append(schema.Enum, s)
			}
		}
	}

	if items, ok := raw["items"].(map[string]any); ok {
		schema.Items = convertSchema(items)
	}

	if props, ok := raw["properties"].(map[string]any); ok {
		schema.Properties = make(map[string]*genai.Schema)
		for name, propRaw := range props {
			if propMap, ok := propRaw.(map[string]any); ok {
				schema.Properties[name] = convertSchema(propMap)
			}
		}
	}

	schema.Required = extractRequired(raw["required"])

	return schema
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
