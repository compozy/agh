package builtin

import toolspkg "github.com/pedronauck/agh/internal/tools"

// Source returns the provenance shared by daemon-compiled AGH tools.
func Source() toolspkg.SourceRef {
	return toolspkg.BuiltinSource()
}
