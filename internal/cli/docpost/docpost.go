// Package docpost transforms raw Cobra-generated markdown into
// Fumadocs-compatible MDX files with YAML frontmatter.
package docpost

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"encoding/json"
)

var (
	autoGenLine = regexp.MustCompile(`(?m)^###### Auto generated.*$\n?`)
	seeAlsoRe   = regexp.MustCompile(`(?ms)^### SEE ALSO\n.*`)
)

// Process reads all .md files from srcDir, transforms them into
// Fumadocs-compatible MDX, and writes the results to dstDir.
// It also generates a meta.json for sidebar ordering.
func Process(srcDir, dstDir string) error {
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		return fmt.Errorf("docpost: create output dir: %w", err)
	}

	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return fmt.Errorf("docpost: read source dir: %w", err)
	}

	var commandNames []string

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		data, err := os.ReadFile(filepath.Join(srcDir, entry.Name()))
		if err != nil {
			return fmt.Errorf("docpost: read %s: %w", entry.Name(), err)
		}

		cmdName := filenameToCommand(entry.Name())
		result := TransformMarkdown(string(data), cmdName)

		mdxName := strings.TrimSuffix(entry.Name(), ".md") + ".mdx"
		if err := os.WriteFile(filepath.Join(dstDir, mdxName), []byte(result), 0o600); err != nil {
			return fmt.Errorf("docpost: write %s: %w", mdxName, err)
		}

		commandNames = append(commandNames, strings.TrimSuffix(entry.Name(), ".md"))
	}

	if err := writeMetaJSON(dstDir, commandNames); err != nil {
		return fmt.Errorf("docpost: write meta.json: %w", err)
	}

	return nil
}

// TransformMarkdown converts raw Cobra markdown to Fumadocs MDX format.
// It strips boilerplate, extracts metadata, and prepends YAML frontmatter.
func TransformMarkdown(raw, cmdName string) string {
	description := extractDescription(raw)
	body := stripBoilerplate(raw)
	body = fenceIndentedBlocks(body)
	body = escapeJSX(body)
	body = rewriteLinks(body)
	body = strings.TrimSpace(body)

	var b strings.Builder
	b.WriteString("---\n")
	fmt.Fprintf(&b, "title: %q\n", cmdName)
	fmt.Fprintf(&b, "description: %q\n", description)
	b.WriteString("---\n\n")
	b.WriteString(body)
	b.WriteString("\n")

	return b.String()
}

// filenameToCommand converts a Cobra-generated filename to a command name.
// e.g. "agh_session_list.md" → "agh session list"
func filenameToCommand(filename string) string {
	name := strings.TrimSuffix(filename, ".md")
	return strings.ReplaceAll(name, "_", " ")
}

// extractDescription pulls the short description from Cobra markdown.
// Cobra generates: ## agh session list\n\nShort description here\n\n### Synopsis
// We grab the first paragraph after the H2 heading.
func extractDescription(raw string) string {
	lines := strings.Split(raw, "\n")
	inDescription := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "## ") {
			inDescription = true
			continue
		}

		if inDescription {
			if trimmed == "" {
				continue
			}
			if strings.HasPrefix(trimmed, "#") {
				break
			}
			return trimmed
		}
	}

	return ""
}

// stripBoilerplate removes Cobra auto-generated artifacts:
// - The "###### Auto generated" footer line
// - The "### SEE ALSO" section (contains local .md file links)
func stripBoilerplate(raw string) string {
	result := autoGenLine.ReplaceAllString(raw, "")
	result = seeAlsoRe.ReplaceAllString(result, "")
	return result
}

// fenceIndentedBlocks converts tab/4-space indented blocks to fenced code
// blocks. MDX does not treat indentation as code, so indented shell snippets
// with `<` or `{` cause parse errors. This function skips lines already
// inside fenced code blocks.
func fenceIndentedBlocks(raw string) string {
	lines := strings.Split(raw, "\n")
	var result []string
	inFence := false
	inIndent := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Track existing fenced code blocks.
		if strings.HasPrefix(trimmed, "```") {
			if inIndent {
				inIndent = false
				result = append(result, "```")
			}
			inFence = !inFence
			result = append(result, line)
			continue
		}

		if inFence {
			result = append(result, line)
			continue
		}

		isIndented := strings.HasPrefix(line, "\t") || strings.HasPrefix(line, "    ")
		isEmpty := trimmed == ""

		switch {
		case !inIndent && isIndented:
			inIndent = true
			result = append(result, "```", stripIndent(line))
		case inIndent && isIndented:
			result = append(result, stripIndent(line))
		case inIndent && isEmpty:
			inIndent = false
			result = append(result, "```", line)
		default:
			if inIndent {
				inIndent = false
				result = append(result, "```")
			}
			result = append(result, line)
		}
	}

	if inIndent {
		result = append(result, "```")
	}

	return strings.Join(result, "\n")
}

// stripIndent removes one level of indentation (tab or 4 spaces).
func stripIndent(line string) string {
	if strings.HasPrefix(line, "\t") {
		return line[1:]
	}
	if strings.HasPrefix(line, "    ") {
		return line[4:]
	}
	return line
}

// escapeJSX escapes bare angle brackets in non-code regions so MDX does not
// parse them as JSX tags. Lines inside fenced code blocks (```) and inline
// code spans are left untouched.
func escapeJSX(raw string) string {
	lines := strings.Split(raw, "\n")
	var result []string
	inFence := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "```") {
			inFence = !inFence
			result = append(result, line)
			continue
		}

		if inFence {
			result = append(result, line)
			continue
		}

		result = append(result, escapeLineJSX(line))
	}

	return strings.Join(result, "\n")
}

// escapeLineJSX escapes `<` and `{` in a single line, skipping inline code.
func escapeLineJSX(line string) string {
	if !strings.ContainsAny(line, "<{") {
		return line
	}

	var b strings.Builder
	inCode := false

	for i := 0; i < len(line); i++ {
		ch := line[i]
		if ch == '`' {
			inCode = !inCode
			b.WriteByte(ch)
			continue
		}
		if !inCode {
			if ch == '<' {
				b.WriteString("\\<")
				continue
			}
			if ch == '{' {
				b.WriteString("\\{")
				continue
			}
		}
		b.WriteByte(ch)
	}

	return b.String()
}

// rewriteLinks converts .md links to .mdx-compatible relative links.
// Cobra generates links like [agh session](agh_session.md) — we strip the
// .md extension since Fumadocs uses slug-based routing.
func rewriteLinks(raw string) string {
	re := regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\.md\)`)
	return re.ReplaceAllString(raw, "[$1]($2)")
}

// writeMetaJSON generates a Fumadocs meta.json for sidebar ordering.
func writeMetaJSON(dir string, commandNames []string) error {
	sort.Strings(commandNames)

	meta := struct {
		Title string   `json:"title"`
		Pages []string `json:"pages"`
	}{
		Title: "CLI Reference",
		Pages: commandNames,
	}

	data, err := json.MarshalIndent(meta, "", "    ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(dir, "meta.json"), append(data, '\n'), 0o600)
}
