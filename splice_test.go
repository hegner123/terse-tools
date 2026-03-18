package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSplice_Append(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source.txt")
	target := filepath.Join(dir, "target.txt")
	os.WriteFile(source, []byte("appended\n"), 0644)
	os.WriteFile(target, []byte("original\n"), 0644)

	result := builtinSplice(context.Background(), map[string]any{
		"source": source,
		"target": target,
		"mode":   "append",
	}, "")

	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Error)
	}

	data, _ := os.ReadFile(target)
	if string(data) != "original\nappended\n" {
		t.Errorf("expected original+appended, got %q", string(data))
	}
}

func TestSplice_Prepend(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source.txt")
	target := filepath.Join(dir, "target.txt")
	os.WriteFile(source, []byte("first\n"), 0644)
	os.WriteFile(target, []byte("second\n"), 0644)

	result := builtinSplice(context.Background(), map[string]any{
		"source": source,
		"target": target,
		"mode":   "prepend",
	}, "")

	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Error)
	}

	data, _ := os.ReadFile(target)
	if string(data) != "first\nsecond\n" {
		t.Errorf("expected first+second, got %q", string(data))
	}
}

func TestSplice_Replace(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source.txt")
	target := filepath.Join(dir, "target.txt")
	os.WriteFile(source, []byte("replacement\n"), 0644)
	os.WriteFile(target, []byte("original\n"), 0644)

	result := builtinSplice(context.Background(), map[string]any{
		"source": source,
		"target": target,
		"mode":   "replace",
	}, "")

	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Error)
	}

	data, _ := os.ReadFile(target)
	if string(data) != "replacement\n" {
		t.Errorf("expected replacement content, got %q", string(data))
	}
}

func TestSplice_Insert(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source.txt")
	target := filepath.Join(dir, "target.txt")
	os.WriteFile(source, []byte("inserted\n"), 0644)
	os.WriteFile(target, []byte("line1\nline2\nline3\n"), 0644)

	result := builtinSplice(context.Background(), map[string]any{
		"source": source,
		"target": target,
		"mode":   "insert",
		"line":   float64(2),
	}, "")

	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Error)
	}

	data, _ := os.ReadFile(target)
	expected := "line1\nline2\ninserted\nline3\n"
	if string(data) != expected {
		t.Errorf("expected %q, got %q", expected, string(data))
	}
}

func TestSplice_InsertMissingLine(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source.txt")
	target := filepath.Join(dir, "target.txt")
	os.WriteFile(source, []byte("x\n"), 0644)
	os.WriteFile(target, []byte("y\n"), 0644)

	result := builtinSplice(context.Background(), map[string]any{
		"source": source,
		"target": target,
		"mode":   "insert",
		// No line param
	}, "")

	if !result.IsError {
		t.Fatal("expected error: insert mode requires line >= 1")
	}
}

func TestSplice_AppendNoTrailingNewline(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source.txt")
	target := filepath.Join(dir, "target.txt")
	os.WriteFile(source, []byte("appended"), 0644)
	os.WriteFile(target, []byte("original"), 0644) // no trailing newline

	result := builtinSplice(context.Background(), map[string]any{
		"source": source,
		"target": target,
		"mode":   "append",
	}, "")

	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Error)
	}

	data, _ := os.ReadFile(target)
	// Should insert a newline between them
	if string(data) != "original\nappended" {
		t.Errorf("expected newline-joined, got %q", string(data))
	}
}

func TestSplice_InvalidMode(t *testing.T) {
	result := builtinSplice(context.Background(), map[string]any{
		"source": "/tmp/a",
		"target": "/tmp/b",
		"mode":   "destroy",
	}, "")
	if !result.IsError {
		t.Fatal("expected error for invalid mode")
	}
}

func TestSplice_MissingParams(t *testing.T) {
	result := builtinSplice(context.Background(), map[string]any{}, "")
	if !result.IsError {
		t.Fatal("expected error for missing params")
	}
}

func TestSplice_ResultFormat(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "s.txt")
	target := filepath.Join(dir, "t.txt")
	os.WriteFile(source, []byte("a\nb\nc\n"), 0644)
	os.WriteFile(target, []byte("x\n"), 0644)

	result := builtinSplice(context.Background(), map[string]any{
		"source": source,
		"target": target,
		"mode":   "append",
	}, "")

	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Error)
	}

	var out spliceResult
	json.Unmarshal([]byte(result.Output), &out)

	if out.Mode != "append" {
		t.Errorf("expected mode append, got %q", out.Mode)
	}
	if out.LinesAdded != 3 {
		t.Errorf("expected 3 lines_added, got %d", out.LinesAdded)
	}
	if out.Summary == "" {
		t.Error("expected non-empty summary")
	}
}
