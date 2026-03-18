package openai_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	tools "github.com/hegner123/terse-tools"
	adapter "github.com/hegner123/terse-tools/provider/openai"

	"github.com/openai/openai-go"
)

func testRegistry() *tools.Registry {
	reg := tools.NewRegistry()
	reg.Register(tools.ToolDef{
		Name:        "echo",
		Description: "Returns the input message",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"message": map[string]any{
					"type":        "string",
					"description": "The message to echo",
				},
			},
			"required": []string{"message"},
		},
		Builtin: func(ctx context.Context, input map[string]any, workDir string) tools.Result {
			msg, _ := input["message"].(string)
			data, _ := json.Marshal(map[string]string{"echo": msg})
			return tools.Result{Output: string(data)}
		},
	})
	reg.Register(tools.ToolDef{
		Name:        "fail",
		Description: "Always fails",
		InputSchema: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
		Builtin: func(ctx context.Context, input map[string]any, workDir string) tools.Result {
			return tools.Result{IsError: true, Error: "intentional failure"}
		},
	})
	return reg
}

func TestConvertToolDef(t *testing.T) {
	def := tools.ToolDef{
		Name:        "test_tool",
		Description: "A test tool",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "File path",
				},
			},
			"required": []string{"path"},
		},
	}

	result := adapter.ConvertToolDef(def)

	if result.Function.Name != "test_tool" {
		t.Errorf("Name = %q, want %q", result.Function.Name, "test_tool")
	}

	// Verify the parameters contain the schema
	params := result.Function.Parameters
	if params == nil {
		t.Fatal("Parameters is nil")
	}
	if params["type"] != "object" {
		t.Errorf("Parameters[type] = %v, want %q", params["type"], "object")
	}
}

func TestParseToolCall(t *testing.T) {
	tc := openai.ChatCompletionMessageToolCall{
		ID: "call_abc123",
		Function: openai.ChatCompletionMessageToolCallFunction{
			Name:      "echo",
			Arguments: `{"message":"hello"}`,
		},
	}

	call := adapter.ParseToolCall(tc)

	if call.ID != "call_abc123" {
		t.Errorf("ID = %q, want %q", call.ID, "call_abc123")
	}
	if call.Name != "echo" {
		t.Errorf("Name = %q, want %q", call.Name, "echo")
	}

	var input map[string]string
	if err := json.Unmarshal(call.Input, &input); err != nil {
		t.Fatalf("unmarshal input: %v", err)
	}
	if input["message"] != "hello" {
		t.Errorf("input[message] = %q, want %q", input["message"], "hello")
	}
}

func TestAdapter_Tools(t *testing.T) {
	reg := testRegistry()
	exec := tools.NewExecutor(reg, t.TempDir())
	runner := tools.NewToolRunner(exec)
	a := adapter.NewAdapter(runner)

	toolParams := a.Tools(reg)
	if len(toolParams) != 2 {
		t.Fatalf("Tools() returned %d params, want 2", len(toolParams))
	}
}

func TestAdapter_Run(t *testing.T) {
	reg := testRegistry()
	exec := tools.NewExecutor(reg, t.TempDir())
	runner := tools.NewToolRunner(exec)
	a := adapter.NewAdapter(runner)

	tc := openai.ChatCompletionMessageToolCall{
		ID: "call_run1",
		Function: openai.ChatCompletionMessageToolCallFunction{
			Name:      "echo",
			Arguments: `{"message":"world"}`,
		},
	}

	result := a.Run(context.Background(), tc)

	if result.OfTool == nil {
		t.Fatal("result.OfTool is nil")
	}
	if result.OfTool.ToolCallID != "call_run1" {
		t.Errorf("ToolCallID = %q, want %q", result.OfTool.ToolCallID, "call_run1")
	}
}

func TestAdapter_Run_ErrorPrefix(t *testing.T) {
	reg := testRegistry()
	exec := tools.NewExecutor(reg, t.TempDir())
	runner := tools.NewToolRunner(exec)
	a := adapter.NewAdapter(runner)

	tc := openai.ChatCompletionMessageToolCall{
		ID: "call_fail",
		Function: openai.ChatCompletionMessageToolCallFunction{
			Name:      "fail",
			Arguments: `{}`,
		},
	}

	result := a.Run(context.Background(), tc)

	if result.OfTool == nil {
		t.Fatal("result.OfTool is nil")
	}

	// Extract content string from the union
	content := result.OfTool.Content.OfString.Value
	if !strings.HasPrefix(content, "[ERROR] ") {
		t.Errorf("error content should be prefixed with [ERROR], got: %q", content)
	}
}

func TestAdapter_RunAll(t *testing.T) {
	reg := testRegistry()
	exec := tools.NewExecutor(reg, t.TempDir())
	runner := tools.NewToolRunner(exec)
	a := adapter.NewAdapter(runner)

	tcs := []openai.ChatCompletionMessageToolCall{
		{
			ID: "call-1",
			Function: openai.ChatCompletionMessageToolCallFunction{
				Name:      "echo",
				Arguments: `{"message":"first"}`,
			},
		},
		{
			ID: "call-2",
			Function: openai.ChatCompletionMessageToolCallFunction{
				Name:      "echo",
				Arguments: `{"message":"second"}`,
			},
		},
	}

	results := a.RunAll(context.Background(), tcs)

	if len(results) != 2 {
		t.Fatalf("RunAll returned %d results, want 2", len(results))
	}
	if results[0].OfTool.ToolCallID != "call-1" {
		t.Errorf("results[0].ToolCallID = %q, want %q", results[0].OfTool.ToolCallID, "call-1")
	}
	if results[1].OfTool.ToolCallID != "call-2" {
		t.Errorf("results[1].ToolCallID = %q, want %q", results[1].OfTool.ToolCallID, "call-2")
	}
}
