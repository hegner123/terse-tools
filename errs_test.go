package tools

import (
	"context"
	"encoding/json"
	"testing"
)

func TestErrs_GoErrors(t *testing.T) {
	input := `./main.go:10:5: undefined: foo
./main.go:15:2: too many arguments
./util.go:3:1: imported and not used: "fmt"
`
	result := builtinErrs(context.Background(), map[string]any{
		"input": input,
	}, "")

	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Error)
	}

	var out errsResult
	json.Unmarshal([]byte(result.Output), &out)

	if out.Count != 3 {
		t.Errorf("expected 3 errors, got %d", out.Count)
	}
	if out.Format != "colon" {
		t.Errorf("expected format colon, got %q", out.Format)
	}
	if out.Files != 2 {
		t.Errorf("expected 2 files, got %d", out.Files)
	}
}

func TestErrs_RustErrors(t *testing.T) {
	input := `error[E0425]: cannot find value ` + "`foo`" + ` in this scope
 --> src/main.rs:10:5
warning: unused variable
 --> src/main.rs:3:9
`
	result := builtinErrs(context.Background(), map[string]any{
		"input": input,
	}, "")

	var out errsResult
	json.Unmarshal([]byte(result.Output), &out)

	if out.Format != "rust" {
		t.Errorf("expected format rust, got %q", out.Format)
	}
	if out.Count != 2 {
		t.Errorf("expected 2 errors, got %d", out.Count)
	}
	if out.Errors[0].Code != "E0425" {
		t.Errorf("expected code E0425, got %q", out.Errors[0].Code)
	}
}

func TestErrs_TypeScriptErrors(t *testing.T) {
	input := `src/app.ts(10,5): error TS2304: Cannot find name 'foo'.
src/app.ts(15,1): error TS2322: Type mismatch.
`
	result := builtinErrs(context.Background(), map[string]any{
		"input": input,
	}, "")

	var out errsResult
	json.Unmarshal([]byte(result.Output), &out)

	if out.Format != "tsc" {
		t.Errorf("expected format tsc, got %q", out.Format)
	}
	if out.Count != 2 {
		t.Errorf("expected 2 errors, got %d", out.Count)
	}
}

func TestErrs_ANSIStripping(t *testing.T) {
	input := "\x1b[31m./main.go:10:5: error message\x1b[0m\n"
	result := builtinErrs(context.Background(), map[string]any{
		"input": input,
	}, "")

	var out errsResult
	json.Unmarshal([]byte(result.Output), &out)

	if out.Count != 1 {
		t.Errorf("expected 1 error after ANSI stripping, got %d", out.Count)
	}
}

func TestErrs_FormatHint(t *testing.T) {
	input := `src/main.rs:10:5: some error`
	result := builtinErrs(context.Background(), map[string]any{
		"input":  input,
		"format": "colon",
	}, "")

	var out errsResult
	json.Unmarshal([]byte(result.Output), &out)

	if out.Format != "colon" {
		t.Errorf("expected format colon (from hint), got %q", out.Format)
	}
}

func TestErrs_Deduplication(t *testing.T) {
	input := `./main.go:10:5: duplicate error
./main.go:10:5: duplicate error
./main.go:10:5: duplicate error
`
	result := builtinErrs(context.Background(), map[string]any{
		"input": input,
	}, "")

	var out errsResult
	json.Unmarshal([]byte(result.Output), &out)

	if out.Count != 1 {
		t.Errorf("expected 1 deduplicated error, got %d", out.Count)
	}
}

func TestErrs_EmptyInput(t *testing.T) {
	result := builtinErrs(context.Background(), map[string]any{
		"input": "",
	}, "")
	if !result.IsError {
		t.Fatal("expected error for empty input")
	}
}

func TestErrs_NoErrors(t *testing.T) {
	result := builtinErrs(context.Background(), map[string]any{
		"input": "Build complete.\nNo errors.\n",
	}, "")

	var out errsResult
	json.Unmarshal([]byte(result.Output), &out)

	if out.Count != 0 {
		t.Errorf("expected 0 errors, got %d", out.Count)
	}
	if out.Summary != "no errors found" {
		t.Errorf("unexpected summary: %q", out.Summary)
	}
}

func TestErrs_MissingParam(t *testing.T) {
	result := builtinErrs(context.Background(), map[string]any{}, "")
	if !result.IsError {
		t.Fatal("expected error for missing input")
	}
}
