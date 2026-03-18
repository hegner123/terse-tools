package tools

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestCleanDiff_EmptyDiff(t *testing.T) {
	// Create a temp git repo with no changes
	dir := t.TempDir()
	exec.Command("git", "-C", dir, "init").Run()
	exec.Command("git", "-C", dir, "config", "user.email", "test@test.com").Run()
	exec.Command("git", "-C", dir, "config", "user.name", "Test").Run()
	os.WriteFile(filepath.Join(dir, "test.txt"), []byte("content\n"), 0644)
	exec.Command("git", "-C", dir, "add", ".").Run()
	exec.Command("git", "-C", dir, "commit", "-m", "init").Run()

	result := builtinCleanDiff(context.Background(), map[string]any{
		"path": dir,
	}, "")

	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Error)
	}

	var out cleanDiffResult
	json.Unmarshal([]byte(result.Output), &out)

	if out.Summary.FilesChanged != 0 {
		t.Errorf("expected 0 files changed, got %d", out.Summary.FilesChanged)
	}
}

func TestCleanDiff_UnstagedChanges(t *testing.T) {
	dir := t.TempDir()
	exec.Command("git", "-C", dir, "init").Run()
	exec.Command("git", "-C", dir, "config", "user.email", "test@test.com").Run()
	exec.Command("git", "-C", dir, "config", "user.name", "Test").Run()
	os.WriteFile(filepath.Join(dir, "test.txt"), []byte("original\n"), 0644)
	exec.Command("git", "-C", dir, "add", ".").Run()
	exec.Command("git", "-C", dir, "commit", "-m", "init").Run()

	// Make unstaged change
	os.WriteFile(filepath.Join(dir, "test.txt"), []byte("modified\n"), 0644)

	result := builtinCleanDiff(context.Background(), map[string]any{
		"path": dir,
	}, "")

	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Error)
	}

	var out cleanDiffResult
	json.Unmarshal([]byte(result.Output), &out)

	if out.Summary.FilesChanged != 1 {
		t.Errorf("expected 1 file changed, got %d", out.Summary.FilesChanged)
	}
	if out.Summary.Insertions != 1 {
		t.Errorf("expected 1 insertion, got %d", out.Summary.Insertions)
	}
	if out.Summary.Deletions != 1 {
		t.Errorf("expected 1 deletion, got %d", out.Summary.Deletions)
	}
}

func TestCleanDiff_StagedChanges(t *testing.T) {
	dir := t.TempDir()
	exec.Command("git", "-C", dir, "init").Run()
	exec.Command("git", "-C", dir, "config", "user.email", "test@test.com").Run()
	exec.Command("git", "-C", dir, "config", "user.name", "Test").Run()
	os.WriteFile(filepath.Join(dir, "test.txt"), []byte("original\n"), 0644)
	exec.Command("git", "-C", dir, "add", ".").Run()
	exec.Command("git", "-C", dir, "commit", "-m", "init").Run()

	// Stage a change
	os.WriteFile(filepath.Join(dir, "test.txt"), []byte("staged\n"), 0644)
	exec.Command("git", "-C", dir, "add", "test.txt").Run()

	result := builtinCleanDiff(context.Background(), map[string]any{
		"path":   dir,
		"staged": true,
	}, "")

	var out cleanDiffResult
	json.Unmarshal([]byte(result.Output), &out)

	if out.Summary.FilesChanged != 1 {
		t.Errorf("expected 1 staged file changed, got %d", out.Summary.FilesChanged)
	}
}

func TestCleanDiff_StatOnly(t *testing.T) {
	dir := t.TempDir()
	exec.Command("git", "-C", dir, "init").Run()
	exec.Command("git", "-C", dir, "config", "user.email", "test@test.com").Run()
	exec.Command("git", "-C", dir, "config", "user.name", "Test").Run()
	os.WriteFile(filepath.Join(dir, "test.txt"), []byte("original\n"), 0644)
	exec.Command("git", "-C", dir, "add", ".").Run()
	exec.Command("git", "-C", dir, "commit", "-m", "init").Run()
	os.WriteFile(filepath.Join(dir, "test.txt"), []byte("modified\n"), 0644)

	result := builtinCleanDiff(context.Background(), map[string]any{
		"path":      dir,
		"stat_only": true,
	}, "")

	var out cleanDiffResult
	json.Unmarshal([]byte(result.Output), &out)

	if len(out.Files) == 0 {
		t.Fatal("expected at least 1 file")
	}
	if out.Files[0].Hunks != nil {
		t.Error("stat_only should omit hunks")
	}
}

func TestCleanDiff_NotAGitRepo(t *testing.T) {
	dir := t.TempDir()

	result := builtinCleanDiff(context.Background(), map[string]any{
		"path": dir,
	}, "")

	if !result.IsError {
		t.Fatal("expected error for non-git directory")
	}
}

func TestCleanDiff_ParseDiffOutput(t *testing.T) {
	raw := `diff --git a/file.go b/file.go
index abc..def 100644
--- a/file.go
+++ b/file.go
@@ -1,3 +1,4 @@ package main
 func main() {
+	fmt.Println("hello")
 }
`

	files := parseUnifiedDiff(raw, false)
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if files[0].Path != "file.go" {
		t.Errorf("expected path file.go, got %q", files[0].Path)
	}
	if files[0].Insertions != 1 {
		t.Errorf("expected 1 insertion, got %d", files[0].Insertions)
	}
}

func TestCleanDiff_ParseRename(t *testing.T) {
	raw := `diff --git a/old.go b/new.go
similarity index 95%
rename from old.go
rename to new.go
`

	files := parseUnifiedDiff(raw, false)
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if files[0].Status != "renamed" {
		t.Errorf("expected status renamed, got %q", files[0].Status)
	}
	if files[0].OldPath != "old.go" {
		t.Errorf("expected old_path old.go, got %q", files[0].OldPath)
	}
	if files[0].Path != "new.go" {
		t.Errorf("expected path new.go, got %q", files[0].Path)
	}
}
