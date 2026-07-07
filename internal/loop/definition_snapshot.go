package loop

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/compozy/agh/internal/loop/dsl"
)

// DefinitionSnapshotJSON returns the canonical JSON body and digest pinned by a run.
func DefinitionSnapshotJSON(def dsl.Definition) (json.RawMessage, string, error) {
	normalized := def
	normalized.Normalize()
	data, err := json.Marshal(normalized)
	if err != nil {
		return nil, "", fmt.Errorf("loop: marshal definition snapshot: %w", err)
	}
	sum := sha256.Sum256(data)
	digest := "sha256:" + hex.EncodeToString(sum[:])
	return json.RawMessage(data), digest, nil
}
