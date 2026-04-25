package daytona

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/environment"
	"github.com/pedronauck/agh/internal/toolruntime"
)

const (
	defaultSDKTimeout    = 60 * time.Second
	defaultCreateTimeout = 5 * time.Minute
)

var _ environment.Provider = (*daytonaProvider)(nil)
var _ environment.Finder = (*daytonaProvider)(nil)

// Option configures the Daytona provider.
type Option func(*daytonaProvider)

type daytonaProvider struct {
	logger            *slog.Logger
	newClient         sandboxClientFactory
	tokenManager      *sshTokenManager
	shellTransport    transport
	launcherTransport transport
	now               func() time.Time
	sdkTimeout        time.Duration
	createTimeout     time.Duration
	sshHost           string
	processRegistry   *toolruntime.Registry
}

// NewProvider returns the Daytona execution environment provider.
func NewProvider(opts ...Option) environment.Provider {
	now := time.Now
	tokenManager := newSSHTokenManager(newRESTSSHTokenSource(now), now)
	provider := &daytonaProvider{
		logger:         slog.Default(),
		newClient:      newSDKClient,
		tokenManager:   tokenManager,
		now:            now,
		sdkTimeout:     defaultSDKTimeout,
		createTimeout:  defaultCreateTimeout,
		sshHost:        defaultSSHHost,
		shellTransport: newSSHTransport(tokenManager),
	}
	for _, opt := range opts {
		if opt != nil {
			opt(provider)
		}
	}
	if provider.logger == nil {
		provider.logger = slog.Default()
	}
	if provider.newClient == nil {
		provider.newClient = newSDKClient
	}
	if provider.now == nil {
		provider.now = time.Now
	}
	if provider.sdkTimeout <= 0 {
		provider.sdkTimeout = defaultSDKTimeout
	}
	if provider.createTimeout <= 0 {
		provider.createTimeout = defaultCreateTimeout
	}
	if provider.tokenManager == nil {
		provider.tokenManager = newSSHTokenManager(newRESTSSHTokenSource(provider.now), provider.now)
	}
	if provider.shellTransport == nil {
		provider.shellTransport = newSSHTransport(provider.tokenManager)
	}
	if provider.launcherTransport == nil {
		provider.launcherTransport = newSidecarTransport(
			provider.logger,
			provider.newClient,
			provider.shellTransport,
		)
	}
	if provider.sshHost == "" {
		provider.sshHost = defaultSSHHost
	}
	return provider
}

func WithLogger(logger *slog.Logger) Option {
	return func(provider *daytonaProvider) {
		provider.logger = logger
	}
}

// WithProcessRegistry injects the shared process registry for environment-owned tool processes.
func WithProcessRegistry(registry *toolruntime.Registry) Option {
	return func(provider *daytonaProvider) {
		provider.processRegistry = registry
	}
}

func withSandboxClientFactory(factory sandboxClientFactory) Option {
	return func(provider *daytonaProvider) {
		provider.newClient = factory
	}
}

func withTransport(transport transport) Option {
	return func(provider *daytonaProvider) {
		provider.shellTransport = transport
		provider.launcherTransport = transport
	}
}

func withTokenManager(manager *sshTokenManager) Option {
	return func(provider *daytonaProvider) {
		provider.tokenManager = manager
	}
}

func withNow(now func() time.Time) Option {
	return func(provider *daytonaProvider) {
		provider.now = now
	}
}

func (p *daytonaProvider) Backend() environment.Backend {
	return environment.BackendDaytona
}

func (p *daytonaProvider) Prepare(
	ctx context.Context,
	req environment.PrepareRequest,
) (environment.Prepared, error) {
	if ctx == nil {
		return environment.Prepared{}, errors.New("environment/daytona: prepare context is required")
	}
	if req.Environment.Backend != environment.BackendDaytona {
		return environment.Prepared{}, fmt.Errorf(
			"environment/daytona: prepare backend = %q, want %q",
			req.Environment.Backend,
			environment.BackendDaytona,
		)
	}
	if req.Environment.Daytona == nil {
		return environment.Prepared{}, errors.New("environment/daytona: Daytona profile is required")
	}
	if err := p.validateNetworkPolicy(req.Environment.Network); err != nil {
		return environment.Prepared{}, err
	}
	daytona := req.Environment.Daytona
	if daytona.StartupSource == "" {
		return environment.Prepared{}, errors.New("environment/daytona: daytona snapshot or image is required")
	}

	existingState, err := decodeProviderState(req.ProviderState)
	if err != nil {
		return environment.Prepared{}, err
	}
	apiURL := normalizeAPIURL(firstNonEmpty(daytona.APIURL, existingState.APIURL))
	client, err := p.newClient(clientConfig{APIURL: apiURL, Target: daytona.Target})
	if err != nil {
		return environment.Prepared{}, err
	}

	sandbox, err := p.prepareSandbox(ctx, client, req, existingState)
	if err != nil {
		return environment.Prepared{}, err
	}

	runtimeRoot := p.runtimeRoot(ctx, sandbox, req.Environment.RuntimeRootDir)
	runtimeAdditional := remoteAdditionalDirs(runtimeRoot, req.LocalAdditionalDirs)
	remoteEnv := remoteEnvMap(req.AgentEnv, req.Environment.Env)
	info := sandboxInfo{
		ID:      sandbox.ID(),
		APIURL:  apiURL,
		SSHHost: p.sshHost,
	}
	access, err := p.tokenManager.Ensure(ctx, apiURL, sandbox.ID(), false)
	if err != nil {
		return environment.Prepared{}, err
	}
	info.SSHAccessExpiresAt = &access.ExpiresAt

	return p.buildPrepared(req, sandbox, info, access, runtimeRoot, runtimeAdditional, remoteEnv)
}

func (p *daytonaProvider) FindEnvironment(
	ctx context.Context,
	req environment.FindEnvironmentRequest,
) (environment.SessionState, error) {
	environmentID, err := validateFindEnvironmentRequest(ctx, req)
	if err != nil {
		return environment.SessionState{}, err
	}

	findConfig, err := newFindEnvironmentConfig(req)
	if err != nil {
		return environment.SessionState{}, err
	}
	client, err := p.newClient(clientConfig{APIURL: findConfig.apiURL, Target: findConfig.target})
	if err != nil {
		return environment.SessionState{}, err
	}

	found, err := p.findByLabels(ctx, client, findEnvironmentLabels(req, environmentID))
	if err != nil {
		if errors.Is(err, errSandboxNotFound) {
			return environment.SessionState{}, fmt.Errorf("%w: %s", environment.ErrEnvironmentNotFound, environmentID)
		}
		return environment.SessionState{}, err
	}
	return p.foundEnvironmentState(ctx, req, findConfig, found, environmentID)
}

type findEnvironmentConfig struct {
	existing      providerState
	apiURL        string
	target        string
	startupSource environment.DaytonaStartupSource
	startupRef    string
}

func validateFindEnvironmentRequest(ctx context.Context, req environment.FindEnvironmentRequest) (string, error) {
	if ctx == nil {
		return "", errors.New("environment/daytona: find context is required")
	}
	if req.Environment.Backend != environment.BackendDaytona {
		return "", fmt.Errorf(
			"environment/daytona: find backend = %q, want %q",
			req.Environment.Backend,
			environment.BackendDaytona,
		)
	}
	environmentID := strings.TrimSpace(req.EnvironmentID)
	if environmentID == "" {
		return "", errors.New("environment/daytona: find environment id is required")
	}
	return environmentID, nil
}

func newFindEnvironmentConfig(req environment.FindEnvironmentRequest) (findEnvironmentConfig, error) {
	existingState, err := decodeProviderState(req.ProviderState)
	if err != nil {
		return findEnvironmentConfig{}, err
	}
	config := findEnvironmentConfig{
		existing:      existingState,
		apiURL:        existingState.APIURL,
		startupSource: existingState.StartupSource,
		startupRef:    existingState.StartupRef,
	}
	daytona := req.Environment.Daytona
	if daytona != nil {
		config.apiURL = firstNonEmpty(daytona.APIURL, config.apiURL)
		config.target = daytona.Target
		if daytona.StartupSource != "" {
			config.startupSource = daytona.StartupSource
		}
		if strings.TrimSpace(daytona.StartupRef) != "" {
			config.startupRef = daytona.StartupRef
		}
	}
	config.apiURL = normalizeAPIURL(config.apiURL)
	return config, nil
}

func findEnvironmentLabels(req environment.FindEnvironmentRequest, environmentID string) map[string]string {
	labels := map[string]string{"agh_environment_id": environmentID}
	if len(req.Labels) > 0 {
		labels = cloneStringMap(req.Labels)
	}
	return labels
}

func (p *daytonaProvider) foundEnvironmentState(
	ctx context.Context,
	req environment.FindEnvironmentRequest,
	config findEnvironmentConfig,
	found sandbox,
	environmentID string,
) (environment.SessionState, error) {
	runtimeRoot := strings.TrimSpace(firstNonEmpty(config.existing.RuntimeRootDir, req.Environment.RuntimeRootDir))
	if runtimeRoot == "" {
		runtimeRoot = p.runtimeRoot(ctx, found, "")
	}
	localRoot := strings.TrimSpace(firstNonEmpty(req.LocalRootDir, config.existing.LocalRootDir))
	localAdditional := cloneStrings(req.LocalAdditionalDirs)
	if len(localAdditional) == 0 {
		localAdditional = cloneStrings(config.existing.LocalAdditionalDirs)
	}
	runtimeAdditional := cloneStrings(config.existing.RuntimeAdditionalDirs)
	if len(runtimeAdditional) == 0 {
		runtimeAdditional = remoteAdditionalDirs(runtimeRoot, localAdditional)
	}

	providerState := providerState{
		Version:               providerStateVersion,
		SandboxID:             found.ID(),
		SandboxName:           found.Name(),
		APIURL:                config.apiURL,
		SSHHost:               p.sshHost,
		LocalRootDir:          localRoot,
		LocalAdditionalDirs:   cloneStrings(localAdditional),
		RuntimeRootDir:        runtimeRoot,
		RuntimeAdditionalDirs: cloneStrings(runtimeAdditional),
		Persistence:           req.Environment.Persistence,
		StartupSource:         config.startupSource,
		StartupRef:            config.startupRef,
		PreparedAt:            p.now().UTC(),
	}
	rawState, err := encodeProviderState(providerState)
	if err != nil {
		return environment.SessionState{}, err
	}

	return environment.SessionState{
		EnvironmentID:         environmentID,
		Backend:               environment.BackendDaytona,
		Profile:               req.Environment.Profile,
		State:                 "found",
		InstanceID:            found.ID(),
		RuntimeRootDir:        runtimeRoot,
		RuntimeAdditionalDirs: cloneStrings(runtimeAdditional),
		ProviderState:         rawState,
		PreparedAt:            providerState.PreparedAt,
	}, nil
}

func (p *daytonaProvider) buildPrepared(
	req environment.PrepareRequest,
	sandbox sandbox,
	info sandboxInfo,
	access sshAccess,
	runtimeRoot string,
	runtimeAdditional []string,
	remoteEnv map[string]string,
) (environment.Prepared, error) {
	daytona := req.Environment.Daytona
	providerState := providerState{
		Version:               providerStateVersion,
		SandboxID:             sandbox.ID(),
		SandboxName:           sandbox.Name(),
		APIURL:                info.APIURL,
		SSHHost:               p.sshHost,
		LocalRootDir:          req.LocalRootDir,
		LocalAdditionalDirs:   cloneStrings(req.LocalAdditionalDirs),
		RuntimeRootDir:        runtimeRoot,
		RuntimeAdditionalDirs: cloneStrings(runtimeAdditional),
		Persistence:           req.Environment.Persistence,
		StartupSource:         daytona.StartupSource,
		StartupRef:            daytona.StartupRef,
		SSHAccessExpiresAt:    &access.ExpiresAt,
		PreparedAt:            p.now().UTC(),
	}
	rawState, err := encodeProviderState(providerState)
	if err != nil {
		return environment.Prepared{}, err
	}

	permission := config.PermissionMode(strings.TrimSpace(req.Permissions))
	toolHost, err := newDaytonaToolHost(sandbox, p.shellTransport, info, runtimeRoot, permission)
	if err != nil {
		return environment.Prepared{}, err
	}
	state := environment.SessionState{
		EnvironmentID:         req.EnvironmentID,
		Backend:               environment.BackendDaytona,
		Profile:               req.Environment.Profile,
		State:                 "ready",
		InstanceID:            sandbox.ID(),
		RuntimeRootDir:        runtimeRoot,
		RuntimeAdditionalDirs: cloneStrings(runtimeAdditional),
		ProviderState:         rawState,
		SSHAccessExpiresAt:    &access.ExpiresAt,
		PreparedAt:            providerState.PreparedAt,
	}
	return environment.Prepared{
		State:                 state,
		RuntimeRootDir:        runtimeRoot,
		RuntimeAdditionalDirs: cloneStrings(runtimeAdditional),
		Launcher:              &daytonaLauncher{transport: p.launcherTransport, sandbox: info},
		Launch: environment.LaunchSpec{
			Command:        req.AgentCommand,
			Cwd:            runtimeRoot,
			AdditionalDirs: cloneStrings(runtimeAdditional),
			Env:            remoteEnvList(remoteEnv),
		},
		ToolHost: toolHost,
	}, nil
}

func (p *daytonaProvider) prepareSandbox(
	ctx context.Context,
	client sandboxClient,
	req environment.PrepareRequest,
	existingState providerState,
) (sandbox, error) {
	sandboxID := strings.TrimSpace(firstNonEmpty(req.InstanceID, existingState.SandboxID))
	if sandboxID != "" {
		return p.getAndStart(ctx, client, sandboxID)
	}

	labels := aghLabels(req)
	if found, err := p.findByLabels(ctx, client, labels); err == nil {
		return p.startSandbox(ctx, found)
	} else if !errors.Is(err, errSandboxNotFound) {
		return nil, err
	}

	return p.createSandbox(ctx, client, req, labels)
}

func (p *daytonaProvider) getAndStart(ctx context.Context, client sandboxClient, sandboxID string) (sandbox, error) {
	var sandbox sandbox
	err := p.withSDKTimeout(ctx, func(ctx context.Context) error {
		var getErr error
		sandbox, getErr = client.Get(ctx, sandboxID)
		return getErr
	})
	if err != nil {
		return nil, fmt.Errorf("environment/daytona: reattach sandbox %q: %w", sandboxID, err)
	}
	return p.startSandbox(ctx, sandbox)
}

func (p *daytonaProvider) findByLabels(
	ctx context.Context,
	client sandboxClient,
	labels map[string]string,
) (sandbox, error) {
	var sandbox sandbox
	err := p.withSDKTimeout(ctx, func(ctx context.Context) error {
		var findErr error
		sandbox, findErr = client.FindOne(ctx, labels)
		return findErr
	})
	if err != nil {
		return nil, err
	}
	return sandbox, nil
}

func (p *daytonaProvider) createSandbox(
	ctx context.Context,
	client sandboxClient,
	req environment.PrepareRequest,
	labels map[string]string,
) (sandbox, error) {
	daytona := req.Environment.Daytona
	createReq := createSandboxRequest{
		Name:               "agh-" + req.EnvironmentID,
		Labels:             labels,
		EnvVars:            remoteEnvMap(req.AgentEnv, req.Environment.Env),
		Public:             req.Environment.Network.AllowPublicIngress,
		AutoStopMinutes:    parseDurationMinutes(daytona.AutoStop),
		AutoArchiveMinutes: parseDurationMinutes(daytona.AutoArchive),
		Timeout:            p.createTimeout,
	}
	switch daytona.StartupSource {
	case environment.DaytonaStartupSourceSnapshot:
		createReq.Snapshot = daytona.StartupRef
	case environment.DaytonaStartupSourceImage:
		createReq.Image = daytona.StartupRef
	default:
		return nil, fmt.Errorf("environment/daytona: unsupported startup source %q", daytona.StartupSource)
	}

	var created sandbox
	err := p.withSDKTimeout(ctx, func(ctx context.Context) error {
		var createErr error
		created, createErr = client.Create(ctx, createReq)
		return createErr
	})
	if err != nil {
		return nil, err
	}
	return created, nil
}

func (p *daytonaProvider) startSandbox(ctx context.Context, sandbox sandbox) (sandbox, error) {
	err := p.withSDKTimeout(ctx, func(ctx context.Context) error {
		return sandbox.Start(ctx)
	})
	if err != nil {
		return nil, err
	}
	return sandbox, nil
}

func (p *daytonaProvider) runtimeRoot(ctx context.Context, sandbox sandbox, configured string) string {
	if strings.TrimSpace(configured) != "" {
		return strings.TrimSpace(configured)
	}
	var workingDir string
	err := p.withSDKTimeout(ctx, func(ctx context.Context) error {
		var workingErr error
		workingDir, workingErr = sandbox.WorkingDir(ctx)
		return workingErr
	})
	if err != nil {
		p.logger.Warn("environment/daytona: get working dir failed; using default runtime root", "error", err)
		return defaultRuntimeRoot
	}
	if strings.TrimSpace(workingDir) == "" {
		return defaultRuntimeRoot
	}
	return strings.TrimSpace(workingDir)
}

func (p *daytonaProvider) Destroy(ctx context.Context, state environment.SessionState) error {
	if ctx == nil {
		return errors.New("environment/daytona: destroy context is required")
	}
	providerState, err := decodeProviderState(state.ProviderState)
	if err != nil {
		return err
	}
	sandboxID := strings.TrimSpace(firstNonEmpty(state.InstanceID, providerState.SandboxID))
	if sandboxID == "" {
		return errors.New("environment/daytona: destroy missing sandbox id")
	}
	client, err := p.newClient(clientConfig{APIURL: providerState.APIURL})
	if err != nil {
		return err
	}
	sandbox, err := p.getSandboxForDestroy(ctx, client, sandboxID)
	if err != nil {
		return err
	}
	switch providerState.Persistence {
	case environment.PersistenceArchive:
		return p.withSDKTimeout(ctx, func(ctx context.Context) error {
			return sandbox.Archive(ctx)
		})
	case environment.PersistenceReuse:
		return nil
	default:
		return p.withSDKTimeout(ctx, func(ctx context.Context) error {
			return sandbox.Delete(ctx)
		})
	}
}

func (p *daytonaProvider) getSandboxForDestroy(
	ctx context.Context,
	client sandboxClient,
	sandboxID string,
) (sandbox, error) {
	var sandbox sandbox
	err := p.withSDKTimeout(ctx, func(ctx context.Context) error {
		var getErr error
		sandbox, getErr = client.Get(ctx, sandboxID)
		return getErr
	})
	if err != nil {
		return nil, fmt.Errorf("environment/daytona: get sandbox %q for destroy: %w", sandboxID, err)
	}
	return sandbox, nil
}

func (p *daytonaProvider) validateNetworkPolicy(policy environment.NetworkPolicy) error {
	unsupported := policy.AllowOutbound || len(policy.AllowList) > 0 || len(policy.DenyList) > 0
	if !unsupported {
		return nil
	}
	message := "environment/daytona: network allow_outbound/allow_list/deny_list " +
		"policies are not enforceable by Daytona alpha provider"
	if policy.Required {
		return errors.New(message)
	}
	p.logger.Warn(message)
	return nil
}

func (p *daytonaProvider) withSDKTimeout(ctx context.Context, fn func(context.Context) error) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, p.sdkTimeout)
	defer cancel()
	if err := fn(timeoutCtx); err != nil {
		return fmt.Errorf("environment/daytona: SDK operation failed: %w", err)
	}
	return nil
}

func aghLabels(req environment.PrepareRequest) map[string]string {
	return map[string]string{
		"agh_session_id":     req.SessionID,
		"agh_environment_id": req.EnvironmentID,
	}
}

func parseDurationMinutes(raw string) *int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	if minutes, err := strconv.Atoi(raw); err == nil {
		return &minutes
	}
	duration, err := time.ParseDuration(raw)
	if err != nil {
		return nil
	}
	minutes := int(duration.Minutes())
	return &minutes
}
