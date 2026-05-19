// Package docpost transforms raw Cobra-generated markdown into
// Fumadocs-compatible MDX files with YAML frontmatter. Output is grouped
// into a nested directory structure by command family so the Fumadocs
// sidebar collapses the CLI reference. The generated root command page is
// written as agh.mdx so index.mdx can remain a hand-authored overview.
package docpost

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const (
	docpostAghKey       = "agh"
	docpostAghMDXPath   = "agh.mdx"
	docpostIndexKey     = "index"
	docpostIndexMDXPath = "index.mdx"
	docpostMetaJSONPath = "meta.json"
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
	segmentRe    = regexp.MustCompile(`^[A-Za-z0-9-]+$`)
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
	if err := prepareOutputDir(dstDir); err != nil {
		return err
	}

	inputs, err := readInputs(ctx, srcDir)
	if err != nil {
		return err
	}

	hasChildren := computeHasChildren(inputs)
	if err := validateOutputPaths(inputs, hasChildren); err != nil {
		return err
	}
	targets := buildTargetMap(inputs)
	if err := cleanOutput(ctx, dstDir); err != nil {
		return err
	}

	for _, in := range inputs {
		if err := ensureContext(ctx, fmt.Sprintf("write %s", in.fileName)); err != nil {
			return err
		}
		body := TransformMarkdown(in.raw, in.commandName())
		body = remapLinks(body, targets)
		body = enrichDocument(body, in, inputs, targets)

		outRel := in.outputPath(hasChildren)
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

func prepareOutputDir(dstDir string) error {
	info, err := os.Stat(dstDir)
	switch {
	case err == nil:
	case errors.Is(err, os.ErrNotExist):
		if err := os.MkdirAll(dstDir, 0o755); err != nil {
			return fmt.Errorf("docpost: create output dir: %w", err)
		}
		return nil
	default:
		return fmt.Errorf("docpost: stat output dir %s: %w", dstDir, err)
	}

	if !info.IsDir() {
		return fmt.Errorf("docpost: output path %s must be a directory", dstDir)
	}

	managed, err := isManagedOutputDir(dstDir)
	if err != nil {
		return err
	}
	if !managed {
		return fmt.Errorf(
			"docpost: refusing to clean non-empty unmanaged output dir %q",
			dstDir,
		)
	}

	return nil
}

func isManagedOutputDir(dstDir string) (bool, error) {
	entries, err := os.ReadDir(dstDir)
	if err != nil {
		return false, fmt.Errorf("docpost: read output dir %s: %w", dstDir, err)
	}
	if len(entries) == 0 {
		return true, nil
	}

	hasEditorialIndex := false
	hasEditorialMeta := false
	hasGeneratedRoot := false

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		switch entry.Name() {
		case docpostIndexMDXPath:
			hasEditorialIndex = true
		case docpostMetaJSONPath:
			hasEditorialMeta = true
		case docpostAghMDXPath:
			hasGeneratedRoot = true
		default:
			if strings.HasSuffix(entry.Name(), ".mdx") {
				continue
			}
			return false, nil
		}
	}

	return hasGeneratedRoot || (hasEditorialIndex && hasEditorialMeta), nil
}

type input struct {
	fileName string
	baseName string
	segments []string
	raw      string
}

func (in input) isRoot() bool {
	return len(in.segments) == 0
}

func (in input) commandName() string {
	return baseNameToCommand(in.baseName)
}

func (in input) targetURL() string {
	if in.isRoot() {
		return linkBasePath + "/agh"
	}
	return linkBasePath + "/" + strings.Join(in.segments, "/")
}

func (in input) outputPath(hasChildren map[string]bool) string {
	return outPath(in, hasChildren)
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
		in, ok, err := readInput(ctx, srcDir, entry)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		inputs = append(inputs, in)
	}
	return inputs, nil
}

func readInput(ctx context.Context, srcDir string, entry fs.DirEntry) (input, bool, error) {
	fullPath := filepath.Join(srcDir, entry.Name())
	if err := ensureContext(ctx, fmt.Sprintf("read %s", fullPath)); err != nil {
		return input{}, false, err
	}
	if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
		return input{}, false, nil
	}
	base := strings.TrimSuffix(entry.Name(), ".md")
	segments, err := commandSegments(entry.Name(), base)
	if err != nil {
		return input{}, false, err
	}
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return input{}, false, fmt.Errorf("docpost: read %s: %w", fullPath, err)
	}
	return input{
		fileName: entry.Name(),
		baseName: base,
		segments: segments,
		raw:      string(data),
	}, true, nil
}

func commandSegments(fileName string, base string) ([]string, error) {
	if base == docpostAghKey {
		return nil, nil
	}
	if !strings.HasPrefix(base, "agh_") {
		return nil, fmt.Errorf("docpost: unexpected filename %q (must be 'agh.md' or start with 'agh_')", fileName)
	}
	segments := strings.Split(base, "_")[1:]
	for _, segment := range segments {
		if !segmentRe.MatchString(segment) {
			return nil, fmt.Errorf("docpost: unexpected filename %q (invalid command segment %q)", fileName, segment)
		}
	}
	return segments, nil
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
		return docpostAghMDXPath
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
func buildTargetMap(inputs []input) map[string]string {
	targets := make(map[string]string, len(inputs))
	for _, in := range inputs {
		targets[in.baseName] = in.targetURL()
	}
	return targets
}

func validateOutputPaths(inputs []input, hasChildren map[string]bool) error {
	seen := make(map[string]string, len(inputs))
	for _, in := range inputs {
		outRel := in.outputPath(hasChildren)
		if previous, ok := seen[outRel]; ok {
			return fmt.Errorf("docpost: output path collision %q for %s and %s", outRel, previous, in.fileName)
		}
		seen[outRel] = in.fileName
	}
	return nil
}

// remapLinks rewrites any `](agh_xxx)` link target in the body to its
// absolute URL under linkBasePath. Runs after TransformMarkdown has already
// stripped `.md` extensions via rewriteLinks.
func remapLinks(body string, targets map[string]string) string {
	return transformMarkdownOutsideCode(body, func(text string) string {
		return strippedLink.ReplaceAllStringFunc(text, func(match string) string {
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
		if e.Name() == docpostIndexMDXPath || e.Name() == docpostMetaJSONPath {
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
		if base == docpostIndexKey {
			hasIndex = true
			continue
		}
		files = append(files, base)
	}
	sort.Strings(files)
	sort.Strings(subdirs)

	pages := make([]string, 0, 1+len(files)+len(subdirs))
	if hasIndex {
		pages = append(pages, docpostIndexKey)
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
	metaPath := filepath.Join(dir, docpostMetaJSONPath)
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

func enrichDocument(
	body string,
	current input,
	inputs []input,
	targets map[string]string,
) string {
	body = strings.TrimSpace(body)

	var sections []string
	if section := renderOutputFormatsSection(body); section != "" {
		sections = append(sections, section)
	}
	if section := renderSubcommandsSection(current, inputs, targets); section != "" {
		sections = append(sections, section)
	}

	if len(sections) == 0 {
		return body + "\n"
	}

	return body + "\n\n" + strings.Join(sections, "\n\n") + "\n"
}

// filenameToCommand converts a Cobra-generated filename to a command name.
// e.g. "agh_session_list.md" → "agh session list"
func filenameToCommand(filename string) string {
	name := strings.TrimSuffix(filename, ".md")
	return baseNameToCommand(name)
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

func renderOutputFormatsSection(body string) string {
	if !strings.Contains(body, "--output string") {
		return ""
	}
	if strings.Contains(body, "## Output Formats") {
		return ""
	}

	var b strings.Builder
	b.WriteString("## Output Formats\n\n")
	b.WriteString("Every AGH command supports `-o, --output`:\n\n")
	b.WriteString("- `human` for interactive terminal use\n")
	b.WriteString("- `json` for scripts and other machine-readable consumers\n")
	b.WriteString("- `jsonl` for wait or streaming commands that emit one JSON record per line\n")
	b.WriteString("- `toon` for compact agent-readable summaries\n")

	if usage := extractUsageLine(body); usage != "" {
		b.WriteString("\nExample:\n\n```bash\n")
		b.WriteString(outputExampleCommand(usage))
		b.WriteString("\n```")
	}

	return b.String()
}

func extractUsageLine(body string) string {
	lines := strings.Split(body, "\n")
	inFence := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			inFence = !inFence
			continue
		}
		if !inFence {
			continue
		}
		if strings.HasPrefix(trimmed, "agh ") {
			return trimmed
		}
	}
	return ""
}

func outputExampleCommand(usage string) string {
	usage = strings.ReplaceAll(usage, "[flags]", "")
	usage = strings.Join(strings.Fields(usage), " ")
	usage = strings.TrimSpace(usage)
	if usage == "" {
		return ""
	}
	return usage + " -o json"
}

func renderSubcommandsSection(current input, inputs []input, targets map[string]string) string {
	children := directChildren(current, inputs)
	if len(children) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("## Subcommands\n\n")
	b.WriteString("| Command | Description |\n")
	b.WriteString("| ------- | ----------- |\n")
	for _, child := range children {
		cmd := child.commandName()
		desc := strings.TrimSpace(extractDescription(child.raw))
		if desc == "" {
			desc = "See command reference."
		}
		desc = strings.ReplaceAll(desc, "|", "\\|")
		target := targets[child.baseName]
		fmt.Fprintf(&b, "| [%s](%s) | %s |\n", cmd, target, desc)
	}

	return strings.TrimSpace(b.String())
}

func directChildren(parent input, inputs []input) []input {
	children := make([]input, 0, 4)
	for _, candidate := range inputs {
		if len(candidate.segments) != len(parent.segments)+1 {
			continue
		}
		match := true
		for i := range parent.segments {
			if candidate.segments[i] != parent.segments[i] {
				match = false
				break
			}
		}
		if match {
			children = append(children, candidate)
		}
	}

	sort.Slice(children, func(i, j int) bool {
		return children[i].commandName() < children[j].commandName()
	})

	return children
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
	return transformMarkdownOutsideCode(raw, func(text string) string {
		return crossLinkRe.ReplaceAllString(text, "[$1]($2)")
	})
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

func baseNameToCommand(baseName string) string {
	return strings.ReplaceAll(baseName, "_", " ")
}

func transformMarkdownOutsideCode(raw string, transform func(string) string) string {
	lines := strings.Split(raw, "\n")
	inFence := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		lines[i] = transformInlineText(line, transform)
	}
	return strings.Join(lines, "\n")
}

func transformInlineText(line string, transform func(string) string) string {
	if !strings.Contains(line, "`") {
		return transform(line)
	}

	var b strings.Builder
	inCode := false
	start := 0
	for i := 0; i < len(line); i++ {
		if line[i] != '`' {
			continue
		}
		if inCode {
			b.WriteString(line[start : i+1])
		} else {
			b.WriteString(transform(line[start:i]))
			b.WriteByte('`')
		}
		inCode = !inCode
		start = i + 1
	}
	if inCode {
		b.WriteString(line[start:])
	} else {
		b.WriteString(transform(line[start:]))
	}
	return b.String()
}
