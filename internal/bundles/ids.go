package bundles

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

func stableID(prefix string, parts ...string) string {
	size := 0
	for idx, part := range parts {
		if idx > 0 {
			size++
		}
		size += len(strings.TrimSpace(part))
	}
	payload := make([]byte, 0, size)
	for idx, part := range parts {
		if idx > 0 {
			payload = append(payload, '\n')
		}
		payload = append(payload, strings.TrimSpace(part)...)
	}
	sum := sha256.Sum256(payload)
	var encoded [16]byte
	hex.Encode(encoded[:], sum[:8])
	return prefix + "_" + string(encoded[:])
}
