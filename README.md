# terse-tools

A Go library that provides a registry of 18 builtin tools for agentic LLM workflows, with provider adapters for Anthropic, OpenAI, and Gemini. Import the core library with zero external dependencies, then add only the provider adapter you need.

## Features

- **18 builtin tools** executing in-process (no external binaries required): file I/O, search, replace, diff, parse, transform, and more
- **Provider adapters** for Anthropic, OpenAI, and Gemini — each in a separate Go module so you only pull the SDK you use
- **Middleware chain** for logging, access control, or custom interception before/after tool execution
- **Registry filtering** with presets like `ExcludeDangerous()` and `ReadOnlyTools()` for sandboxing agents at different trust levels
- **Read-before-write safety** — the executor tracks file reads and blocks blind overwrites
- **Thread-safe** registry with concurrent access via `sync.RWMutex`

## Installation

Core library (zero external dependencies):

```bash
go get github.com/hegner123/terse-tools
```

Add a provider adapter (pulls only that SDK):

```bash
go get github.com/hegner123/terse-tools/provider/anthropic
go get github.com/hegner123/terse-tools/provider/openai
go get github.com/hegner123/terse-tools/provider/gemini
```

## Usage

### Basic: Registry + Executor

```go
package main

import (
    "context"
    "fmt"

    tools "github.com/hegner123/terse-tools"
)

func main() {
    reg := tools.DefaultRegistry()  // All 18 tools
    exec := tools.NewExecutor(reg, "/workspace")

    result := exec.Execute(context.Background(), "checkfor", map[string]any{
        "search": "TODO",
        "dir":    ".",
        "ext":    ".go",
    })

    if result.IsError {
        fmt.Println("Error:", result.Error)
        return
    }
    fmt.Println(result.Output)
}
```

### With Middleware

```go
runner := tools.NewToolRunner(exec)

// Log every tool call
runner.Use(tools.LogMiddleware(nil, func(ctx context.Context, call tools.ToolCall, result tools.ToolResult, elapsed time.Duration) {
    log.Printf("[%s] %s %v", call.Name, elapsed, result.IsError)
}))

// Block dangerous tools
runner.Use(tools.DenyListMiddleware("bash", "delete", "write"))
```

### With Provider Adapters (Anthropic example)

```go
import (
    tools "github.com/hegner123/terse-tools"
    adapter "github.com/hegner123/terse-tools/provider/anthropic"
    "github.com/anthropics/anthropic-sdk-go"
)

// Set up
reg := tools.TerseRegistry().Filter(tools.ExcludeDangerous())
exec := tools.NewExecutor(reg, "/workspace")
runner := tools.NewToolRunner(exec)
a := adapter.NewAdapter(runner)

// Convert tools for the API request
toolParams := a.Tools(reg)
// Pass toolParams to anthropic.MessageNewParams{Tools: toolParams}

// When the response contains tool_use blocks:
// results := a.RunAll(ctx, toolUseBlocks)
// Append results to the next user message
```

### Registry Filtering

```go
// Only read-only tools (safe for untrusted agents)
safe := tools.DefaultRegistry().Filter(tools.ReadOnlyTools())

// Remove specific tools
limited := tools.DefaultRegistry().Filter(tools.ExcludeNames("bash", "write"))

// Custom filter
goOnly := tools.DefaultRegistry().Filter(func(def tools.ToolDef) bool {
    return def.Name != "bash"
})
```

## Builtin Tools

| Tool | Description |
|------|-------------|
| `read` | Read file contents with optional line offset/limit |
| `write` | Write content to a file (enforces read-before-write) |
| `bash` | Execute shell commands |
| `checkfor` | Search for text across files and directories |
| `repfor` | Replace text across multiple files |
| `stump` | Directory tree visualization |
| `sig` | Extract function signatures and type definitions (Go, TypeScript, C#) |
| `errs` | Parse compiler/linter error output into structured JSON |
| `imports` | Map import/dependency statements across 11 languages |
| `conflicts` | Parse git merge conflict markers into structured JSON |
| `cleanDiff` | Compact git diff as structured JSON |
| `transform` | Composable JSON array pipeline (group, sort, filter, format) |
| `split` | Split a file at specified line numbers |
| `splice` | Insert, append, prepend, or replace content in a file |
| `tabcount` | Count tab characters in a file |
| `notab` | Convert between tabs and spaces |
| `utf8` | Fix malformed UTF-8/UTF-16 encoding |
| `delete` | Move files to trash (macOS) |

## API Reference

### Core Types

- **`ToolDef`** — Defines a tool: name, description, JSON Schema, and either a builtin function or binary path
- **`Registry`** — Thread-safe collection of tool definitions with `Register`, `Get`, `Remove`, `Filter`, `Defs`
- **`Executor`** — Runs tools by name, handling timeout, output truncation, and read-before-write tracking
- **`ToolRunner`** — Wraps an Executor with a middleware chain
- **`ToolCall`** / **`ToolResult`** — Provider-agnostic types that adapters convert to/from SDK-specific types

### Registry Factories

- `DefaultRegistry()` — All 18 tools
- `TerseRegistry()` — 15 domain tools (no read/write/bash)
- `ToolRegistry(toolSets...)` — Custom composition from tool slices

### Filter Presets

- `ExcludeDangerous()` — Removes bash, delete, write, transform, splice, split, repfor
- `ReadOnlyTools()` — Keeps only non-mutating tools
- `ExcludeNames(names...)` / `IncludeNames(names...)` — Explicit allow/deny lists

### Built-in Middleware

- `LogMiddleware(before, after)` — Hook into tool execution lifecycle
- `DenyListMiddleware(names...)` — Block specific tools at runtime

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

MIT License. See [LICENSE](LICENSE).
