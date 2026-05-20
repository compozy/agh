package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	burnttoml "github.com/BurntSushi/toml"
	"github.com/kballard/go-shellquote"
	"github.com/pedronauck/agh/internal/api/contract"
	aghconfig "github.com/pedronauck/agh/internal/config"
	settingspkg "github.com/pedronauck/agh/internal/settings"
	"github.com/spf13/cobra"
)

const (
	configManagedValue         = "Managed"
	configManagerValue         = "Manager"
	configPathValue            = "Path"
	configRedactedValue        = "Redacted"
	configScopeValue           = "Scope"
	configStatusValue          = "Status"
	configTargetValue          = "Target"
	configValueValue           = "Value"
	configWorkspaceValue       = "Workspace"
	configBackendKey           = "backend"
	configCommandKey           = "command"
	configConfigKey            = "config"
	configDaemonKey            = "daemon"
	configDefaultsProviderPath = "defaults.provider"
	configEditKey              = "edit"
	configEnabledKey           = "enabled"
	configInvalidKey           = "invalid"
	configListKey              = "list"
	configManagedKey           = "managed"
	configManagerKey           = "manager"
	configMemoryKey            = "memory"
	configNetworkKey           = "network"
	configPathKey              = "path"
	configReadKey              = "read"
	configRedactedKey          = "redacted"
	configRequiredKey          = "required"
	configScopeKey             = "scope"
	configShowKey              = "show"
	configSkillsKey            = "skills"
	configStatusKey            = "status"
	configTargetKey            = "target"
	configWorkspaceRootKey     = "workspace_root"
)

const (
	configEnvKey        = "env"
	configSecretEnvKey  = "secret_env"
	configProvidersKey  = "providers"
	configModelsKey     = "models"
	configDiscoveryKey  = "discovery"
	configDefaultKey    = "default"
	configSessionMCPKey = "session_mcp"
)

type configEntry struct {
	Path     string `json:"path"`
	Value    any    `json:"value"`
	Redacted bool   `json:"redacted"`
}

type configShowRecord struct {
	Scope         string         `json:"scope"`
	WorkspaceRoot string         `json:"workspace_root,omitempty"`
	Redacted      bool           `json:"redacted"`
	Config        map[string]any `json:"config"`
}

type configListRecord struct {
	Scope         string        `json:"scope"`
	WorkspaceRoot string        `json:"workspace_root,omitempty"`
	Redacted      bool          `json:"redacted"`
	Entries       []configEntry `json:"entries"`
}

type configValueRecord struct {
	Path     string `json:"path"`
	Value    any    `json:"value"`
	Redacted bool   `json:"redacted"`
}

type configSetRecord struct {
	Path             string `json:"path"`
	Value            any    `json:"value"`
	Scope            string `json:"scope"`
	Target           string `json:"target"`
	Redacted         bool   `json:"redacted"`
	Lifecycle        string `json:"lifecycle"`
	ApplyRecordID    string `json:"apply_record_id,omitempty"`
	Applied          bool   `json:"applied"`
	ActiveGeneration int64  `json:"active_generation,omitempty"`
	ActiveConfigHash string `json:"active_config_hash,omitempty"`
	NextAction       string `json:"next_action,omitempty"`
	RestartRequired  bool   `json:"restart_required"`
	RestartScope     string `json:"restart_scope,omitempty"`
}

type configApplyHistoryRecord struct {
	Entries []contract.ConfigApplyRecordPayload `json:"entries"`
}

type configPathRecord struct {
	HomeDir              string `json:"home_dir"`
	GlobalConfig         string `json:"global_config"`
	GlobalMCPJSON        string `json:"global_mcp_json"`
	Scope                string `json:"scope"`
	WorkspaceRoot        string `json:"workspace_root,omitempty"`
	WorkspaceConfig      string `json:"workspace_config,omitempty"`
	WorkspaceMCPJSON     string `json:"workspace_mcp_json,omitempty"`
	Managed              bool   `json:"managed"`
	Manager              string `json:"manager,omitempty"`
	SelectedConfigTarget string `json:"selected_config_target"`
}

type configValidateRecord struct {
	Status        string                        `json:"status"`
	Scope         string                        `json:"scope"`
	WorkspaceRoot string                        `json:"workspace_root,omitempty"`
	ConfigFile    string                        `json:"config_file"`
	Redacted      bool                          `json:"redacted"`
	Errors        []configValidationError       `json:"errors,omitempty"`
	DotEnv        *aghconfig.DotEnvRepairReport `json:"dot_env,omitempty"`
}

type configValidationError struct {
	Code    string `json:"code"`
	Path    string `json:"path,omitempty"`
	File    string `json:"file,omitempty"`
	Line    int    `json:"line,omitempty"`
	Column  int    `json:"column,omitempty"`
	Message string `json:"message"`
}

type configValidationFailedError struct {
	err error
}

type configMutationLifecycle struct {
	Lifecycle       string
	Applied         bool
	RestartRequired bool
	RestartScope    string
	NextAction      string
}

func (e configValidationFailedError) Error() string {
	return e.err.Error()
}

func (e configValidationFailedError) Unwrap() error {
	return e.err
}

type configSetValueKind int

const (
	configSetString configSetValueKind = iota
	configSetBool
	configSetInt
	configSetInt64
	configSetFloat
	configSetDuration
	configSetStringSlice
)

var (
	configDurationType = reflect.TypeFor[time.Duration]()

	configScalarMutationKinds = map[string]configSetValueKind{
		"daemon.socket":                                   configSetString,
		"daemon.reload_timeouts.providers":                configSetDuration,
		"daemon.reload_timeouts.mcp":                      configSetDuration,
		"daemon.reload_timeouts.bridges":                  configSetDuration,
		"http.host":                                       configSetString,
		"http.port":                                       configSetInt,
		"defaults.agent":                                  configSetString,
		configDefaultsProviderPath:                        configSetString,
		"defaults.sandbox":                                configSetString,
		"limits.max_concurrent_agents":                    configSetInt,
		"session.limits.timeout":                          configSetDuration,
		"session.supervision.activity_heartbeat_interval": configSetDuration,
		"session.supervision.prompt_deadline":             configSetDuration,
		"session.supervision.progress_notify_interval":    configSetDuration,
		"session.supervision.inactivity_warning_after":    configSetDuration,
		"session.supervision.inactivity_timeout":          configSetDuration,
		"session.supervision.timeout_cancel_grace":        configSetDuration,
		"permissions.mode":                                configSetString,
		"observability.enabled":                           configSetBool,
		"observability.retention_days":                    configSetInt,
		"observability.max_global_bytes":                  configSetInt64,
		"observability.transcripts.enabled":               configSetBool,
		"observability.transcripts.segment_bytes":         configSetInt,
		"observability.transcripts.max_bytes_per_session": configSetInt64,
		"log.level":                                               configSetString,
		"memory.enabled":                                          configSetBool,
		"memory.global_dir":                                       configSetString,
		"memory.controller.mode":                                  configSetString,
		"memory.controller.max_latency":                           configSetDuration,
		"memory.controller.default_op_on_fail":                    configSetString,
		"memory.controller.llm.enabled":                           configSetBool,
		"memory.controller.llm.model":                             configSetString,
		"memory.controller.llm.top_k":                             configSetInt,
		"memory.controller.llm.prompt_version":                    configSetString,
		"memory.controller.llm.timeout":                           configSetDuration,
		"memory.controller.llm.max_tokens_out":                    configSetInt,
		"memory.controller.policy.max_content_chars":              configSetInt,
		"memory.controller.policy.max_writes_per_min":             configSetInt,
		"memory.controller.policy.allow_origins":                  configSetStringSlice,
		"memory.recall.top_k":                                     configSetInt,
		"memory.recall.raw_candidates":                            configSetInt,
		"memory.recall.fusion":                                    configSetString,
		"memory.recall.include_already_surfaced":                  configSetBool,
		"memory.recall.include_system":                            configSetBool,
		"memory.recall.weights.bm25_unicode":                      configSetFloat,
		"memory.recall.weights.bm25_trigram":                      configSetFloat,
		"memory.recall.weights.recency":                           configSetFloat,
		"memory.recall.weights.recall_signal":                     configSetFloat,
		"memory.recall.freshness.banner_after_days":               configSetInt,
		"memory.recall.signals.queue_capacity":                    configSetInt,
		"memory.recall.signals.worker_retry_max":                  configSetInt,
		"memory.recall.signals.metrics_enabled":                   configSetBool,
		"memory.decisions.prune_after_applied_days":               configSetInt,
		"memory.decisions.keep_audit_summary":                     configSetBool,
		"memory.decisions.max_post_content_bytes":                 configSetInt64,
		"memory.extractor.enabled":                                configSetBool,
		"memory.extractor.mode":                                   configSetString,
		"memory.extractor.throttle_turns":                         configSetInt,
		"memory.extractor.deadline":                               configSetDuration,
		"memory.extractor.sandbox_inbox_only":                     configSetBool,
		"memory.extractor.inbox_path":                             configSetString,
		"memory.extractor.dlq_path":                               configSetString,
		"memory.extractor.model":                                  configSetString,
		"memory.extractor.queue.capacity":                         configSetInt,
		"memory.extractor.queue.coalesce_max":                     configSetInt,
		"memory.dream.enabled":                                    configSetBool,
		"memory.dream.agent":                                      configSetString,
		"memory.dream.min_hours":                                  configSetFloat,
		"memory.dream.min_sessions":                               configSetInt,
		"memory.dream.debounce":                                   configSetDuration,
		"memory.dream.prompt_version":                             configSetString,
		"memory.dream.check_interval":                             configSetDuration,
		"memory.dream.gates.min_unpromoted":                       configSetInt,
		"memory.dream.gates.min_recall_count":                     configSetInt,
		"memory.dream.gates.min_score":                            configSetFloat,
		"memory.dream.scoring.recency_half_life_days":             configSetInt,
		"memory.dream.scoring.weights.frequency":                  configSetFloat,
		"memory.dream.scoring.weights.relevance":                  configSetFloat,
		"memory.dream.scoring.weights.recency":                    configSetFloat,
		"memory.dream.scoring.weights.freshness":                  configSetFloat,
		"memory.session.ledger_format":                            configSetString,
		"memory.session.ledger_root":                              configSetString,
		"memory.session.events_purge_grace":                       configSetDuration,
		"memory.session.cold_archive_days":                        configSetInt,
		"memory.session.hard_delete_days":                         configSetInt,
		"memory.session.max_archive_bytes":                        configSetInt64,
		"memory.session.unbound_partition":                        configSetString,
		"memory.daily.max_bytes":                                  configSetInt64,
		"memory.daily.max_lines":                                  configSetInt,
		"memory.daily.rotate_format":                              configSetString,
		"memory.daily.dreaming_window":                            configSetInt,
		"memory.daily.cold_archive_days":                          configSetInt,
		"memory.daily.hard_delete_days":                           configSetInt,
		"memory.daily.max_archive_bytes":                          configSetInt64,
		"memory.daily.sweep_hour":                                 configSetInt,
		"memory.daily.archive_path":                               configSetString,
		"memory.file.max_lines":                                   configSetInt,
		"memory.file.max_bytes":                                   configSetInt64,
		"memory.provider.name":                                    configSetString,
		"memory.provider.timeout":                                 configSetDuration,
		"memory.provider.failure_threshold":                       configSetInt,
		"memory.provider.cooldown":                                configSetDuration,
		"memory.workspace.auto_create":                            configSetBool,
		"skills.enabled":                                          configSetBool,
		"skills.disabled_skills":                                  configSetStringSlice,
		"skills.poll_interval":                                    configSetDuration,
		"skills.allowed_marketplace_mcp":                          configSetStringSlice,
		"skills.allowed_marketplace_hooks":                        configSetStringSlice,
		"skills.marketplace.registry":                             configSetString,
		"skills.marketplace.base_url":                             configSetString,
		"extensions.marketplace.registry":                         configSetString,
		"extensions.marketplace.base_url":                         configSetString,
		"extensions.resources.allowed_kinds":                      configSetStringSlice,
		"extensions.resources.max_scope":                          configSetString,
		"extensions.resources.snapshot_rate_limit.requests":       configSetInt,
		"extensions.resources.snapshot_rate_limit.window":         configSetDuration,
		"extensions.resources.snapshot_rate_limit.queue":          configSetInt,
		"extensions.resources.operator_write_rate_limit.requests": configSetInt,
		"extensions.resources.operator_write_rate_limit.window":   configSetDuration,
		"extensions.resources.operator_write_rate_limit.queue":    configSetInt,
		"model_catalog.sources.models_dev.enabled":                configSetBool,
		"model_catalog.sources.models_dev.endpoint":               configSetString,
		"model_catalog.sources.models_dev.ttl":                    configSetDuration,
		"model_catalog.sources.models_dev.timeout":                configSetDuration,
		"automation.enabled":                                      configSetBool,
		"automation.timezone":                                     configSetString,
		"automation.max_concurrent_jobs":                          configSetInt,
		"agents.soul.enabled":                                     configSetBool,
		"agents.soul.max_body_bytes":                              configSetInt64,
		"agents.soul.context_projection_bytes":                    configSetInt64,
		"agents.heartbeat.enabled":                                configSetBool,
		"agents.heartbeat.max_body_bytes":                         configSetInt64,
		"agents.heartbeat.context_projection_bytes":               configSetInt64,
		"agents.heartbeat.min_interval":                           configSetDuration,
		"agents.heartbeat.default_interval":                       configSetDuration,
		"agents.heartbeat.wake_cooldown":                          configSetDuration,
		"agents.heartbeat.max_wakes_per_cycle":                    configSetInt,
		"agents.heartbeat.active_session_only":                    configSetBool,
		"agents.heartbeat.allow_active_hours_preferences":         configSetBool,
		"agents.heartbeat.wake_event_retention":                   configSetDuration,
		"agents.heartbeat.session_health_stale_after":             configSetDuration,
		"agents.heartbeat.session_health_hook_min_interval":       configSetDuration,
		"network.enabled":                                         configSetBool,
		"network.default_channel":                                 configSetString,
		"network.port":                                            configSetInt,
		"network.max_payload":                                     configSetInt,
		"network.greet_interval":                                  configSetInt,
		"network.max_replay_age":                                  configSetInt,
		"network.max_queue_depth":                                 configSetInt,
	}
)

func newConfigCommand(deps commandDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   configConfigKey,
		Short: "Inspect and mutate AGH configuration",
	}
	cmd.AddCommand(newConfigShowCommand(deps))
	cmd.AddCommand(newConfigListCommand(deps))
	cmd.AddCommand(newConfigGetCommand(deps))
	cmd.AddCommand(newConfigSetCommand(deps))
	cmd.AddCommand(newConfigPathCommand(deps))
	cmd.AddCommand(newConfigValidateCommand(deps))
	cmd.AddCommand(newConfigCheckCommand(deps))
	cmd.AddCommand(newConfigEditCommand(deps))
	cmd.AddCommand(newConfigReloadCommand(deps))
	cmd.AddCommand(newConfigApplyHistoryCommand(deps))
	return cmd
}

func newConfigShowCommand(deps commandDeps) *cobra.Command {
	var workspaceRoot string
	cmd := &cobra.Command{
		Use:   configShowKey,
		Short: "Show the redacted effective config",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, resolvedWorkspace, err := loadConfigForDisplay(deps, workspaceRoot)
			if err != nil {
				return err
			}
			configMap := redactedConfigMap(&cfg)
			entries := flattenConfigEntries(configMap)
			record := configShowRecord{
				Scope:         scopeForWorkspace(resolvedWorkspace),
				WorkspaceRoot: resolvedWorkspace,
				Redacted:      true,
				Config:        configMap,
			}
			return writeCommandOutput(cmd, configShowBundle(record, entries))
		},
	}
	cmd.Flags().StringVar(&workspaceRoot, "workspace", "", "Workspace root whose overlay should be included")
	return cmd
}

func newConfigListCommand(deps commandDeps) *cobra.Command {
	var workspaceRoot string
	cmd := &cobra.Command{
		Use:   configListKey,
		Short: "List redacted effective config values",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, resolvedWorkspace, err := loadConfigForDisplay(deps, workspaceRoot)
			if err != nil {
				return err
			}
			record := configListRecord{
				Scope:         scopeForWorkspace(resolvedWorkspace),
				WorkspaceRoot: resolvedWorkspace,
				Redacted:      true,
				Entries:       flattenConfigEntries(redactedConfigMap(&cfg)),
			}
			return writeCommandOutput(cmd, configListBundle(record))
		},
	}
	cmd.Flags().StringVar(&workspaceRoot, "workspace", "", "Workspace root whose overlay should be included")
	return cmd
}

func newConfigGetCommand(deps commandDeps) *cobra.Command {
	var workspaceRoot string
	cmd := &cobra.Command{
		Use:   "get <path>",
		Short: "Get one redacted effective config value",
		Args:  exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _, err := loadConfigForDisplay(deps, workspaceRoot)
			if err != nil {
				return err
			}
			path := strings.TrimSpace(args[0])
			for _, entry := range flattenConfigEntries(redactedConfigMap(&cfg)) {
				if entry.Path == path {
					return writeCommandOutput(cmd, configValueBundle(configValueRecord(entry)))
				}
			}
			return fmt.Errorf("cli: config path %q not found", path)
		},
	}
	cmd.Flags().StringVar(&workspaceRoot, "workspace", "", "Workspace root whose overlay should be included")
	return cmd
}

func newConfigSetCommand(deps commandDeps) *cobra.Command {
	var (
		scopeRaw      string
		workspaceRoot string
	)
	cmd := &cobra.Command{
		Use:   "set <path> <value>",
		Short: "Set one config value through the validated config writer",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireUnmanagedForMutation(deps, "set config values"); err != nil {
				return err
			}
			homePaths, target, workspace, err := configWriteTarget(deps, scopeRaw, workspaceRoot)
			if err != nil {
				return err
			}
			if err := ensureWriteTargetParent(target); err != nil {
				return err
			}

			path, kind, redacted, err := configMutationPath(args[0])
			if err != nil {
				return err
			}
			lifecycle, err := classifyConfigSetLifecycle(path)
			if err != nil {
				return err
			}
			value, err := parseConfigSetValue(kind, args[1])
			if err != nil {
				return err
			}
			if liveRecord, err := maybeApplyConfigSetViaDaemon(
				cmd.Context(),
				deps,
				homePaths,
				target,
				path,
				value,
				redacted,
			); err != nil {
				return err
			} else if liveRecord != nil {
				return writeCommandOutput(cmd, configSetBundle(*liveRecord))
			}
			if _, err := aghconfig.EditConfigOverlay(
				homePaths,
				workspace,
				target,
				func(editor *aghconfig.OverlayEditor) error {
					return editor.SetValue(path, value)
				},
			); err != nil {
				return err
			}

			outputValue := value
			if redacted {
				outputValue = aghconfig.RedactedValue()
			}
			record := configSetRecord{
				Path:            strings.Join(path, "."),
				Value:           outputValue,
				Scope:           string(target.Scope()),
				Target:          target.Path(),
				Redacted:        redacted,
				Lifecycle:       lifecycle.Lifecycle,
				Applied:         lifecycle.Applied,
				NextAction:      lifecycle.NextAction,
				RestartRequired: lifecycle.RestartRequired,
				RestartScope:    lifecycle.RestartScope,
			}
			reloadRecord, err := maybeReloadConfigAfterLocalWrite(cmd.Context(), deps, homePaths, target, record)
			if err != nil {
				return err
			}
			if reloadRecord != nil {
				record = *reloadRecord
			}
			return writeCommandOutput(cmd, configSetBundle(record))
		},
	}
	cmd.Flags().
		StringVar(&scopeRaw, configScopeKey, string(aghconfig.WriteScopeGlobal), "Write scope: global or workspace")
	cmd.Flags().StringVar(&workspaceRoot, "workspace", "", "Workspace root for workspace-scoped writes")
	return cmd
}

func newConfigPathCommand(deps commandDeps) *cobra.Command {
	var (
		scopeRaw      string
		workspaceRoot string
	)
	cmd := &cobra.Command{
		Use:   configPathKey,
		Short: "Show resolved AGH config paths",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			scope, err := parseWriteScope(scopeRaw)
			if err != nil {
				return err
			}
			homeWorkspace := ""
			if scope == aghconfig.WriteScopeWorkspace || strings.TrimSpace(workspaceRoot) != "" {
				homeWorkspace, err = resolveConfigWorkspaceRoot(deps, workspaceRoot)
				if err != nil {
					return err
				}
			} else {
				homeWorkspace, err = currentWorkingDirectory(deps)
				if err != nil {
					return err
				}
			}
			homePaths, err := deps.resolveHomeForWorkspace(homeWorkspace)
			if err != nil {
				return err
			}
			globalMCP, err := aghconfig.ResolveMCPSidecarWriteTarget(homePaths, "", aghconfig.WriteScopeGlobal)
			if err != nil {
				return err
			}
			selected, err := aghconfig.ResolveConfigWriteTarget(homePaths, "", aghconfig.WriteScopeGlobal)
			if err != nil {
				return err
			}
			record := configPathRecord{
				HomeDir:              homePaths.HomeDir,
				GlobalConfig:         homePaths.ConfigFile,
				GlobalMCPJSON:        globalMCP.Path(),
				Scope:                string(scope),
				Managed:              detectManagedState(deps).Managed,
				Manager:              detectManagedState(deps).Manager,
				SelectedConfigTarget: selected.Path(),
			}
			if scope == aghconfig.WriteScopeWorkspace || strings.TrimSpace(workspaceRoot) != "" {
				workspace := homeWorkspace
				workspaceConfig, err := aghconfig.ResolveConfigWriteTarget(
					homePaths,
					workspace,
					aghconfig.WriteScopeWorkspace,
				)
				if err != nil {
					return err
				}
				workspaceMCP, err := aghconfig.ResolveMCPSidecarWriteTarget(
					homePaths,
					workspace,
					aghconfig.WriteScopeWorkspace,
				)
				if err != nil {
					return err
				}
				record.WorkspaceRoot = workspace
				record.WorkspaceConfig = workspaceConfig.Path()
				record.WorkspaceMCPJSON = workspaceMCP.Path()
				if scope == aghconfig.WriteScopeWorkspace {
					record.SelectedConfigTarget = workspaceConfig.Path()
				}
			}
			return writeCommandOutput(cmd, configPathBundle(record))
		},
	}
	cmd.Flags().
		StringVar(&scopeRaw, configScopeKey, string(aghconfig.WriteScopeGlobal), "Path scope: global or workspace")
	cmd.Flags().StringVar(&workspaceRoot, "workspace", "", "Workspace root for workspace-scoped paths")
	return cmd
}

func newConfigValidateCommand(deps commandDeps) *cobra.Command {
	return newConfigValidateCommandNamed(deps, "validate")
}

func newConfigCheckCommand(deps commandDeps) *cobra.Command {
	cmd := newConfigValidateCommandNamed(deps, "check")
	cmd.Short = "Alias for config validate"
	return cmd
}

func newConfigValidateCommandNamed(deps commandDeps, name string) *cobra.Command {
	var workspaceRoot string
	var repairEnv bool
	cmd := &cobra.Command{
		Use:   name,
		Short: "Validate AGH configuration",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			workspace, err := resolveOptionalConfigWorkspaceRoot(workspaceRoot)
			if err != nil {
				return err
			}
			homeWorkspace := workspace
			if homeWorkspace == "" {
				homeWorkspace, err = currentWorkingDirectory(deps)
				if err != nil {
					return err
				}
			}
			homePaths, err := deps.resolveHomeForWorkspace(homeWorkspace)
			if err != nil {
				return err
			}
			var dotenvReport *aghconfig.DotEnvRepairReport
			if repairEnv {
				if workspace == "" {
					workspace, err = currentWorkingDirectory(deps)
					if err != nil {
						return err
					}
				}
				report, err := aghconfig.RepairDotEnvFile(aghconfig.WorkspaceDotEnvFile(workspace))
				dotenvReport = &report
				if err != nil {
					return err
				}
			}
			loadOptions := []aghconfig.LoadOption{}
			if workspace != "" {
				loadOptions = append(loadOptions, aghconfig.WithWorkspaceRoot(workspace))
			}
			if _, err := aghconfig.LoadForHome(homePaths, loadOptions...); err != nil {
				record := configValidateRecord{
					Status:        configInvalidKey,
					Scope:         scopeForWorkspace(workspace),
					WorkspaceRoot: workspace,
					ConfigFile:    homePaths.ConfigFile,
					Redacted:      true,
					Errors:        configValidationErrors(err),
					DotEnv:        dotenvReport,
				}
				if writeErr := writeCommandOutput(cmd, configValidateBundle(record)); writeErr != nil {
					return writeErr
				}
				return configValidationFailedError{err: err}
			}
			return writeCommandOutput(cmd, configValidateBundle(configValidateRecord{
				Status:        configValidKey,
				Scope:         scopeForWorkspace(workspace),
				WorkspaceRoot: workspace,
				ConfigFile:    homePaths.ConfigFile,
				Redacted:      true,
				DotEnv:        dotenvReport,
			}))
		},
	}
	cmd.Flags().StringVar(&workspaceRoot, "workspace", "", "Workspace root whose overlay should be validated")
	cmd.Flags().BoolVar(&repairEnv, "repair-env", false, "Repair a structured workspace .env before validating")
	return cmd
}

func configValidationErrors(err error) []configValidationError {
	record := configValidationError{
		Code:    "config.invalid",
		Message: err.Error(),
	}
	if fileErr, ok := errors.AsType[aghconfig.FileError](err); ok {
		record.File = fileErr.Path
		switch fileErr.Op {
		case "decode":
			record.Code = "config.decode"
		case configReadKey:
			record.Code = "config.read"
		default:
			record.Code = "config.file"
		}
	}
	if parseErr, ok := errors.AsType[burnttoml.ParseError](err); ok {
		record.Code = "config.parse"
		record.Line = parseErr.Position.Line
		record.Column = parseErr.Position.Col
		record.Message = parseErr.Message
	}
	if validationErr, ok := errors.AsType[aghconfig.ValidationError](err); ok {
		record.Code = "config.validation"
		record.Path = validationErr.Path
		record.Message = validationErr.Error()
	}
	return []configValidationError{record}
}

func newConfigEditCommand(deps commandDeps) *cobra.Command {
	var (
		scopeRaw      string
		workspaceRoot string
	)
	cmd := &cobra.Command{
		Use:   configEditKey,
		Short: "Open the selected config overlay in $VISUAL or $EDITOR",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := requireUnmanagedForMutation(deps, "edit config"); err != nil {
				return err
			}
			homePaths, target, workspace, err := configWriteTarget(deps, scopeRaw, workspaceRoot)
			if err != nil {
				return err
			}
			if err := ensureWriteTargetParent(target); err != nil {
				return err
			}
			if err := ensureEditableConfigFile(target.Path()); err != nil {
				return err
			}
			if err := runConfigEditor(cmd, deps, target.Path()); err != nil {
				return err
			}
			loadOptions := []aghconfig.LoadOption{}
			if workspace != "" {
				loadOptions = append(loadOptions, aghconfig.WithWorkspaceRoot(workspace))
			}
			if _, err := aghconfig.LoadForHome(homePaths, loadOptions...); err != nil {
				return fmt.Errorf("cli: edited config failed validation: %w", err)
			}
			return writeCommandOutput(cmd, configSetBundle(configSetRecord{
				Path:            "",
				Value:           "edited",
				Scope:           string(target.Scope()),
				Target:          target.Path(),
				Lifecycle:       string(contract.SettingsApplyLifecycleRestartRequired),
				NextAction:      string(contract.SettingsApplyNextActionRestartDaemon),
				RestartRequired: true,
				RestartScope:    configDaemonKey,
			}))
		},
	}
	cmd.Flags().
		StringVar(&scopeRaw, configScopeKey, string(aghconfig.WriteScopeGlobal), "Edit scope: global or workspace")
	cmd.Flags().StringVar(&workspaceRoot, "workspace", "", "Workspace root for workspace-scoped edits")
	return cmd
}

func newConfigReloadCommand(deps commandDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reload",
		Short: "Reconcile config.toml with the daemon active generation",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return fmt.Errorf("cli: create daemon client for config reload: %w", err)
			}
			result, err := client.ReloadSettings(cmd.Context())
			if err != nil {
				return fmt.Errorf("cli: reload config via daemon settings surface: %w", err)
			}
			return writeCommandOutput(cmd, configSetBundle(configSetRecord{
				Path:             "config.toml",
				Value:            "reload",
				Scope:            string(aghconfig.WriteScopeGlobal),
				Target:           "daemon",
				Lifecycle:        string(result.Lifecycle),
				ApplyRecordID:    result.ApplyRecordID,
				Applied:          result.Applied,
				ActiveGeneration: result.ActiveGeneration,
				ActiveConfigHash: result.ActiveConfigHash,
				NextAction:       string(result.NextAction),
				RestartRequired:  result.RestartRequired,
				RestartScope:     result.RestartScope,
			}))
		},
	}
	return cmd
}

func newConfigApplyHistoryCommand(deps commandDeps) *cobra.Command {
	var status string
	var actor string
	var limit int
	cmd := &cobra.Command{
		Use:   "apply-history",
		Short: "List persisted config apply records",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return fmt.Errorf("cli: create daemon client for config apply-history: %w", err)
			}
			history, err := client.ListSettingsApplyRecords(cmd.Context(), SettingsApplyHistoryQuery{
				Status: status,
				Actor:  actor,
				Limit:  limit,
			})
			if err != nil {
				return fmt.Errorf("cli: list config apply-history: %w", err)
			}
			return writeCommandOutput(cmd, configApplyHistoryBundle(history))
		},
	}
	cmd.Flags().StringVar(&status, configStatusKey, "", "Filter by apply status")
	cmd.Flags().StringVar(&actor, "actor", "", "Filter by apply actor")
	cmd.Flags().IntVar(&limit, "limit", 50, "Maximum records to return")
	return cmd
}

func loadConfigForDisplay(deps commandDeps, workspaceRoot string) (aghconfig.Config, string, error) {
	workspace, err := resolveOptionalConfigWorkspaceRoot(workspaceRoot)
	if err != nil {
		return aghconfig.Config{}, "", err
	}
	homeWorkspace := workspace
	if homeWorkspace == "" {
		homeWorkspace, err = currentWorkingDirectory(deps)
		if err != nil {
			return aghconfig.Config{}, "", err
		}
	}
	homePaths, err := deps.resolveHomeForWorkspace(homeWorkspace)
	if err != nil {
		return aghconfig.Config{}, "", err
	}
	loadOptions := []aghconfig.LoadOption{}
	if workspace != "" {
		loadOptions = append(loadOptions, aghconfig.WithWorkspaceRoot(workspace))
	}
	cfg, err := aghconfig.LoadForHome(homePaths, loadOptions...)
	if err != nil {
		return aghconfig.Config{}, "", err
	}
	return cfg, workspace, nil
}

func configWriteTarget(
	deps commandDeps,
	scopeRaw string,
	workspaceRoot string,
) (aghconfig.HomePaths, aghconfig.WriteTarget, string, error) {
	scope, err := parseWriteScope(scopeRaw)
	if err != nil {
		return aghconfig.HomePaths{}, aghconfig.WriteTarget{}, "", err
	}
	workspace := ""
	if scope == aghconfig.WriteScopeWorkspace {
		workspace, err = resolveConfigWorkspaceRoot(deps, workspaceRoot)
		if err != nil {
			return aghconfig.HomePaths{}, aghconfig.WriteTarget{}, "", err
		}
	} else {
		workspace, err = currentWorkingDirectory(deps)
		if err != nil {
			return aghconfig.HomePaths{}, aghconfig.WriteTarget{}, "", err
		}
	}
	homePaths, err := deps.resolveHomeForWorkspace(workspace)
	if err != nil {
		return aghconfig.HomePaths{}, aghconfig.WriteTarget{}, "", err
	}
	writeWorkspace := ""
	if scope == aghconfig.WriteScopeWorkspace {
		writeWorkspace = workspace
	}
	target, err := aghconfig.ResolveConfigWriteTarget(homePaths, writeWorkspace, scope)
	if err != nil {
		return aghconfig.HomePaths{}, aghconfig.WriteTarget{}, "", err
	}
	return homePaths, target, writeWorkspace, nil
}

func parseWriteScope(raw string) (aghconfig.WriteScope, error) {
	scope := aghconfig.WriteScope(strings.ToLower(strings.TrimSpace(raw)))
	if scope == "" {
		scope = aghconfig.WriteScopeGlobal
	}
	if err := scope.Validate(); err != nil {
		return "", err
	}
	return scope, nil
}

func resolveConfigWorkspaceRoot(deps commandDeps, raw string) (string, error) {
	if strings.TrimSpace(raw) != "" {
		return aghconfig.ResolvePath(raw)
	}
	return currentWorkingDirectory(deps)
}

func resolveOptionalConfigWorkspaceRoot(raw string) (string, error) {
	if strings.TrimSpace(raw) == "" {
		return "", nil
	}
	return aghconfig.ResolvePath(raw)
}

func scopeForWorkspace(workspaceRoot string) string {
	if strings.TrimSpace(workspaceRoot) == "" {
		return string(aghconfig.WriteScopeGlobal)
	}
	return string(aghconfig.WriteScopeWorkspace)
}

func redactedConfigMap(cfg *aghconfig.Config) map[string]any {
	node, ok := configNodeFromValue(reflect.ValueOf(cfg), "")
	if !ok {
		return map[string]any{}
	}
	values, ok := node.(map[string]any)
	if !ok {
		return map[string]any{}
	}
	return values
}

func configNodeFromValue(value reflect.Value, fieldName string) (any, bool) {
	value, ok := indirectConfigValue(value)
	if !ok {
		return nil, false
	}
	if value.Type() == configDurationType {
		return time.Duration(value.Int()).String(), true
	}
	switch value.Kind() {
	case reflect.Struct:
		return configStructNode(value)
	case reflect.Map:
		return configMapNode(value, fieldName)
	case reflect.Slice, reflect.Array:
		return configSequenceNode(value, fieldName)
	default:
		return configScalarNode(value)
	}
}

func indirectConfigValue(value reflect.Value) (reflect.Value, bool) {
	if !value.IsValid() {
		return reflect.Value{}, false
	}
	for value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return reflect.Value{}, false
		}
		value = value.Elem()
	}
	return value, true
}

func configStructNode(value reflect.Value) (any, bool) {
	result := make(map[string]any)
	valueType := value.Type()
	for i := 0; i < value.NumField(); i++ {
		field := valueType.Field(i)
		if field.PkgPath != "" {
			continue
		}
		name, omitEmpty, ok := tomlFieldName(field)
		if !ok {
			continue
		}
		fieldValue := value.Field(i)
		if omitEmpty && fieldValue.IsZero() {
			continue
		}
		node, hasValue := configNodeFromValue(fieldValue, name)
		if hasValue {
			result[name] = node
		}
	}
	return result, true
}

func configMapNode(value reflect.Value, fieldName string) (any, bool) {
	if value.IsNil() {
		return map[string]any{}, true
	}
	result := make(map[string]any, value.Len())
	for _, key := range sortedReflectMapKeys(value) {
		mapKey := fmt.Sprint(key.Interface())
		if strings.EqualFold(fieldName, configEnvKey) || strings.EqualFold(fieldName, configSecretEnvKey) {
			result[mapKey] = aghconfig.RedactedValue()
			continue
		}
		node, hasValue := configNodeFromValue(value.MapIndex(key), "")
		if hasValue {
			result[mapKey] = node
		}
	}
	return result, true
}

func sortedReflectMapKeys(value reflect.Value) []reflect.Value {
	keys := value.MapKeys()
	sort.Slice(keys, func(i int, j int) bool {
		return fmt.Sprint(keys[i].Interface()) < fmt.Sprint(keys[j].Interface())
	})
	return keys
}

func configSequenceNode(value reflect.Value, fieldName string) (any, bool) {
	items := make([]any, 0, value.Len())
	for i := 0; i < value.Len(); i++ {
		node, hasValue := configNodeFromValue(value.Index(i), fieldName)
		if hasValue {
			items = append(items, node)
		}
	}
	return items, true
}

func configScalarNode(value reflect.Value) (any, bool) {
	switch value.Kind() {
	case reflect.String:
		return value.String(), true
	case reflect.Bool:
		return value.Bool(), true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return value.Int(), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return value.Uint(), true
	case reflect.Float32, reflect.Float64:
		return value.Float(), true
	default:
		if value.CanInterface() {
			return fmt.Sprint(value.Interface()), true
		}
		return nil, false
	}
}

func tomlFieldName(field reflect.StructField) (string, bool, bool) {
	tag := field.Tag.Get("toml")
	if tag == "-" {
		return "", false, false
	}
	if tag == "" {
		return strings.ToLower(field.Name), false, true
	}
	parts := strings.Split(tag, ",")
	name := strings.TrimSpace(parts[0])
	if name == "" {
		return "", false, false
	}
	omitEmpty := false
	for _, part := range parts[1:] {
		if strings.TrimSpace(part) == "omitempty" {
			omitEmpty = true
			break
		}
	}
	return name, omitEmpty, true
}

func flattenConfigEntries(configMap map[string]any) []configEntry {
	entries := make([]configEntry, 0)
	flattenConfigValue(&entries, "", configMap, false)
	sort.Slice(entries, func(i int, j int) bool {
		return entries[i].Path < entries[j].Path
	})
	return entries
}

func flattenConfigValue(entries *[]configEntry, path string, value any, redacted bool) {
	switch typed := value.(type) {
	case map[string]any:
		if len(typed) == 0 {
			if path != "" {
				*entries = append(*entries, configEntry{Path: path, Value: map[string]any{}, Redacted: redacted})
			}
			return
		}
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			nextPath := key
			if path != "" {
				nextPath = path + "." + key
			}
			flattenConfigValue(
				entries,
				nextPath,
				typed[key],
				redacted || key == configEnvKey || key == configSecretEnvKey,
			)
		}
	case []any:
		if len(typed) == 0 {
			if path != "" {
				*entries = append(*entries, configEntry{Path: path, Value: []any{}, Redacted: redacted})
			}
			return
		}
		for i, item := range typed {
			flattenConfigValue(entries, fmt.Sprintf("%s[%d]", path, i), item, redacted)
		}
	default:
		if path != "" {
			*entries = append(*entries, configEntry{Path: path, Value: typed, Redacted: redacted})
		}
	}
}

func configShowBundle(record configShowRecord, entries []configEntry) outputBundle {
	return outputBundle{
		jsonValue: record,
		human: func() (string, error) {
			return renderConfigEntries("Config", entries), nil
		},
		toon: func() (string, error) {
			return renderConfigEntriesToon(configConfigKey, entries), nil
		},
	}
}

func configListBundle(record configListRecord) outputBundle {
	return outputBundle{
		jsonValue: record,
		human: func() (string, error) {
			return renderConfigEntries("Config", record.Entries), nil
		},
		toon: func() (string, error) {
			return renderConfigEntriesToon(configConfigKey, record.Entries), nil
		},
	}
}

func configValueBundle(record configValueRecord) outputBundle {
	return outputBundle{
		jsonValue: record,
		human: func() (string, error) {
			return fmt.Sprintf("%s: %s", record.Path, formatConfigValue(record.Value)), nil
		},
		toon: func() (string, error) {
			return renderToonObject("config_value", []string{configPathKey, hooksValueKey, configRedactedKey}, []string{
				record.Path,
				formatConfigValue(record.Value),
				strconv.FormatBool(record.Redacted),
			}), nil
		},
	}
}

func configSetBundle(record configSetRecord) outputBundle {
	rows := []keyValue{
		{Label: configPathValue, Value: stringOrDash(record.Path)},
		{Label: configValueValue, Value: formatConfigValue(record.Value)},
		{Label: configScopeValue, Value: stringOrDash(record.Scope)},
		{Label: configTargetValue, Value: stringOrDash(record.Target)},
		{Label: configRedactedValue, Value: strconv.FormatBool(record.Redacted)},
		{Label: "Lifecycle", Value: stringOrDash(record.Lifecycle)},
		{Label: "Applied", Value: strconv.FormatBool(record.Applied)},
		{Label: "Next Action", Value: stringOrDash(record.NextAction)},
		{Label: "Apply Record", Value: stringOrDash(record.ApplyRecordID)},
		{Label: "Active Generation", Value: strconv.FormatInt(record.ActiveGeneration, 10)},
		{Label: "Restart Required", Value: strconv.FormatBool(record.RestartRequired)},
		{Label: "Restart Scope", Value: stringOrDash(record.RestartScope)},
	}
	return outputBundle{
		jsonValue: record,
		human: func() (string, error) {
			return renderHumanSection("Config", rows), nil
		},
		toon: func() (string, error) {
			return renderToonObject("config_set", []string{
				configPathKey,
				hooksValueKey,
				configScopeKey,
				configTargetKey,
				configRedactedKey,
				"lifecycle",
				"applied",
				"next_action",
				"apply_record_id",
				"active_generation",
				"restart_required",
				"restart_scope",
			}, []string{
				record.Path,
				formatConfigValue(record.Value),
				record.Scope,
				record.Target,
				strconv.FormatBool(record.Redacted),
				record.Lifecycle,
				strconv.FormatBool(record.Applied),
				record.NextAction,
				record.ApplyRecordID,
				strconv.FormatInt(record.ActiveGeneration, 10),
				strconv.FormatBool(record.RestartRequired),
				record.RestartScope,
			}), nil
		},
	}
}

func maybeApplyConfigSetViaDaemon(
	ctx context.Context,
	deps commandDeps,
	homePaths aghconfig.HomePaths,
	target aghconfig.WriteTarget,
	path []string,
	value any,
	redacted bool,
) (*configSetRecord, error) {
	if !supportsDaemonManagedConfigSet(path, target) {
		return nil, nil
	}

	_, running, err := daemonInfo(homePaths, deps)
	if err != nil {
		return nil, fmt.Errorf("cli: inspect daemon state for config set: %w", err)
	}
	if !running {
		return nil, nil
	}

	disabledSkills, ok := value.([]string)
	if !ok {
		return nil, fmt.Errorf(
			"cli: config set %q expects a string slice payload, got %T",
			strings.Join(path, "."),
			value,
		)
	}

	cfg, err := deps.loadConfig()
	if err != nil {
		return nil, fmt.Errorf("cli: load current config for daemon-backed config set: %w", err)
	}
	cfg.Skills.DisabledSkills = append([]string(nil), disabledSkills...)

	client, err := clientFromDeps(deps)
	if err != nil {
		return nil, fmt.Errorf("cli: create daemon client for config set: %w", err)
	}
	result, err := client.UpdateSettingsSkills(ctx, UpdateSettingsSkillsRequest{
		Config: settingsSkillsPayloadFromConfig(cfg.Skills),
	})
	if err != nil {
		return nil, fmt.Errorf("cli: apply %q via daemon settings surface: %w", strings.Join(path, "."), err)
	}

	outputValue := value
	if redacted {
		outputValue = aghconfig.RedactedValue()
	}
	return &configSetRecord{
		Path:             strings.Join(path, "."),
		Value:            outputValue,
		Scope:            string(target.Scope()),
		Target:           target.Path(),
		Redacted:         redacted,
		Lifecycle:        string(result.Lifecycle),
		ApplyRecordID:    result.ApplyRecordID,
		Applied:          result.Applied,
		ActiveGeneration: result.ActiveGeneration,
		ActiveConfigHash: result.ActiveConfigHash,
		NextAction:       string(result.NextAction),
		RestartRequired:  result.RestartRequired,
		RestartScope:     result.RestartScope,
	}, nil
}

func maybeReloadConfigAfterLocalWrite(
	ctx context.Context,
	deps commandDeps,
	homePaths aghconfig.HomePaths,
	target aghconfig.WriteTarget,
	record configSetRecord,
) (*configSetRecord, error) {
	if target.Scope() != aghconfig.WriteScopeGlobal {
		return nil, nil
	}
	_, running, err := daemonInfo(homePaths, deps)
	if err != nil {
		return nil, fmt.Errorf("cli: inspect daemon state for config reload: %w", err)
	}
	if !running {
		return nil, nil
	}
	client, err := clientFromDeps(deps)
	if err != nil {
		return nil, fmt.Errorf("cli: create daemon client for config reload: %w", err)
	}
	result, err := client.ReloadSettings(ctx)
	if err != nil {
		return nil, fmt.Errorf("cli: reload config via daemon settings surface: %w", err)
	}
	record.Lifecycle = string(result.Lifecycle)
	record.ApplyRecordID = result.ApplyRecordID
	record.Applied = result.Applied
	record.ActiveGeneration = result.ActiveGeneration
	record.ActiveConfigHash = result.ActiveConfigHash
	record.NextAction = string(result.NextAction)
	record.RestartRequired = result.RestartRequired
	record.RestartScope = result.RestartScope
	return &record, nil
}

func supportsDaemonManagedConfigSet(path []string, target aghconfig.WriteTarget) bool {
	return target.Scope() == aghconfig.WriteScopeGlobal &&
		len(path) == 2 &&
		path[0] == configSkillsKey &&
		path[1] == "disabled_skills"
}

func settingsSkillsPayloadFromConfig(cfg aghconfig.SkillsConfig) contract.SettingsSkillsConfigPayload {
	return contract.SettingsSkillsConfigPayload{
		Enabled:                 cfg.Enabled,
		DisabledSkills:          append([]string(nil), cfg.DisabledSkills...),
		PollInterval:            cfg.PollInterval.String(),
		AllowedMarketplaceMCP:   append([]string(nil), cfg.AllowedMarketplaceMCP...),
		AllowedMarketplaceHooks: append([]string(nil), cfg.AllowedMarketplaceHooks...),
		Marketplace: contract.SettingsMarketplacePayload{
			Registry: cfg.Marketplace.Registry,
			BaseURL:  cfg.Marketplace.BaseURL,
		},
	}
}

func configApplyHistoryBundle(record SettingsApplyHistoryRecord) outputBundle {
	rows := make([][]string, 0, len(record.Entries))
	for _, entry := range record.Entries {
		rows = append(rows, []string{
			entry.ID,
			string(entry.Status),
			string(entry.Lifecycle),
			strconv.FormatInt(entry.Generation, 10),
			entry.Actor,
			string(entry.NextAction),
			entry.UpdatedAt.Format(time.RFC3339),
		})
	}
	return outputBundle{
		jsonValue: record,
		human: func() (string, error) {
			return renderHumanTable("Config Apply History", []string{
				"ID",
				"Status",
				"Lifecycle",
				"Generation",
				"Actor",
				"Next Action",
				"Updated",
			}, rows), nil
		},
		toon: func() (string, error) {
			data, err := json.Marshal(record)
			if err != nil {
				return "", fmt.Errorf("cli: marshal config apply-history toon payload: %w", err)
			}
			return renderToonObject("config_apply_history", []string{"entries"}, []string{string(data)}), nil
		},
	}
}

func configPathBundle(record configPathRecord) outputBundle {
	rows := []keyValue{
		{Label: "Home", Value: stringOrDash(record.HomeDir)},
		{Label: "Global Config", Value: stringOrDash(record.GlobalConfig)},
		{Label: "Global MCP JSON", Value: stringOrDash(record.GlobalMCPJSON)},
		{Label: configScopeValue, Value: stringOrDash(record.Scope)},
		{Label: "Selected Config Target", Value: stringOrDash(record.SelectedConfigTarget)},
		{Label: configManagedValue, Value: strconv.FormatBool(record.Managed)},
		{Label: configManagerValue, Value: stringOrDash(record.Manager)},
	}
	if record.WorkspaceRoot != "" {
		rows = append(rows,
			keyValue{Label: configWorkspaceValue, Value: record.WorkspaceRoot},
			keyValue{Label: "Workspace Config", Value: record.WorkspaceConfig},
			keyValue{Label: "Workspace MCP JSON", Value: record.WorkspaceMCPJSON},
		)
	}
	return outputBundle{
		jsonValue: record,
		human: func() (string, error) {
			return renderHumanSection("Config Paths", rows), nil
		},
		toon: func() (string, error) {
			return renderToonObject(
				"config_paths",
				[]string{
					"home_dir",
					"global_config",
					"global_mcp_json",
					configScopeKey,
					configWorkspaceRootKey,
					"selected_config_target",
					configManagedKey,
					configManagerKey,
				},
				[]string{
					record.HomeDir,
					record.GlobalConfig,
					record.GlobalMCPJSON,
					record.Scope,
					record.WorkspaceRoot,
					record.SelectedConfigTarget,
					strconv.FormatBool(record.Managed),
					record.Manager,
				},
			), nil
		},
	}
}

func configValidateBundle(record configValidateRecord) outputBundle {
	rows := []keyValue{
		{Label: configStatusValue, Value: stringOrDash(record.Status)},
		{Label: configScopeValue, Value: stringOrDash(record.Scope)},
		{Label: configWorkspaceValue, Value: stringOrDash(record.WorkspaceRoot)},
		{Label: "Config File", Value: stringOrDash(record.ConfigFile)},
		{Label: configRedactedValue, Value: strconv.FormatBool(record.Redacted)},
	}
	if record.DotEnv != nil {
		rows = append(rows,
			keyValue{Label: ".env Path", Value: stringOrDash(record.DotEnv.Path)},
			keyValue{Label: ".env Status", Value: stringOrDash(record.DotEnv.Status)},
			keyValue{Label: ".env Repaired", Value: strconv.FormatBool(record.DotEnv.Repaired)},
		)
		if len(record.DotEnv.Diagnostics) > 0 {
			rows = append(rows, keyValue{
				Label: ".env Diagnostics",
				Value: strings.Join(dotEnvDiagnosticSummaries(record.DotEnv.Diagnostics), "; "),
			})
		}
	}
	return outputBundle{
		jsonValue: record,
		human: func() (string, error) {
			return renderHumanSection("Config Validation", rows), nil
		},
		toon: func() (string, error) {
			fields := []string{
				configStatusKey,
				configScopeKey,
				configWorkspaceRootKey,
				"config_file",
				configRedactedKey,
			}
			values := []string{
				record.Status,
				record.Scope,
				record.WorkspaceRoot,
				record.ConfigFile,
				strconv.FormatBool(record.Redacted),
			}
			if record.DotEnv != nil {
				fields = append(fields, "dot_env_status", "dot_env_repaired")
				values = append(values, record.DotEnv.Status, strconv.FormatBool(record.DotEnv.Repaired))
			}
			return renderToonObject("config_validation", fields, values), nil
		},
	}
}

func dotEnvDiagnosticSummaries(diagnostics []aghconfig.DotEnvDiagnostic) []string {
	summaries := make([]string, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		location := ""
		if diagnostic.Line > 0 {
			location = fmt.Sprintf("line %d", diagnostic.Line)
		}
		if diagnostic.Key != "" {
			if location != "" {
				location += " "
			}
			location += diagnostic.Key
		}
		if location == "" {
			location = "file"
		}
		summaries = append(summaries, location+": "+diagnostic.Message)
	}
	return summaries
}

func renderConfigEntries(title string, entries []configEntry) string {
	return renderHumanTable(
		title,
		[]string{configPathValue, configValueValue, configRedactedValue},
		configEntryRows(entries),
	)
}

func configEntryRows(entries []configEntry) [][]string {
	rows := make([][]string, 0, len(entries))
	for _, entry := range entries {
		rows = append(rows, []string{
			entry.Path,
			formatConfigValue(entry.Value),
			strconv.FormatBool(entry.Redacted),
		})
	}
	return rows
}

func renderConfigEntriesToon(name string, entries []configEntry) string {
	rows := make([][]string, 0, len(entries))
	for _, entry := range entries {
		rows = append(rows, []string{entry.Path, formatConfigValue(entry.Value), strconv.FormatBool(entry.Redacted)})
	}
	return renderToonArray(name, []string{configPathKey, hooksValueKey, configRedactedKey}, rows)
}

func formatConfigValue(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case bool:
		return strconv.FormatBool(typed)
	case int:
		return strconv.Itoa(typed)
	case int64:
		return strconv.FormatInt(typed, 10)
	case uint64:
		return strconv.FormatUint(typed, 10)
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	default:
		payload, err := json.Marshal(typed)
		if err != nil {
			return fmt.Sprint(typed)
		}
		return compactJSON(payload)
	}
}

func configMutationPath(raw string) ([]string, configSetValueKind, bool, error) {
	segments, err := parseDottedConfigPath(raw)
	if err != nil {
		return nil, configSetString, false, err
	}
	kind, redacted, err := classifyConfigMutationPath(segments)
	if err != nil {
		return nil, configSetString, false, err
	}
	return segments, kind, redacted, nil
}

func parseDottedConfigPath(raw string) ([]string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, errors.New("cli: config path is required")
	}
	if strings.ContainsAny(trimmed, "[]") {
		return nil, fmt.Errorf("cli: config set does not support array paths: %q", trimmed)
	}
	parts := strings.Split(trimmed, ".")
	for _, part := range parts {
		if strings.TrimSpace(part) == "" {
			return nil, fmt.Errorf("cli: config path %q contains an empty segment", trimmed)
		}
	}
	return parts, nil
}

func classifyConfigMutationPath(path []string) (configSetValueKind, bool, error) {
	joined := strings.Join(path, ".")
	if kind, ok := configScalarMutationKinds[joined]; ok {
		return kind, false, nil
	}
	if len(path) == 3 && path[0] == configProvidersKey && path[2] == configSessionMCPKey {
		return configSetBool, false, nil
	}
	if len(path) == 5 &&
		path[0] == configProvidersKey &&
		path[2] == configModelsKey &&
		path[3] == configDiscoveryKey &&
		path[4] == configEnabledKey {
		return configSetBool, false, nil
	}
	if isProviderMutationPath(path) {
		return configSetString, false, nil
	}
	if kind, redacted, ok := classifySandboxMutationPath(path); ok {
		return kind, redacted, nil
	}

	return configSetString, false, fmt.Errorf("cli: config path %q is not supported by config set", joined)
}

func classifyConfigSetLifecycle(path []string) (configMutationLifecycle, error) {
	field := strings.Join(path, ".")
	section := settingsSectionForConfigMutation(path)
	if section == "" {
		return restartRequiredConfigLifecycle(), nil
	}
	classification, err := settingspkg.ClassifyMutation(settingspkg.MutationDescriptor{
		Section:       section,
		ChangedFields: []string{field},
	})
	if err != nil {
		return configMutationLifecycle{}, fmt.Errorf(
			"cli: classify lifecycle for config path %q: %w",
			field,
			err,
		)
	}
	return configLifecycleFromSettings(classification), nil
}

func settingsSectionForConfigMutation(path []string) settingspkg.SectionName {
	if len(path) == 0 {
		return ""
	}
	switch path[0] {
	case configDaemonKey, "defaults", "http", "limits", configPermissionsKey, sessionSessionKey:
		return settingspkg.SectionGeneral
	case configMemoryKey:
		return settingspkg.SectionMemory
	case configSkillsKey:
		return settingspkg.SectionSkills
	case "automation":
		return settingspkg.SectionAutomation
	case configNetworkKey:
		return settingspkg.SectionNetwork
	case "log", "observability":
		return settingspkg.SectionObservability
	case "extensions", "hooks":
		return settingspkg.SectionHooksExtensions
	case configProvidersKey:
		return settingspkg.SectionName(settingspkg.CollectionProviders)
	case "mcp-servers":
		return settingspkg.SectionName(settingspkg.CollectionMCPServers)
	case configPathSandboxes:
		return settingspkg.SectionName(settingspkg.CollectionSandboxes)
	default:
		return ""
	}
}

func configLifecycleFromSettings(classification settingspkg.MutationClassification) configMutationLifecycle {
	nextAction := contract.SettingsApplyNextActionNone
	if classification.RestartRequired {
		nextAction = contract.SettingsApplyNextActionRestartDaemon
	}
	return configMutationLifecycle{
		Lifecycle:       string(classification.Lifecycle),
		Applied:         classification.Applied,
		RestartRequired: classification.RestartRequired,
		RestartScope:    classification.RestartScope,
		NextAction:      string(nextAction),
	}
}

func restartRequiredConfigLifecycle() configMutationLifecycle {
	return configMutationLifecycle{
		Lifecycle:       string(contract.SettingsApplyLifecycleRestartRequired),
		RestartRequired: true,
		RestartScope:    configDaemonKey,
		NextAction:      string(contract.SettingsApplyNextActionRestartDaemon),
	}
}

const configPathSandboxes = "sandboxes"

func isProviderMutationPath(path []string) bool {
	if len(path) == 3 && path[0] == configProvidersKey {
		switch path[2] {
		case configCommandKey,
			providerAuthModeKey,
			"env_policy",
			"home_policy",
			"runtime_provider",
			"transport",
			"base_url",
			"auth_status_command",
			"auth_login_command":
			return true
		}
	}
	if len(path) == 4 && path[0] == configProvidersKey && path[2] == configModelsKey {
		if path[3] == configDefaultKey {
			return true
		}
	}
	if len(path) == 5 &&
		path[0] == configProvidersKey &&
		path[2] == configModelsKey &&
		path[3] == configDiscoveryKey {
		switch path[4] {
		case configCommandKey, "endpoint", "timeout":
			return true
		}
	}
	return false
}

func classifySandboxMutationPath(path []string) (configSetValueKind, bool, bool) {
	if len(path) == 4 && path[0] == configPathSandboxes {
		switch path[2] {
		case configEnvKey, configSecretEnvKey:
			return configSetString, true, true
		case configNetworkKey:
			return classifySandboxNetworkMutationPath(path[3])
		case "daytona":
			return classifySandboxDaytonaMutationPath(path[3])
		}
	}
	if len(path) == 3 && path[0] == configPathSandboxes {
		switch path[2] {
		case configBackendKey, "sync_mode", "persistence", "runtime_root":
			return configSetString, false, true
		}
	}
	return configSetString, false, false
}

func classifySandboxNetworkMutationPath(name string) (configSetValueKind, bool, bool) {
	switch name {
	case "allow_public_ingress", "allow_outbound", configRequiredKey:
		return configSetBool, false, true
	case "allow_list", "deny_list":
		return configSetStringSlice, false, true
	default:
		return configSetString, false, false
	}
}

func classifySandboxDaytonaMutationPath(name string) (configSetValueKind, bool, bool) {
	switch name {
	case "api_url", configTargetKey, "image", "snapshot", "class", "auto_stop", "auto_archive":
		return configSetString, false, true
	default:
		return configSetString, false, false
	}
}

func parseConfigSetValue(kind configSetValueKind, raw string) (any, error) {
	trimmed := strings.TrimSpace(raw)
	switch kind {
	case configSetString:
		return raw, nil
	case configSetBool:
		value, err := strconv.ParseBool(trimmed)
		if err != nil {
			return nil, fmt.Errorf("cli: parse bool value %q: %w", raw, err)
		}
		return value, nil
	case configSetInt:
		value, err := strconv.Atoi(trimmed)
		if err != nil {
			return nil, fmt.Errorf("cli: parse integer value %q: %w", raw, err)
		}
		return value, nil
	case configSetInt64:
		value, err := strconv.ParseInt(trimmed, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("cli: parse integer value %q: %w", raw, err)
		}
		return value, nil
	case configSetFloat:
		value, err := strconv.ParseFloat(trimmed, 64)
		if err != nil {
			return nil, fmt.Errorf("cli: parse float value %q: %w", raw, err)
		}
		return value, nil
	case configSetDuration:
		if _, err := time.ParseDuration(trimmed); err != nil {
			return nil, fmt.Errorf("cli: parse duration value %q: %w", raw, err)
		}
		return trimmed, nil
	case configSetStringSlice:
		return parseStringSliceValue(trimmed)
	default:
		return nil, fmt.Errorf("cli: unsupported config value kind %d", kind)
	}
}

func parseStringSliceValue(raw string) ([]string, error) {
	if strings.TrimSpace(raw) == "" {
		return []string{}, nil
	}
	if strings.HasPrefix(strings.TrimSpace(raw), "[") {
		var values []string
		if err := json.Unmarshal([]byte(raw), &values); err != nil {
			return nil, fmt.Errorf("cli: parse string array %q: %w", raw, err)
		}
		return values, nil
	}
	parts := strings.Split(raw, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			values = append(values, trimmed)
		}
	}
	return values, nil
}

func ensureEditableConfigFile(path string) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return fmt.Errorf("cli: create editable config file %q: %w", path, err)
	}
	return file.Close()
}

func runConfigEditor(cmd *cobra.Command, deps commandDeps, path string) error {
	editor := strings.TrimSpace(deps.getenv("VISUAL"))
	if editor == "" {
		editor = strings.TrimSpace(deps.getenv("EDITOR"))
	}
	if editor == "" {
		return errors.New("cli: config edit requires VISUAL or EDITOR")
	}
	parts, err := shellquote.Split(editor)
	if err != nil {
		return fmt.Errorf("cli: parse config editor command: %w", err)
	}
	if len(parts) == 0 {
		return errors.New("cli: config editor command is empty")
	}
	//nolint:gosec // VISUAL/EDITOR intentionally selects the local editor for config edit.
	editorCmd := exec.CommandContext(cmd.Context(), parts[0], append(parts[1:], path)...)
	editorCmd.Stdin = os.Stdin
	editorCmd.Stdout = cmd.OutOrStdout()
	editorCmd.Stderr = cmd.ErrOrStderr()
	if err := editorCmd.Run(); err != nil {
		return fmt.Errorf("cli: run config editor %q: %w", parts[0], err)
	}
	return nil
}
