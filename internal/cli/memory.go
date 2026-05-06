package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/pedronauck/agh/internal/api/contract"
	memcontract "github.com/pedronauck/agh/internal/memory/contract"

	"github.com/spf13/cobra"
)

type memorySelectorFlags struct {
	Scope     string
	Workspace string
	Agent     string
	AgentTier string
}

type memorySelectorOptions struct {
	DefaultScope     memcontract.Scope
	DefaultWorkspace bool
}

type memoryListItem struct {
	Filename        string                `json:"filename"`
	Name            string                `json:"name"`
	Type            memcontract.Type      `json:"type"`
	Scope           memcontract.Scope     `json:"scope"`
	WorkspaceID     string                `json:"workspace_id,omitempty"`
	AgentName       string                `json:"agent_name,omitempty"`
	AgentTier       memcontract.AgentTier `json:"agent_tier,omitempty"`
	Age             string                `json:"age"`
	Description     string                `json:"description,omitempty"`
	StalenessBanner string                `json:"staleness_banner,omitempty"`
	ModTime         time.Time             `json:"mod_time"`
}

type memorySearchItem struct {
	Filename      string                `json:"filename"`
	Name          string                `json:"name"`
	Type          memcontract.Type      `json:"type"`
	Scope         memcontract.Scope     `json:"scope"`
	AgentTier     memcontract.AgentTier `json:"agent_tier,omitempty"`
	Score         float64               `json:"score"`
	Snippet       string                `json:"snippet,omitempty"`
	WhyRecalled   []string              `json:"why_recalled,omitempty"`
	ShadowedBy    string                `json:"shadowed_by,omitempty"`
	AlreadyShown  bool                  `json:"already_shown"`
	StalenessNote string                `json:"staleness_banner,omitempty"`
}

type memoryHistoryItem struct {
	ID          string                `json:"id"`
	Operation   memcontract.Operation `json:"operation"`
	Scope       memcontract.Scope     `json:"scope,omitempty"`
	WorkspaceID string                `json:"workspace_id,omitempty"`
	Filename    string                `json:"filename,omitempty"`
	AgentName   string                `json:"agent_name,omitempty"`
	AgentTier   memcontract.AgentTier `json:"agent_tier,omitempty"`
	Summary     string                `json:"summary,omitempty"`
	Age         string                `json:"age"`
	Timestamp   time.Time             `json:"timestamp"`
}

func newMemoryCommand(deps commandDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "memory",
		Short: "Show, write, search, and operate Memory v2 durable context",
	}

	cmd.AddCommand(newMemoryListCommand(deps))
	cmd.AddCommand(newMemoryShowCommand(deps))
	cmd.AddCommand(newMemoryWriteCommand(deps))
	cmd.AddCommand(newMemoryEditCommand(deps))
	cmd.AddCommand(newMemoryDeleteCommand(deps))
	cmd.AddCommand(newMemorySearchCommand(deps))
	cmd.AddCommand(newMemoryReindexCommand(deps))
	cmd.AddCommand(newMemoryHistoryCommand(deps))
	cmd.AddCommand(newMemoryHealthCommand(deps))
	cmd.AddCommand(newMemoryPromoteCommand(deps))
	cmd.AddCommand(newMemoryResetCommand(deps))
	cmd.AddCommand(newMemoryReloadCommand(deps))
	cmd.AddCommand(newMemoryScopeShowCommand(deps))
	cmd.AddCommand(newMemoryDecisionsCommand(deps))
	cmd.AddCommand(newMemoryRecallCommand(deps))
	cmd.AddCommand(newMemoryDreamCommand(deps))
	cmd.AddCommand(newMemoryDailyCommand(deps))
	cmd.AddCommand(newMemoryExtractorCommand(deps))
	cmd.AddCommand(newMemoryProviderCommand(deps))
	cmd.AddCommand(newMemoryAdhocCommand())
	return cmd
}

func newMemoryListCommand(deps commandDeps) *cobra.Command {
	var flags memorySelectorFlags
	var typeRaw string
	var includeShadowed bool
	var includeSystem bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List Memory v2 entries",
		Example: `  # List global and current-workspace memories
  agh memory list

  # List agent-workspace memories
  agh memory list --scope agent --agent reviewer --agent-tier workspace`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			selector, err := resolveMemorySelectorFlags(deps, flags, memorySelectorOptions{DefaultWorkspace: true})
			if err != nil {
				return err
			}
			typ, err := parseOptionalMemoryType(typeRaw)
			if err != nil {
				return err
			}
			selector.IncludeSystem = includeSystem
			response, err := client.ListMemory(cmd.Context(), MemoryListQuery{
				MemorySelectorQuery: selector,
				Type:                typ,
				IncludeShadowed:     includeShadowed,
			})
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, memoryListBundle(response, deps.now))
		},
	}
	addMemorySelectorFlags(cmd, &flags)
	cmd.Flags().StringVar(&typeRaw, "type", "", "Memory type: user, feedback, project, or reference")
	cmd.Flags().BoolVar(&includeShadowed, "include-shadowed", false, "Include shadowed entries")
	cmd.Flags().BoolVar(&includeSystem, "include-system", false, "Include _system memory entries")
	return cmd
}

func newMemoryShowCommand(deps commandDeps) *cobra.Command {
	var flags memorySelectorFlags
	var includeSystem bool

	cmd := &cobra.Command{
		Use:   "show <filename>",
		Short: "Show one Memory v2 entry",
		Example: `  # Show a workspace memory entry
  agh memory show runtime-notes.md --scope workspace

  # Show an agent-global memory entry as JSON
  agh memory show prefs.md --scope agent --agent reviewer --agent-tier global -o json`,
		Args: exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			selector, err := resolveMemorySelectorFlags(deps, flags, memorySelectorOptions{DefaultWorkspace: true})
			if err != nil {
				return err
			}
			selector.IncludeSystem = includeSystem
			response, err := client.ShowMemory(cmd.Context(), strings.TrimSpace(args[0]), selector)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, memoryEntryBundle(response))
		},
	}
	addMemorySelectorFlags(cmd, &flags)
	cmd.Flags().BoolVar(&includeSystem, "include-system", false, "Allow showing _system memory entries")
	return cmd
}

func newMemoryWriteCommand(deps commandDeps) *cobra.Command {
	var flags memorySelectorFlags
	var typeRaw string
	var name string
	var description string
	var contentFlag string
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "write --type <type> --name <name> --content <@file|text>",
		Short: "Create a Memory v2 entry through the controller",
		Example: `  # Write workspace-scoped project memory from a file
  agh memory write --scope workspace --type project --name "Runtime docs" --content @runtime.md

  # Write agent-global feedback
  agh memory write --scope agent --agent reviewer --agent-tier global \
    --type feedback --name "Review tone" --content @feedback.md`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			typ, err := parseRequiredMemoryType(typeRaw)
			if err != nil {
				return err
			}
			defaultScope, err := memcontract.DefaultScopeForType(typ)
			if err != nil {
				return err
			}
			selector, err := resolveMemorySelectorFlags(deps, flags, memorySelectorOptions{DefaultScope: defaultScope})
			if err != nil {
				return err
			}
			content, err := resolveMemoryContent(cmd, deps, contentFlag)
			if err != nil {
				return err
			}
			if strings.TrimSpace(name) == "" {
				return errors.New("memory.name_required: --name is required")
			}
			response, err := client.CreateMemory(cmd.Context(), MemoryCreateRequest{
				Scope:       selector.Scope,
				WorkspaceID: selector.WorkspaceID,
				AgentName:   selector.AgentName,
				AgentTier:   selector.AgentTier,
				Origin:      memcontract.OriginCLI,
				Type:        typ,
				Name:        strings.TrimSpace(name),
				Description: strings.TrimSpace(description),
				Content:     content,
				DryRun:      dryRun,
			})
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, memoryMutationBundle("Memory Write", response))
		},
	}
	addMemorySelectorFlags(cmd, &flags)
	cmd.Flags().StringVar(&typeRaw, "type", "", "Memory type: user, feedback, project, or reference")
	cmd.Flags().StringVar(&name, "name", "", "Memory display name")
	cmd.Flags().StringVar(&description, "description", "", "One-line durable memory description")
	cmd.Flags().StringVar(&contentFlag, "content", "", "Memory content; use @file to read from disk or - for stdin")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Ask the controller for a decision without applying it")
	mustMarkFlagRequired(cmd, "type")
	mustMarkFlagRequired(cmd, "name")
	mustMarkFlagRequired(cmd, "content")
	return cmd
}

func newMemoryEditCommand(deps commandDeps) *cobra.Command {
	var flags memorySelectorFlags
	var typeRaw string
	var name string
	var description string
	var contentFlag string
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "edit <filename> --content <@file|text>",
		Short: "Edit a Memory v2 entry through the controller",
		Args:  exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			selector, err := resolveMemorySelectorFlags(deps, flags, memorySelectorOptions{DefaultWorkspace: true})
			if err != nil {
				return err
			}
			typ, err := parseOptionalMemoryType(typeRaw)
			if err != nil {
				return err
			}
			content, err := resolveMemoryContent(cmd, deps, contentFlag)
			if err != nil {
				return err
			}
			response, err := client.EditMemory(cmd.Context(), strings.TrimSpace(args[0]), MemoryEditRequest{
				Scope:       selector.Scope,
				WorkspaceID: selector.WorkspaceID,
				AgentName:   selector.AgentName,
				AgentTier:   selector.AgentTier,
				Type:        typ,
				Name:        strings.TrimSpace(name),
				Description: strings.TrimSpace(description),
				Content:     content,
				DryRun:      dryRun,
			})
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, memoryMutationBundle("Memory Edit", response))
		},
	}
	addMemorySelectorFlags(cmd, &flags)
	cmd.Flags().StringVar(&typeRaw, "type", "", "Memory type override")
	cmd.Flags().StringVar(&name, "name", "", "Memory display name override")
	cmd.Flags().StringVar(&description, "description", "", "Memory description override")
	cmd.Flags().StringVar(&contentFlag, "content", "", "Memory content; use @file to read from disk or - for stdin")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Ask the controller for a decision without applying it")
	mustMarkFlagRequired(cmd, "content")
	return cmd
}

func newMemoryDeleteCommand(deps commandDeps) *cobra.Command {
	var flags memorySelectorFlags

	cmd := &cobra.Command{
		Use:   "delete <filename>",
		Short: "Delete a Memory v2 entry through the controller",
		Args:  exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			selector, err := resolveMemorySelectorFlags(deps, flags, memorySelectorOptions{DefaultWorkspace: true})
			if err != nil {
				return err
			}
			response, err := client.DeleteMemory(cmd.Context(), strings.TrimSpace(args[0]), selector)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, memoryDeleteBundle(response))
		},
	}
	addMemorySelectorFlags(cmd, &flags)
	return cmd
}

func newMemorySearchCommand(deps commandDeps) *cobra.Command {
	var flags memorySelectorFlags
	var topK int
	var includeSystem bool

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search deterministic Memory v2 recall",
		Example: `  # Search global and current-workspace memories
  agh memory search "auth sessions"

  # Search agent memory with system entries included
  agh memory search "review tone" --scope agent --agent reviewer --agent-tier global --include-system`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			selector, err := resolveMemorySelectorFlags(deps, flags, memorySelectorOptions{DefaultWorkspace: true})
			if err != nil {
				return err
			}
			query := strings.TrimSpace(strings.Join(args, " "))
			if query == "" {
				return errors.New("memory.query_required: query is required")
			}
			response, err := client.SearchMemory(cmd.Context(), MemorySearchRequest{
				QueryText:     query,
				Scope:         selector.Scope,
				WorkspaceID:   selector.WorkspaceID,
				AgentName:     selector.AgentName,
				AgentTier:     selector.AgentTier,
				TopK:          topK,
				IncludeSystem: includeSystem,
			})
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, memorySearchBundle(response))
		},
	}
	addMemorySelectorFlags(cmd, &flags)
	cmd.Flags().IntVar(&topK, "top-k", 0, "Maximum number of recalled entries")
	cmd.Flags().BoolVar(&includeSystem, "include-system", false, "Include _system memory entries")
	return cmd
}

func newMemoryReindexCommand(deps commandDeps) *cobra.Command {
	var flags memorySelectorFlags
	var includeSystem bool

	cmd := &cobra.Command{
		Use:   "reindex",
		Short: "Rebuild the derived Memory v2 search catalog",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			selector, err := resolveMemorySelectorFlags(deps, flags, memorySelectorOptions{DefaultWorkspace: true})
			if err != nil {
				return err
			}
			response, err := client.ReindexMemory(cmd.Context(), MemoryReindexRequest{
				Scope:         selector.Scope,
				WorkspaceID:   selector.WorkspaceID,
				AgentName:     selector.AgentName,
				AgentTier:     selector.AgentTier,
				IncludeSystem: includeSystem,
			})
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, memoryReindexBundle(response))
		},
	}
	addMemorySelectorFlags(cmd, &flags)
	cmd.Flags().BoolVar(&includeSystem, "include-system", false, "Include _system memory entries")
	return cmd
}

func newMemoryHistoryCommand(deps commandDeps) *cobra.Command {
	var flags memorySelectorFlags
	var operation string
	var sinceRaw string
	var limit int

	cmd := &cobra.Command{
		Use:   "history",
		Short: "Show redaction-safe Memory v2 operation history",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			selector, err := resolveMemorySelectorFlags(deps, flags, memorySelectorOptions{})
			if err != nil {
				return err
			}
			since, err := parseSinceFlag(sinceRaw, deps.now)
			if err != nil {
				return err
			}
			records, err := client.MemoryHistory(cmd.Context(), MemoryHistoryQuery{
				Scope:       selector.Scope,
				WorkspaceID: selector.WorkspaceID,
				AgentName:   selector.AgentName,
				AgentTier:   selector.AgentTier,
				Operation:   strings.TrimSpace(operation),
				Since:       since,
				Limit:       limit,
			})
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, memoryHistoryBundle(records, deps.now))
		},
	}
	addMemorySelectorFlags(cmd, &flags)
	cmd.Flags().StringVar(&operation, "operation", "", "Memory operation type, for example memory.write")
	cmd.Flags().StringVar(&sinceRaw, "since", "", "Show operations since an RFC3339 timestamp or relative duration")
	cmd.Flags().IntVar(&limit, "limit", 25, "Maximum number of operations to return")
	return cmd
}

func newMemoryHealthCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "health",
		Short: "Show Memory v2 health",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			workspace, err := currentWorkingDirectory(deps)
			if err != nil {
				return err
			}
			health, err := client.MemoryHealth(cmd.Context(), workspace)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, memoryHealthBundle(health))
		},
	}
}

func newMemoryPromoteCommand(deps commandDeps) *cobra.Command {
	var flags memorySelectorFlags
	var fromRaw string
	var toRaw string
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "promote <filename> --from <scope[:tier]> --to <scope[:tier]>",
		Short: "Promote a memory entry across Memory v2 scopes",
		Args:  exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			from, err := parseMemoryPromotionSelector(deps, flags, fromRaw)
			if err != nil {
				return err
			}
			to, err := parseMemoryPromotionSelector(deps, flags, toRaw)
			if err != nil {
				return err
			}
			response, err := client.PromoteMemory(cmd.Context(), MemoryPromoteRequest{
				Filename: strings.TrimSpace(args[0]),
				From:     from,
				To:       to,
				DryRun:   dryRun,
			})
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, memoryPromoteBundle(response))
		},
	}
	addMemorySelectorFlags(cmd, &flags)
	cmd.Flags().StringVar(&fromRaw, "from", "", "Source scope: global, workspace, agent:workspace, or agent:global")
	cmd.Flags().StringVar(&toRaw, "to", "", "Destination scope: global, workspace, agent:workspace, or agent:global")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Ask for a promotion decision without applying it")
	mustMarkFlagRequired(cmd, "from")
	mustMarkFlagRequired(cmd, "to")
	return cmd
}

func newMemoryResetCommand(deps commandDeps) *cobra.Command {
	var flags memorySelectorFlags
	var includeDaily bool
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "reset",
		Short: "Reset derived Memory v2 state through the daemon",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			selector, err := resolveMemorySelectorFlags(deps, flags, memorySelectorOptions{DefaultWorkspace: true})
			if err != nil {
				return err
			}
			response, err := client.ResetMemory(cmd.Context(), MemoryResetRequest{
				Scope:       selector.Scope,
				WorkspaceID: selector.WorkspaceID,
				AgentName:   selector.AgentName,
				AgentTier:   selector.AgentTier,
				DerivedOnly: !includeDaily,
				Confirm:     !dryRun,
			})
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, memoryObjectBundle("Memory Reset", response))
		},
	}
	addMemorySelectorFlags(cmd, &flags)
	cmd.Flags().BoolVar(&includeDaily, "include-daily", false, "Include daily memory artifacts")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show reset work without applying it")
	return cmd
}

func newMemoryReloadCommand(deps commandDeps) *cobra.Command {
	var flags memorySelectorFlags

	cmd := &cobra.Command{
		Use:   "reload",
		Short: "Invalidate frozen memory snapshots for future session boots",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			selector, err := resolveMemorySelectorFlags(deps, flags, memorySelectorOptions{})
			if err != nil {
				return err
			}
			response, err := client.ReloadMemory(cmd.Context(), selector)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, memoryObjectBundle("Memory Reload", response))
		},
	}
	addMemorySelectorFlags(cmd, &flags)
	return cmd
}

func newMemoryScopeShowCommand(deps commandDeps) *cobra.Command {
	var flags memorySelectorFlags

	cmd := &cobra.Command{
		Use:   "scope-show",
		Short: "Show resolved Memory v2 precedence for a selector",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			selector, err := resolveMemorySelectorFlags(deps, flags, memorySelectorOptions{DefaultWorkspace: true})
			if err != nil {
				return err
			}
			response, err := client.MemoryScopeShow(cmd.Context(), selector)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, memoryScopeShowBundle(response))
		},
	}
	addMemorySelectorFlags(cmd, &flags)
	return cmd
}

func newMemoryDecisionsCommand(deps commandDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "decisions",
		Short: "Inspect and revert Memory v2 controller decisions",
	}
	cmd.AddCommand(newMemoryDecisionsListCommand(deps))
	cmd.AddCommand(newMemoryDecisionsShowCommand(deps))
	cmd.AddCommand(newMemoryDecisionsRevertCommand(deps))
	return cmd
}

func newMemoryDecisionsListCommand(deps commandDeps) *cobra.Command {
	var flags memorySelectorFlags
	var op string
	var sinceRaw string
	var reason string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List Memory v2 controller decisions",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			selector, err := resolveMemorySelectorFlags(deps, flags, memorySelectorOptions{})
			if err != nil {
				return err
			}
			since, err := parseSinceFlag(sinceRaw, deps.now)
			if err != nil {
				return err
			}
			response, err := client.ListMemoryDecisions(cmd.Context(), MemoryDecisionListQuery{
				Scope:       selector.Scope,
				WorkspaceID: selector.WorkspaceID,
				AgentName:   selector.AgentName,
				AgentTier:   selector.AgentTier,
				Operation:   op,
				Since:       since,
				Reason:      reason,
			})
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, memoryDecisionListBundle(response))
		},
	}
	addMemorySelectorFlags(cmd, &flags)
	cmd.Flags().StringVar(&op, "op", "", "Decision operation filter")
	cmd.Flags().StringVar(&sinceRaw, "since", "", "Show decisions since an RFC3339 timestamp or relative duration")
	cmd.Flags().StringVar(&reason, "reason", "", "Reason substring filter")
	return cmd
}

func newMemoryDecisionsShowCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "show <id>",
		Short: "Show one Memory v2 controller decision",
		Args:  exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			response, err := client.GetMemoryDecision(cmd.Context(), strings.TrimSpace(args[0]))
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, memoryDecisionBundle("Memory Decision", response.Decision, response))
		},
	}
}

func newMemoryDecisionsRevertCommand(deps commandDeps) *cobra.Command {
	var reason string
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "revert <id>",
		Short: "Revert one Memory v2 controller decision",
		Args:  exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			response, err := client.RevertMemoryDecision(
				cmd.Context(),
				strings.TrimSpace(args[0]),
				MemoryDecisionRevertRequest{
					Reason: strings.TrimSpace(reason),
					DryRun: dryRun,
				},
			)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, memoryDecisionRevertBundle(response))
		},
	}
	cmd.Flags().StringVar(&reason, "reason", "", "Operator-visible revert reason")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Return the revert decision without applying it")
	return cmd
}

func newMemoryRecallCommand(deps commandDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "recall",
		Short: "Inspect Memory v2 recall traces",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "trace <session_id> <turn_seq>",
		Short: "Show one redaction-safe recall trace",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			turnSeq, err := strconv.ParseInt(strings.TrimSpace(args[1]), 10, 64)
			if err != nil || turnSeq <= 0 {
				return errors.New("memory.recall.turn_seq_invalid: turn_seq must be a positive integer")
			}
			response, err := client.GetMemoryRecallTrace(cmd.Context(), strings.TrimSpace(args[0]), turnSeq)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, memoryObjectBundle("Memory Recall Trace", response))
		},
	})
	return cmd
}

func newMemoryDreamCommand(deps commandDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dream",
		Short: "Operate Memory v2 dreaming runs",
	}
	cmd.AddCommand(newMemoryDreamShowCommand(deps))
	cmd.AddCommand(newMemoryDreamRetryCommand(deps))
	cmd.AddCommand(newMemoryDreamTriggerCommand(deps))
	cmd.AddCommand(newMemoryDreamStatusCommand(deps))
	return cmd
}

func newMemoryDreamShowCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "show <date-or-run-id>",
		Short: "Show one Memory v2 dreaming run",
		Args:  exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			response, err := client.GetMemoryDream(cmd.Context(), strings.TrimSpace(args[0]))
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, memoryObjectBundle("Memory Dream", response))
		},
	}
}

func newMemoryDreamRetryCommand(deps commandDeps) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "retry <run_id>",
		Short: "Retry one failed Memory v2 dreaming run",
		Args:  exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			response, err := client.RetryMemoryDream(cmd.Context(), strings.TrimSpace(args[0]), MemoryDreamRetryRequest{
				Force: force,
			})
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, memoryObjectBundle("Memory Dream Retry", response))
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Retry even if normal gates would skip the run")
	return cmd
}

func newMemoryDreamTriggerCommand(deps commandDeps) *cobra.Command {
	var flags memorySelectorFlags
	var force bool

	cmd := &cobra.Command{
		Use:   "trigger",
		Short: "Trigger Memory v2 dreaming",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			selector, err := resolveMemorySelectorFlags(deps, flags, memorySelectorOptions{DefaultWorkspace: true})
			if err != nil {
				return err
			}
			response, err := client.TriggerMemoryDream(cmd.Context(), MemoryDreamTriggerRequest{
				Scope:       selector.Scope,
				WorkspaceID: selector.WorkspaceID,
				AgentName:   selector.AgentName,
				AgentTier:   selector.AgentTier,
				Force:       force,
			})
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, memoryDreamTriggerBundle(response))
		},
	}
	addMemorySelectorFlags(cmd, &flags)
	cmd.Flags().BoolVar(&force, "force", false, "Trigger even if normal gates would skip the run")
	return cmd
}

func newMemoryDreamStatusCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show Memory v2 dreaming runtime status",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			response, err := client.GetMemoryDreamStatus(cmd.Context())
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, memoryDreamListBundle(response))
		},
	}
}

func newMemoryDailyCommand(deps commandDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "daily",
		Short: "Inspect Memory v2 daily operation logs",
	}
	cmd.AddCommand(newMemoryDailyListCommand(deps))
	cmd.AddCommand(
		newMemoryDailyUnsupportedSelectorCommand(
			"show <date>",
			"memory.unsupported: daily show is not registered in Slice 1 API",
			exactOneNonBlankArg(),
		),
	)
	cmd.AddCommand(
		newMemoryDailyRetentionCommand("archive", "memory.unsupported: daily archive is not registered in Slice 1 API"),
	)
	cmd.AddCommand(
		newMemoryDailyUnsupportedSelectorCommand(
			"restore <date>",
			"memory.unsupported: daily restore is not registered in Slice 1 API",
			exactOneNonBlankArg(),
		),
	)
	cmd.AddCommand(
		newMemoryDailyRetentionCommand("purge", "memory.unsupported: daily purge is not registered in Slice 1 API"),
	)
	return cmd
}

func newMemoryDailyListCommand(deps commandDeps) *cobra.Command {
	var flags memorySelectorFlags

	cmd := &cobra.Command{
		Use:   "ls",
		Short: "List Memory v2 daily operation logs",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			selector, err := resolveMemorySelectorFlags(deps, flags, memorySelectorOptions{DefaultWorkspace: true})
			if err != nil {
				return err
			}
			response, err := client.ListMemoryDailyLogs(cmd.Context(), selector)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, memoryObjectBundle("Memory Daily Logs", response))
		},
	}
	addMemorySelectorFlags(cmd, &flags)
	return cmd
}

func newMemoryExtractorCommand(deps commandDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "extractor",
		Short: "Operate Memory v2 extractor runtime",
	}
	cmd.AddCommand(newMemoryExtractorStatusCommand(deps))
	cmd.AddCommand(newMemoryExtractorListPendingCommand(deps))
	cmd.AddCommand(newMemoryExtractorReplayCommand(deps))
	cmd.AddCommand(newMemoryExtractorDrainCommand(deps))
	cmd.AddCommand(newMemoryExtractorDisableCommand())
	return cmd
}

func newMemoryExtractorStatusCommand(deps commandDeps) *cobra.Command {
	var sessionID string

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show Memory v2 extractor runtime status",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			response, err := client.GetMemoryExtractorStatus(cmd.Context(), sessionID)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, memoryObjectBundle("Memory Extractor Status", response))
		},
	}
	cmd.Flags().StringVar(&sessionID, "session", "", "Filter extractor status by session")
	return cmd
}

func newMemoryExtractorListPendingCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "list-pending",
		Short: "List Memory v2 extractor pending/DLQ records",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			response, err := client.ListMemoryExtractorFailures(cmd.Context())
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, memoryObjectBundle("Memory Extractor Pending", response))
		},
	}
}

func newMemoryExtractorReplayCommand(deps commandDeps) *cobra.Command {
	var sessionID string

	cmd := &cobra.Command{
		Use:   "replay --session <id>",
		Short: "Replay Memory v2 extractor work",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			if strings.TrimSpace(sessionID) == "" {
				return errors.New("memory.extractor.session_required: --session is required")
			}
			response, err := client.RetryMemoryExtractor(cmd.Context(), MemoryExtractorRetryRequest{
				SessionID: strings.TrimSpace(sessionID),
			})
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, memoryObjectBundle("Memory Extractor Replay", response))
		},
	}
	cmd.Flags().StringVar(&sessionID, "session", "", "Session whose extractor work should be replayed")
	mustMarkFlagRequired(cmd, "session")
	return cmd
}

func newMemoryExtractorDrainCommand(deps commandDeps) *cobra.Command {
	var timeoutRaw string

	cmd := &cobra.Command{
		Use:   "drain",
		Short: "Drain Memory v2 extractor work",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			timeout := 60 * time.Second
			if strings.TrimSpace(timeoutRaw) != "" {
				parsed, err := time.ParseDuration(strings.TrimSpace(timeoutRaw))
				if err != nil {
					return fmt.Errorf("memory.extractor.timeout_invalid: %w", err)
				}
				timeout = parsed
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
			defer cancel()
			response, err := client.DrainMemoryExtractor(ctx)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, memoryObjectBundle("Memory Extractor Drain", response))
		},
	}
	cmd.Flags().StringVar(&timeoutRaw, "timeout", "60s", "Maximum drain wait duration")
	return cmd
}

func newMemoryExtractorDisableCommand() *cobra.Command {
	var sessionID string
	cmd := newUnsupportedMemoryCommand(
		"disable --session <id>",
		"memory.unsupported: extractor disable is not registered in Slice 1 API",
		cobra.NoArgs,
	)
	cmd.Flags().StringVar(&sessionID, "session", "", "Session whose extractor should be disabled")
	mustMarkFlagRequired(cmd, "session")
	return cmd
}

func newMemoryProviderCommand(deps commandDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "provider",
		Short: "Operate Memory v2 providers",
	}
	cmd.AddCommand(newMemoryProviderListCommand(deps))
	cmd.AddCommand(newMemoryProviderEnableCommand(deps))
	cmd.AddCommand(newMemoryProviderDisableCommand(deps))
	return cmd
}

func newMemoryProviderListCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List registered Memory v2 providers",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			response, err := client.ListMemoryProviders(cmd.Context())
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, memoryProviderListBundle(response))
		},
	}
}

func newMemoryProviderEnableCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "enable <name>",
		Short: "Enable and select one Memory v2 provider",
		Args:  exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			name := strings.TrimSpace(args[0])
			response, err := client.EnableMemoryProvider(
				cmd.Context(),
				name,
				MemoryProviderLifecycleRequest{},
			)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, memoryObjectBundle("Memory Provider", response))
		},
	}
}

func newMemoryProviderDisableCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "disable <name>",
		Short: "Disable one Memory v2 provider",
		Args:  exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			name := strings.TrimSpace(args[0])
			response, err := client.DisableMemoryProvider(
				cmd.Context(),
				name,
				MemoryProviderLifecycleRequest{},
			)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, memoryObjectBundle("Memory Provider", response))
		},
	}
}

func newMemoryAdhocCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "adhoc",
		Short: "Inspect ad-hoc Memory v2 notes",
	}
	cmd.AddCommand(newMemoryAdhocListCommand())
	cmd.AddCommand(
		newUnsupportedMemoryCommand(
			"show <slug>",
			"memory.unsupported: adhoc show is not registered in Slice 1 API",
			exactOneNonBlankArg(),
		),
	)
	return cmd
}

func newMemoryAdhocListCommand() *cobra.Command {
	var flags memorySelectorFlags
	cmd := newUnsupportedMemoryCommand(
		"list",
		"memory.unsupported: adhoc list is not registered in Slice 1 API",
		cobra.NoArgs,
	)
	addMemorySelectorFlags(cmd, &flags)
	return cmd
}

func newMemoryDailyUnsupportedSelectorCommand(
	use string,
	message string,
	args cobra.PositionalArgs,
) *cobra.Command {
	var flags memorySelectorFlags
	cmd := newUnsupportedMemoryCommand(use, message, args)
	addMemorySelectorFlags(cmd, &flags)
	return cmd
}

func newMemoryDailyRetentionCommand(use string, message string) *cobra.Command {
	var olderThan string
	var dryRun bool
	cmd := newUnsupportedMemoryCommand(use, message, cobra.NoArgs)
	cmd.Flags().StringVar(&olderThan, "older-than", "", "Retention age threshold")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show retention work without applying it")
	return cmd
}

func newUnsupportedMemoryCommand(use string, message string, args cobra.PositionalArgs) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: "Reserved Memory v2 command",
		Args:  args,
		RunE: func(*cobra.Command, []string) error {
			return errors.New(message)
		},
	}
}

func addMemorySelectorFlags(cmd *cobra.Command, flags *memorySelectorFlags) {
	cmd.Flags().StringVar(&flags.Scope, "scope", "", "Memory scope: global, workspace, or agent")
	cmd.Flags().StringVar(&flags.Workspace, "workspace", "", "Workspace ID or path for workspace-bound memory")
	cmd.Flags().StringVar(&flags.Agent, "agent", "", "Agent name for agent-scoped memory")
	cmd.Flags().StringVar(&flags.AgentTier, "agent-tier", "", "Agent memory tier: workspace or global")
}

func resolveMemorySelectorFlags(
	deps commandDeps,
	flags memorySelectorFlags,
	opts memorySelectorOptions,
) (MemorySelectorQuery, error) {
	scope, err := parseOptionalCLIMemoryScope(flags.Scope)
	if err != nil {
		return MemorySelectorQuery{}, err
	}
	agent := strings.TrimSpace(flags.Agent)
	tier, err := parseOptionalCLIAgentTier(flags.AgentTier)
	if err != nil {
		return MemorySelectorQuery{}, err
	}
	if scope == "" && (agent != "" || tier != "") {
		scope = memcontract.ScopeAgent
	}
	if scope == "" {
		scope = opts.DefaultScope
	}
	if scope != memcontract.ScopeAgent && (agent != "" || tier != "") {
		return MemorySelectorQuery{}, errors.New(
			"memory.scope.agent_flags_invalid: --agent and --agent-tier require --scope agent",
		)
	}
	if scope == memcontract.ScopeAgent {
		if agent == "" {
			return MemorySelectorQuery{}, errors.New(
				"memory.scope.agent_required: --agent is required when --scope agent",
			)
		}
		if tier == "" {
			return MemorySelectorQuery{}, errors.New(
				"memory.scope.agent_tier_required: --agent-tier is required when --scope agent",
			)
		}
	}

	workspace := strings.TrimSpace(flags.Workspace)
	needsWorkspace := scope == memcontract.ScopeWorkspace || (scope == "" && opts.DefaultWorkspace) ||
		(scope == memcontract.ScopeAgent && tier == memcontract.AgentTierWorkspace)
	if workspace == "" && needsWorkspace {
		var err error
		workspace, err = currentWorkingDirectory(deps)
		if err != nil {
			return MemorySelectorQuery{}, err
		}
	}
	return MemorySelectorQuery{
		Scope:       scope,
		WorkspaceID: workspace,
		AgentName:   agent,
		AgentTier:   tier,
	}, nil
}

func parseOptionalCLIMemoryScope(raw string) (memcontract.Scope, error) {
	scope := memcontract.Scope(strings.TrimSpace(raw)).Normalize()
	switch scope {
	case "":
		return "", nil
	case memcontract.ScopeGlobal, memcontract.ScopeWorkspace, memcontract.ScopeAgent:
		return scope, nil
	default:
		return "", errors.New("memory.scope.invalid: scope must be one of global, workspace, or agent")
	}
}

func parseOptionalCLIAgentTier(raw string) (memcontract.AgentTier, error) {
	tier := memcontract.AgentTier(strings.TrimSpace(raw)).Normalize()
	switch tier {
	case "":
		return "", nil
	case memcontract.AgentTierWorkspace, memcontract.AgentTierGlobal:
		return tier, nil
	default:
		return "", errors.New("memory.scope.agent_tier_invalid: agent-tier must be one of workspace or global")
	}
}

func parseRequiredMemoryType(raw string) (memcontract.Type, error) {
	typ, err := parseOptionalMemoryType(raw)
	if err != nil {
		return "", err
	}
	if typ == "" {
		return "", errors.New("memory.type_required: --type is required")
	}
	return typ, nil
}

func parseOptionalMemoryType(raw string) (memcontract.Type, error) {
	typ := memcontract.Type(strings.TrimSpace(raw)).Normalize()
	if typ == "" {
		return "", nil
	}
	if err := typ.Validate(); err != nil {
		return "", err
	}
	return typ, nil
}

func resolveMemoryContent(cmd *cobra.Command, deps commandDeps, raw string) (string, error) {
	flag := cmd.Flags().Lookup("content")
	flagChanged := flag != nil && flag.Changed
	stdinContent, err := readOptionalCommandInput(cmd.InOrStdin())
	if err != nil {
		return "", err
	}
	if flagChanged && strings.TrimSpace(stdinContent) != "" && strings.TrimSpace(raw) != "-" {
		return "", errors.New("memory.content_conflict: provide memory content via --content or stdin, not both")
	}
	if flagChanged {
		if strings.TrimSpace(raw) == "-" {
			if strings.TrimSpace(stdinContent) == "" {
				return "", errors.New("memory.content_required: stdin content is required")
			}
			return stdinContent, nil
		}
		return resolveMemoryContentValue(deps, raw, cmd.InOrStdin())
	}
	if strings.TrimSpace(stdinContent) != "" {
		return stdinContent, nil
	}
	return "", errors.New("memory.content_required: content is required via --content or stdin")
}

func resolveMemoryContentValue(deps commandDeps, raw string, stdin io.Reader) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", errors.New("memory.content_required: content is required")
	}
	if trimmed == "-" {
		content, err := readOptionalCommandInput(stdin)
		if err != nil {
			return "", err
		}
		if strings.TrimSpace(content) == "" {
			return "", errors.New("memory.content_required: stdin content is required")
		}
		return content, nil
	}
	if after, ok := strings.CutPrefix(trimmed, "@"); ok {
		path := strings.TrimSpace(after)
		if path == "" {
			return "", errors.New("memory.content_path_required: @ content path is required")
		}
		cleaned := filepath.Clean(path)
		if !filepath.IsAbs(cleaned) && deps.getwd != nil {
			wd, err := currentWorkingDirectory(deps)
			if err != nil {
				return "", err
			}
			cleaned = filepath.Join(wd, cleaned)
		}
		data, err := os.ReadFile(cleaned)
		if err != nil {
			return "", fmt.Errorf("memory.content_read_failed: read %s: %w", cleaned, err)
		}
		if strings.TrimSpace(string(data)) == "" {
			return "", errors.New("memory.content_required: file content is required")
		}
		return string(data), nil
	}
	return raw, nil
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

func parseMemoryPromotionSelector(
	deps commandDeps,
	flags memorySelectorFlags,
	raw string,
) (contract.MemoryScopeSelectorPayload, error) {
	parts := strings.Split(strings.TrimSpace(raw), ":")
	if len(parts) > 2 || strings.TrimSpace(parts[0]) == "" {
		return contract.MemoryScopeSelectorPayload{}, errors.New(
			"memory.promote.selector_invalid: selector must be scope[:tier]",
		)
	}
	scope, err := parseOptionalCLIMemoryScope(parts[0])
	if err != nil {
		return contract.MemoryScopeSelectorPayload{}, err
	}
	if scope == "" {
		return contract.MemoryScopeSelectorPayload{}, errors.New(
			"memory.promote.selector_invalid: selector scope is required",
		)
	}
	promoteFlags := flags
	promoteFlags.Scope = string(scope)
	if len(parts) == 2 {
		promoteFlags.AgentTier = strings.TrimSpace(parts[1])
	}
	if scope != memcontract.ScopeAgent {
		promoteFlags.Agent = ""
		promoteFlags.AgentTier = ""
	}
	selector, err := resolveMemorySelectorFlags(deps, promoteFlags, memorySelectorOptions{DefaultWorkspace: true})
	if err != nil {
		return contract.MemoryScopeSelectorPayload{}, err
	}
	return contract.MemoryScopeSelectorPayload{
		Scope:       selector.Scope,
		WorkspaceID: selector.WorkspaceID,
		AgentName:   selector.AgentName,
		AgentTier:   selector.AgentTier,
	}, nil
}

func boolStatus(value bool) string {
	if value {
		return toolBoolTrue
	}
	return toolBoolFalse
}

func memoryListBundle(response MemoryListRecord, now func() time.Time) outputBundle {
	items := make([]memoryListItem, 0, len(response.Memories))
	for _, memory := range response.Memories {
		items = append(items, memoryListItem{
			Filename:        memory.Filename,
			Name:            memory.Name,
			Type:            memory.Type,
			Scope:           memory.Scope,
			WorkspaceID:     memory.WorkspaceID,
			AgentName:       memory.AgentName,
			AgentTier:       memory.AgentTier,
			Age:             formatAge(now, memory.ModTime),
			Description:     memory.Description,
			StalenessBanner: memory.StalenessBanner,
			ModTime:         memory.ModTime,
		})
	}
	bundle := listBundle(
		response,
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
				stringOrDash(memoryScopeLabel(item.Scope, item.AgentTier)),
				stringOrDash(item.Age),
				stringOrDash(item.Description),
			}
		},
		func(item memoryListItem) []string {
			return []string{
				item.Filename,
				item.Name,
				string(item.Type),
				memoryScopeLabel(item.Scope, item.AgentTier),
				item.Age,
				item.Description,
			}
		},
	)
	bundle.jsonl = func(cmd *cobra.Command) error {
		return writeJSONLines(cmd, response.Memories)
	}
	return bundle
}

func memoryEntryBundle(response MemoryEntryRecord) outputBundle {
	return outputBundle{
		jsonValue: response,
		jsonl: func(cmd *cobra.Command) error {
			return writeJSONLine(cmd, response)
		},
		human: func() (string, error) {
			return strings.TrimRight(response.Memory.Content, "\n"), nil
		},
		toon: func() (string, error) {
			summary := response.Memory.Summary
			return renderToonObject("memory", []string{"filename", "scope", "content"}, []string{
				summary.Filename,
				memoryScopeLabel(summary.Scope, summary.AgentTier),
				response.Memory.Content,
			}), nil
		},
	}
}

func memorySearchBundle(response MemorySearchRecord) outputBundle {
	items := make([]memorySearchItem, 0, len(response.Results))
	for _, result := range response.Results {
		items = append(items, memorySearchItem{
			Filename:      result.Memory.Filename,
			Name:          result.Memory.Name,
			Type:          result.Memory.Type,
			Scope:         result.Memory.Scope,
			AgentTier:     result.Memory.AgentTier,
			Score:         result.Score,
			Snippet:       result.Snippet,
			WhyRecalled:   result.WhyRecalled,
			ShadowedBy:    result.ShadowedBy,
			AlreadyShown:  result.AlreadyShown,
			StalenessNote: result.Memory.StalenessBanner,
		})
	}
	bundle := listBundle(
		response,
		items,
		"Memory Search",
		[]string{"Filename", "Name", "Scope", "Score", "Snippet"},
		"results",
		[]string{"filename", "name", "scope", "score", "snippet"},
		func(item memorySearchItem) []string {
			return []string{
				stringOrDash(item.Filename),
				stringOrDash(item.Name),
				stringOrDash(memoryScopeLabel(item.Scope, item.AgentTier)),
				fmt.Sprintf("%.2f", item.Score),
				stringOrDash(item.Snippet),
			}
		},
		func(item memorySearchItem) []string {
			return []string{
				item.Filename,
				item.Name,
				memoryScopeLabel(item.Scope, item.AgentTier),
				fmt.Sprintf("%.2f", item.Score),
				item.Snippet,
			}
		},
	)
	bundle.jsonl = func(cmd *cobra.Command) error {
		return writeJSONLines(cmd, response.Results)
	}
	return bundle
}

func memoryHealthBundle(view MemoryHealthRecord) outputBundle {
	return outputBundle{
		jsonValue: view,
		jsonl: func(cmd *cobra.Command) error {
			return writeJSONLine(cmd, view)
		},
		human: func() (string, error) {
			return renderHumanSection("Memory Health", []keyValue{
				{Label: "Status", Value: stringOrDash(view.Status)},
				{Label: "Reason", Value: stringOrDash(view.Reason)},
				{Label: "Enabled", Value: fmt.Sprintf("%t", view.Enabled)},
				{Label: "Configured", Value: fmt.Sprintf("%t", view.Configured)},
				{Label: "Global Dir", Value: stringOrDash(view.GlobalDir)},
				{Label: "Global Files", Value: fmt.Sprintf("%d", view.GlobalFiles)},
				{Label: "Workspace Files", Value: fmt.Sprintf("%d", view.WorkspaceFiles)},
				{Label: "Workspace Count", Value: fmt.Sprintf("%d", view.WorkspaceCount)},
				{Label: "Indexed Files", Value: fmt.Sprintf("%d", view.IndexedFiles)},
				{Label: "Orphaned Files", Value: fmt.Sprintf("%d", view.OrphanedFiles)},
				{Label: "Operation Count", Value: fmt.Sprintf("%d", view.OperationCount)},
				{Label: "Last Operation", Value: stringOrDash(formatMemoryOptionalTime(view.LastOperationAt))},
				{Label: "Dream Enabled", Value: fmt.Sprintf("%t", view.DreamEnabled)},
				{Label: "Dream Agent", Value: stringOrDash(view.DreamAgent)},
			}), nil
		},
		toon: func() (string, error) {
			return renderToonObject(
				"memory_health",
				[]string{"status", "enabled", "configured", "global_files", "workspace_files", "operation_count"},
				[]string{
					view.Status,
					fmt.Sprintf("%t", view.Enabled),
					fmt.Sprintf("%t", view.Configured),
					fmt.Sprintf("%d", view.GlobalFiles),
					fmt.Sprintf("%d", view.WorkspaceFiles),
					fmt.Sprintf("%d", view.OperationCount),
				},
			), nil
		},
	}
}

func memoryHistoryBundle(records []MemoryHistoryRecord, now func() time.Time) outputBundle {
	items := make([]memoryHistoryItem, 0, len(records))
	for _, record := range records {
		items = append(items, memoryHistoryItem{
			ID:          record.ID,
			Operation:   record.Operation,
			Scope:       record.Scope,
			WorkspaceID: record.WorkspaceID,
			Filename:    record.Filename,
			AgentName:   record.AgentName,
			AgentTier:   record.AgentTier,
			Summary:     record.Summary,
			Age:         formatAge(now, record.Timestamp),
			Timestamp:   record.Timestamp,
		})
	}
	return listBundle(
		contract.MemoryOperationHistoryResponse{Operations: records},
		items,
		"Memory History",
		[]string{"Time", "Operation", "Scope", "Filename", "Summary"},
		"operations",
		[]string{"timestamp", "operation", "scope", "filename", "summary"},
		func(item memoryHistoryItem) []string {
			return []string{
				stringOrDash(formatTime(item.Timestamp)),
				stringOrDash(string(item.Operation)),
				stringOrDash(memoryScopeLabel(item.Scope, item.AgentTier)),
				stringOrDash(item.Filename),
				stringOrDash(item.Summary),
			}
		},
		func(item memoryHistoryItem) []string {
			return []string{
				formatTime(item.Timestamp),
				string(item.Operation),
				memoryScopeLabel(item.Scope, item.AgentTier),
				item.Filename,
				item.Summary,
			}
		},
	)
}

func memoryMutationBundle(title string, response MemoryMutationRecord) outputBundle {
	return memoryDecisionBundle(title, response.Decision, response)
}

func memoryDeleteBundle(response MemoryDeleteRecord) outputBundle {
	return memoryDecisionBundle("Memory Delete", response.Decision, response)
}

func memoryPromoteBundle(response MemoryPromoteRecord) outputBundle {
	return memoryDecisionBundle("Memory Promote", response.Decision, response)
}

func memoryDecisionRevertBundle(response MemoryDecisionRevertRecord) outputBundle {
	return memoryDecisionBundle("Memory Decision Revert", response.Decision, response)
}

func memoryDecisionBundle(title string, decision contract.MemoryDecisionPayload, jsonValue any) outputBundle {
	return outputBundle{
		jsonValue: jsonValue,
		jsonl: func(cmd *cobra.Command) error {
			return writeJSONLine(cmd, jsonValue)
		},
		human: func() (string, error) {
			return renderHumanSection(title, []keyValue{
				{Label: "Decision ID", Value: stringOrDash(decision.ID)},
				{Label: "Operation", Value: stringOrDash(string(decision.Op))},
				{Label: "Scope", Value: stringOrDash(memoryScopeLabel(decision.Scope, decision.AgentTier))},
				{Label: "Filename", Value: stringOrDash(decision.TargetFilename)},
				{Label: "Confidence", Value: fmt.Sprintf("%.2f", decision.Confidence)},
				{Label: "Source", Value: stringOrDash(string(decision.Source))},
				{Label: "Reason", Value: stringOrDash(decision.Reason)},
			}), nil
		},
		toon: func() (string, error) {
			return renderToonObject("memory_decision", []string{"id", "op", "scope", "filename", "reason"}, []string{
				decision.ID,
				string(decision.Op),
				memoryScopeLabel(decision.Scope, decision.AgentTier),
				decision.TargetFilename,
				decision.Reason,
			}), nil
		},
	}
}

func memoryReindexBundle(view MemoryReindexRecord) outputBundle {
	return outputBundle{
		jsonValue: view,
		jsonl: func(cmd *cobra.Command) error {
			return writeJSONLine(cmd, view)
		},
		human: func() (string, error) {
			return renderHumanSection("Memory Reindex", []keyValue{
				{Label: "Scope", Value: stringOrDash(memoryScopeLabel(view.Scope, view.AgentTier))},
				{Label: "Workspace ID", Value: stringOrDash(view.WorkspaceID)},
				{Label: "Agent", Value: stringOrDash(view.AgentName)},
				{Label: "Indexed Files", Value: fmt.Sprintf("%d", view.IndexedFiles)},
				{Label: "Completed At", Value: view.CompletedAt.Format(time.RFC3339)},
			}), nil
		},
		toon: func() (string, error) {
			return renderToonObject(
				"memory_reindex",
				[]string{"scope", "workspace_id", "agent_name", "indexed_files", "completed_at"},
				[]string{
					memoryScopeLabel(view.Scope, view.AgentTier),
					view.WorkspaceID,
					view.AgentName,
					fmt.Sprintf("%d", view.IndexedFiles),
					view.CompletedAt.Format(time.RFC3339),
				},
			), nil
		},
	}
}

func memoryScopeShowBundle(response MemoryScopeShowRecord) outputBundle {
	return memoryObjectBundle("Memory Scope", response)
}

func memoryDecisionListBundle(response MemoryDecisionListRecord) outputBundle {
	bundle := memoryObjectBundle("Memory Decisions", response)
	bundle.jsonl = func(cmd *cobra.Command) error {
		return writeJSONLines(cmd, response.Decisions)
	}
	return bundle
}

func memoryDreamTriggerBundle(response MemoryDreamTriggerRecord) outputBundle {
	return outputBundle{
		jsonValue: response,
		jsonl: func(cmd *cobra.Command) error {
			return writeJSONLine(cmd, response)
		},
		human: func() (string, error) {
			return renderHumanSection("Memory Dream Trigger", []keyValue{
				{Label: "Triggered", Value: boolStatus(response.Triggered)},
				{Label: "Status", Value: stringOrDash(string(response.Dream.Status))},
				{Label: "Scope", Value: stringOrDash(memoryScopeLabel(response.Dream.Scope, response.Dream.AgentTier))},
				{Label: "Reason", Value: stringOrDash(response.Reason)},
			}), nil
		},
		toon: func() (string, error) {
			return renderToonObject(
				"memory_dream_trigger",
				[]string{"triggered", "status", "scope", "reason"},
				[]string{
					boolStatus(response.Triggered),
					string(response.Dream.Status),
					memoryScopeLabel(response.Dream.Scope, response.Dream.AgentTier),
					response.Reason,
				},
			), nil
		},
	}
}

func memoryDreamListBundle(response MemoryDreamListRecord) outputBundle {
	bundle := memoryObjectBundle("Memory Dreams", response)
	bundle.jsonl = func(cmd *cobra.Command) error {
		return writeJSONLines(cmd, response.Dreams)
	}
	return bundle
}

func memoryProviderListBundle(response MemoryProviderListRecord) outputBundle {
	bundle := listBundle(
		response,
		response.Providers,
		"Memory Providers",
		[]string{"Name", "Status", "Active", "Builtin", "Failure Count"},
		"providers",
		[]string{"name", "status", "active", "builtin", "failure_count"},
		func(item contract.MemoryProviderPayload) []string {
			return []string{
				stringOrDash(item.Name),
				stringOrDash(string(item.Status)),
				boolStatus(item.Active),
				boolStatus(item.Builtin),
				fmt.Sprintf("%d", item.FailureCount),
			}
		},
		func(item contract.MemoryProviderPayload) []string {
			return []string{
				item.Name,
				string(item.Status),
				boolStatus(item.Active),
				boolStatus(item.Builtin),
				fmt.Sprintf("%d", item.FailureCount),
			}
		},
	)
	bundle.jsonl = func(cmd *cobra.Command) error {
		return writeJSONLines(cmd, response.Providers)
	}
	return bundle
}

func memoryObjectBundle(title string, value any) outputBundle {
	return outputBundle{
		jsonValue: value,
		jsonl: func(cmd *cobra.Command) error {
			return writeJSONLine(cmd, value)
		},
		human: func() (string, error) {
			data, err := json.MarshalIndent(value, "", "  ")
			if err != nil {
				return "", fmt.Errorf("cli: render %s: %w", title, err)
			}
			return renderHumanBlocks(title, string(data)), nil
		},
		toon: func() (string, error) {
			data, err := json.Marshal(value)
			if err != nil {
				return "", fmt.Errorf("cli: render %s toon: %w", title, err)
			}
			return renderToonObject("memory", []string{"payload"}, []string{string(data)}), nil
		},
	}
}

func memoryScopeLabel(scope memcontract.Scope, tier memcontract.AgentTier) string {
	normalized := scope.Normalize()
	if normalized == memcontract.ScopeAgent && tier.Normalize() != "" {
		return string(normalized) + ":" + string(tier.Normalize())
	}
	return string(normalized)
}

func formatMemoryOptionalTime(value *time.Time) string {
	if value == nil {
		return ""
	}
	return formatTime(*value)
}
