package sse

import "strings"

// MemoryContextRedaction is emitted when prompt-only memory fences are removed.
const MemoryContextRedaction = "[memory-context redacted]"

var memoryContextOpenMarkers = []string{
	"<memory-context",
	"<memory_context",
	"\\u003cmemory-context",
	"\\u003cmemory_context",
}

var memoryContextCloseMarkers = []string{
	"</memory-context>",
	"</memory_context>",
	"\\u003c/memory-context\\u003e",
	"\\u003c/memory_context\\u003e",
}

// ScrubMemoryContextBytes removes prompt-only memory fences from raw SSE bytes.
func ScrubMemoryContextBytes(raw []byte) []byte {
	if len(raw) == 0 {
		return raw
	}
	scrubbed := ScrubMemoryContextString(string(raw))
	if scrubbed == string(raw) {
		return raw
	}
	return []byte(scrubbed)
}

// ScrubMemoryContextString removes literal and JSON-escaped memory context fences.
func ScrubMemoryContextString(value string) string {
	if strings.TrimSpace(value) == "" {
		return value
	}

	result := value
	for {
		start, ok := nextMemoryContextOpen(result)
		if !ok {
			return result
		}
		end := len(result)
		if closeStart, closeLen, found := nextMemoryContextClose(result[start:]); found {
			end = start + closeStart + closeLen
		}
		result = result[:start] + MemoryContextRedaction + result[end:]
	}
}

func nextMemoryContextOpen(value string) (int, bool) {
	lower := strings.ToLower(value)
	best := -1
	for _, marker := range memoryContextOpenMarkers {
		offset := 0
		for {
			idx := strings.Index(lower[offset:], marker)
			if idx < 0 {
				break
			}
			candidate := offset + idx
			if memoryContextOpenBoundary(lower, candidate+len(marker)) &&
				(best < 0 || candidate < best) {
				best = candidate
			}
			offset = candidate + len(marker)
		}
	}
	if best < 0 {
		return 0, false
	}
	return best, true
}

func memoryContextOpenBoundary(lower string, after int) bool {
	if after >= len(lower) {
		return true
	}
	switch lower[after] {
	case '>', '/', ' ', '\t', '\r', '\n':
		return true
	default:
		return strings.HasPrefix(lower[after:], "\\u003e")
	}
}

func nextMemoryContextClose(value string) (int, int, bool) {
	lower := strings.ToLower(value)
	best := -1
	bestLen := 0
	for _, marker := range memoryContextCloseMarkers {
		idx := strings.Index(lower, marker)
		if idx >= 0 && (best < 0 || idx < best) {
			best = idx
			bestLen = len(marker)
		}
	}
	if best < 0 {
		return 0, 0, false
	}
	return best, bestLen, true
}
