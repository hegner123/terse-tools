package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConflicts_StandardConflict(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "conflict.txt")
	content := `before
<<<<<<< HEAD
our changes
=======
their changes
>>>>>>> feature
after
`
	os.WriteFile(path, []byte(content), 0644)

	result := builtinConflicts(context.Background(), map[string]any{
		"file": []any{path},
	}, "")

	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Error)
	}

	var out conflictsResult
	json.Unmarshal([]byte(result.Output), &out)

	if out.Total != 1 {
		t.Fatalf("expected 1 conflict, got %d", out.Total)
	}

	conflict := out.Files[0].Conflicts[0]
	if conflict.OursRef != "HEAD" {
		t.Errorf("expected ours_ref HEAD, got %q", conflict.OursRef)
	}
	if conflict.TheirsRef != "feature" {
		t.Errorf("expected theirs_ref feature, got %q", conflict.TheirsRef)
	}
	if conflict.Ours != "our changes" {
		t.Errorf("expected ours 'our changes', got %q", conflict.Ours)
	}
	if conflict.Theirs != "their changes" {
		t.Errorf("expected theirs 'their changes', got %q", conflict.Theirs)
	}
}

func TestConflicts_Diff3Style(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "conflict.txt")
	content := `<<<<<<< HEAD
our version
||||||| parent
original version
=======
their version
>>>>>>> feature
`
	os.WriteFile(path, []byte(content), 0644)

	result := builtinConflicts(context.Background(), map[string]any{
		"file": []any{path},
	}, "")

	var out conflictsResult
	json.Unmarshal([]byte(result.Output), &out)

	if !out.HasDiff3 {
		t.Error("expected has_diff3=true")
	}

	conflict := out.Files[0].Conflicts[0]
	if conflict.Base != "original version" {
		t.Errorf("expected base 'original version', got %q", conflict.Base)
	}
}

func TestConflicts_MultipleConflicts(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "multi.txt")
	content := `<<<<<<< HEAD
a
=======
b
>>>>>>> branch
middle
<<<<<<< HEAD
c
=======
d
>>>>>>> branch
`
	os.WriteFile(path, []byte(content), 0644)

	result := builtinConflicts(context.Background(), map[string]any{
		"file": []any{path},
	}, "")

	var out conflictsResult
	json.Unmarshal([]byte(result.Output), &out)

	if out.Total != 2 {
		t.Errorf("expected 2 conflicts, got %d", out.Total)
	}
}

func TestConflicts_ContextLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ctx.txt")
	content := `above1
above2
<<<<<<< HEAD
ours
=======
theirs
>>>>>>> feature
below1
below2
`
	os.WriteFile(path, []byte(content), 0644)

	result := builtinConflicts(context.Background(), map[string]any{
		"file":          []any{path},
		"context_lines": float64(2),
	}, "")

	var out conflictsResult
	json.Unmarshal([]byte(result.Output), &out)

	conflict := out.Files[0].Conflicts[0]
	if !strings.Contains(conflict.ContextAbove, "above1") {
		t.Errorf("expected context_above to contain 'above1', got %q", conflict.ContextAbove)
	}
	if !strings.Contains(conflict.ContextBelow, "below1") {
		t.Errorf("expected context_below to contain 'below1', got %q", conflict.ContextBelow)
	}
}

func TestConflicts_NoConflicts(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "clean.txt")
	os.WriteFile(path, []byte("no conflicts here\n"), 0644)

	result := builtinConflicts(context.Background(), map[string]any{
		"file": []any{path},
	}, "")

	var out conflictsResult
	json.Unmarshal([]byte(result.Output), &out)

	if out.Total != 0 {
		t.Errorf("expected 0 conflicts, got %d", out.Total)
	}
}

func TestConflicts_MultipleFiles(t *testing.T) {
	dir := t.TempDir()
	file1 := filepath.Join(dir, "a.txt")
	file2 := filepath.Join(dir, "b.txt")
	os.WriteFile(file1, []byte("<<<<<<< HEAD\na\n=======\nb\n>>>>>>> br\n"), 0644)
	os.WriteFile(file2, []byte("no conflicts\n"), 0644)

	result := builtinConflicts(context.Background(), map[string]any{
		"file": []any{file1, file2},
	}, "")

	var out conflictsResult
	json.Unmarshal([]byte(result.Output), &out)

	if len(out.Files) != 2 {
		t.Errorf("expected 2 file results, got %d", len(out.Files))
	}
	if out.Total != 1 {
		t.Errorf("expected 1 total conflict, got %d", out.Total)
	}
	if !strings.Contains(out.Summary, "1 conflict") {
		t.Errorf("unexpected summary: %q", out.Summary)
	}
}

func TestConflicts_NonexistentFile(t *testing.T) {
	result := builtinConflicts(context.Background(), map[string]any{
		"file": []any{"/nonexistent/file.txt"},
	}, "")
	if !result.IsError {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestConflicts_MissingFileParam(t *testing.T) {
	result := builtinConflicts(context.Background(), map[string]any{}, "")
	if !result.IsError {
		t.Fatal("expected error for missing file param")
	}
}

func TestConflicts_SingleStringFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	os.WriteFile(path, []byte("<<<<<<< H\nx\n=======\ny\n>>>>>>> B\n"), 0644)

	// Accept single string instead of array
	result := builtinConflicts(context.Background(), map[string]any{
		"file": path,
	}, "")

	if result.IsError {
		t.Fatalf("single string file should work: %s", result.Error)
	}

	var out conflictsResult
	json.Unmarshal([]byte(result.Output), &out)
	if out.Total != 1 {
		t.Errorf("expected 1 conflict, got %d", out.Total)
	}
}

func TestConflicts_CRLFLineEndings(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "crlf.txt")
	os.WriteFile(path, []byte("<<<<<<< HEAD\r\nours\r\n=======\r\ntheirs\r\n>>>>>>> br\r\n"), 0644)

	result := builtinConflicts(context.Background(), map[string]any{
		"file": []any{path},
	}, "")

	var out conflictsResult
	json.Unmarshal([]byte(result.Output), &out)

	if out.Total != 1 {
		t.Errorf("expected 1 conflict with CRLF, got %d", out.Total)
	}
	if out.Files[0].Conflicts[0].Ours != "ours" {
		t.Errorf("expected ours='ours' (stripped \\r), got %q", out.Files[0].Conflicts[0].Ours)
	}
}
