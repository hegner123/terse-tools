package tools

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestTabcount_LineRange(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	os.WriteFile(path, []byte("a\n\tb\n\t\tc\n\t\t\td\ne\n"), 0644)

	result := builtinTabcount(context.Background(), map[string]any{
		"file":       path,
		"start_line": float64(2),
		"end_line":   float64(4),
	}, "")

	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Error)
	}

	var out tabcountResult
	json.Unmarshal([]byte(result.Output), &out)

	if out.TotalLines != 3 {
		t.Errorf("expected 3 lines in range, got %d", out.TotalLines)
	}
	if out.StartLine != 2 {
		t.Errorf("expected start_line 2, got %d", out.StartLine)
	}
	if out.EndLine != 4 {
		t.Errorf("expected end_line 4, got %d", out.EndLine)
	}

	// Should only have lines 2, 3, 4
	for _, li := range out.Lines {
		if li.Line < 2 || li.Line > 4 {
			t.Errorf("unexpected line %d in range [2,4]", li.Line)
		}
	}
}

func TestTabcount_NonexistentFile(t *testing.T) {
	result := builtinTabcount(context.Background(), map[string]any{
		"file": "/nonexistent/file.txt",
	}, "")
	if !result.IsError {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestTabcount_MissingFile(t *testing.T) {
	result := builtinTabcount(context.Background(), map[string]any{}, "")
	if !result.IsError {
		t.Fatal("expected error for missing file param")
	}
}

func TestTabcount_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.txt")
	os.WriteFile(path, []byte{}, 0644)

	result := builtinTabcount(context.Background(), map[string]any{
		"file": path,
	}, "")

	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Error)
	}

	var out tabcountResult
	json.Unmarshal([]byte(result.Output), &out)
	if out.TotalLines != 0 {
		t.Errorf("expected 0 total_lines for empty file, got %d", out.TotalLines)
	}
}

func TestTabcount_SpacesNotTabs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "spaces.py")
	os.WriteFile(path, []byte("def foo():\n    return 1\n        nested\n"), 0644)

	result := builtinTabcount(context.Background(), map[string]any{
		"file": path,
	}, "")

	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Error)
	}

	var out tabcountResult
	json.Unmarshal([]byte(result.Output), &out)

	// All lines should have 0 indentation (spaces, not tabs)
	for _, li := range out.Lines {
		if li.Indentation != 0 {
			t.Errorf("line %d: expected 0 tabs (spaces only), got %d", li.Line, li.Indentation)
		}
	}
}

func TestTabcount_RelativePath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	os.WriteFile(path, []byte("\tindented\n"), 0644)

	result := builtinTabcount(context.Background(), map[string]any{
		"file": "test.txt",
	}, dir)

	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Error)
	}
}

func TestTabcount_MatchesCLI(t *testing.T) {
	if _, err := exec.LookPath("tabcount"); err != nil {
		t.Skip("tabcount binary not found on PATH")
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "test.go")
	os.WriteFile(path, []byte("package main\n\nfunc main() {\n\tfmt.Println()\n\t\tx := 1\n}\n"), 0644)

	// Run builtin
	builtinOut := builtinTabcount(context.Background(), map[string]any{
		"file": path,
	}, "")
	if builtinOut.IsError {
		t.Fatalf("builtin error: %s", builtinOut.Error)
	}

	// Run CLI
	cmd := exec.Command("tabcount", "--cli", "--file", path)
	cliBytes, err := cmd.Output()
	if err != nil {
		t.Fatalf("CLI error: %s", err)
	}

	// Compare JSON structures
	var builtinJSON, cliJSON tabcountResult
	json.Unmarshal([]byte(builtinOut.Output), &builtinJSON)
	json.Unmarshal(cliBytes, &cliJSON)

	if builtinJSON.TotalLines != cliJSON.TotalLines {
		t.Errorf("total_lines mismatch: builtin=%d cli=%d", builtinJSON.TotalLines, cliJSON.TotalLines)
	}
	if len(builtinJSON.Lines) != len(cliJSON.Lines) {
		t.Fatalf("lines count mismatch: builtin=%d cli=%d", len(builtinJSON.Lines), len(cliJSON.Lines))
	}
	for i := range builtinJSON.Lines {
		if builtinJSON.Lines[i] != cliJSON.Lines[i] {
			t.Errorf("line %d mismatch: builtin=%+v cli=%+v", i, builtinJSON.Lines[i], cliJSON.Lines[i])
		}
	}
}
