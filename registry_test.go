package tools

import (
	"testing"
)

func TestPhase3_BuiltinsInRegistry(t *testing.T) {
	reg := DefaultRegistry()

	for _, name := range []string{"checkfor", "conflicts"} {
		def, ok := reg.Get(name)
		if !ok {
			t.Errorf("%q not found in default registry", name)
			continue
		}
		if !def.IsBuiltinTool() {
			t.Errorf("%q should be a builtin", name)
		}
	}
}

func TestPhase3_TotalToolCount(t *testing.T) {
	reg := DefaultRegistry()
	// 7 subprocess + 11 builtins
	if reg.Len() != 18 {
		t.Errorf("expected 18 tools, got %d", reg.Len())
		t.Logf("tools: %v", reg.Names())
	}
}

func TestPhase4_BuiltinsInRegistry(t *testing.T) {
	reg := DefaultRegistry()

	for _, name := range []string{"repfor", "cleanDiff"} {
		def, ok := reg.Get(name)
		if !ok {
			t.Errorf("%q not found in default registry", name)
			continue
		}
		if !def.IsBuiltinTool() {
			t.Errorf("%q should be a builtin", name)
		}
	}
}

func TestPhase4_TotalToolCount(t *testing.T) {
	reg := DefaultRegistry()
	// 5 subprocess + 13 builtins
	if reg.Len() != 18 {
		t.Errorf("expected 18 tools, got %d", reg.Len())
		t.Logf("tools: %v", reg.Names())
	}
}

func TestPhase5_BuiltinsInRegistry(t *testing.T) {
	reg := DefaultRegistry()

	for _, name := range []string{"utf8", "imports"} {
		def, ok := reg.Get(name)
		if !ok {
			t.Errorf("%q not found in default registry", name)
			continue
		}
		if !def.IsBuiltinTool() {
			t.Errorf("%q should be a builtin", name)
		}
	}
}

func TestPhase5_TotalToolCount(t *testing.T) {
	reg := DefaultRegistry()
	// 3 subprocess (sig, errs, transform) + 15 builtins
	if reg.Len() != 18 {
		t.Errorf("expected 18 tools, got %d", reg.Len())
		t.Logf("tools: %v", reg.Names())
	}
}

func TestPhase6_BuiltinsInRegistry(t *testing.T) {
	reg := DefaultRegistry()

	for _, name := range []string{"sig", "transform", "errs"} {
		def, ok := reg.Get(name)
		if !ok {
			t.Errorf("%q not found in default registry", name)
			continue
		}
		if !def.IsBuiltinTool() {
			t.Errorf("%q should be a builtin", name)
		}
	}
}

func TestPhase6_AllToolsAreBuiltins(t *testing.T) {
	reg := DefaultRegistry()

	// All 18 tools should be builtins (15 terse + 3 core)
	if reg.Len() != 18 {
		t.Errorf("expected 18 tools, got %d", reg.Len())
		t.Logf("tools: %v", reg.Names())
	}

	// Every tool should be a builtin
	for _, name := range reg.Names() {
		def, _ := reg.Get(name)
		if !def.IsBuiltinTool() {
			t.Errorf("tool %q is not a builtin", name)
		}
	}
}

func TestPhase6_NoBinaryDependencies(t *testing.T) {
	reg := DefaultRegistry()
	missing := reg.CheckBinaries()

	// Should be empty — all builtins, no binaries needed
	if len(missing) != 0 {
		t.Errorf("expected no missing binaries, got %v", missing)
	}
}

func TestPhase1_BuiltinsInRegistry(t *testing.T) {
	reg := DefaultRegistry()

	for _, name := range []string{"tabcount", "notab", "delete"} {
		def, ok := reg.Get(name)
		if !ok {
			t.Errorf("%q not found in default registry", name)
			continue
		}
		if !def.IsBuiltinTool() {
			t.Errorf("%q should be a builtin, not subprocess", name)
		}
		if def.Binary != "" {
			t.Errorf("%q should have empty Binary field, got %q", name, def.Binary)
		}
	}
}

func TestPhase1_TotalToolCount(t *testing.T) {
	reg := DefaultRegistry()
	// 12 subprocess + 6 builtins (read, write, bash, tabcount, notab, delete)
	if reg.Len() != 18 {
		t.Errorf("expected 18 tools, got %d", reg.Len())
		t.Logf("tools: %v", reg.Names())
	}
}

func TestPhase2_BuiltinsInRegistry(t *testing.T) {
	reg := DefaultRegistry()

	for _, name := range []string{"split", "splice", "stump"} {
		def, ok := reg.Get(name)
		if !ok {
			t.Errorf("%q not found in default registry", name)
			continue
		}
		if !def.IsBuiltinTool() {
			t.Errorf("%q should be a builtin", name)
		}
	}
}

func TestPhase2_TotalToolCount(t *testing.T) {
	reg := DefaultRegistry()
	// 9 subprocess + 9 builtins (read, write, bash, tabcount, notab, delete, split, splice, stump)
	if reg.Len() != 18 {
		t.Errorf("expected 18 tools, got %d", reg.Len())
		t.Logf("tools: %v", reg.Names())
	}
}
