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
	"github.com/pedronauck/agh/internal/sandbox"
	"github.com/pedronauck/agh/internal/toolruntime"
)

const (
	defaultSDKTimeout    = 60 * time.Second
	defaultCreateTimeout = 5 * time.Minute
)

var _ sandbox.Provider = (*daytonaProvider)(nil)
var _ sandbox.Finder = (*daytonaProvider)(nil)

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

// NewProvider returns the Daytona execution sandbox provider.
func NewProvider(opts ...Option) sandbox.Provider {
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

// WithProcessRegistry injects the shared process registry for sandbox-owned tool processes.
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

func (p *daytonaProvider) Backend() sandbox.Backend {
	return sandbox.BackendDaytona
}

func (p *daytonaProvider) Prepare(
	ctx context.Context,
	req sandbox.PrepareRequest,
) (sandbox.Prepared, error) {
	if ctx == nil {
		return sandbox.Prepared{}, errors.New("sandbox/daytona: prepare context is required")
	}
	if req.Sandbox.Backend != sandbox.BackendDaytona {
		return sandbox.Prepared{}, fmt.Errorf(
			"sandbox/daytona: prepare backend = %q, want %q",
			req.Sandbox.Backend,
			sandbox.BackendDaytona,
		)
	}
	if req.Sandbox.Daytona == nil {
		return sandbox.Prepared{}, errors.New("sandbox/daytona: Daytona profile is required")
	}
	if err := p.validateNetworkPolicy(req.Sandbox.Network); err != nil {
		return sandbox.Prepared{}, err
	}
	daytona := req.Sandbox.Daytona
	if daytona.StartupSource == "" {
		return sandbox.Prepared{}, errors.New("sandbox/daytona: daytona snapshot or image is required")
	}

	existingState, err := decodeProviderState(req.ProviderState)
	if err != nil {
		return sandbox.Prepared{}, err
	}
	apiURL := normalizeAPIURL(firstNonEmpty(daytona.APIURL, existingState.APIURL))
	client, err := p.newClient(clientConfig{APIURL: apiURL, Target: daytona.Target})
	if err != nil {
		return sandbox.Prepared{}, err
	}

	instance, err := p.prepareSandbox(ctx, client, req, existingState)
	if err != nil {
		return sandbox.Prepared{}, err
	}

	runtimeRoot := p.runtimeRoot(ctx, instance, req.Sandbox.RuntimeRootDir)
	runtimeAdditional := remoteAdditionalDirs(runtimeRoot, req.LocalAdditionalDirs)
	remoteEnv := remoteEnvMap(req.AgentEnv, req.Sandbox.Env)
	info := sandboxInfo{
		ID:      instance.ID(),
		APIURL:  apiURL,
		SSHHost: p.sshHost,
	}
	access, err := p.tokenManager.Ensure(ctx, apiURL, instance.ID(), false)
	if err != nil {
		return sandbox.Prepared{}, err
	}
	info.SSHAccessExpiresAt = &access.ExpiresAt

	return p.buildPrepared(req, instance, info, access, runtimeRoot, runtimeAdditional, remoteEnv)
}

func (p *daytonaProvider) FindSandbox(
	ctx context.Context,
	req sandbox.FindSandboxRequest,
) (sandbox.SessionState, error) {
	sandboxID, err := validateFindSandboxRequest(ctx, req)
	if err != nil {
		return sandbox.SessionState{}, err
	}

	findConfig, err := newFindSandboxConfig(req)
	if err != nil {
		return sandbox.SessionState{}, err
	}
	client, err := p.newClient(clientConfig{APIURL: findConfig.apiURL, Target: findConfig.target})
	if err != nil {
		return sandbox.SessionState{}, err
	}

	found, err := p.findByLabels(ctx, client, findSandboxLabels(req, sandboxID))
	if err != nil {
		if errors.Is(err, errSandboxNotFound) {
			return sandbox.SessionState{}, fmt.Errorf("%w: %s", sandbox.ErrSandboxNotFound, sandboxID)
		}
		return sandbox.SessionState{}, err
	}
	return p.foundSandboxState(ctx, req, findConfig, found, sandboxID)
}

type findSandboxConfig struct {
	existing      providerState
	apiURL        string
	target        string
	startupSource sandbox.DaytonaStartupSource
	startupRef    string
}

func validateFindSandboxRequest(ctx context.Context, req sandbox.FindSandboxRequest) (string, error) {
	if ctx == nil {
		return "", errors.New("sandbox/daytona: find context is required")
	}
	if req.Sandbox.Backend != sandbox.BackendDaytona {
		return "", fmt.Errorf(
			"sandbox/daytona: find backend = %q, want %q",
			req.Sandbox.Backend,
			sandbox.BackendDaytona,
		)
	}
	sandboxID := strings.TrimSpace(req.SandboxID)
	if sandboxID == "" {
		return "", errors.New("sandbox/daytona: find sandbox id is required")
	}
	return sandboxID, nil
}

func newFindSandboxConfig(req sandbox.FindSandboxRequest) (findSandboxConfig, error) {
	existingState, err := decodeProviderState(req.ProviderState)
	if err != nil {
		return findSandboxConfig{}, err
	}
	config := findSandboxConfig{
		existing:      existingState,
		apiURL:        existingState.APIURL,
		startupSource: existingState.StartupSource,
		startupRef:    existingState.StartupRef,
	}
	daytona := req.Sandbox.Daytona
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

func findSandboxLabels(req sandbox.FindSandboxRequest, sandboxID string) map[string]string {
	labels := map[string]string{"agh_sandbox_id": sandboxID}
	if len(req.Labels) > 0 {
		labels = cloneStringMap(req.Labels)
	}
	return labels
}

func (p *daytonaProvider) foundSandboxState(
	ctx context.Context,
	req sandbox.FindSandboxRequest,
	config findSandboxConfig,
	found daytonaSandbox,
	sandboxID string,
) (sandbox.SessionState, error) {
	runtimeRoot := strings.TrimSpace(firstNonEmpty(config.existing.RuntimeRootDir, req.Sandbox.RuntimeRootDir))
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
		Persistence:           req.Sandbox.Persistence,
		StartupSource:         config.startupSource,
		StartupRef:            config.startupRef,
		PreparedAt:            p.now().UTC(),
	}
	rawState, err := encodeProviderState(providerState)
	if err != nil {
		return sandbox.SessionState{}, err
	}

	return sandbox.SessionState{
		SandboxID:             sandboxID,
		Backend:               sandbox.BackendDaytona,
		Profile:               req.Sandbox.Profile,
		State:                 "found",
		InstanceID:            found.ID(),
		RuntimeRootDir:        runtimeRoot,
		RuntimeAdditionalDirs: cloneStrings(runtimeAdditional),
		ProviderState:         rawState,
		PreparedAt:            providerState.PreparedAt,
	}, nil
}

func (p *daytonaProvider) buildPrepared(
	req sandbox.PrepareRequest,
	instance daytonaSandbox,
	info sandboxInfo,
	access sshAccess,
	runtimeRoot string,
	runtimeAdditional []string,
	remoteEnv map[string]string,
) (sandbox.Prepared, error) {
	daytona := req.Sandbox.Daytona
	providerState := providerState{
		Version:               providerStateVersion,
		SandboxID:             instance.ID(),
		SandboxName:           instance.Name(),
		APIURL:                info.APIURL,
		SSHHost:               p.sshHost,
		LocalRootDir:          req.LocalRootDir,
		LocalAdditionalDirs:   cloneStrings(req.LocalAdditionalDirs),
		RuntimeRootDir:        runtimeRoot,
		RuntimeAdditionalDirs: cloneStrings(runtimeAdditional),
		Persistence:           req.Sandbox.Persistence,
		StartupSource:         daytona.StartupSource,
		StartupRef:            daytona.StartupRef,
		SSHAccessExpiresAt:    &access.ExpiresAt,
		PreparedAt:            p.now().UTC(),
	}
	rawState, err := encodeProviderState(providerState)
	if err != nil {
		return sandbox.Prepared{}, err
	}

	permission := config.PermissionMode(strings.TrimSpace(req.Permissions))
	toolHost, err := newDaytonaToolHost(
		instance,
		p.shellTransport,
		info,
		runtimeRoot,
		permission,
		withDaytonaToolHostProcessRegistry(p.processRegistry),
	)
	if err != nil {
		return sandbox.Prepared{}, err
	}
	state := sandbox.SessionState{
		SandboxID:             req.SandboxID,
		Backend:               sandbox.BackendDaytona,
		Profile:               req.Sandbox.Profile,
		State:                 "ready",
		InstanceID:            instance.ID(),
		RuntimeRootDir:        runtimeRoot,
		RuntimeAdditionalDirs: cloneStrings(runtimeAdditional),
		ProviderState:         rawState,
		SSHAccessExpiresAt:    &access.ExpiresAt,
		PreparedAt:            providerState.PreparedAt,
	}
	return sandbox.Prepared{
		State:                 state,
		RuntimeRootDir:        runtimeRoot,
		RuntimeAdditionalDirs: cloneStrings(runtimeAdditional),
		Launcher:              &daytonaLauncher{transport: p.launcherTransport, sandbox: info},
		Launch: sandbox.LaunchSpec{
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
	req sandbox.PrepareRequest,
	existingState providerState,
) (daytonaSandbox, error) {
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

func (p *daytonaProvider) getAndStart(
	ctx context.Context,
	client sandboxClient,
	sandboxID string,
) (daytonaSandbox, error) {
	var instance daytonaSandbox
	err := p.withSDKTimeout(ctx, func(ctx context.Context) error {
		var getErr error
		instance, getErr = client.Get(ctx, sandboxID)
		return getErr
	})
	if err != nil {
		return nil, fmt.Errorf("sandbox/daytona: reattach sandbox %q: %w", sandboxID, err)
	}
	return p.startSandbox(ctx, instance)
}

func (p *daytonaProvider) findByLabels(
	ctx context.Context,
	client sandboxClient,
	labels map[string]string,
) (daytonaSandbox, error) {
	var instance daytonaSandbox
	err := p.withSDKTimeout(ctx, func(ctx context.Context) error {
		var findErr error
		instance, findErr = client.FindOne(ctx, labels)
		return findErr
	})
	if err != nil {
		return nil, err
	}
	return instance, nil
}

func (p *daytonaProvider) createSandbox(
	ctx context.Context,
	client sandboxClient,
	req sandbox.PrepareRequest,
	labels map[string]string,
) (daytonaSandbox, error) {
	daytona := req.Sandbox.Daytona
	createReq := createSandboxRequest{
		Name:               "agh-" + req.SandboxID,
		Labels:             labels,
		EnvVars:            remoteEnvMap(req.AgentEnv, req.Sandbox.Env),
		Public:             req.Sandbox.Network.AllowPublicIngress,
		AutoStopMinutes:    parseDurationMinutes(daytona.AutoStop),
		AutoArchiveMinutes: parseDurationMinutes(daytona.AutoArchive),
		Timeout:            p.createTimeout,
	}
	switch daytona.StartupSource {
	case sandbox.DaytonaStartupSourceSnapshot:
		createReq.Snapshot = daytona.StartupRef
	case sandbox.DaytonaStartupSourceImage:
		createReq.Image = daytona.StartupRef
	default:
		return nil, fmt.Errorf("sandbox/daytona: unsupported startup source %q", daytona.StartupSource)
	}

	var created daytonaSandbox
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

func (p *daytonaProvider) startSandbox(ctx context.Context, instance daytonaSandbox) (daytonaSandbox, error) {
	err := p.withSDKTimeout(ctx, func(ctx context.Context) error {
		return instance.Start(ctx)
	})
	if err != nil {
		return nil, err
	}
	return instance, nil
}

func (p *daytonaProvider) runtimeRoot(ctx context.Context, instance daytonaSandbox, configured string) string {
	if strings.TrimSpace(configured) != "" {
		return strings.TrimSpace(configured)
	}
	var workingDir string
	err := p.withSDKTimeout(ctx, func(ctx context.Context) error {
		var workingErr error
		workingDir, workingErr = instance.WorkingDir(ctx)
		return workingErr
	})
	if err != nil {
		p.logger.Warn("sandbox/daytona: get working dir failed; using default runtime root", "error", err)
		return defaultRuntimeRoot
	}
	if strings.TrimSpace(workingDir) == "" {
		return defaultRuntimeRoot
	}
	return strings.TrimSpace(workingDir)
}

func (p *daytonaProvider) Destroy(ctx context.Context, state sandbox.SessionState) error {
	if ctx == nil {
		return errors.New("sandbox/daytona: destroy context is required")
	}
	providerState, err := decodeProviderState(state.ProviderState)
	if err != nil {
		return err
	}
	sandboxID := strings.TrimSpace(firstNonEmpty(state.InstanceID, providerState.SandboxID))
	if sandboxID == "" {
		return errors.New("sandbox/daytona: destroy missing sandbox id")
	}
	client, err := p.newClient(clientConfig{APIURL: providerState.APIURL})
	if err != nil {
		return err
	}
	instance, err := p.getSandboxForDestroy(ctx, client, sandboxID)
	if err != nil {
		return err
	}
	switch providerState.Persistence {
	case sandbox.PersistenceArchive:
		return p.withSDKTimeout(ctx, func(ctx context.Context) error {
			return instance.Archive(ctx)
		})
	case sandbox.PersistenceReuse:
		return nil
	default:
		return p.withSDKTimeout(ctx, func(ctx context.Context) error {
			return instance.Delete(ctx)
		})
	}
}

func (p *daytonaProvider) getSandboxForDestroy(
	ctx context.Context,
	client sandboxClient,
	sandboxID string,
) (daytonaSandbox, error) {
	var instance daytonaSandbox
	err := p.withSDKTimeout(ctx, func(ctx context.Context) error {
		var getErr error
		instance, getErr = client.Get(ctx, sandboxID)
		return getErr
	})
	if err != nil {
		return nil, fmt.Errorf("sandbox/daytona: get sandbox %q for destroy: %w", sandboxID, err)
	}
	return instance, nil
}

func (p *daytonaProvider) validateNetworkPolicy(policy sandbox.NetworkPolicy) error {
	unsupported := policy.AllowOutbound || len(policy.AllowList) > 0 || len(policy.DenyList) > 0
	if !unsupported {
		return nil
	}
	message := "sandbox/daytona: network allow_outbound/allow_list/deny_list " +
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
		return fmt.Errorf("sandbox/daytona: SDK operation failed: %w", err)
	}
	return nil
}

func aghLabels(req sandbox.PrepareRequest) map[string]string {
	return map[string]string{
		"agh_session_id": req.SessionID,
		"agh_sandbox_id": req.SandboxID,
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
