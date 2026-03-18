// Package tools provides a tool registry and executor for agentic tool use.
// Each tool is either a standalone CLI binary invoked via exec.Command, or a builtin
// that executes directly in-process.
package tools

import (
	"context"
	"fmt"
	"os/exec"
	"sort"
	"sync"
	"time"
)

// BuiltinFunc is the signature for tools that execute in-process rather than
// spawning a subprocess. The workDir parameter is the executor's working directory.
type BuiltinFunc func(ctx context.Context, input map[string]any, workDir string) Result

// ToolDef defines a tool that can be invoked during agentic conversations.
// A tool is either a subprocess tool (Binary set) or a builtin (Builtin set).
type ToolDef struct {
	// Name is the tool name (e.g. "checkfor").
	Name string

	// Description is shown to the LLM to help it decide when to use the tool.
	Description string

	// InputSchema is the JSON Schema for the tool's parameters.
	InputSchema map[string]any

	// Builtin, when non-nil, means this tool runs in-process instead of
	// spawning a subprocess. Binary/FlagMap/NeedsCLI/StdinParam are ignored.
	Builtin BuiltinFunc

	// Binary is the executable name resolved via PATH (e.g. "checkfor", "stump-core").
	// Empty for builtin tools.
	Binary string

	// NeedsCLI prepends --cli to the argument list when true.
	NeedsCLI bool

	// FlagMap maps input parameter names to CLI flag specifications.
	FlagMap map[string]FlagSpec

	// StdinParam names the input parameter whose value is piped to stdin
	// instead of passed as a flag. Empty means no stdin piping.
	StdinParam string

	// Timeout is the per-invocation timeout. Zero uses the executor's default.
	Timeout time.Duration
}

// IsBuiltinTool returns true if this tool executes in-process.
func (d ToolDef) IsBuiltinTool() bool {
	return d.Builtin != nil
}

// FlagSpec describes how a single input parameter maps to a CLI flag.
type FlagSpec struct {
	// Flag is the CLI flag name (e.g. "--file", "--dir").
	Flag string

	// Type controls serialization: "string", "bool", "int", "array".
	Type string

	// Positional appends the value as a positional arg without a flag prefix.
	Positional bool
}

// APITool is a generic tool representation for API requests.
type APITool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"input_schema"`
}

// Registry holds tool definitions and converts them to API-ready format.
// All methods are safe for concurrent use via sync.RWMutex.
type Registry struct {
	mu    sync.RWMutex
	tools map[string]ToolDef
	order []string // insertion order for stable iteration
}

// NewRegistry creates an empty tool registry.
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]ToolDef),
	}
}

// Register adds a tool definition to the registry.
// If a tool with the same name exists, it is replaced.
func (r *Registry) Register(def ToolDef) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tools[def.Name]; !exists {
		r.order = append(r.order, def.Name)
	}
	r.tools[def.Name] = def
}

// Get returns a tool definition by name.
func (r *Registry) Get(name string) (ToolDef, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	def, ok := r.tools[name]
	return def, ok
}

// Remove removes a tool from the registry.
func (r *Registry) Remove(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tools[name]; !exists {
		return
	}
	delete(r.tools, name)
	filtered := r.order[:0]
	for _, n := range r.order {
		if n != name {
			filtered = append(filtered, n)
		}
	}
	r.order = filtered
}

// Names returns all registered tool names in sorted order.
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// APITools converts all registered tool definitions to APITool slices
// suitable for inclusion in API requests.
func (r *Registry) APITools() []APITool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	apiTools := make([]APITool, 0, len(r.tools))
	for _, name := range r.order {
		def := r.tools[name]
		apiTools = append(apiTools, APITool{
			Name:        def.Name,
			Description: def.Description,
			InputSchema: def.InputSchema,
		})
	}
	return apiTools
}

// Len returns the number of registered tools.
func (r *Registry) Len() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.tools)
}

// CheckBinaries verifies that each registered subprocess tool's binary is on PATH.
// Builtin tools are skipped. Returns a map of tool name to error for any
// missing binaries.
func (r *Registry) CheckBinaries() map[string]error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	missing := make(map[string]error)
	for name, def := range r.tools {
		if def.IsBuiltinTool() {
			continue
		}
		if _, err := exec.LookPath(def.Binary); err != nil {
			missing[name] = fmt.Errorf("binary %q not found on PATH: %w", def.Binary, err)
		}
	}
	return missing
}

// DefaultRegistry creates a registry with all builtin tools registered.
func DefaultRegistry() *Registry {
	return ToolRegistry(BuiltinTools())
}

// TerseRegistry creates a registry with only the 15 terse tools (no read/write/bash).
// Use this when your agent framework already provides its own file I/O and shell.
func TerseRegistry() *Registry {
	return ToolRegistry(TerseTools())
}

// ToolRegistry creates a registry from one or more tool definition slices.
//
//	reg := tools.ToolRegistry(tools.CoreTools(), tools.TerseTools())
func ToolRegistry(toolSets ...[]ToolDef) *Registry {
	reg := NewRegistry()
	for _, set := range toolSets {
		for _, def := range set {
			reg.Register(def)
		}
	}
	return reg
}

// Defs returns all registered ToolDef values in insertion order.
func (r *Registry) Defs() []ToolDef {
	r.mu.RLock()
	defer r.mu.RUnlock()

	defs := make([]ToolDef, 0, len(r.tools))
	for _, name := range r.order {
		defs = append(defs, r.tools[name])
	}
	return defs
}
