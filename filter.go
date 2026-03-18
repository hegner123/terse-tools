package tools

// FilterFunc decides whether a ToolDef should be included in a filtered registry.
// Return true to include, false to exclude.
type FilterFunc func(def ToolDef) bool

// Filter returns a new Registry containing only tools that pass fn.
// The original registry is not modified.
func (r *Registry) Filter(fn FilterFunc) *Registry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	filtered := NewRegistry()
	for _, name := range r.order {
		def := r.tools[name]
		if fn(def) {
			filtered.Register(def)
		}
	}
	return filtered
}

// ExcludeNames returns a FilterFunc that excludes tools with any of the given names.
func ExcludeNames(names ...string) FilterFunc {
	excluded := make(map[string]bool, len(names))
	for _, n := range names {
		excluded[n] = true
	}
	return func(def ToolDef) bool {
		return !excluded[def.Name]
	}
}

// IncludeNames returns a FilterFunc that includes only tools with one of the given names.
func IncludeNames(names ...string) FilterFunc {
	included := make(map[string]bool, len(names))
	for _, n := range names {
		included[n] = true
	}
	return func(def ToolDef) bool {
		return included[def.Name]
	}
}

// ExcludeDangerous returns a FilterFunc that removes tools capable of
// writing, deleting, or executing arbitrary commands.
func ExcludeDangerous() FilterFunc {
	return ExcludeNames("bash", "delete", "write", "transform", "splice", "split", "repfor")
}

// ReadOnlyTools returns a FilterFunc that keeps only read-only,
// non-destructive tools.
func ReadOnlyTools() FilterFunc {
	return IncludeNames(
		"read", "checkfor", "stump", "sig", "errs",
		"imports", "conflicts", "cleanDiff", "tabcount",
	)
}
