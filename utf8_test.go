package tools

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUTF8_AlreadyUTF8(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "clean.txt")
	os.WriteFile(path, []byte("Hello, world!\n"), 0644)

	result := builtinUTF8(context.Background(), map[string]any{
		"file": path,
	}, "")

	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Error)
	}

	var out utf8Result
	json.Unmarshal([]byte(result.Output), &out)

	if out.Status != "already_utf8" {
		t.Errorf("expected status already_utf8, got %q", out.Status)
	}
	if out.Detected != utf8EncUTF8 {
		t.Errorf("expected detected utf8, got %q", out.Detected)
	}
}

func TestUTF8_UTF8BOM(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bom.txt")
	bom := []byte{0xEF, 0xBB, 0xBF}
	os.WriteFile(path, append(bom, []byte("Hello BOM\n")...), 0644)

	result := builtinUTF8(context.Background(), map[string]any{
		"file":   path,
		"backup": false,
	}, "")

	var out utf8Result
	json.Unmarshal([]byte(result.Output), &out)

	if out.Status != "converted" {
		t.Errorf("expected converted, got %q", out.Status)
	}
	if out.Detected != utf8EncUTF8BOM {
		t.Errorf("expected detected utf8_bom, got %q", out.Detected)
	}

	// File should have BOM stripped
	data, _ := os.ReadFile(path)
	if data[0] == 0xEF {
		t.Error("BOM should have been stripped")
	}
}

func TestUTF8_NullLaced(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nulls.txt")
	// Null-laced ASCII: 'H' 0x00 'i' 0x00
	os.WriteFile(path, []byte{'H', 0, 'i', 0, '\n', 0}, 0644)

	result := builtinUTF8(context.Background(), map[string]any{
		"file":   path,
		"backup": false,
	}, "")

	var out utf8Result
	json.Unmarshal([]byte(result.Output), &out)

	if out.Status != "converted" {
		t.Errorf("expected converted, got %q", out.Status)
	}

	data, _ := os.ReadFile(path)
	if strings.Contains(string(data), "\x00") {
		t.Error("null bytes should have been stripped")
	}
}

func TestUTF8_UTF16LE(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "utf16le.txt")

	// UTF-16 LE BOM + "Hi"
	var content []byte
	content = append(content, 0xFF, 0xFE) // BOM
	buf := make([]byte, 2)
	for _, r := range "Hi\n" {
		binary.LittleEndian.PutUint16(buf, uint16(r))
		content = append(content, buf...)
	}
	os.WriteFile(path, content, 0644)

	result := builtinUTF8(context.Background(), map[string]any{
		"file":   path,
		"backup": false,
	}, "")

	var out utf8Result
	json.Unmarshal([]byte(result.Output), &out)

	if out.Status != "converted" {
		t.Fatalf("expected converted, got %q", out.Status)
	}

	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "Hi") {
		t.Errorf("expected 'Hi' in converted output, got %q", string(data))
	}
}

func TestUTF8_Backup(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	bom := []byte{0xEF, 0xBB, 0xBF}
	os.WriteFile(path, append(bom, []byte("content")...), 0644)

	result := builtinUTF8(context.Background(), map[string]any{
		"file":   path,
		"backup": true,
	}, "")

	var out utf8Result
	json.Unmarshal([]byte(result.Output), &out)

	if out.Backup == "" {
		t.Error("expected backup path")
	}

	// Backup should exist
	if _, err := os.Stat(out.Backup); err != nil {
		t.Errorf("backup file should exist: %s", err)
	}
}

func TestUTF8_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.txt")
	os.WriteFile(path, []byte{}, 0644)

	result := builtinUTF8(context.Background(), map[string]any{
		"file": path,
	}, "")

	var out utf8Result
	json.Unmarshal([]byte(result.Output), &out)

	if out.Status != "already_utf8" {
		t.Errorf("empty file should be already_utf8, got %q", out.Status)
	}
}

func TestUTF8_MissingFile(t *testing.T) {
	result := builtinUTF8(context.Background(), map[string]any{
		"file": "/nonexistent/file.txt",
	}, "")
	if !result.IsError {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestUTF8_MissingParam(t *testing.T) {
	result := builtinUTF8(context.Background(), map[string]any{}, "")
	if !result.IsError {
		t.Fatal("expected error for missing file param")
	}
}
