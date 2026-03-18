package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTransform_Count(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.json")
	os.WriteFile(path, []byte(`[{"a":1},{"a":2},{"a":3}]`), 0644)

	result := builtinTransform(context.Background(), map[string]any{
		"file": path,
		"pipeline": []any{
			map[string]any{"op": "count"},
		},
	}, "")

	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Error)
	}

	if !strings.Contains(result.Output, "3") {
		t.Errorf("expected count of 3, got %s", result.Output)
	}
}

func TestTransform_Filter(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.json")
	os.WriteFile(path, []byte(`[{"name":"a","val":1},{"name":"b","val":2},{"name":"c","val":3}]`), 0644)

	result := builtinTransform(context.Background(), map[string]any{
		"file": path,
		"pipeline": []any{
			map[string]any{"op": "filter", "key": "name", "eq": "b"},
		},
	}, "")

	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Error)
	}

	var out []map[string]any
	json.Unmarshal([]byte(result.Output), &out)
	if len(out) != 1 || out[0]["name"] != "b" {
		t.Errorf("expected filtered to 'b', got %s", result.Output)
	}
}

func TestTransform_SortBy(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.json")
	os.WriteFile(path, []byte(`[{"n":3},{"n":1},{"n":2}]`), 0644)

	result := builtinTransform(context.Background(), map[string]any{
		"file": path,
		"pipeline": []any{
			map[string]any{"op": "sort_by", "key": "n"},
		},
	}, "")

	var out []map[string]any
	json.Unmarshal([]byte(result.Output), &out)
	if len(out) != 3 || out[0]["n"] != float64(1) {
		t.Errorf("expected sorted [1,2,3], got %s", result.Output)
	}
}

func TestTransform_GroupBy(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.json")
	os.WriteFile(path, []byte(`[{"t":"a","v":1},{"t":"b","v":2},{"t":"a","v":3}]`), 0644)

	result := builtinTransform(context.Background(), map[string]any{
		"file": path,
		"pipeline": []any{
			map[string]any{"op": "group_by", "key": "t"},
		},
	}, "")

	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Error)
	}

	var out []map[string]any
	json.Unmarshal([]byte(result.Output), &out)
	if len(out) != 2 {
		t.Errorf("expected 2 groups, got %d", len(out))
	}
}

func TestTransform_Format(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.json")
	os.WriteFile(path, []byte(`[{"name":"Alice","age":30},{"name":"Bob","age":25}]`), 0644)

	result := builtinTransform(context.Background(), map[string]any{
		"file": path,
		"pipeline": []any{
			map[string]any{"op": "format", "template": "{name} is {age}"},
		},
	}, "")

	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Error)
	}

	if !strings.Contains(result.Output, "Alice is 30") {
		t.Errorf("expected formatted output, got %s", result.Output)
	}
}

func TestTransform_ExecMode(t *testing.T) {
	result := builtinTransform(context.Background(), map[string]any{
		"exec": `echo '[{"x":1},{"x":2}]'`,
		"pipeline": []any{
			map[string]any{"op": "count"},
		},
	}, "")

	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Error)
	}

	if !strings.Contains(result.Output, "2") {
		t.Errorf("expected count 2, got %s", result.Output)
	}
}

func TestTransform_DotPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.json")
	os.WriteFile(path, []byte(`[{"user":{"name":"A"}},{"user":{"name":"B"}}]`), 0644)

	result := builtinTransform(context.Background(), map[string]any{
		"file": path,
		"pipeline": []any{
			map[string]any{"op": "format", "template": "{user.name}"},
		},
	}, "")

	if !strings.Contains(result.Output, "A") || !strings.Contains(result.Output, "B") {
		t.Errorf("expected dot-path resolution, got %s", result.Output)
	}
}

func TestTransform_MissingPipeline(t *testing.T) {
	result := builtinTransform(context.Background(), map[string]any{
		"file": "/tmp/test.json",
	}, "")
	if !result.IsError {
		t.Fatal("expected error for missing pipeline")
	}
}

func TestTransform_MissingInput(t *testing.T) {
	result := builtinTransform(context.Background(), map[string]any{
		"pipeline": []any{map[string]any{"op": "count"}},
	}, "")
	if !result.IsError {
		t.Fatal("expected error for missing exec/file")
	}
}
