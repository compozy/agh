package modelcatalog

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/kballard/go-shellquote"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/procutil"
	"github.com/pedronauck/agh/internal/providerenv"
	"github.com/pedronauck/agh/internal/vault"
)

const (
	defaultLiveDiscoveryTimeout = 10 * time.Second
	maxLiveDiscoveryPayloadSize = 8 << 20
)

// ProviderSecretResolver resolves provider credential refs for live discovery.
type ProviderSecretResolver interface {
	ResolveRef(ctx context.Context, ref string) (string, error)
}

// EnvSecretResolver resolves env: secret refs from an environment lookup.
type EnvSecretResolver struct {
	LookupEnv func(string) (string, bool)
}

var _ ProviderSecretResolver = EnvSecretResolver{}

// ResolveRef resolves one env-backed provider credential ref.
func (r EnvSecretResolver) ResolveRef(ctx context.Context, ref string) (string, error) {
	if ctx == nil {
		return "", fmt.Errorf("model catalog: provider secret context is required")
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	normalized := vault.NormalizeRef(ref)
	if !vault.IsEnvRef(normalized) {
		return "", fmt.Errorf("%w: %s", vault.ErrUnsupportedSecretRef, normalized)
	}
	envName, err := vault.EnvNameFromRef(normalized)
	if err != nil {
		return "", err
	}
	lookup := r.LookupEnv
	if lookup == nil {
		lookup = os.LookupEnv
	}
	value, ok := lookup(envName)
	if !ok || strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("%w: env:%s", vault.ErrMissingSecret, envName)
	}
	return value, nil
}

// DiscoveryCommandRequest describes one timeout-bound discovery subprocess.
type DiscoveryCommandRequest struct {
	ProviderID string
	Command    string
	Args       []string
	Env        []string
	Timeout    time.Duration
}

// DiscoveryCommandResult captures safe subprocess output for parsing.
type DiscoveryCommandResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// DiscoveryCommandExecutor runs a provider discovery command.
type DiscoveryCommandExecutor interface {
	RunDiscoveryCommand(ctx context.Context, req DiscoveryCommandRequest) (DiscoveryCommandResult, error)
}

// ExecDiscoveryCommandExecutor runs discovery commands as subprocesses.
type ExecDiscoveryCommandExecutor struct{}

var _ DiscoveryCommandExecutor = ExecDiscoveryCommandExecutor{}

// RunDiscoveryCommand runs one subprocess with the caller-supplied deadline.
func (ExecDiscoveryCommandExecutor) RunDiscoveryCommand(
	ctx context.Context,
	req DiscoveryCommandRequest,
) (DiscoveryCommandResult, error) {
	if ctx == nil {
		return DiscoveryCommandResult{}, fmt.Errorf("model catalog: discovery command context is required")
	}
	if strings.TrimSpace(req.Command) == "" {
		return DiscoveryCommandResult{}, fmt.Errorf("model catalog: discovery command is required")
	}
	// #nosec G204 -- discovery commands come from validated provider model discovery config.
	cmd := exec.CommandContext(ctx, req.Command, req.Args...)
	cmd.Env = append([]string(nil), req.Env...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	result := DiscoveryCommandResult{
		Stdout: strings.TrimSpace(stdout.String()),
		Stderr: strings.TrimSpace(stderr.String()),
	}
	if cmd.ProcessState != nil {
		result.ExitCode = cmd.ProcessState.ExitCode()
	}
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return result, fmt.Errorf("model catalog: discovery command timed out after %s: %w", req.Timeout, ctx.Err())
		}
		return result, fmt.Errorf("model catalog: discovery command failed: %w", err)
	}
	return result, nil
}

// LiveProviderSourcesConfig configures built-in provider live discovery sources.
type LiveProviderSourcesConfig struct {
	Providers       map[string]aghconfig.ProviderConfig
	HomePaths       aghconfig.HomePaths
	BaseEnv         []string
	SecretResolver  ProviderSecretResolver
	HTTPClient      *http.Client
	CommandExecutor DiscoveryCommandExecutor
	DefaultTimeout  time.Duration
}

// NewLiveProviderSources creates provider_live sources for known provider adapters.
func NewLiveProviderSources(cfg LiveProviderSourcesConfig) ([]Source, error) {
	providers := aghconfig.BuiltinProviders()
	maps.Copy(providers, cfg.Providers)
	providerIDs := make([]string, 0, len(providers))
	for providerID := range providers {
		if _, ok := liveProviderAdapters[providerID]; ok {
			providerIDs = append(providerIDs, providerID)
		}
	}
	sort.Strings(providerIDs)
	sources := make([]Source, 0, len(providerIDs))
	for _, providerID := range providerIDs {
		source, err := NewLiveProviderSource(providerID, providers[providerID], cfg)
		if err != nil {
			return nil, err
		}
		sources = append(sources, source)
	}
	return sources, nil
}

// NewLiveProviderSource creates one provider_live source.
func NewLiveProviderSource(
	providerID string,
	provider aghconfig.ProviderConfig,
	cfg LiveProviderSourcesConfig,
) (*LiveProviderSource, error) {
	trimmedProviderID := strings.TrimSpace(providerID)
	adapter, ok := liveProviderAdapters[trimmedProviderID]
	if !ok {
		return nil, fmt.Errorf(
			"model catalog: live discovery adapter for provider %q is not registered",
			trimmedProviderID,
		)
	}
	sourceID := SourceKindProviderLiveID(trimmedProviderID)
	if err := ValidateSourceIdentity(sourceID, SourceKindProviderLive); err != nil {
		return nil, err
	}
	timeout := cfg.DefaultTimeout
	if timeout <= 0 {
		timeout = defaultLiveDiscoveryTimeout
	}
	executor := cfg.CommandExecutor
	if executor == nil {
		executor = ExecDiscoveryCommandExecutor{}
	}
	secretResolver := cfg.SecretResolver
	if secretResolver == nil {
		secretResolver = EnvSecretResolver{}
	}
	return &LiveProviderSource{
		providerID:      trimmedProviderID,
		provider:        provider,
		adapter:         adapter,
		sourceID:        sourceID,
		homePaths:       cfg.HomePaths,
		baseEnv:         append([]string(nil), cfg.BaseEnv...),
		secretResolver:  secretResolver,
		httpClient:      cfg.HTTPClient,
		commandExecutor: executor,
		defaultTimeout:  timeout,
	}, nil
}

// SourceKindProviderLiveID returns the stable source id for a live provider source.
func SourceKindProviderLiveID(providerID string) string {
	return string(SourceKindProviderLive) + ":" + strings.TrimSpace(providerID)
}

// LiveProviderSource performs side-effect-free model discovery for one provider.
type LiveProviderSource struct {
	providerID      string
	provider        aghconfig.ProviderConfig
	adapter         liveProviderAdapter
	sourceID        string
	homePaths       aghconfig.HomePaths
	baseEnv         []string
	secretResolver  ProviderSecretResolver
	httpClient      *http.Client
	commandExecutor DiscoveryCommandExecutor
	defaultTimeout  time.Duration
}

var _ Source = (*LiveProviderSource)(nil)

// ID returns the provider_live source id.
func (s *LiveProviderSource) ID() string {
	return s.sourceID
}

// Kind returns provider_live.
func (s *LiveProviderSource) Kind() SourceKind {
	return SourceKindProviderLive
}

// Priority returns the provider_live merge priority.
func (s *LiveProviderSource) Priority() int {
	return PriorityProviderLive
}

// ProviderIDs returns the single AGH provider id this source owns.
func (s *LiveProviderSource) ProviderIDs() []string {
	return []string{s.providerID}
}

// ListModels discovers live provider models without touching ACP sessions.
func (s *LiveProviderSource) ListModels(ctx context.Context, opts ListOptions) ([]ModelRow, error) {
	if ctx == nil {
		return nil, fmt.Errorf("model catalog: live provider context is required")
	}
	if requested := strings.TrimSpace(opts.ProviderID); requested != "" && requested != s.providerID {
		return nil, nil
	}
	target, err := s.discoveryTarget()
	if err != nil {
		return nil, err
	}
	env, err := s.discoveryEnv(ctx)
	if err != nil {
		return nil, err
	}
	timeout := target.timeout
	if timeout <= 0 {
		timeout = s.defaultTimeout
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	now := defaultNow(opts.Now)
	switch target.kind {
	case liveDiscoveryHTTP:
		rows, err := s.listHTTP(runCtx, target.endpoint, env, timeout, now)
		if err != nil {
			return nil, err
		}
		return rows, nil
	case liveDiscoveryCommand:
		rows, err := s.listCommand(runCtx, target.command, env, timeout, now)
		if err != nil {
			return nil, err
		}
		return rows, nil
	default:
		return nil, fmt.Errorf("model catalog: provider %q has no side-effect-free model discovery path", s.providerID)
	}
}

type liveDiscoveryKind string

const (
	liveDiscoveryNone    liveDiscoveryKind = ""
	liveDiscoveryHTTP    liveDiscoveryKind = "http"
	liveDiscoveryCommand liveDiscoveryKind = "command"
)

type liveAuthScheme string

const (
	liveAuthNone      liveAuthScheme = ""
	liveAuthBearer    liveAuthScheme = "bearer"
	liveAuthAnthropic liveAuthScheme = "anthropic"
	liveAuthGemini    liveAuthScheme = "gemini"
)

type liveProviderAdapter struct {
	defaultKind       liveDiscoveryKind
	defaultEndpoint   string
	defaultCommand    string
	authScheme        liveAuthScheme
	authRequired      bool
	credentialEnvKeys []string
	headers           map[string]string
}

type liveDiscoveryTarget struct {
	kind     liveDiscoveryKind
	endpoint string
	command  string
	timeout  time.Duration
}

var liveProviderAdapters = map[string]liveProviderAdapter{
	"codex": {
		defaultKind:       liveDiscoveryHTTP,
		defaultEndpoint:   "https://api.openai.com/v1/models",
		authScheme:        liveAuthBearer,
		authRequired:      true,
		credentialEnvKeys: []string{"OPENAI_API_KEY"},
	},
	"openai": {
		defaultKind:       liveDiscoveryHTTP,
		defaultEndpoint:   "https://api.openai.com/v1/models",
		authScheme:        liveAuthBearer,
		authRequired:      true,
		credentialEnvKeys: []string{"OPENAI_API_KEY"},
	},
	"claude": {
		defaultKind:       liveDiscoveryHTTP,
		defaultEndpoint:   "https://api.anthropic.com/v1/models",
		authScheme:        liveAuthAnthropic,
		authRequired:      true,
		credentialEnvKeys: []string{"ANTHROPIC_API_KEY"},
		headers:           map[string]string{"anthropic-version": "2023-06-01"},
	},
	"anthropic": {
		defaultKind:       liveDiscoveryHTTP,
		defaultEndpoint:   "https://api.anthropic.com/v1/models",
		authScheme:        liveAuthAnthropic,
		authRequired:      true,
		credentialEnvKeys: []string{"ANTHROPIC_API_KEY"},
		headers:           map[string]string{"anthropic-version": "2023-06-01"},
	},
	"gemini": {
		defaultKind:       liveDiscoveryHTTP,
		defaultEndpoint:   "https://generativelanguage.googleapis.com/v1beta/models",
		authScheme:        liveAuthGemini,
		authRequired:      true,
		credentialEnvKeys: []string{"GEMINI_API_KEY", "GOOGLE_API_KEY"},
	},
	"openrouter": {
		defaultKind:       liveDiscoveryHTTP,
		defaultEndpoint:   "https://openrouter.ai/api/v1/models",
		authScheme:        liveAuthBearer,
		authRequired:      true,
		credentialEnvKeys: []string{"OPENROUTER_API_KEY"},
	},
	"vercel-ai-gateway": {
		defaultKind:       liveDiscoveryHTTP,
		defaultEndpoint:   "https://ai-gateway.vercel.sh/v1/models",
		authScheme:        liveAuthBearer,
		authRequired:      false,
		credentialEnvKeys: []string{"AI_GATEWAY_API_KEY", "VERCEL_AI_GATEWAY_API_KEY"},
	},
	"ollama": {
		defaultKind:     liveDiscoveryHTTP,
		defaultEndpoint: "http://localhost:11434/api/tags",
	},
	"opencode": {
		defaultKind:    liveDiscoveryCommand,
		defaultCommand: "opencode models",
	},
	"openclaw": {
		defaultKind: liveDiscoveryNone,
	},
	"hermes": {
		defaultKind: liveDiscoveryNone,
	},
	"pi": {
		defaultKind: liveDiscoveryNone,
	},
}

func (s *LiveProviderSource) discoveryTarget() (liveDiscoveryTarget, error) {
	discovery := s.provider.Models.Discovery
	configuredCommand := strings.TrimSpace(discovery.Command)
	configuredEndpoint := strings.TrimSpace(discovery.Endpoint)
	hasConfiguredPath := configuredCommand != "" || configuredEndpoint != ""
	if discovery.Enabled != nil && !*discovery.Enabled {
		return liveDiscoveryTarget{}, ErrSourceDisabled
	}
	if s.adapter.defaultKind == liveDiscoveryNone && discovery.Enabled == nil {
		if hasConfiguredPath {
			return liveDiscoveryTarget{}, ErrSourceDisabled
		}
		return liveDiscoveryTarget{}, fmt.Errorf(
			"model catalog: provider %q has no configured side-effect-free model discovery command or endpoint",
			s.providerID,
		)
	}
	timeout, err := s.discoveryTimeout(discovery.Timeout)
	if err != nil {
		return liveDiscoveryTarget{}, err
	}
	if configuredEndpoint != "" {
		return liveDiscoveryTarget{kind: liveDiscoveryHTTP, endpoint: configuredEndpoint, timeout: timeout}, nil
	}
	if configuredCommand != "" {
		return liveDiscoveryTarget{kind: liveDiscoveryCommand, command: configuredCommand, timeout: timeout}, nil
	}
	switch s.adapter.defaultKind {
	case liveDiscoveryHTTP:
		return liveDiscoveryTarget{
			kind:     liveDiscoveryHTTP,
			endpoint: s.defaultEndpoint(),
			timeout:  timeout,
		}, nil
	case liveDiscoveryCommand:
		return liveDiscoveryTarget{
			kind:    liveDiscoveryCommand,
			command: s.adapter.defaultCommand,
			timeout: timeout,
		}, nil
	default:
		return liveDiscoveryTarget{}, fmt.Errorf(
			"model catalog: provider %q has no configured side-effect-free model discovery command or endpoint",
			s.providerID,
		)
	}
}

func (s *LiveProviderSource) discoveryTimeout(raw string) (time.Duration, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return s.defaultTimeout, nil
	}
	timeout, err := time.ParseDuration(trimmed)
	if err != nil || timeout <= 0 {
		return 0, fmt.Errorf("model catalog: provider %q discovery timeout must be a positive duration", s.providerID)
	}
	return timeout, nil
}

func (s *LiveProviderSource) defaultEndpoint() string {
	baseURL := strings.TrimSpace(s.provider.BaseURL)
	if baseURL == "" {
		return s.adapter.defaultEndpoint
	}
	return joinEndpoint(baseURL, defaultEndpointPath(s.adapter.defaultEndpoint))
}

func joinEndpoint(baseURL string, path string) string {
	trimmedBase := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if trimmedBase == "" {
		return path
	}
	trimmedPath := strings.TrimSpace(path)
	if trimmedPath == "" {
		return trimmedBase
	}
	if parsed, err := url.Parse(trimmedBase); err == nil {
		basePath := strings.TrimRight(parsed.Path, "/")
		switch {
		case strings.HasSuffix(basePath, "/v1") && strings.HasPrefix(trimmedPath, "/v1/"):
			trimmedPath = strings.TrimPrefix(trimmedPath, "/v1")
		case strings.HasSuffix(basePath, "/v1beta") && strings.HasPrefix(trimmedPath, "/v1beta/"):
			trimmedPath = strings.TrimPrefix(trimmedPath, "/v1beta")
		case strings.HasSuffix(basePath, "/api/v1") && strings.HasPrefix(trimmedPath, "/api/v1/"):
			trimmedPath = strings.TrimPrefix(trimmedPath, "/api/v1")
		case strings.HasSuffix(basePath, "/api") && strings.HasPrefix(trimmedPath, "/api/"):
			trimmedPath = strings.TrimPrefix(trimmedPath, "/api")
		}
	}
	if strings.HasPrefix(trimmedPath, "/") {
		return trimmedBase + trimmedPath
	}
	return trimmedBase + "/" + trimmedPath
}

func defaultEndpointPath(endpoint string) string {
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Path == "" {
		return ""
	}
	if parsed.RawQuery != "" {
		return parsed.Path + "?" + parsed.RawQuery
	}
	return parsed.Path
}

func (s *LiveProviderSource) discoveryEnv(ctx context.Context) ([]string, error) {
	env := append([]string(nil), s.baseEnv...)
	switch s.provider.EffectiveEnvPolicy() {
	case aghconfig.ProviderEnvPolicyIsolated:
		env = procutil.IsolatedDaemonEnv(env)
	default:
		env = procutil.FilteredDaemonEnv(env)
	}
	env = providerenv.SetEnvValue(env, "AGH_PROVIDER", s.providerID)
	env = providerenv.SetEnvValue(env, "AGH_PROVIDER_AUTH_MODE", string(s.provider.EffectiveAuthMode()))
	env = providerenv.SetEnvValue(env, "AGH_PROVIDER_ENV_POLICY", string(s.provider.EffectiveEnvPolicy()))
	env = providerenv.SetEnvValue(env, "AGH_PROVIDER_HOME_POLICY", string(s.provider.EffectiveHomePolicy()))

	var err error
	env, err = providerenv.ApplyHomePolicy(s.homePaths, s.providerID, s.provider.EffectiveHomePolicy(), env)
	if err != nil {
		return nil, fmt.Errorf("model catalog: apply provider home policy for %q: %w", s.providerID, err)
	}
	if s.provider.EffectiveHarness() == aghconfig.ProviderHarnessPiACP &&
		s.provider.EffectiveAuthMode() == aghconfig.ProviderAuthModeNativeCLI {
		env, err = providerenv.ApplyPiAgentDirPolicy(s.homePaths, s.providerID, s.provider.EffectiveHomePolicy(), env)
		if err != nil {
			return nil, fmt.Errorf("model catalog: apply pi discovery home policy for %q: %w", s.providerID, err)
		}
	}
	if s.provider.EffectiveAuthMode() != aghconfig.ProviderAuthModeBoundSecret {
		return env, nil
	}
	for _, slot := range s.provider.EffectiveCredentialSlots() {
		next, err := s.injectProviderSecret(ctx, env, slot)
		if err != nil {
			return nil, err
		}
		env = next
	}
	return env, nil
}

func (s *LiveProviderSource) injectProviderSecret(
	ctx context.Context,
	env []string,
	slot aghconfig.ProviderCredentialSlot,
) ([]string, error) {
	targetEnv := strings.TrimSpace(slot.TargetEnv)
	secretRef := vault.NormalizeRef(slot.SecretRef)
	if targetEnv == "" || secretRef == "" {
		return env, nil
	}
	value, err := s.secretResolver.ResolveRef(ctx, secretRef)
	if err != nil {
		if !slot.Required && (errors.Is(err, vault.ErrMissingSecret) || errors.Is(err, vault.ErrSecretNotFound)) {
			return env, nil
		}
		return nil, fmt.Errorf("model catalog: resolve provider credential %q for %q: %w", slot.Name, s.providerID, err)
	}
	return providerenv.SetEnvValue(env, targetEnv, value), nil
}

func (s *LiveProviderSource) listHTTP(
	ctx context.Context,
	endpoint string,
	env []string,
	timeout time.Duration,
	now time.Time,
) (rows []ModelRow, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("model catalog: create live discovery request for %q: %w", s.providerID, err)
	}
	for key, value := range s.adapter.headers {
		req.Header.Set(key, value)
	}
	if err := s.applyRequestAuth(req, env); err != nil {
		return nil, err
	}
	client := s.httpClient
	if client == nil {
		client = &http.Client{Timeout: timeout}
	}
	resp, err := client.Do(req)
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, fmt.Errorf(
				"model catalog: live discovery for %q timed out after %s: %w",
				s.providerID,
				timeout,
				ctx.Err(),
			)
		}
		return nil, fmt.Errorf("model catalog: fetch live models for %q: %w", s.providerID, err)
	}
	defer func() {
		if _, copyErr := io.Copy(io.Discard, resp.Body); copyErr != nil && err == nil {
			err = fmt.Errorf("model catalog: drain live discovery response for %q: %w", s.providerID, copyErr)
		}
		if closeErr := resp.Body.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("model catalog: close live discovery response for %q: %w", s.providerID, closeErr)
		}
	}()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("model catalog: live discovery for %q returned HTTP %d", s.providerID, resp.StatusCode)
	}
	payload, err := io.ReadAll(io.LimitReader(resp.Body, maxLiveDiscoveryPayloadSize))
	if err != nil {
		return nil, fmt.Errorf("model catalog: read live discovery response for %q: %w", s.providerID, err)
	}
	rows, err = parseLiveModelPayload(s.providerID, payload, now)
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *LiveProviderSource) applyRequestAuth(req *http.Request, env []string) error {
	credential := firstEnvValue(env, s.adapter.credentialEnvKeys...)
	if credential == "" && s.adapter.authRequired {
		return fmt.Errorf(
			"model catalog: provider %q live discovery requires a bound_secret credential",
			s.providerID,
		)
	}
	if credential == "" {
		return nil
	}
	switch s.adapter.authScheme {
	case liveAuthBearer:
		req.Header.Set("Authorization", "Bearer "+credential)
	case liveAuthAnthropic:
		req.Header.Set("x-api-key", credential)
	case liveAuthGemini:
		req.Header.Set("x-goog-api-key", credential)
	case liveAuthNone:
		return nil
	default:
		return fmt.Errorf("model catalog: unsupported live discovery auth scheme %q", s.adapter.authScheme)
	}
	return nil
}

func (s *LiveProviderSource) listCommand(
	ctx context.Context,
	command string,
	env []string,
	timeout time.Duration,
	now time.Time,
) ([]ModelRow, error) {
	bin, args, err := parseDiscoveryCommand(command)
	if err != nil {
		return nil, err
	}
	result, err := s.commandExecutor.RunDiscoveryCommand(ctx, DiscoveryCommandRequest{
		ProviderID: s.providerID,
		Command:    bin,
		Args:       args,
		Env:        env,
		Timeout:    timeout,
	})
	if err != nil {
		detail := firstNonEmptyLine(result.Stderr)
		if detail == "" {
			detail = firstNonEmptyLine(result.Stdout)
		}
		if detail != "" {
			return nil, fmt.Errorf("%w: %s", err, RedactString(detail))
		}
		return nil, err
	}
	if result.ExitCode != 0 {
		detail := firstNonEmptyLine(result.Stderr)
		if detail == "" {
			detail = firstNonEmptyLine(result.Stdout)
		}
		if detail == "" {
			detail = "no diagnostic output"
		}
		return nil, fmt.Errorf(
			"model catalog: discovery command for %q exited %d: %s",
			s.providerID,
			result.ExitCode,
			RedactString(detail),
		)
	}
	rows, err := parseLiveModelPayload(s.providerID, []byte(result.Stdout), now)
	if err == nil {
		return rows, nil
	}
	lineRows := parseLineModelRows(s.providerID, result.Stdout, now)
	if len(lineRows) > 0 {
		return lineRows, nil
	}
	return nil, fmt.Errorf("model catalog: parse discovery command output for %q: %w", s.providerID, err)
}

func parseDiscoveryCommand(command string) (string, []string, error) {
	parts, err := shellquote.Split(command)
	if err != nil {
		return "", nil, fmt.Errorf("model catalog: parse discovery command %q: %w", command, err)
	}
	if len(parts) == 0 {
		return "", nil, fmt.Errorf("model catalog: discovery command is empty")
	}
	return parts[0], parts[1:], nil
}

type livePayloadEnvelope struct {
	Data   []liveRawModel `json:"data"`
	Models []liveRawModel `json:"models"`
}

type liveRawModel struct {
	ID                         string          `json:"id"`
	Name                       string          `json:"name"`
	Model                      string          `json:"model"`
	Value                      string          `json:"value"`
	Label                      string          `json:"label"`
	DisplayName                string          `json:"display_name"`
	DisplayNameCamel           string          `json:"displayName"`
	ContextWindow              *int64          `json:"context_window"`
	ContextLength              *int64          `json:"context_length"`
	MaxTokens                  *int64          `json:"max_tokens"`
	MaxInputTokens             *int64          `json:"max_input_tokens"`
	MaxInputTokensCamel        *int64          `json:"maxInputTokens"`
	MaxOutputTokens            *int64          `json:"max_output_tokens"`
	MaxOutputTokensCamel       *int64          `json:"maxOutputTokens"`
	InputTokenLimit            *int64          `json:"inputTokenLimit"`
	OutputTokenLimit           *int64          `json:"outputTokenLimit"`
	SupportedGenerationMethods []string        `json:"supportedGenerationMethods"`
	SupportedParameters        []string        `json:"supported_parameters"`
	SupportsTools              *bool           `json:"supports_tools"`
	SupportsToolsCamel         *bool           `json:"supportsTools"`
	ToolCall                   *bool           `json:"tool_call"`
	SupportsReasoning          *bool           `json:"supports_reasoning"`
	SupportsReasoningCamel     *bool           `json:"supportsReasoning"`
	SupportsEffort             *bool           `json:"supportsEffort"`
	ReasoningEfforts           []string        `json:"reasoning_efforts"`
	SupportedEffortLevels      []string        `json:"supportedEffortLevels"`
	DefaultReasoningEffort     string          `json:"default_reasoning_effort"`
	Pricing                    liveRawPricing  `json:"pricing"`
	Cost                       liveRawPricing  `json:"cost"`
	Raw                        json.RawMessage `json:"-"`
}

type liveRawPricing struct {
	Input      json.RawMessage `json:"input"`
	Output     json.RawMessage `json:"output"`
	Prompt     json.RawMessage `json:"prompt"`
	Completion json.RawMessage `json:"completion"`
}

func parseLiveModelPayload(providerID string, payload []byte, now time.Time) ([]ModelRow, error) {
	trimmed := bytes.TrimSpace(payload)
	if len(trimmed) == 0 {
		return nil, fmt.Errorf("model catalog: live discovery for %q returned empty payload", providerID)
	}
	rawModels, err := decodeLiveRawModels(trimmed)
	if err != nil {
		return nil, err
	}
	rows := make([]ModelRow, 0, len(rawModels))
	seen := make(map[string]struct{}, len(rawModels))
	for index := range rawModels {
		row, ok := liveModelRow(providerID, &rawModels[index], now)
		if !ok {
			continue
		}
		if _, exists := seen[row.ModelID]; exists {
			continue
		}
		seen[row.ModelID] = struct{}{}
		rows = append(rows, row)
	}
	sortModelRowsByID(rows)
	return rows, nil
}

func decodeLiveRawModels(payload []byte) ([]liveRawModel, error) {
	var array []liveRawModel
	if err := json.Unmarshal(payload, &array); err == nil && len(array) > 0 {
		return array, nil
	}
	var envelope livePayloadEnvelope
	if err := json.Unmarshal(payload, &envelope); err == nil {
		if len(envelope.Data) > 0 {
			return envelope.Data, nil
		}
		if len(envelope.Models) > 0 {
			return envelope.Models, nil
		}
	}
	var objectMap map[string]liveRawModel
	if err := json.Unmarshal(payload, &objectMap); err == nil && len(objectMap) > 0 {
		keys := make([]string, 0, len(objectMap))
		for key := range objectMap {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		models := make([]liveRawModel, 0, len(keys))
		for _, key := range keys {
			model := objectMap[key]
			if strings.TrimSpace(model.ID) == "" {
				model.ID = key
			}
			models = append(models, model)
		}
		return models, nil
	}
	return nil, fmt.Errorf("model catalog: live discovery payload did not contain model rows")
}

func liveModelRow(providerID string, raw *liveRawModel, now time.Time) (ModelRow, bool) {
	if raw == nil {
		return ModelRow{}, false
	}
	modelID := firstNonBlank(raw.ID, raw.Model, raw.Value, raw.Name)
	modelID = strings.TrimPrefix(modelID, "models/")
	if modelID == "" {
		return ModelRow{}, false
	}
	available := true
	row := ModelRow{
		ProviderID:     providerID,
		ModelID:        modelID,
		DisplayName:    firstNonBlank(raw.DisplayName, raw.DisplayNameCamel, raw.Label, raw.Name),
		SourceID:       SourceKindProviderLiveID(providerID),
		SourceKind:     SourceKindProviderLive,
		Priority:       PriorityProviderLive,
		Available:      &available,
		RefreshedAt:    now,
		ContextWindow:  firstInt64(raw.ContextWindow, raw.ContextLength),
		MaxInputTokens: firstInt64(raw.MaxInputTokens, raw.MaxInputTokensCamel, raw.InputTokenLimit),
		MaxOutputTokens: firstInt64(
			raw.MaxOutputTokens,
			raw.MaxOutputTokensCamel,
			raw.MaxTokens,
			raw.OutputTokenLimit,
		),
		SupportsTools:          liveSupportsTools(raw),
		SupportsReasoning:      firstBool(raw.SupportsReasoning, raw.SupportsReasoningCamel, raw.SupportsEffort),
		ReasoningEfforts:       normalizedReasoningEfforts(raw.ReasoningEfforts, raw.SupportedEffortLevels),
		CostInputPerMillion:    livePricePerMillion(raw.Cost.Input, raw.Pricing.Input, raw.Pricing.Prompt),
		CostOutputPerMillion:   livePricePerMillion(raw.Cost.Output, raw.Pricing.Output, raw.Pricing.Completion),
		DefaultReasoningEffort: normalizedDefaultReasoningEffort(raw.DefaultReasoningEffort),
	}
	if row.SupportsReasoning == nil && len(row.ReasoningEfforts) > 0 {
		value := true
		row.SupportsReasoning = &value
	}
	return row, true
}

func parseLineModelRows(providerID string, stdout string, now time.Time) []ModelRow {
	lines := strings.Split(stdout, "\n")
	rows := make([]ModelRow, 0, len(lines))
	seen := make(map[string]struct{}, len(lines))
	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}
		firstToken := strings.Fields(line)[0]
		if strings.EqualFold(firstToken, "id") || strings.EqualFold(firstToken, "model") {
			continue
		}
		if !strings.Contains(firstToken, "/") && providerID == "opencode" {
			continue
		}
		modelID := strings.TrimSpace(firstToken)
		if modelID == "" {
			continue
		}
		if _, exists := seen[modelID]; exists {
			continue
		}
		seen[modelID] = struct{}{}
		available := true
		rows = append(rows, ModelRow{
			ProviderID:  providerID,
			ModelID:     modelID,
			DisplayName: modelID,
			SourceID:    SourceKindProviderLiveID(providerID),
			SourceKind:  SourceKindProviderLive,
			Priority:    PriorityProviderLive,
			Available:   &available,
			RefreshedAt: now,
		})
	}
	sortModelRowsByID(rows)
	return rows
}

func liveSupportsTools(raw *liveRawModel) *bool {
	if raw == nil {
		return nil
	}
	if value := firstBool(raw.SupportsTools, raw.SupportsToolsCamel, raw.ToolCall); value != nil {
		return value
	}
	for _, parameter := range raw.SupportedParameters {
		normalized := strings.ToLower(strings.TrimSpace(parameter))
		if normalized == "tools" || normalized == "tool_choice" {
			value := true
			return &value
		}
	}
	for _, method := range raw.SupportedGenerationMethods {
		if strings.EqualFold(strings.TrimSpace(method), "generateContent") {
			value := true
			return &value
		}
	}
	return nil
}

func normalizedReasoningEfforts(groups ...[]string) []ReasoningEffort {
	efforts := make([]ReasoningEffort, 0)
	seen := make(map[ReasoningEffort]struct{})
	for _, group := range groups {
		for _, raw := range group {
			effort, ok := normalizeReasoningEffort(raw)
			if !ok {
				continue
			}
			if _, exists := seen[effort]; exists {
				continue
			}
			seen[effort] = struct{}{}
			efforts = append(efforts, effort)
		}
	}
	return efforts
}

func normalizedDefaultReasoningEffort(raw string) *ReasoningEffort {
	effort, ok := normalizeReasoningEffort(raw)
	if !ok {
		return nil
	}
	return &effort
}

func normalizeReasoningEffort(raw string) (ReasoningEffort, bool) {
	switch ReasoningEffort(strings.ToLower(strings.TrimSpace(raw))) {
	case ReasoningEffortMinimal:
		return ReasoningEffortMinimal, true
	case ReasoningEffortLow:
		return ReasoningEffortLow, true
	case ReasoningEffortMedium:
		return ReasoningEffortMedium, true
	case ReasoningEffortHigh:
		return ReasoningEffortHigh, true
	case ReasoningEffortXHigh:
		return ReasoningEffortXHigh, true
	default:
		return "", false
	}
}

func livePricePerMillion(values ...json.RawMessage) *float64 {
	for _, raw := range values {
		if len(bytes.TrimSpace(raw)) == 0 {
			continue
		}
		value, ok := parseJSONFloat(raw)
		if !ok {
			continue
		}
		perMillion := value * 1_000_000
		return &perMillion
	}
	return nil
}

func parseJSONFloat(raw json.RawMessage) (float64, bool) {
	var number float64
	if err := json.Unmarshal(raw, &number); err == nil {
		return number, true
	}
	var text string
	if err := json.Unmarshal(raw, &text); err != nil {
		return 0, false
	}
	parsed, err := strconv.ParseFloat(strings.TrimSpace(text), 64)
	if err != nil {
		return 0, false
	}
	return parsed, true
}

func firstEnvValue(env []string, keys ...string) string {
	keySet := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		if trimmed := strings.TrimSpace(key); trimmed != "" {
			keySet[trimmed] = struct{}{}
		}
	}
	for _, entry := range env {
		key, value, ok := strings.Cut(entry, "=")
		if !ok {
			continue
		}
		if _, exists := keySet[key]; exists && strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func firstNonBlank(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func firstNonEmptyLine(text string) string {
	for line := range strings.SplitSeq(text, "\n") {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func sortModelRowsByID(rows []ModelRow) {
	sort.SliceStable(rows, func(i, j int) bool {
		return rows[i].ModelID < rows[j].ModelID
	})
}
