package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDelete_File(t *testing.T) {
	// Use HOME-based temp dir to avoid /var/folders being blocked
	home := os.Getenv("HOME")
	dir, err := os.MkdirTemp(home, ".nostop-test-delete-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %s", err)
	}
	defer os.RemoveAll(dir)

	trashDir := filepath.Join(dir, "trash")
	filePath := filepath.Join(dir, "victim.txt")
	os.WriteFile(filePath, []byte("delete me"), 0644)

	result, err := trashPathTo(filePath, trashDir)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if result.Type != "file" {
		t.Errorf("expected type 'file', got %q", result.Type)
	}
	if result.OriginalPath != filePath {
		t.Errorf("expected original_path %q, got %q", filePath, result.OriginalPath)
	}

	// Original should be gone
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Error("original file should not exist after trashing")
	}

	// Should exist in trash
	if _, err := os.Stat(result.TrashPath); err != nil {
		t.Errorf("file should exist in trash at %q: %s", result.TrashPath, err)
	}
}

func TestDelete_Directory(t *testing.T) {
	home := os.Getenv("HOME")
	dir, _ := os.MkdirTemp(home, ".nostop-test-delete-*")
	defer os.RemoveAll(dir)

	trashDir := filepath.Join(dir, "trash")
	victim := filepath.Join(dir, "victim_dir")
	os.MkdirAll(filepath.Join(victim, "sub"), 0755)
	os.WriteFile(filepath.Join(victim, "a.txt"), []byte("a"), 0644)
	os.WriteFile(filepath.Join(victim, "sub", "b.txt"), []byte("b"), 0644)

	result, err := trashPathTo(victim, trashDir)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if result.Type != "directory" {
		t.Errorf("expected type 'directory', got %q", result.Type)
	}
	if result.Items < 3 { // dir itself + sub + 2 files = at least 3
		t.Errorf("expected at least 3 items, got %d", result.Items)
	}
}

func TestDelete_NameCollision(t *testing.T) {
	home := os.Getenv("HOME")
	dir, _ := os.MkdirTemp(home, ".nostop-test-delete-*")
	defer os.RemoveAll(dir)

	trashDir := filepath.Join(dir, "trash")
	os.MkdirAll(trashDir, 0755)

	// Create file and a collision in trash
	filePath := filepath.Join(dir, "test.txt")
	os.WriteFile(filePath, []byte("original"), 0644)
	os.WriteFile(filepath.Join(trashDir, "test.txt"), []byte("existing"), 0644)

	result, err := trashPathTo(filePath, trashDir)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	// Trash path should NOT be the plain name (collision handling)
	if filepath.Base(result.TrashPath) == "test.txt" {
		t.Error("expected collision-handled name, got plain 'test.txt'")
	}
}

func TestDelete_NonexistentPath(t *testing.T) {
	dir := t.TempDir()
	trashDir := filepath.Join(dir, "trash")

	_, err := trashPathTo("/nonexistent/path", trashDir)
	if err == nil {
		t.Fatal("expected error for nonexistent path")
	}
	if !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("unexpected error: %s", err)
	}
}

func TestDelete_BlockedSystemPath(t *testing.T) {
	dir := t.TempDir()
	trashDir := filepath.Join(dir, "trash")

	for _, blocked := range []string{"/System/Library", "/usr/bin/ls", "/bin/bash"} {
		_, err := trashPathTo(blocked, trashDir)
		if err == nil {
			t.Errorf("expected error for blocked path %q", blocked)
		}
		if !strings.Contains(err.Error(), "refusing") {
			t.Errorf("expected 'refusing' error for %q, got: %s", blocked, err)
		}
	}
}

func TestDelete_BlockTrashItself(t *testing.T) {
	dir := t.TempDir()
	trashDir := filepath.Join(dir, "trash")
	os.MkdirAll(trashDir, 0755)

	_, err := trashPathTo(trashDir, trashDir)
	if err == nil {
		t.Fatal("expected error for trashing Trash itself")
	}
}

func TestDelete_MissingPath(t *testing.T) {
	result := builtinDelete(context.Background(), map[string]any{}, "")
	if !result.IsError {
		t.Fatal("expected error for missing path param")
	}
}

func TestDelete_ViaExecutor(t *testing.T) {
	home := os.Getenv("HOME")
	dir, _ := os.MkdirTemp(home, ".nostop-test-delete-*")
	defer os.RemoveAll(dir)

	filePath := filepath.Join(dir, "exec_test.txt")
	os.WriteFile(filePath, []byte("executor delete"), 0644)

	reg := NewRegistry()
	reg.Register(DeleteDef)
	exec := NewExecutor(reg, "")

	result := exec.Execute(context.Background(), "delete", map[string]any{
		"path": filePath,
	})

	if result.IsError {
		t.Fatalf("executor delete failed: %s", result.Error)
	}

	// File should be gone
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Error("file should be deleted")
	}
}
