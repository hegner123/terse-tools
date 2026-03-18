package tools

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestCheckfor_BasicSearch(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.go"), []byte("package main\nfunc hello() {}\nfunc world() {}\n"), 0644)
	os.WriteFile(filepath.Join(dir, "b.go"), []byte("package lib\nfunc helper() {}\n"), 0644)

	result := builtinCheckfor(context.Background(), map[string]any{
		"search": "func",
		"dir":    []any{dir},
	}, "")

	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Error)
	}

	var out checkforResult
	json.Unmarshal([]byte(result.Output), &out)

	if len(out.Directories) != 1 {
		t.Fatalf("expected 1 directory result, got %d", len(out.Directories))
	}

	totalMatches := out.Directories[0].MatchesFound
	if totalMatches != 3 { // 2 in a.go + 1 in b.go
		t.Errorf("expected 3 matches, got %d", totalMatches)
	}
}

func TestCheckfor_ExtFilter(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.go"), []byte("target\n"), 0644)
	os.WriteFile(filepath.Join(dir, "b.txt"), []byte("target\n"), 0644)

	result := builtinCheckfor(context.Background(), map[string]any{
		"search": "target",
		"dir":    []any{dir},
		"ext":    ".go",
	}, "")

	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Error)
	}

	var out checkforResult
	json.Unmarshal([]byte(result.Output), &out)

	if out.Directories[0].MatchesFound != 1 {
		t.Errorf("expected 1 match (only .go), got %d", out.Directories[0].MatchesFound)
	}
}

func TestCheckfor_CaseInsensitive(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "test.txt"), []byte("Hello\nhello\nHELLO\n"), 0644)

	result := builtinCheckfor(context.Background(), map[string]any{
		"search":           "hello",
		"dir":              []any{dir},
		"case_insensitive": true,
	}, "")

	var out checkforResult
	json.Unmarshal([]byte(result.Output), &out)

	if out.Directories[0].MatchesFound != 3 {
		t.Errorf("expected 3 case-insensitive matches, got %d", out.Directories[0].MatchesFound)
	}
}

func TestCheckfor_WholeWord(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "test.txt"), []byte("foo foobar barfoo foo_bar\n"), 0644)

	result := builtinCheckfor(context.Background(), map[string]any{
		"search":     "foo",
		"dir":        []any{dir},
		"whole_word": true,
	}, "")

	var out checkforResult
	json.Unmarshal([]byte(result.Output), &out)

	// "foo" appears as whole word only once (foobar and barfoo don't count; foo_bar: _ is word char)
	if out.Directories[0].MatchesFound != 1 {
		t.Errorf("expected 1 whole-word match, got %d", out.Directories[0].MatchesFound)
	}
}

func TestCheckfor_ContextLines(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "test.txt"), []byte("line1\nline2\nTARGET\nline4\nline5\n"), 0644)

	result := builtinCheckfor(context.Background(), map[string]any{
		"search":  "TARGET",
		"dir":     []any{dir},
		"context": float64(1),
	}, "")

	var out checkforResult
	json.Unmarshal([]byte(result.Output), &out)

	match := out.Directories[0].Files[0].Matches[0]
	if len(match.ContextBefore) != 1 || match.ContextBefore[0] != "line2" {
		t.Errorf("expected context_before [line2], got %v", match.ContextBefore)
	}
	if len(match.ContextAfter) != 1 || match.ContextAfter[0] != "line4" {
		t.Errorf("expected context_after [line4], got %v", match.ContextAfter)
	}
}

func TestCheckfor_Exclude(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "test.txt"), []byte("func main() {}\nfunc test_helper() {}\nfunc run() {}\n"), 0644)

	result := builtinCheckfor(context.Background(), map[string]any{
		"search":  "func",
		"dir":     []any{dir},
		"exclude": []any{"test_"},
	}, "")

	var out checkforResult
	json.Unmarshal([]byte(result.Output), &out)

	if out.Directories[0].MatchesFound != 2 { // "func main" and "func run", not "func test_helper"
		t.Errorf("expected 2 matches after exclude, got %d", out.Directories[0].MatchesFound)
	}
}

func TestCheckfor_DirectFileSearch(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "specific.txt")
	os.WriteFile(path, []byte("needle in haystack\n"), 0644)

	result := builtinCheckfor(context.Background(), map[string]any{
		"search": "needle",
		"file":   []any{path},
	}, "")

	var out checkforResult
	json.Unmarshal([]byte(result.Output), &out)

	// Direct file search uses "(files)" as the dir name
	found := false
	for _, d := range out.Directories {
		if d.MatchesFound > 0 {
			found = true
		}
	}
	if !found {
		t.Error("expected match in direct file search")
	}
}

func TestCheckfor_NoMatches(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "test.txt"), []byte("nothing here\n"), 0644)

	result := builtinCheckfor(context.Background(), map[string]any{
		"search": "nonexistent_string_xyz",
		"dir":    []any{dir},
	}, "")

	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Error)
	}

	var out checkforResult
	json.Unmarshal([]byte(result.Output), &out)
	if out.Directories[0].MatchesFound != 0 {
		t.Errorf("expected 0 matches, got %d", out.Directories[0].MatchesFound)
	}
}

func TestCheckfor_MultilineSearch(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "test.txt"), []byte("first\nsecond\nthird\n"), 0644)

	result := builtinCheckfor(context.Background(), map[string]any{
		"search": "first\nsecond",
		"dir":    []any{dir},
	}, "")

	var out checkforResult
	json.Unmarshal([]byte(result.Output), &out)

	if out.Directories[0].MatchesFound != 1 {
		t.Errorf("expected 1 multiline match, got %d", out.Directories[0].MatchesFound)
	}

	match := out.Directories[0].Files[0].Matches[0]
	if match.Line != 1 {
		t.Errorf("expected match at line 1, got %d", match.Line)
	}
	if match.EndLine != 2 {
		t.Errorf("expected end_line 2, got %d", match.EndLine)
	}
}

func TestCheckfor_MissingSearch(t *testing.T) {
	result := builtinCheckfor(context.Background(), map[string]any{}, "")
	if !result.IsError {
		t.Fatal("expected error for missing search param")
	}
}

func TestCheckfor_DefaultDirectory(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "test.txt"), []byte("findme\n"), 0644)

	// No dir or file specified — should use workDir
	result := builtinCheckfor(context.Background(), map[string]any{
		"search": "findme",
	}, dir)

	var out checkforResult
	json.Unmarshal([]byte(result.Output), &out)

	if len(out.Directories) == 0 || out.Directories[0].MatchesFound != 1 {
		t.Error("expected 1 match using default workDir")
	}
}

func TestCheckfor_MatchesCLI(t *testing.T) {
	if _, err := exec.LookPath("checkfor"); err != nil {
		t.Skip("checkfor binary not found on PATH")
	}

	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "test.go"), []byte("package main\n\nfunc main() {\n\tfmt.Println(\"hello\")\n}\n"), 0644)

	// Run builtin
	builtinOut := builtinCheckfor(context.Background(), map[string]any{
		"search": "func",
		"dir":    []any{dir},
	}, "")
	if builtinOut.IsError {
		t.Fatalf("builtin error: %s", builtinOut.Error)
	}

	// Run CLI
	cmd := exec.Command("checkfor", "--cli", "--search", "func", "--dir", dir)
	cliBytes, err := cmd.Output()
	if err != nil {
		t.Fatalf("CLI error: %s", err)
	}

	var builtinJSON, cliJSON checkforResult
	json.Unmarshal([]byte(builtinOut.Output), &builtinJSON)
	json.Unmarshal(cliBytes, &cliJSON)

	if len(builtinJSON.Directories) != len(cliJSON.Directories) {
		t.Fatalf("directory count mismatch: builtin=%d cli=%d", len(builtinJSON.Directories), len(cliJSON.Directories))
	}
	if builtinJSON.Directories[0].MatchesFound != cliJSON.Directories[0].MatchesFound {
		t.Errorf("matches_found mismatch: builtin=%d cli=%d",
			builtinJSON.Directories[0].MatchesFound, cliJSON.Directories[0].MatchesFound)
	}
}
