package toolmeta

import "strings"

func terminalPreview(input map[string]any) string {
	command := scalarString(input["command"])
	if command == "" {
		command = scalarString(input["cmd"])
	}
	parts := splitShellOutsideQuotes(command, true)
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		pipeline := splitShellOutsideQuotes(part, false)
		if len(pipeline) == 0 {
			continue
		}
		if head := oneLine(pipeline[0]); head != "" {
			result = append(result, head)
		}
	}
	return strings.Join(result, " · ")
}

func splitShellOutsideQuotes(value string, compounds bool) []string {
	parts := make([]string, 0, 2)
	start := 0
	var quote rune
	escaped := false
	runes := []rune(value)
	for idx := 0; idx < len(runes); idx++ {
		char := runes[idx]
		if escaped {
			escaped = false
			continue
		}
		if char == '\\' && quote != '\'' {
			escaped = true
			continue
		}
		if quote != 0 {
			if char == quote {
				quote = 0
			}
			continue
		}
		if char == '\'' || char == '"' {
			quote = char
			continue
		}

		width := 0
		if compounds {
			switch {
			case char == ';':
				width = 1
			case idx+1 < len(runes) && char == '&' && runes[idx+1] == '&':
				width = 2
			case idx+1 < len(runes) && char == '|' && runes[idx+1] == '|':
				width = 2
			}
		} else if char == '|' {
			width = 1
		}
		if width == 0 {
			continue
		}
		parts = append(parts, string(runes[start:idx]))
		idx += width - 1
		start = idx + 1
	}
	parts = append(parts, string(runes[start:]))
	return parts
}
