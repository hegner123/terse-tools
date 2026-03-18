package tools

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"
)

func TestToolRunner_ExecutesTool(t *testing.T) {
	reg := NewRegistry()
	reg.Register(ToolDef{
		Name:        "echo",
		Description: "echoes input",
		InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
		Builtin: func(ctx context.Context, input map[string]any, workDir string) Result {
			return resultJSON(map[string]string{"msg": "hello"})
		},
	})

	exec := NewExecutor(reg, t.TempDir())
	runner := NewToolRunner(exec)

	result := runner.Run(context.Background(), ToolCall{
		ID:    "call-1",
		Name:  "echo",
		Input: json.RawMessage(`{}`),
	})

	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content)
	}
	if result.ID != "call-1" {
		t.Errorf("ID = %q, want %q", result.ID, "call-1")
	}
	if result.Name != "echo" {
		t.Errorf("Name = %q, want %q", result.Name, "echo")
	}
}

func TestToolRunner_UnknownTool(t *testing.T) {
	reg := NewRegistry()
	exec := NewExecutor(reg, t.TempDir())
	runner := NewToolRunner(exec)

	result := runner.Run(context.Background(), ToolCall{
		ID:   "call-1",
		Name: "nonexistent",
	})

	if !result.IsError {
		t.Fatal("expected error for unknown tool")
	}
}

func TestToolRunner_MiddlewareOrder(t *testing.T) {
	reg := NewRegistry()
	reg.Register(ToolDef{
		Name:        "noop",
		Description: "does nothing",
		InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
		Builtin: func(ctx context.Context, input map[string]any, workDir string) Result {
			return Result{Output: "ok"}
		},
	})

	exec := NewExecutor(reg, t.TempDir())
	runner := NewToolRunner(exec)

	var order []string
	var mu sync.Mutex

	appendOrder := func(s string) {
		mu.Lock()
		order = append(order, s)
		mu.Unlock()
	}

	runner.Use(func(ctx context.Context, call ToolCall, next func(context.Context, ToolCall) ToolResult) ToolResult {
		appendOrder("A-before")
		result := next(ctx, call)
		appendOrder("A-after")
		return result
	})

	runner.Use(func(ctx context.Context, call ToolCall, next func(context.Context, ToolCall) ToolResult) ToolResult {
		appendOrder("B-before")
		result := next(ctx, call)
		appendOrder("B-after")
		return result
	})

	runner.Run(context.Background(), ToolCall{Name: "noop", Input: json.RawMessage(`{}`)})

	expected := []string{"A-before", "B-before", "B-after", "A-after"}
	if len(order) != len(expected) {
		t.Fatalf("order = %v, want %v", order, expected)
	}
	for i, v := range expected {
		if order[i] != v {
			t.Errorf("order[%d] = %q, want %q", i, order[i], v)
		}
	}
}

func TestToolRunner_MiddlewareShortCircuit(t *testing.T) {
	reg := NewRegistry()
	reg.Register(ToolDef{
		Name:        "noop",
		Description: "does nothing",
		InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
		Builtin: func(ctx context.Context, input map[string]any, workDir string) Result {
			t.Fatal("executor should not be called when middleware short-circuits")
			return Result{}
		},
	})

	exec := NewExecutor(reg, t.TempDir())
	runner := NewToolRunner(exec)

	runner.Use(func(ctx context.Context, call ToolCall, next func(context.Context, ToolCall) ToolResult) ToolResult {
		return ToolResult{
			ID:      call.ID,
			Name:    call.Name,
			Content: "blocked",
			IsError: true,
		}
	})

	result := runner.Run(context.Background(), ToolCall{ID: "x", Name: "noop"})
	if !result.IsError {
		t.Fatal("expected error from short-circuit middleware")
	}
	if result.Content != "blocked" {
		t.Errorf("Content = %q, want %q", result.Content, "blocked")
	}
}

func TestLogMiddleware(t *testing.T) {
	reg := NewRegistry()
	reg.Register(ToolDef{
		Name:        "slow",
		Description: "sleeps briefly",
		InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
		Builtin: func(ctx context.Context, input map[string]any, workDir string) Result {
			time.Sleep(5 * time.Millisecond)
			return Result{Output: "done"}
		},
	})

	exec := NewExecutor(reg, t.TempDir())
	runner := NewToolRunner(exec)

	var beforeCalled bool
	var afterElapsed time.Duration
	var afterResult ToolResult

	runner.Use(LogMiddleware(
		func(ctx context.Context, call ToolCall) {
			beforeCalled = true
			if call.Name != "slow" {
				t.Errorf("before: Name = %q, want %q", call.Name, "slow")
			}
		},
		func(ctx context.Context, call ToolCall, result ToolResult, elapsed time.Duration) {
			afterElapsed = elapsed
			afterResult = result
		},
	))

	runner.Run(context.Background(), ToolCall{Name: "slow", Input: json.RawMessage(`{}`)})

	if !beforeCalled {
		t.Error("before hook was not called")
	}
	if afterElapsed < 5*time.Millisecond {
		t.Errorf("elapsed = %v, expected >= 5ms", afterElapsed)
	}
	if afterResult.IsError {
		t.Errorf("expected no error, got: %s", afterResult.Content)
	}
}

func TestLogMiddleware_NilHooks(t *testing.T) {
	reg := NewRegistry()
	reg.Register(ToolDef{
		Name:        "noop",
		Description: "does nothing",
		InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
		Builtin: func(ctx context.Context, input map[string]any, workDir string) Result {
			return Result{Output: "ok"}
		},
	})

	exec := NewExecutor(reg, t.TempDir())
	runner := NewToolRunner(exec)
	runner.Use(LogMiddleware(nil, nil))

	result := runner.Run(context.Background(), ToolCall{Name: "noop", Input: json.RawMessage(`{}`)})
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content)
	}
}

func TestDenyListMiddleware(t *testing.T) {
	reg := NewRegistry()
	reg.Register(ToolDef{
		Name:        "safe",
		Description: "allowed tool",
		InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
		Builtin: func(ctx context.Context, input map[string]any, workDir string) Result {
			return Result{Output: "ok"}
		},
	})
	reg.Register(ToolDef{
		Name:        "dangerous",
		Description: "blocked tool",
		InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
		Builtin: func(ctx context.Context, input map[string]any, workDir string) Result {
			t.Fatal("dangerous tool should not execute")
			return Result{}
		},
	})

	exec := NewExecutor(reg, t.TempDir())
	runner := NewToolRunner(exec)
	runner.Use(DenyListMiddleware("dangerous", "other"))

	// Allowed tool works
	result := runner.Run(context.Background(), ToolCall{Name: "safe", Input: json.RawMessage(`{}`)})
	if result.IsError {
		t.Fatalf("safe tool should not be blocked: %s", result.Content)
	}

	// Denied tool is blocked
	result = runner.Run(context.Background(), ToolCall{ID: "call-2", Name: "dangerous"})
	if !result.IsError {
		t.Fatal("dangerous tool should be blocked")
	}
	if result.ID != "call-2" {
		t.Errorf("ID = %q, want %q", result.ID, "call-2")
	}
}

func TestToolRunner_Executor(t *testing.T) {
	reg := NewRegistry()
	exec := NewExecutor(reg, t.TempDir())
	runner := NewToolRunner(exec)

	if runner.Executor() != exec {
		t.Error("Executor() should return the original executor")
	}
}

func TestToolRunner_EmptyID(t *testing.T) {
	reg := NewRegistry()
	reg.Register(ToolDef{
		Name:        "noop",
		Description: "does nothing",
		InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
		Builtin: func(ctx context.Context, input map[string]any, workDir string) Result {
			return Result{Output: "ok"}
		},
	})

	exec := NewExecutor(reg, t.TempDir())
	runner := NewToolRunner(exec)

	// Gemini-style call with no ID
	result := runner.Run(context.Background(), ToolCall{Name: "noop", Input: json.RawMessage(`{}`)})
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content)
	}
	if result.ID != "" {
		t.Errorf("ID = %q, want empty", result.ID)
	}
}
