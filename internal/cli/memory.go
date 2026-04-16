package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/pedronauck/agh/internal/memory"
	"github.com/spf13/cobra"
)

type memoryListItem struct {
	Filename    string       `json:"filename"`
	Name        string       `json:"name"`
	Type        memory.Type  `json:"type"`
	Scope       memory.Scope `json:"scope"`
	Age         string       `json:"age"`
	Description string       `json:"description,omitempty"`
	ModTime     time.Time    `json:"mod_time"`
}

type memoryReadView struct {
	Filename string       `json:"filename"`
	Scope    memory.Scope `json:"scope"`
	Content  string       `json:"content"`
}

type memoryMutationView struct {
	Filename string       `json:"filename"`
	Scope    memory.Scope `json:"scope"`
	Type     memory.Type  `json:"type,omitempty"`
	Status   string       `json:"status"`
	Reason   string       `json:"reason,omitempty"`
}

var memoryWriteExample = strings.Join([]string{
	"  # Write workspace-scoped project memory from a flag",
	`  agh memory write runtime-notes.md --type project --description "Runtime docs live in the site package" ` +
		`--content "Runtime docs are authored under packages/site/content/runtime."`,
	"",
	"  # Write global user memory from stdin",
	`  printf "Prefer concise PR summaries.\n" | agh memory write review-style.md --type user ` +
		`--description "User wants concise PR summaries"`,
}, "\n")

type memoryLocation struct {
	Scope     memory.Scope
	Workspace string
	Header    MemoryHeaderRecord
}

func newMemoryCommand(deps commandDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "memory",
		Short: "Manage persistent cross-session memories",
	}

	cmd.AddCommand(newMemoryListCommand(deps))
	cmd.AddCommand(newMemoryReadCommand(deps))
	cmd.AddCommand(newMemoryWriteCommand(deps))
	cmd.AddCommand(newMemoryDeleteCommand(deps))
	cmd.AddCommand(newMemoryConsolidateCommand(deps))
	return cmd
}

func newMemoryListCommand(deps commandDeps) *cobra.Command {
	var scope string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List persistent memories",
		Example: `  # List global and workspace memories visible from the current directory
  agh memory list

  # List only workspace-scoped memories
  agh memory list --scope workspace`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			locations, err := listMemoryLocations(cmd.Context(), client, deps, scope)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, memoryListBundle(locations, deps.now))
		},
	}
	cmd.Flags().StringVar(&scope, "scope", "", "Memory scope: global or workspace")
	return cmd
}

func newMemoryReadCommand(deps commandDeps) *cobra.Command {
	var scope string

	cmd := &cobra.Command{
		Use:   "read <filename>",
		Short: "Read a persistent memory file",
		Example: `  # Read a workspace memory file
  agh memory read runtime-notes.md --scope workspace

  # Read a global memory file as JSON
  agh memory read review-style.md --scope global -o json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			filename := strings.TrimSpace(args[0])
			location, err := resolveMemoryLocation(cmd.Context(), client, deps, scope, filename)
			if err != nil {
				return err
			}

			record, err := client.ReadMemory(cmd.Context(), filename, location.Scope, location.Workspace)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, memoryReadBundle(memoryReadView{
				Filename: filename,
				Scope:    location.Scope,
				Content:  record.Content,
			}))
		},
	}
	cmd.Flags().StringVar(&scope, "scope", "", "Memory scope: global or workspace")
	return cmd
}

func newMemoryWriteCommand(deps commandDeps) *cobra.Command {
	var (
		scope       string
		typeRaw     string
		description string
		contentFlag string
	)

	cmd := &cobra.Command{
		Use:     "write <filename> --type <type> --description <description>",
		Short:   "Write or update a persistent memory file",
		Example: memoryWriteExample,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			filename := strings.TrimSpace(args[0])
			memoryType, err := parseMemoryType(typeRaw)
			if err != nil {
				return err
			}
			if strings.TrimSpace(description) == "" {
				return errors.New("memory description is required")
			}

			content, err := resolveMemoryWriteContent(cmd, contentFlag)
			if err != nil {
				return err
			}

			resolvedScope, err := resolveCLIMemoryWriteScope(scope, memoryType)
			if err != nil {
				return err
			}
			workspace, err := memoryWorkspaceForScope(deps, resolvedScope)
			if err != nil {
				return err
			}

			payload, err := formatMemoryDocument(filename, memoryType, description, content)
			if err != nil {
				return err
			}

			result, err := client.WriteMemory(cmd.Context(), filename, MemoryWriteRequest{
				Content:   payload,
				Scope:     string(resolvedScope),
				Workspace: workspace,
			})
			if err != nil {
				return err
			}
			if !result.OK {
				return errors.New("cli: memory write was not acknowledged")
			}

			return writeCommandOutput(cmd, memoryMutationBundle(memoryMutationView{
				Filename: filename,
				Scope:    resolvedScope,
				Type:     memoryType,
				Status:   "written",
			}))
		},
	}
	cmd.Flags().StringVar(&scope, "scope", "", "Memory scope: global or workspace")
	cmd.Flags().StringVar(&typeRaw, "type", "", "Memory type: user, feedback, project, or reference")
	cmd.Flags().StringVar(&description, "description", "", "One-line durable memory description")
	cmd.Flags().StringVar(&contentFlag, "content", "", "Memory body content (alternative to stdin)")
	mustMarkFlagRequired(cmd, "type")
	mustMarkFlagRequired(cmd, "description")
	return cmd
}

func newMemoryDeleteCommand(deps commandDeps) *cobra.Command {
	var scope string

	cmd := &cobra.Command{
		Use:   "delete <filename>",
		Short: "Delete a persistent memory file",
		Example: `  # Delete a workspace memory file
  agh memory delete runtime-notes.md --scope workspace`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			filename := strings.TrimSpace(args[0])
			location, err := resolveMemoryLocation(cmd.Context(), client, deps, scope, filename)
			if err != nil {
				return err
			}

			result, err := client.DeleteMemory(cmd.Context(), filename, location.Scope, location.Workspace)
			if err != nil {
				return err
			}
			if !result.OK {
				return errors.New("cli: memory delete was not acknowledged")
			}

			return writeCommandOutput(cmd, memoryMutationBundle(memoryMutationView{
				Filename: filename,
				Scope:    location.Scope,
				Status:   "deleted",
			}))
		},
	}
	cmd.Flags().StringVar(&scope, "scope", "", "Memory scope: global or workspace")
	return cmd
}

func newMemoryConsolidateCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "consolidate",
		Short: "Trigger manual memory consolidation",
		Example: `  # Ask the daemon to consolidate memory for the current workspace
  agh memory consolidate`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			workspace, err := currentWorkingDirectory(deps)
			if err != nil {
				return err
			}

			result, err := client.ConsolidateMemory(cmd.Context(), workspace)
			if err != nil {
				return err
			}

			return writeCommandOutput(cmd, memoryMutationBundle(memoryMutationView{
				Filename: "",
				Scope:    memory.ScopeWorkspace,
				Status:   boolStatus(result.Triggered),
				Reason:   result.Reason,
			}))
		},
	}
}

func listMemoryLocations(
	ctx context.Context,
	client DaemonClient,
	deps commandDeps,
	rawScope string,
) ([]memoryLocation, error) {
	scope, err := parseOptionalCLIMemoryScope(rawScope)
	if err != nil {
		return nil, err
	}

	scopes := []memory.Scope{memory.ScopeGlobal, memory.ScopeWorkspace}
	if scope != "" {
		scopes = []memory.Scope{scope}
	}

	locations := make([]memoryLocation, 0, len(scopes))
	for _, currentScope := range scopes {
		workspace, err := memoryWorkspaceForScope(deps, currentScope)
		if err != nil {
			return nil, err
		}

		headers, err := client.ListMemory(ctx, currentScope, workspace)
		if err != nil {
			return nil, err
		}

		for _, header := range headers {
			item := header
			locations = append(locations, memoryLocation{
				Scope:     currentScope,
				Workspace: workspace,
				Header:    item,
			})
		}
	}

	sort.SliceStable(locations, func(i, j int) bool {
		if locations[i].Header.ModTime.Equal(locations[j].Header.ModTime) {
			return locations[i].Header.Filename < locations[j].Header.Filename
		}
		return locations[i].Header.ModTime.After(locations[j].Header.ModTime)
	})

	return locations, nil
}

func resolveMemoryLocation(
	ctx context.Context,
	client DaemonClient,
	deps commandDeps,
	rawScope string,
	filename string,
) (memoryLocation, error) {
	scope, err := parseOptionalCLIMemoryScope(rawScope)
	if err != nil {
		return memoryLocation{}, err
	}
	filename = strings.TrimSpace(filename)
	if filename == "" {
		return memoryLocation{}, errors.New("memory filename is required")
	}

	if scope != "" {
		workspace, err := memoryWorkspaceForScope(deps, scope)
		if err != nil {
			return memoryLocation{}, err
		}
		headers, err := client.ListMemory(ctx, scope, workspace)
		if err != nil {
			return memoryLocation{}, err
		}
		for _, header := range headers {
			if strings.TrimSpace(header.Filename) == filename {
				return memoryLocation{Scope: scope, Workspace: workspace, Header: header}, nil
			}
		}
		return memoryLocation{}, fmt.Errorf("%w: memory %q not found", os.ErrNotExist, filename)
	}

	locations, err := listMemoryLocations(ctx, client, deps, "")
	if err != nil {
		return memoryLocation{}, err
	}

	matches := make([]memoryLocation, 0, 2)
	for _, location := range locations {
		if strings.TrimSpace(location.Header.Filename) == filename {
			matches = append(matches, location)
		}
	}

	switch len(matches) {
	case 0:
		return memoryLocation{}, fmt.Errorf("%w: memory %q not found", os.ErrNotExist, filename)
	case 1:
		return matches[0], nil
	default:
		return memoryLocation{}, fmt.Errorf("memory %q exists in multiple scopes; set --scope explicitly", filename)
	}
}

func parseMemoryType(raw string) (memory.Type, error) {
	typ := memory.Type(strings.TrimSpace(raw)).Normalize()
	if err := typ.Validate(); err != nil {
		return "", err
	}
	return typ, nil
}

func resolveCLIMemoryWriteScope(rawScope string, memoryType memory.Type) (memory.Scope, error) {
	scope, err := parseOptionalCLIMemoryScope(rawScope)
	if err != nil {
		return "", err
	}
	if scope != "" {
		return scope, nil
	}
	return memory.DefaultScopeForType(memoryType)
}

func resolveMemoryWriteContent(cmd *cobra.Command, contentFlag string) (string, error) {
	stdinContent, err := readOptionalCommandInput(cmd.InOrStdin())
	if err != nil {
		return "", err
	}

	flagChanged := cmd.Flags().Lookup("content") != nil && cmd.Flags().Lookup("content").Changed
	switch {
	case flagChanged && strings.TrimSpace(stdinContent) != "":
		return "", errors.New("memory content must be provided via --content or stdin, not both")
	case flagChanged:
		if strings.TrimSpace(contentFlag) == "" {
			return "", errors.New("memory content is required via --content or stdin")
		}
		return contentFlag, nil
	case strings.TrimSpace(stdinContent) != "":
		return stdinContent, nil
	default:
		return "", errors.New("memory content is required via --content or stdin")
	}
}

func readOptionalCommandInput(reader io.Reader) (string, error) {
	if reader == nil {
		return "", nil
	}
	if file, ok := reader.(*os.File); ok {
		info, err := file.Stat()
		if err != nil {
			return "", fmt.Errorf("cli: stat stdin: %w", err)
		}
		if info.Mode()&os.ModeCharDevice != 0 {
			return "", nil
		}
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("cli: read stdin: %w", err)
	}
	return string(data), nil
}

func memoryWorkspaceForScope(deps commandDeps, scope memory.Scope) (string, error) {
	if scope != memory.ScopeWorkspace {
		return "", nil
	}
	return currentWorkingDirectory(deps)
}

func parseOptionalCLIMemoryScope(raw string) (memory.Scope, error) {
	scope := memory.Scope(strings.TrimSpace(raw)).Normalize()
	switch scope {
	case "":
		return "", nil
	case memory.ScopeGlobal, memory.ScopeWorkspace:
		return scope, nil
	default:
		return "", errors.New("memory scope must be one of global or workspace")
	}
}

func formatMemoryDocument(
	filename string,
	memoryType memory.Type,
	description string,
	body string,
) (string, error) {
	header := memory.Header{
		Name:        memoryNameFromFilename(filename),
		Description: strings.TrimSpace(description),
		Type:        memoryType,
	}
	if err := header.Validate(); err != nil {
		return "", err
	}

	metadata, err := yaml.Marshal(header)
	if err != nil {
		return "", fmt.Errorf("cli: encode memory frontmatter: %w", err)
	}

	var buffer bytes.Buffer
	buffer.WriteString("---\n")
	buffer.Write(metadata)
	buffer.WriteString("---\n\n")
	buffer.WriteString(body)
	return buffer.String(), nil
}

func memoryNameFromFilename(filename string) string {
	base := strings.TrimSuffix(filepath.Base(strings.TrimSpace(filename)), filepath.Ext(strings.TrimSpace(filename)))
	if base == "" {
		return ""
	}

	normalized := strings.NewReplacer("-", " ", "_", " ", ".", " ").Replace(base)
	parts := strings.Fields(normalized)
	for idx, part := range parts {
		parts[idx] = titleCaseWord(part)
	}
	return strings.Join(parts, " ")
}

func titleCaseWord(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	if len(trimmed) == 1 {
		return strings.ToUpper(trimmed)
	}
	return strings.ToUpper(trimmed[:1]) + strings.ToLower(trimmed[1:])
}

func boolStatus(value bool) string {
	if value {
		return "triggered"
	}
	return "not-triggered"
}

func memoryListBundle(locations []memoryLocation, now func() time.Time) outputBundle {
	items := make([]memoryListItem, 0, len(locations))
	for _, location := range locations {
		items = append(items, memoryListItem{
			Filename:    location.Header.Filename,
			Name:        location.Header.Name,
			Type:        location.Header.Type,
			Scope:       location.Scope,
			Age:         formatAge(now, location.Header.ModTime),
			Description: location.Header.Description,
			ModTime:     location.Header.ModTime,
		})
	}

	return listBundle(
		items,
		items,
		"Memories",
		[]string{"Filename", "Name", "Type", "Scope", "Age", "Description"},
		"memories",
		[]string{"filename", "name", "type", "scope", "age", "description"},
		func(item memoryListItem) []string {
			return []string{
				stringOrDash(item.Filename),
				stringOrDash(item.Name),
				stringOrDash(string(item.Type)),
				stringOrDash(string(item.Scope)),
				stringOrDash(item.Age),
				stringOrDash(item.Description),
			}
		},
		func(item memoryListItem) []string {
			return []string{
				item.Filename,
				item.Name,
				string(item.Type),
				string(item.Scope),
				item.Age,
				item.Description,
			}
		},
	)
}

func memoryReadBundle(view memoryReadView) outputBundle {
	return outputBundle{
		jsonValue: view,
		human: func() (string, error) {
			return strings.TrimRight(view.Content, "\n"), nil
		},
		toon: func() (string, error) {
			return renderToonObject("memory", []string{"filename", "scope", "content"}, []string{
				view.Filename,
				string(view.Scope),
				view.Content,
			}), nil
		},
	}
}

func memoryMutationBundle(view memoryMutationView) outputBundle {
	return outputBundle{
		jsonValue: view,
		human: func() (string, error) {
			rows := []keyValue{
				{Label: "Filename", Value: stringOrDash(view.Filename)},
				{Label: "Scope", Value: stringOrDash(string(view.Scope))},
				{Label: "Type", Value: stringOrDash(string(view.Type))},
				{Label: "Status", Value: stringOrDash(view.Status)},
			}
			if strings.TrimSpace(view.Reason) != "" {
				rows = append(rows, keyValue{Label: "Reason", Value: view.Reason})
			}
			return renderHumanSection("Memory", rows), nil
		},
		toon: func() (string, error) {
			return renderToonObject("memory", []string{"filename", "scope", "type", "status", "reason"}, []string{
				view.Filename,
				string(view.Scope),
				string(view.Type),
				view.Status,
				view.Reason,
			}), nil
		},
	}
}
