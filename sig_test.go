package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSig_TypeScript(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.ts")
	os.WriteFile(path, []byte(`export interface User {
  id: number;
  name: string;
}

export function greet(user: User): string {
  return "Hello " + user.name;
}

export const MAX_USERS = 100;
`), 0644)

	result := builtinSig(context.Background(), map[string]any{
		"file": path,
	}, "")

	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Error)
	}

	var out FileShape
	json.Unmarshal([]byte(result.Output), &out)

	if len(out.Types) == 0 {
		t.Error("expected types from TS file")
	}
	if len(out.Functions) == 0 {
		t.Error("expected functions from TS file")
	}
}

func TestSig_CSharp(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "Test.cs")
	os.WriteFile(path, []byte(`using System;

namespace MyApp
{
    public class User
    {
        public int Id { get; set; }
        public string Name { get; set; }
    }

    public interface IService
    {
        void Process();
    }
}
`), 0644)

	result := builtinSig(context.Background(), map[string]any{
		"file": path,
	}, "")

	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Error)
	}

	var out FileShape
	json.Unmarshal([]byte(result.Output), &out)

	if len(out.Types) == 0 {
		t.Error("expected types from C# file")
	}
}

func TestSig_UnsupportedExt(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.rb")
	os.WriteFile(path, []byte("class Foo; end"), 0644)

	result := builtinSig(context.Background(), map[string]any{
		"file": path,
	}, "")

	if !result.IsError {
		t.Fatal("expected error for unsupported extension")
	}
}

func TestSig_MissingFile(t *testing.T) {
	result := builtinSig(context.Background(), map[string]any{
		"file": "/nonexistent/file.go",
	}, "")
	if !result.IsError {
		t.Fatal("expected error for missing file")
	}
}

func TestSig_MissingParam(t *testing.T) {
	result := builtinSig(context.Background(), map[string]any{}, "")
	if !result.IsError {
		t.Fatal("expected error for missing file param")
	}
}
