package anthropic_test

import (
	"context"
	"encoding/json"
	"testing"

	tools "github.com/hegner123/terse-tools"
	adapter "github.com/hegner123/terse-tools/provider/anthropic"

	"github.com/anthropics/anthropic-sdk-go"
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
				"verbose": map[string]any{
					"type":        "boolean",
					"description": "Enable verbose output",
				},
			},
			"required": []string{"path"},
		},
	}

	result := adapter.ConvertToolDef(def)

	if result.OfTool == nil {
		t.Fatal("OfTool is nil")
	}
	if result.OfTool.Name != "test_tool" {
		t.Errorf("Name = %q, want %q", result.OfTool.Name, "test_tool")
	}

	// Verify required field
	if len(result.OfTool.InputSchema.Required) != 1 {
		t.Fatalf("Required length = %d, want 1", len(result.OfTool.InputSchema.Required))
	}
	if result.OfTool.InputSchema.Required[0] != "path" {
		t.Errorf("Required[0] = %q, want %q", result.OfTool.InputSchema.Required[0], "path")
	}

	// Verify properties exist
	props, ok := result.OfTool.InputSchema.Properties.(map[string]any)
	if !ok {
		t.Fatal("Properties is not map[string]any")
	}
	if _, ok := props["path"]; !ok {
		t.Error("Properties missing 'path'")
	}
	if _, ok := props["verbose"]; !ok {
		t.Error("Properties missing 'verbose'")
	}
}

func TestConvertToolDef_EmptySchema(t *testing.T) {
	def := tools.ToolDef{
		Name:        "no_params",
		Description: "Tool with no parameters",
		InputSchema: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
	}

	result := adapter.ConvertToolDef(def)
	if result.OfTool.Name != "no_params" {
		t.Errorf("Name = %q, want %q", result.OfTool.Name, "no_params")
	}
	if len(result.OfTool.InputSchema.Required) != 0 {
		t.Errorf("Required should be empty, got %v", result.OfTool.InputSchema.Required)
	}
}

func TestParseToolUse(t *testing.T) {
	block := anthropic.ToolUseBlock{
		ID:    "toolu_abc123",
		Name:  "echo",
		Input: json.RawMessage(`{"message":"hello"}`),
	}

	call := adapter.ParseToolUse(block)

	if call.ID != "toolu_abc123" {
		t.Errorf("ID = %q, want %q", call.ID, "toolu_abc123")
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

func TestParseToolUse_RoundTrip(t *testing.T) {
	original := anthropic.ToolUseBlock{
		ID:    "toolu_xyz",
		Name:  "test",
		Input: json.RawMessage(`{"key":"value","num":42}`),
	}

	call := adapter.ParseToolUse(original)

	if call.ID != original.ID {
		t.Errorf("ID mismatch: %q != %q", call.ID, original.ID)
	}
	if call.Name != original.Name {
		t.Errorf("Name mismatch: %q != %q", call.Name, original.Name)
	}
	if string(call.Input) != string(original.Input) {
		t.Errorf("Input mismatch: %s != %s", call.Input, original.Input)
	}
}

func TestAdapter_Tools(t *testing.T) {
	reg := testRegistry()
	exec := tools.NewExecutor(reg, t.TempDir())
	runner := tools.NewToolRunner(exec)
	a := adapter.NewAdapter(runner)

	toolParams := a.Tools(reg)
	if len(toolParams) != 1 {
		t.Fatalf("Tools() returned %d params, want 1", len(toolParams))
	}
	if toolParams[0].OfTool.Name != "echo" {
		t.Errorf("Name = %q, want %q", toolParams[0].OfTool.Name, "echo")
	}
}

func TestAdapter_Run(t *testing.T) {
	reg := testRegistry()
	exec := tools.NewExecutor(reg, t.TempDir())
	runner := tools.NewToolRunner(exec)
	a := adapter.NewAdapter(runner)

	block := anthropic.ToolUseBlock{
		ID:    "toolu_run1",
		Name:  "echo",
		Input: json.RawMessage(`{"message":"world"}`),
	}

	result := a.Run(context.Background(), block)

	// Verify the result is a tool_result block
	if result.OfToolResult == nil {
		t.Fatal("result.OfToolResult is nil")
	}
	if result.OfToolResult.ToolUseID != "toolu_run1" {
		t.Errorf("ToolUseID = %q, want %q", result.OfToolResult.ToolUseID, "toolu_run1")
	}
}

func TestAdapter_Run_UnknownTool(t *testing.T) {
	reg := testRegistry()
	exec := tools.NewExecutor(reg, t.TempDir())
	runner := tools.NewToolRunner(exec)
	a := adapter.NewAdapter(runner)

	block := anthropic.ToolUseBlock{
		ID:    "toolu_err",
		Name:  "nonexistent",
		Input: json.RawMessage(`{}`),
	}

	result := a.Run(context.Background(), block)

	if result.OfToolResult == nil {
		t.Fatal("result.OfToolResult is nil")
	}
	if result.OfToolResult.ToolUseID != "toolu_err" {
		t.Errorf("ToolUseID = %q, want %q", result.OfToolResult.ToolUseID, "toolu_err")
	}
}

func TestAdapter_RunAll(t *testing.T) {
	reg := testRegistry()
	exec := tools.NewExecutor(reg, t.TempDir())
	runner := tools.NewToolRunner(exec)
	a := adapter.NewAdapter(runner)

	blocks := []anthropic.ToolUseBlock{
		{ID: "call-1", Name: "echo", Input: json.RawMessage(`{"message":"first"}`)},
		{ID: "call-2", Name: "echo", Input: json.RawMessage(`{"message":"second"}`)},
	}

	results := a.RunAll(context.Background(), blocks)

	if len(results) != 2 {
		t.Fatalf("RunAll returned %d results, want 2", len(results))
	}
	if results[0].OfToolResult.ToolUseID != "call-1" {
		t.Errorf("results[0].ToolUseID = %q, want %q", results[0].OfToolResult.ToolUseID, "call-1")
	}
	if results[1].OfToolResult.ToolUseID != "call-2" {
		t.Errorf("results[1].ToolUseID = %q, want %q", results[1].OfToolResult.ToolUseID, "call-2")
	}
}

func TestAdapter_WithMiddleware(t *testing.T) {
	reg := testRegistry()
	exec := tools.NewExecutor(reg, t.TempDir())
	runner := tools.NewToolRunner(exec)
	runner.Use(tools.DenyListMiddleware("echo"))
	a := adapter.NewAdapter(runner)

	block := anthropic.ToolUseBlock{
		ID:    "toolu_blocked",
		Name:  "echo",
		Input: json.RawMessage(`{"message":"blocked"}`),
	}

	result := a.Run(context.Background(), block)

	if result.OfToolResult == nil {
		t.Fatal("result.OfToolResult is nil")
	}
	if result.OfToolResult.ToolUseID != "toolu_blocked" {
		t.Errorf("ToolUseID = %q, want %q", result.OfToolResult.ToolUseID, "toolu_blocked")
	}
}
