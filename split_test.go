package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSplit_BasicSplit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	os.WriteFile(path, []byte("line1\nline2\nline3\nline4\nline5\nline6\n"), 0644)

	result := builtinSplit(context.Background(), map[string]any{
		"file":  path,
		"lines": []any{float64(2), float64(4)},
	}, "")

	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Error)
	}

	var out splitResult
	json.Unmarshal([]byte(result.Output), &out)

	if out.TotalLines != 6 {
		t.Errorf("expected 6 total_lines, got %d", out.TotalLines)
	}
	if len(out.Parts) != 3 {
		t.Fatalf("expected 3 parts, got %d", len(out.Parts))
	}

	// Part 1: lines 1-2
	if out.Parts[0].StartLine != 1 || out.Parts[0].EndLine != 2 {
		t.Errorf("part 1: expected lines 1-2, got %d-%d", out.Parts[0].StartLine, out.Parts[0].EndLine)
	}
	// Part 2: lines 3-4
	if out.Parts[1].StartLine != 3 || out.Parts[1].EndLine != 4 {
		t.Errorf("part 2: expected lines 3-4, got %d-%d", out.Parts[1].StartLine, out.Parts[1].EndLine)
	}
	// Part 3: lines 5-6
	if out.Parts[2].StartLine != 5 || out.Parts[2].EndLine != 6 {
		t.Errorf("part 3: expected lines 5-6, got %d-%d", out.Parts[2].StartLine, out.Parts[2].EndLine)
	}

	// Verify file content
	data1, _ := os.ReadFile(out.Parts[0].Path)
	if strings.TrimSpace(string(data1)) != "line1\nline2" {
		t.Errorf("part 1 content: %q", string(data1))
	}
}

func TestSplit_OutputNaming(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "code.go")
	os.WriteFile(path, []byte("a\nb\nc\nd\n"), 0644)

	result := builtinSplit(context.Background(), map[string]any{
		"file":  path,
		"lines": []any{float64(2)},
	}, "")

	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Error)
	}

	var out splitResult
	json.Unmarshal([]byte(result.Output), &out)

	// Should produce code_001.go and code_002.go
	if !strings.HasSuffix(out.Parts[0].Path, "code_001.go") {
		t.Errorf("expected code_001.go, got %s", filepath.Base(out.Parts[0].Path))
	}
	if !strings.HasSuffix(out.Parts[1].Path, "code_002.go") {
		t.Errorf("expected code_002.go, got %s", filepath.Base(out.Parts[1].Path))
	}
}

func TestSplit_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.txt")
	os.WriteFile(path, []byte{}, 0644)

	result := builtinSplit(context.Background(), map[string]any{
		"file":  path,
		"lines": []any{float64(5)},
	}, "")

	if !result.IsError {
		t.Fatal("expected error for empty file")
	}
}

func TestSplit_InvalidSplitPoints(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	os.WriteFile(path, []byte("a\nb\nc\n"), 0644)

	// Split point beyond file length
	result := builtinSplit(context.Background(), map[string]any{
		"file":  path,
		"lines": []any{float64(100)},
	}, "")

	if !result.IsError {
		t.Fatal("expected error: split point beyond file length should yield no valid points")
	}
}

func TestSplit_DuplicatePoints(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	os.WriteFile(path, []byte("a\nb\nc\nd\n"), 0644)

	result := builtinSplit(context.Background(), map[string]any{
		"file":  path,
		"lines": []any{float64(2), float64(2), float64(2)},
	}, "")

	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Error)
	}

	var out splitResult
	json.Unmarshal([]byte(result.Output), &out)

	// Duplicates should be deduped — only 2 parts
	if len(out.Parts) != 2 {
		t.Errorf("expected 2 parts (deduped), got %d", len(out.Parts))
	}
}

func TestSplit_MissingParams(t *testing.T) {
	result := builtinSplit(context.Background(), map[string]any{}, "")
	if !result.IsError {
		t.Fatal("expected error for missing params")
	}

	result = builtinSplit(context.Background(), map[string]any{
		"file": "/tmp/test.txt",
	}, "")
	if !result.IsError {
		t.Fatal("expected error for missing lines")
	}
}

func TestSplit_RelativePath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	os.WriteFile(path, []byte("a\nb\nc\nd\n"), 0644)

	result := builtinSplit(context.Background(), map[string]any{
		"file":  "test.txt",
		"lines": []any{float64(2)},
	}, dir)

	if result.IsError {
		t.Fatalf("relative path should work: %s", result.Error)
	}
}
