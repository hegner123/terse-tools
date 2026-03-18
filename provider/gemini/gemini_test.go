package gemini_test

import (
	"context"
	"encoding/json"
	"testing"

	tools "github.com/hegner123/terse-tools"
	adapter "github.com/hegner123/terse-tools/provider/gemini"

	"github.com/google/generative-ai-go/genai"
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

func TestConvertToolDef_SimpleTypes(t *testing.T) {
	def := tools.ToolDef{
		Name:        "test_tool",
		Description: "A test tool",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name": map[string]any{
					"type":        "string",
					"description": "A name",
				},
				"count": map[string]any{
					"type":        "integer",
					"description": "A count",
				},
				"ratio": map[string]any{
					"type":        "number",
					"description": "A ratio",
				},
				"verbose": map[string]any{
					"type":        "boolean",
					"description": "Verbose mode",
				},
			},
			"required": []string{"name"},
		},
	}

	decl := adapter.ConvertToolDef(def)

	if decl.Name != "test_tool" {
		t.Errorf("Name = %q, want %q", decl.Name, "test_tool")
	}
	if decl.Description != "A test tool" {
		t.Errorf("Description = %q, want %q", decl.Description, "A test tool")
	}

	schema := decl.Parameters
	if schema == nil {
		t.Fatal("Parameters is nil")
	}
	if schema.Type != genai.TypeObject {
		t.Errorf("Type = %v, want TypeObject", schema.Type)
	}

	// Verify each property type
	typeTests := map[string]genai.Type{
		"name":    genai.TypeString,
		"count":   genai.TypeInteger,
		"ratio":   genai.TypeNumber,
		"verbose": genai.TypeBoolean,
	}
	for propName, expectedType := range typeTests {
		prop, ok := schema.Properties[propName]
		if !ok {
			t.Errorf("missing property %q", propName)
			continue
		}
		if prop.Type != expectedType {
			t.Errorf("property %q: Type = %v, want %v", propName, prop.Type, expectedType)
		}
	}

	// Verify required
	if len(schema.Required) != 1 || schema.Required[0] != "name" {
		t.Errorf("Required = %v, want [name]", schema.Required)
	}
}

func TestConvertToolDef_NestedArray(t *testing.T) {
	def := tools.ToolDef{
		Name:        "array_tool",
		Description: "Tool with array param",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"items": map[string]any{
					"type":        "array",
					"description": "List of items",
					"items": map[string]any{
						"type": "string",
					},
				},
			},
		},
	}

	decl := adapter.ConvertToolDef(def)
	itemsProp := decl.Parameters.Properties["items"]

	if itemsProp == nil {
		t.Fatal("items property is nil")
	}
	if itemsProp.Type != genai.TypeArray {
		t.Errorf("items Type = %v, want TypeArray", itemsProp.Type)
	}
	if itemsProp.Items == nil {
		t.Fatal("items.Items is nil")
	}
	if itemsProp.Items.Type != genai.TypeString {
		t.Errorf("items.Items.Type = %v, want TypeString", itemsProp.Items.Type)
	}
}

func TestConvertToolDef_NestedObject(t *testing.T) {
	def := tools.ToolDef{
		Name:        "nested_tool",
		Description: "Tool with nested object",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"config": map[string]any{
					"type":        "object",
					"description": "Configuration",
					"properties": map[string]any{
						"depth": map[string]any{
							"type":        "integer",
							"description": "Depth level",
						},
					},
					"required": []string{"depth"},
				},
			},
		},
	}

	decl := adapter.ConvertToolDef(def)
	configProp := decl.Parameters.Properties["config"]

	if configProp == nil {
		t.Fatal("config property is nil")
	}
	if configProp.Type != genai.TypeObject {
		t.Errorf("config Type = %v, want TypeObject", configProp.Type)
	}

	depthProp, ok := configProp.Properties["depth"]
	if !ok {
		t.Fatal("config.Properties missing 'depth'")
	}
	if depthProp.Type != genai.TypeInteger {
		t.Errorf("depth Type = %v, want TypeInteger", depthProp.Type)
	}
	if len(configProp.Required) != 1 || configProp.Required[0] != "depth" {
		t.Errorf("config Required = %v, want [depth]", configProp.Required)
	}
}

func TestConvertToolDef_EmptySchema(t *testing.T) {
	def := tools.ToolDef{
		Name:        "no_params",
		Description: "No parameters",
		InputSchema: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
	}

	decl := adapter.ConvertToolDef(def)
	if decl.Parameters.Type != genai.TypeObject {
		t.Errorf("Type = %v, want TypeObject", decl.Parameters.Type)
	}
}

func TestConvertToolDef_NilSchema(t *testing.T) {
	def := tools.ToolDef{
		Name:        "nil_schema",
		Description: "Nil schema",
	}

	decl := adapter.ConvertToolDef(def)
	if decl.Parameters != nil {
		t.Errorf("Parameters should be nil for nil InputSchema, got %v", decl.Parameters)
	}
}

func TestParseFunctionCall(t *testing.T) {
	fc := genai.FunctionCall{
		Name: "echo",
		Args: map[string]any{
			"message": "hello",
		},
	}

	call, err := adapter.ParseFunctionCall(fc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if call.ID != "" {
		t.Errorf("ID = %q, want empty (Gemini has no call IDs)", call.ID)
	}
	if call.Name != "echo" {
		t.Errorf("Name = %q, want %q", call.Name, "echo")
	}

	var input map[string]any
	if err := json.Unmarshal(call.Input, &input); err != nil {
		t.Fatalf("unmarshal input: %v", err)
	}
	if input["message"] != "hello" {
		t.Errorf("input[message] = %v, want %q", input["message"], "hello")
	}
}

func TestParseFunctionCall_NilArgs(t *testing.T) {
	fc := genai.FunctionCall{
		Name: "no_args",
		Args: nil,
	}

	call, err := adapter.ParseFunctionCall(fc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if call.Name != "no_args" {
		t.Errorf("Name = %q, want %q", call.Name, "no_args")
	}
	if string(call.Input) != "null" {
		t.Errorf("Input = %q, want %q", string(call.Input), "null")
	}
}

func TestAdapter_Tools(t *testing.T) {
	reg := testRegistry()
	exec := tools.NewExecutor(reg, t.TempDir())
	runner := tools.NewToolRunner(exec)
	a := adapter.NewAdapter(runner)

	tool := a.Tools(reg)
	if tool == nil {
		t.Fatal("Tools() returned nil")
	}
	if len(tool.FunctionDeclarations) != 1 {
		t.Fatalf("FunctionDeclarations length = %d, want 1", len(tool.FunctionDeclarations))
	}
	if tool.FunctionDeclarations[0].Name != "echo" {
		t.Errorf("Name = %q, want %q", tool.FunctionDeclarations[0].Name, "echo")
	}
}

func TestAdapter_Run(t *testing.T) {
	reg := testRegistry()
	exec := tools.NewExecutor(reg, t.TempDir())
	runner := tools.NewToolRunner(exec)
	a := adapter.NewAdapter(runner)

	fc := genai.FunctionCall{
		Name: "echo",
		Args: map[string]any{"message": "world"},
	}

	resp, err := a.Run(context.Background(), fc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Name != "echo" {
		t.Errorf("Name = %q, want %q", resp.Name, "echo")
	}
	if resp.Response == nil {
		t.Fatal("Response is nil")
	}
	if resp.Response["is_error"] != false {
		t.Errorf("is_error = %v, want false", resp.Response["is_error"])
	}
}

func TestAdapter_Run_UnknownTool(t *testing.T) {
	reg := testRegistry()
	exec := tools.NewExecutor(reg, t.TempDir())
	runner := tools.NewToolRunner(exec)
	a := adapter.NewAdapter(runner)

	fc := genai.FunctionCall{
		Name: "nonexistent",
		Args: map[string]any{},
	}

	resp, err := a.Run(context.Background(), fc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Response["is_error"] != true {
		t.Errorf("is_error = %v, want true for unknown tool", resp.Response["is_error"])
	}
}

func TestAdapter_RunAll(t *testing.T) {
	reg := testRegistry()
	exec := tools.NewExecutor(reg, t.TempDir())
	runner := tools.NewToolRunner(exec)
	a := adapter.NewAdapter(runner)

	calls := []genai.FunctionCall{
		{Name: "echo", Args: map[string]any{"message": "first"}},
		{Name: "echo", Args: map[string]any{"message": "second"}},
	}

	results, err := a.RunAll(context.Background(), calls)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("RunAll returned %d results, want 2", len(results))
	}
	if results[0].Name != "echo" {
		t.Errorf("results[0].Name = %q, want %q", results[0].Name, "echo")
	}
	if results[1].Name != "echo" {
		t.Errorf("results[1].Name = %q, want %q", results[1].Name, "echo")
	}
}

func TestConvertToolDef_WithEnum(t *testing.T) {
	def := tools.ToolDef{
		Name:        "enum_tool",
		Description: "Tool with enum",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"color": map[string]any{
					"type":        "string",
					"description": "A color",
					"enum":        []any{"red", "green", "blue"},
				},
			},
		},
	}

	decl := adapter.ConvertToolDef(def)
	colorProp := decl.Parameters.Properties["color"]

	if colorProp == nil {
		t.Fatal("color property is nil")
	}
	if len(colorProp.Enum) != 3 {
		t.Fatalf("Enum length = %d, want 3", len(colorProp.Enum))
	}
	expected := []string{"red", "green", "blue"}
	for i, v := range expected {
		if colorProp.Enum[i] != v {
			t.Errorf("Enum[%d] = %q, want %q", i, colorProp.Enum[i], v)
		}
	}
}
