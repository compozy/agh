// Package docpost transforms raw Cobra-generated markdown into
// Fumadocs-compatible MDX files with YAML frontmatter. Output is grouped
// into a nested directory structure by command family so the Fumadocs
// sidebar collapses the CLI reference. The generated root command page is
// written as agh.mdx so index.mdx can remain a hand-authored overview.
package docpost

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// linkBasePath is the URL prefix the site router mounts the CLI reference at.
// We rewrite inter-command links to use absolute paths under this prefix so
// they resolve the same regardless of which nested page they live on.
const linkBasePath = "/runtime/cli-reference"

var (
	autoGenLine  = regexp.MustCompile(`(?m)^###### Auto generated.*$\n?`)
	seeAlsoRe    = regexp.MustCompile(`(?ms)^### SEE ALSO\n.*`)
	crossLinkRe  = regexp.MustCompile(`\[([^\]]+)\]\((agh[A-Za-z0-9_\-]*)\.md\)`)
	strippedLink = regexp.MustCompile(`\]\((agh[A-Za-z0-9_\-]*)\)`)
)

// Process reads all agh*.md files from srcDir, transforms them into
// Fumadocs-compatible MDX, and writes them to dstDir using a nested
// directory layout: `agh` → agh.mdx, `agh_agent` → agent/index.mdx,
// `agh_agent_list` → agent/list.mdx, and so on.
//
// The root-level index.mdx and meta.json of dstDir are hand-maintained and
// never touched by Process. Subdirectory meta.json files are regenerated on
// each run.
// Stale files from prior runs are removed before writing.
func Process(ctx context.Context, srcDir, dstDir string) error {
	if err := ensureContext(ctx, "start doc post-processing"); err != nil {
		return err
	}
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		return fmt.Errorf("docpost: create output dir: %w", err)
	}
	if err := cleanOutput(ctx, dstDir); err != nil {
		return err
	}

	inputs, err := readInputs(ctx, srcDir)
	if err != nil {
		return err
	}

	hasChildren := computeHasChildren(inputs)
	targets := buildTargetMap(inputs, hasChildren)

	for _, in := range inputs {
		if err := ensureContext(ctx, fmt.Sprintf("write %s", in.fileName)); err != nil {
			return err
		}
		cmdName := strings.ReplaceAll(in.baseName, "_", " ")
		body := TransformMarkdown(in.raw, cmdName)
		body = remapLinks(body, targets)

		outRel := outPath(in, hasChildren)
		dst := filepath.Join(dstDir, filepath.FromSlash(outRel))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return fmt.Errorf("docpost: mkdir %s: %w", dst, err)
		}
		if err := os.WriteFile(dst, []byte(body), 0o600); err != nil {
			return fmt.Errorf("docpost: write %s: %w", dst, err)
		}
	}

	return writeSubdirMetas(ctx, dstDir)
}

type input struct {
	fileName string
	baseName string
	segments []string
	raw      string
}

func readInputs(ctx context.Context, srcDir string) ([]input, error) {
	if err := ensureContext(ctx, fmt.Sprintf("read source dir %s", srcDir)); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return nil, fmt.Errorf("docpost: read source dir %s: %w", srcDir, err)
	}

	var inputs []input
	for _, entry := range entries {
		fullPath := filepath.Join(srcDir, entry.Name())
		if err := ensureContext(ctx, fmt.Sprintf("read %s", fullPath)); err != nil {
			return nil, err
		}
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		data, err := os.ReadFile(fullPath)
		if err != nil {
			return nil, fmt.Errorf("docpost: read %s: %w", fullPath, err)
		}
		base := strings.TrimSuffix(entry.Name(), ".md")
		if !strings.HasPrefix(base, "agh") {
			return nil, fmt.Errorf("docpost: unexpected filename %q (must start with 'agh')", entry.Name())
		}
		var segs []string
		if base != "agh" {
			parts := strings.Split(base, "_")
			segs = parts[1:]
		}
		inputs = append(inputs, input{
			fileName: entry.Name(),
			baseName: base,
			segments: segs,
			raw:      string(data),
		})
	}
	return inputs, nil
}

// computeHasChildren marks a baseName as a "parent" when any other input's
// baseName starts with baseName + "_". Parents render as index.mdx in their
// own directory; leaves render as <segment>.mdx under the parent directory.
func computeHasChildren(inputs []input) map[string]bool {
	result := make(map[string]bool, len(inputs))
	for _, a := range inputs {
		prefix := a.baseName + "_"
		for _, b := range inputs {
			if a.baseName != b.baseName && strings.HasPrefix(b.baseName, prefix) {
				result[a.baseName] = true
				break
			}
		}
	}
	return result
}

// outPath returns the output file path for an input, relative to dstDir, using
// forward slashes.
func outPath(in input, hasChildren map[string]bool) string {
	if len(in.segments) == 0 {
		return "agh.mdx"
	}
	if hasChildren[in.baseName] {
		return path.Join(in.segments...) + "/index.mdx"
	}
	if len(in.segments) == 1 {
		return in.segments[0] + ".mdx"
	}
	parent := path.Join(in.segments[:len(in.segments)-1]...)
	return parent + "/" + in.segments[len(in.segments)-1] + ".mdx"
}

// buildTargetMap builds a baseName -> absolute URL map used by remapLinks.
// The root command maps to the generated agh page; every other command maps
// to linkBasePath + its segment path.
func buildTargetMap(inputs []input, _ map[string]bool) map[string]string {
	targets := make(map[string]string, len(inputs))
	for _, in := range inputs {
		if len(in.segments) == 0 {
			targets[in.baseName] = linkBasePath + "/agh"
			continue
		}
		targets[in.baseName] = linkBasePath + "/" + strings.Join(in.segments, "/")
	}
	return targets
}

// remapLinks rewrites any `](agh_xxx)` link target in the body to its
// absolute URL under linkBasePath. Runs after TransformMarkdown has already
// stripped `.md` extensions via rewriteLinks.
func remapLinks(body string, targets map[string]string) string {
	return strippedLink.ReplaceAllStringFunc(body, func(match string) string {
		m := strippedLink.FindStringSubmatch(match)
		if m == nil {
			return match
		}
		target, ok := targets[m[1]]
		if !ok {
			return match
		}
		return "](" + target + ")"
	})
}

// cleanOutput removes generated files in dstDir while preserving the root
// hand-maintained index.mdx and meta.json.
func cleanOutput(ctx context.Context, dstDir string) error {
	if err := ensureContext(ctx, fmt.Sprintf("clean output dir %s", dstDir)); err != nil {
		return err
	}
	entries, err := os.ReadDir(dstDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("docpost: read output dir %s: %w", dstDir, err)
	}
	for _, e := range entries {
		if e.Name() == "index.mdx" || e.Name() == "meta.json" {
			continue
		}
		target := filepath.Join(dstDir, e.Name())
		if err := ensureContext(ctx, fmt.Sprintf("remove stale entry %s", target)); err != nil {
			return err
		}
		if err := os.RemoveAll(target); err != nil {
			return fmt.Errorf("docpost: remove stale entry %s: %w", target, err)
		}
	}
	return nil
}

// writeSubdirMetas walks dstDir and emits a meta.json in every subdirectory
// (never the root) listing its direct children alphabetically, with an
// optional leading "index" when an index.mdx is present.
func writeSubdirMetas(ctx context.Context, dstDir string) error {
	return filepath.WalkDir(dstDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("docpost: walk %s: %w", p, err)
		}
		if err := ensureContext(ctx, fmt.Sprintf("write meta for %s", p)); err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dstDir, p)
		if err != nil {
			return fmt.Errorf("docpost: relative path from %s to %s: %w", dstDir, p, err)
		}
		if rel == "." {
			return nil // root is hand-maintained
		}
		return writeDirMeta(ctx, p)
	})
}

func writeDirMeta(ctx context.Context, dir string) error {
	if err := ensureContext(ctx, fmt.Sprintf("write meta for %s", dir)); err != nil {
		return err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("docpost: read dir %s: %w", dir, err)
	}
	var files, subdirs []string
	hasIndex := false
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() {
			subdirs = append(subdirs, name)
			continue
		}
		if !strings.HasSuffix(name, ".mdx") {
			continue
		}
		base := strings.TrimSuffix(name, ".mdx")
		if base == "index" {
			hasIndex = true
			continue
		}
		files = append(files, base)
	}
	sort.Strings(files)
	sort.Strings(subdirs)

	pages := make([]string, 0, 1+len(files)+len(subdirs))
	if hasIndex {
		pages = append(pages, "index")
	}
	pages = append(pages, files...)
	pages = append(pages, subdirs...)

	meta := struct {
		Title string   `json:"title"`
		Pages []string `json:"pages"`
	}{
		Title: titleCase(filepath.Base(dir)),
		Pages: pages,
	}
	data, err := json.MarshalIndent(meta, "", "    ")
	if err != nil {
		return fmt.Errorf("docpost: marshal meta for %s: %w", dir, err)
	}
	metaPath := filepath.Join(dir, "meta.json")
	if err := os.WriteFile(metaPath, append(data, '\n'), 0o600); err != nil {
		return fmt.Errorf("docpost: write %s: %w", metaPath, err)
	}
	return nil
}

// titleCase renders a lowercase command segment as a display title.
// Replaces dashes with spaces and capitalises each word.
func titleCase(seg string) string {
	parts := strings.FieldsFunc(seg, func(r rune) bool { return r == '-' || r == '_' })
	for i, w := range parts {
		if w == "" {
			continue
		}
		parts[i] = strings.ToUpper(w[:1]) + w[1:]
	}
	return strings.Join(parts, " ")
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
			result = append(result, line)
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

// rewriteLinks strips the `.md` extension from Cobra-generated cross-command
// links. The stripped form is then remapped to an absolute URL by remapLinks
// during Process.
func rewriteLinks(raw string) string {
	return crossLinkRe.ReplaceAllString(raw, "[$1]($2)")
}

func ensureContext(ctx context.Context, action string) error {
	if ctx == nil {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("docpost: %s: %w", action, err)
	}
	return nil
}
