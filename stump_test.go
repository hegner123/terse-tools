package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStump_BasicDirectory(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "sub"), 0755)
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a"), 0644)
	os.WriteFile(filepath.Join(dir, "sub", "b.txt"), []byte("b"), 0644)

	result := builtinStump(context.Background(), map[string]any{
		"dir": dir,
	}, "")

	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Error)
	}

	var out stumpResult
	json.Unmarshal([]byte(result.Output), &out)

	if out.Root != dir {
		t.Errorf("expected root %q, got %q", dir, out.Root)
	}
	if out.Stats.Dirs < 1 {
		t.Error("expected at least 1 directory")
	}
	if out.Stats.Files < 2 {
		t.Errorf("expected at least 2 files, got %d", out.Stats.Files)
	}

	// Verify tree entries exist
	paths := make(map[string]bool)
	for _, e := range out.Tree {
		paths[e.Path] = true
	}
	if !paths["a.txt"] {
		t.Error("expected a.txt in tree")
	}
	if !paths["sub"] {
		t.Error("expected sub directory in tree")
	}
}

func TestStump_DepthLimit(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "a", "b", "c"), 0755)
	os.WriteFile(filepath.Join(dir, "a", "b", "c", "deep.txt"), []byte("deep"), 0644)

	result := builtinStump(context.Background(), map[string]any{
		"dir":   dir,
		"depth": float64(1),
	}, "")

	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Error)
	}

	var out stumpResult
	json.Unmarshal([]byte(result.Output), &out)

	// At depth 1, should see "a" dir and its immediate children, but not c/deep.txt
	for _, e := range out.Tree {
		if strings.Count(e.Path, string(filepath.Separator)) > 1 {
			t.Errorf("depth=1 should not include deeply nested paths, got %q", e.Path)
		}
	}
}

func TestStump_ExtensionFilter(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.go"), []byte("go"), 0644)
	os.WriteFile(filepath.Join(dir, "b.txt"), []byte("txt"), 0644)
	os.WriteFile(filepath.Join(dir, "c.go"), []byte("go"), 0644)

	result := builtinStump(context.Background(), map[string]any{
		"dir":         dir,
		"include_ext": []any{".go"},
	}, "")

	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Error)
	}

	var out stumpResult
	json.Unmarshal([]byte(result.Output), &out)

	for _, e := range out.Tree {
		if e.Type == "f" && !strings.HasSuffix(e.Path, ".go") {
			t.Errorf("expected only .go files, got %q", e.Path)
		}
	}
	if out.Stats.Files != 2 {
		t.Errorf("expected 2 .go files, got %d", out.Stats.Files)
	}
}

func TestStump_ExcludePatterns(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "node_modules", "pkg"), 0755)
	os.MkdirAll(filepath.Join(dir, "src"), 0755)
	os.WriteFile(filepath.Join(dir, "node_modules", "pkg", "index.js"), []byte("x"), 0644)
	os.WriteFile(filepath.Join(dir, "src", "main.go"), []byte("x"), 0644)

	result := builtinStump(context.Background(), map[string]any{
		"dir":              dir,
		"exclude_patterns": []any{"node_modules"},
	}, "")

	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Error)
	}

	var out stumpResult
	json.Unmarshal([]byte(result.Output), &out)

	for _, e := range out.Tree {
		if strings.Contains(e.Path, "node_modules") {
			t.Errorf("node_modules should be excluded, got %q", e.Path)
		}
	}
}

func TestStump_ShowSize(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "test.txt"), []byte("hello world"), 0644)

	result := builtinStump(context.Background(), map[string]any{
		"dir":       dir,
		"show_size": true,
	}, "")

	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Error)
	}

	var out stumpResult
	json.Unmarshal([]byte(result.Output), &out)

	for _, e := range out.Tree {
		if e.Type == "f" && e.Size == 0 {
			t.Errorf("expected non-zero size with show_size=true for %q", e.Path)
		}
	}
}

func TestStump_HiddenFiles(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, ".hidden"), []byte("x"), 0644)
	os.WriteFile(filepath.Join(dir, "visible"), []byte("x"), 0644)

	// Without show_hidden: should skip .hidden
	result := builtinStump(context.Background(), map[string]any{
		"dir": dir,
	}, "")

	var out stumpResult
	json.Unmarshal([]byte(result.Output), &out)
	for _, e := range out.Tree {
		if strings.HasPrefix(e.Path, ".") {
			t.Errorf("hidden files should be excluded by default, got %q", e.Path)
		}
	}

	// With show_hidden: should include .hidden
	result2 := builtinStump(context.Background(), map[string]any{
		"dir":         dir,
		"show_hidden": true,
	}, "")

	var out2 stumpResult
	json.Unmarshal([]byte(result2.Output), &out2)

	found := false
	for _, e := range out2.Tree {
		if e.Path == ".hidden" {
			found = true
		}
	}
	if !found {
		t.Error("expected .hidden with show_hidden=true")
	}
}

func TestStump_NonexistentDir(t *testing.T) {
	result := builtinStump(context.Background(), map[string]any{
		"dir": "/nonexistent/dir",
	}, "")
	if !result.IsError {
		t.Fatal("expected error for nonexistent directory")
	}
}

func TestStump_FileNotDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")
	os.WriteFile(path, []byte("x"), 0644)

	result := builtinStump(context.Background(), map[string]any{
		"dir": path,
	}, "")
	if !result.IsError {
		t.Fatal("expected error for file-as-dir")
	}
}

func TestStump_MissingDir(t *testing.T) {
	result := builtinStump(context.Background(), map[string]any{}, "")
	if !result.IsError {
		t.Fatal("expected error for missing dir param")
	}
}
