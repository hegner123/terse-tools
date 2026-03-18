package tools

import (
	"context"
	"encoding/json"
	"time"
)

// ToolCall is the provider-agnostic representation of a tool invocation from an LLM.
// Each provider adapter converts SDK-specific types to/from ToolCall.
type ToolCall struct {
	// ID is the round-trip correlation ID from the provider.
	// Empty for providers that don't use call IDs (e.g. Gemini).
	ID string

	// Name is the tool name (must match a registered ToolDef.Name).
	Name string

	// Input is the raw JSON arguments from the LLM.
	Input json.RawMessage
}

// ToolResult is the provider-agnostic execution result.
// Each provider adapter converts ToolResult to its SDK-specific result type.
type ToolResult struct {
	// ID is echoed from the originating ToolCall.ID.
	ID string

	// Name is echoed from the originating ToolCall.Name.
	Name string

	// Content is the text output sent back to the LLM.
	Content string

	// IsError indicates the tool execution failed.
	IsError bool
}

// Middleware intercepts tool calls before execution.
// Call next to continue the chain, or return directly to short-circuit.
type Middleware func(ctx context.Context, call ToolCall, next func(context.Context, ToolCall) ToolResult) ToolResult

// ToolRunner wraps an Executor with a middleware chain.
type ToolRunner struct {
	executor    *Executor
	middlewares []Middleware
}

// NewToolRunner creates a ToolRunner that dispatches calls through the given Executor.
func NewToolRunner(executor *Executor) *ToolRunner {
	return &ToolRunner{executor: executor}
}

// Use appends one or more middleware to the chain. Middleware execute in
// the order they are added (first added = outermost).
func (r *ToolRunner) Use(mw ...Middleware) {
	r.middlewares = append(r.middlewares, mw...)
}

// Run executes a ToolCall through the middleware chain, then dispatches
// to the Executor. Returns a ToolResult with the call's ID and Name echoed back.
func (r *ToolRunner) Run(ctx context.Context, call ToolCall) ToolResult {
	// Build the terminal handler that calls the executor
	terminal := func(ctx context.Context, c ToolCall) ToolResult {
		result := r.executor.ExecuteJSON(ctx, c.Name, c.Input)
		return ToolResult{
			ID:      c.ID,
			Name:    c.Name,
			Content: result.Content(),
			IsError: result.IsError,
		}
	}

	// Wrap middleware in reverse order so the first middleware added is outermost
	handler := terminal
	for i := len(r.middlewares) - 1; i >= 0; i-- {
		mw := r.middlewares[i]
		next := handler
		handler = func(ctx context.Context, c ToolCall) ToolResult {
			return mw(ctx, c, next)
		}
	}

	return handler(ctx, call)
}

// Executor returns the underlying Executor for direct access.
func (r *ToolRunner) Executor() *Executor {
	return r.executor
}

// --- Built-in middleware ---

// BeforeFunc is called before a tool executes.
type BeforeFunc func(ctx context.Context, call ToolCall)

// AfterFunc is called after a tool executes with elapsed time.
type AfterFunc func(ctx context.Context, call ToolCall, result ToolResult, elapsed time.Duration)

// LogMiddleware returns middleware that calls hooks before and after each tool execution.
// Either hook may be nil.
func LogMiddleware(before BeforeFunc, after AfterFunc) Middleware {
	return func(ctx context.Context, call ToolCall, next func(context.Context, ToolCall) ToolResult) ToolResult {
		if before != nil {
			before(ctx, call)
		}
		start := time.Now()
		result := next(ctx, call)
		if after != nil {
			after(ctx, call, result, time.Since(start))
		}
		return result
	}
}

// DenyListMiddleware returns middleware that blocks calls to the named tools,
// returning an error result without executing.
func DenyListMiddleware(names ...string) Middleware {
	denied := make(map[string]bool, len(names))
	for _, n := range names {
		denied[n] = true
	}
	return func(ctx context.Context, call ToolCall, next func(context.Context, ToolCall) ToolResult) ToolResult {
		if denied[call.Name] {
			return ToolResult{
				ID:      call.ID,
				Name:    call.Name,
				Content: "tool " + call.Name + " is not allowed",
				IsError: true,
			}
		}
		return next(ctx, call)
	}
}
