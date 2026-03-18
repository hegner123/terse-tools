package tools

import (
	"testing"
)

func TestFilter_ExcludeNames(t *testing.T) {
	reg := DefaultRegistry()
	filtered := reg.Filter(ExcludeNames("bash", "write"))

	if _, ok := filtered.Get("bash"); ok {
		t.Error("bash should be excluded")
	}
	if _, ok := filtered.Get("write"); ok {
		t.Error("write should be excluded")
	}
	if _, ok := filtered.Get("read"); !ok {
		t.Error("read should still be present")
	}

	// Original registry unmodified
	if _, ok := reg.Get("bash"); !ok {
		t.Error("original registry should still have bash")
	}
}

func TestFilter_IncludeNames(t *testing.T) {
	reg := DefaultRegistry()
	filtered := reg.Filter(IncludeNames("read", "checkfor"))

	if filtered.Len() != 2 {
		t.Errorf("Len() = %d, want 2", filtered.Len())
	}
	if _, ok := filtered.Get("read"); !ok {
		t.Error("read should be included")
	}
	if _, ok := filtered.Get("checkfor"); !ok {
		t.Error("checkfor should be included")
	}
}

func TestFilter_ExcludeDangerous(t *testing.T) {
	reg := DefaultRegistry()
	filtered := reg.Filter(ExcludeDangerous())

	dangerous := []string{"bash", "delete", "write", "transform", "splice", "split", "repfor"}
	for _, name := range dangerous {
		if _, ok := filtered.Get(name); ok {
			t.Errorf("%s should be excluded by ExcludeDangerous", name)
		}
	}

	// Safe tools remain
	safe := []string{"read", "checkfor", "stump", "sig"}
	for _, name := range safe {
		if _, ok := filtered.Get(name); !ok {
			t.Errorf("%s should remain after ExcludeDangerous", name)
		}
	}
}

func TestFilter_ReadOnlyTools(t *testing.T) {
	reg := DefaultRegistry()
	filtered := reg.Filter(ReadOnlyTools())

	expected := map[string]bool{
		"read": true, "checkfor": true, "stump": true, "sig": true,
		"errs": true, "imports": true, "conflicts": true, "cleanDiff": true,
		"tabcount": true,
	}

	if filtered.Len() != len(expected) {
		t.Errorf("Len() = %d, want %d", filtered.Len(), len(expected))
	}

	for name := range expected {
		if _, ok := filtered.Get(name); !ok {
			t.Errorf("%s should be in ReadOnlyTools", name)
		}
	}
}

func TestFilter_PreservesOrder(t *testing.T) {
	reg := NewRegistry()
	reg.Register(ToolDef{Name: "alpha"})
	reg.Register(ToolDef{Name: "beta"})
	reg.Register(ToolDef{Name: "gamma"})
	reg.Register(ToolDef{Name: "delta"})

	filtered := reg.Filter(ExcludeNames("beta"))
	defs := filtered.Defs()

	expected := []string{"alpha", "gamma", "delta"}
	if len(defs) != len(expected) {
		t.Fatalf("got %d defs, want %d", len(defs), len(expected))
	}
	for i, name := range expected {
		if defs[i].Name != name {
			t.Errorf("defs[%d].Name = %q, want %q", i, defs[i].Name, name)
		}
	}
}

func TestFilter_EmptyResult(t *testing.T) {
	reg := DefaultRegistry()
	filtered := reg.Filter(func(def ToolDef) bool { return false })

	if filtered.Len() != 0 {
		t.Errorf("Len() = %d, want 0", filtered.Len())
	}
}

func TestFilter_NonMutating(t *testing.T) {
	reg := DefaultRegistry()
	originalLen := reg.Len()

	reg.Filter(ExcludeDangerous())

	if reg.Len() != originalLen {
		t.Errorf("original Len() changed from %d to %d", originalLen, reg.Len())
	}
}
