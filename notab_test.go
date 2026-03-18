package tools

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestNotab_TabsToSpaces(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.go")
	os.WriteFile(path, []byte("\tfirst\n\t\tsecond\nthird\n"), 0644)

	result := builtinNotab(context.Background(), map[string]any{
		"file":   path,
		"spaces": float64(4),
	}, "")

	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Error)
	}

	var out notabResult
	json.Unmarshal([]byte(result.Output), &out)

	if out.Replacements != 3 { // 1 tab + 2 tabs = 3 replacements
		t.Errorf("expected 3 replacements, got %d", out.Replacements)
	}
	if out.LinesAffected != 2 {
		t.Errorf("expected 2 lines affected, got %d", out.LinesAffected)
	}
	if out.Direction != "tabs_to_spaces" {
		t.Errorf("expected direction tabs_to_spaces, got %q", out.Direction)
	}

	// Verify file content
	data, _ := os.ReadFile(path)
	content := string(data)
	if strings.Contains(content, "\t") {
		t.Error("file should not contain tabs after normalization")
	}
	if !strings.HasPrefix(content, "    first") {
		t.Errorf("expected 4-space indent, got: %q", content[:20])
	}
}

func TestNotab_SpacesToTabs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "Makefile")
	os.WriteFile(path, []byte("target:\n    echo hello\n        echo nested\n"), 0644)

	result := builtinNotab(context.Background(), map[string]any{
		"file":   path,
		"spaces": float64(4),
		"tabs":   true,
	}, "")

	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Error)
	}

	var out notabResult
	json.Unmarshal([]byte(result.Output), &out)

	if out.Direction != "spaces_to_tabs" {
		t.Errorf("expected direction spaces_to_tabs, got %q", out.Direction)
	}
	if out.LinesAffected != 2 {
		t.Errorf("expected 2 lines affected, got %d", out.LinesAffected)
	}

	data, _ := os.ReadFile(path)
	lines := strings.Split(string(data), "\n")
	if lines[1] != "\techo hello" {
		t.Errorf("expected tab indent on line 2, got %q", lines[1])
	}
	if lines[2] != "\t\techo nested" {
		t.Errorf("expected 2 tab indent on line 3, got %q", lines[2])
	}
}

func TestNotab_NoChanges(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "clean.txt")
	os.WriteFile(path, []byte("no tabs here\njust spaces\n"), 0644)

	result := builtinNotab(context.Background(), map[string]any{
		"file": path,
	}, "")

	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Error)
	}

	var out notabResult
	json.Unmarshal([]byte(result.Output), &out)

	if out.Replacements != 0 {
		t.Errorf("expected 0 replacements, got %d", out.Replacements)
	}
}

func TestNotab_MissingFile(t *testing.T) {
	result := builtinNotab(context.Background(), map[string]any{}, "")
	if !result.IsError {
		t.Fatal("expected error for missing file param")
	}
}

func TestNotab_NonexistentFile(t *testing.T) {
	result := builtinNotab(context.Background(), map[string]any{
		"file": "/nonexistent/file.txt",
	}, "")
	if !result.IsError {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestNotab_CustomSpaces(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	os.WriteFile(path, []byte("\tindented\n"), 0644)

	result := builtinNotab(context.Background(), map[string]any{
		"file":   path,
		"spaces": float64(2),
	}, "")

	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Error)
	}

	data, _ := os.ReadFile(path)
	if !strings.HasPrefix(string(data), "  indented") {
		t.Errorf("expected 2-space indent, got: %q", string(data))
	}
}

func TestNotab_PreservesPermissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	os.WriteFile(path, []byte("\ttab\n"), 0755)

	builtinNotab(context.Background(), map[string]any{
		"file": path,
	}, "")

	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0755 {
		t.Errorf("expected 0755 permissions preserved, got %o", info.Mode().Perm())
	}
}

func TestNotab_MatchesCLI(t *testing.T) {
	if _, err := exec.LookPath("notab"); err != nil {
		t.Skip("notab binary not found on PATH")
	}

	// Create two identical files — one for builtin, one for CLI
	dir := t.TempDir()
	builtinPath := filepath.Join(dir, "builtin.go")
	cliPath := filepath.Join(dir, "cli.go")
	content := []byte("\tpackage main\n\n\tfunc main() {\n\t\tfmt.Println()\n\t}\n")
	os.WriteFile(builtinPath, content, 0644)
	os.WriteFile(cliPath, content, 0644)

	// Run builtin
	builtinOut := builtinNotab(context.Background(), map[string]any{
		"file":   builtinPath,
		"spaces": float64(4),
	}, "")
	if builtinOut.IsError {
		t.Fatalf("builtin error: %s", builtinOut.Error)
	}

	// Run CLI
	cmd := exec.Command("notab", "--cli", "--file", cliPath, "--spaces", "4")
	cmd.Run() // ignore exit code (2 = no changes, which won't happen here)

	// Compare file contents
	builtinData, _ := os.ReadFile(builtinPath)
	cliData, _ := os.ReadFile(cliPath)
	if string(builtinData) != string(cliData) {
		t.Error("builtin and CLI produced different file contents")
		t.Logf("builtin: %q", string(builtinData))
		t.Logf("cli:     %q", string(cliData))
	}
}
