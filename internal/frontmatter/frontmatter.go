package frontmatter

import (
	"bytes"
	"errors"
	"strings"
)

const delimiter = "---"

var (
	// ErrMissing reports content that does not start with a valid YAML frontmatter block.
	ErrMissing = errors.New("frontmatter: missing YAML frontmatter")
	// ErrUnterminated reports content whose opening delimiter has no matching closing delimiter.
	ErrUnterminated = errors.New("frontmatter: unterminated YAML frontmatter")
)

// Parts contains the parsed metadata bytes and normalized markdown body.
type Parts struct {
	Metadata []byte
	Body     string
}

// Split normalizes line endings and separates YAML frontmatter from the body.
func Split(content []byte) (Parts, error) {
	normalized := normalizeLineEndings(content)
	if !bytes.HasPrefix(normalized, []byte(delimiter)) {
		return Parts{}, ErrMissing
	}

	openLineEnd := nextLineBoundary(normalized, 0)
	if string(normalized[:openLineEnd]) != delimiter {
		return Parts{}, ErrMissing
	}

	offset := openLineEnd
	if offset < len(normalized) && normalized[offset] == '\n' {
		offset++
	}

	closeStart, closeEnd, ok := findClosingDelimiter(normalized, offset)
	if !ok {
		return Parts{}, ErrUnterminated
	}

	bodyStart := closeEnd
	if bodyStart < len(normalized) && normalized[bodyStart] == '\n' {
		bodyStart++
	}

	return Parts{
		Metadata: normalized[offset:closeStart],
		Body:     string(normalized[bodyStart:]),
	}, nil
}

// Decode splits frontmatter and delegates metadata decoding to the supplied callback.
func Decode(content []byte, decode func([]byte) error) (string, error) {
	parts, err := Split(content)
	if err != nil {
		return "", err
	}
	if err := decode(parts.Metadata); err != nil {
		return "", err
	}

	return parts.Body, nil
}

func normalizeLineEndings(content []byte) []byte {
	return []byte(strings.ReplaceAll(string(content), "\r\n", "\n"))
}

func nextLineBoundary(content []byte, start int) int {
	if start >= len(content) {
		return len(content)
	}

	if idx := bytes.IndexByte(content[start:], '\n'); idx >= 0 {
		return start + idx
	}

	return len(content)
}

func findClosingDelimiter(content []byte, start int) (int, int, bool) {
	lineStart := start
	for lineStart <= len(content) {
		lineEnd := nextLineBoundary(content, lineStart)
		if string(content[lineStart:lineEnd]) == delimiter {
			return lineStart, lineEnd, true
		}
		if lineEnd == len(content) {
			break
		}
		lineStart = lineEnd + 1
	}

	return 0, 0, false
}
