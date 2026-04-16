package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	bridgepkg "github.com/pedronauck/agh/internal/bridges"
	"github.com/pedronauck/agh/internal/bridgesdk"
	extensioncontract "github.com/pedronauck/agh/internal/extension/contract"
	"github.com/pedronauck/agh/internal/subprocess"
)

const (
	githubListenAddrEnv = "AGH_BRIDGE_GITHUB_LISTEN_ADDR"
	githubAPIBaseEnv    = "AGH_BRIDGE_GITHUB_API_BASE_URL"

	githubDefaultAPIBaseURL        = "https://api.github.com"
	githubWebhookReadHeaderTimeout = 10 * time.Second
	githubWebhookIdleTimeout       = 2 * time.Minute

	githubModePAT = "pat"
	githubModeApp = "app"

	rpcCodeNotInitialized = -32003
)

var (
	githubReviewThreadPattern = regexp.MustCompile(`^github:([^/]+)/([^:]+):([0-9]+):rc:([0-9]+)$`)
	githubIssueThreadPattern  = regexp.MustCompile(`^github:([^/]+)/([^:]+):issue:([0-9]+)$`)
	githubPRThreadPattern     = regexp.MustCompile(`^github:([^/]+)/([^:]+):([0-9]+)$`)
)

type githubProvider struct {
	sdk     *bridgesdk.Runtime
	stderr  io.Writer
	env     markerEnv
	now     func() time.Time
	session *bridgesdk.Session

	mu                sync.RWMutex
	lastError         string
	server            *http.Server
	serverAddr        string
	listenAddr        string
	routes            map[string]resolvedInstanceConfig
	deliveries        map[string]deliveryState
	reportedStatus    map[string]bridgepkg.BridgeStatus
	installationCache map[string]int64
	apiClients        map[string]githubAPI
	apiFactory        func(resolvedInstanceConfig) githubAPI
	rateLimiter       *bridgesdk.FixedWindowRateLimiter
	inFlightLimiter   *bridgesdk.InFlightLimiter
	initReady         chan struct{}
	initReadyOnce     sync.Once

	stopCh   chan struct{}
	stopOnce sync.Once
	wg       sync.WaitGroup
}

type deliveryState struct {
	LastSeq                int64
	RemoteMessageID        string
	ReplaceRemoteMessageID string
}

type githubProviderConfig struct {
	APIBaseURL     string `json:"api_base_url,omitempty"`
	Mode           string `json:"mode,omitempty"`
	InstallationID int64  `json:"installation_id,omitempty"`
	BotLogin       string `json:"bot_login,omitempty"`
	Webhook        struct {
		ListenAddr string `json:"listen_addr,omitempty"`
		Path       string `json:"path,omitempty"`
	} `json:"webhook"`
	Repository struct {
		Owner    string `json:"owner,omitempty"`
		Name     string `json:"name,omitempty"`
		FullName string `json:"full_name,omitempty"`
	} `json:"repository"`
}

type resolvedInstanceConfig struct {
	managed            subprocess.InitializeBridgeManagedInstance
	instanceID         string
	listenAddr         string
	webhookPath        string
	apiBaseURL         string
	mode               string
	repoOwner          string
	repoName           string
	repoFullName       string
	installationID     int64
	webhookSecret      string
	token              string
	appID              string
	privateKey         string
	botLogin           string
	dedup              *bridgesdk.DedupCache
	configError        error
	initialDegradation *bridgepkg.BridgeDegradation
	initialStatus      bridgepkg.BridgeStatus
}

type githubRepository struct {
	ID       int64      `json:"id,omitempty"`
	Name     string     `json:"name,omitempty"`
	FullName string     `json:"full_name,omitempty"`
	Owner    githubUser `json:"owner"`
}

type githubUser struct {
	ID    int64  `json:"id,omitempty"`
	Login string `json:"login,omitempty"`
	Type  string `json:"type,omitempty"`
}

type githubIssueComment struct {
	ID        int64      `json:"id,omitempty"`
	Body      string     `json:"body,omitempty"`
	CreatedAt string     `json:"created_at,omitempty"`
	UpdatedAt string     `json:"updated_at,omitempty"`
	HTMLURL   string     `json:"html_url,omitempty"`
	User      githubUser `json:"user"`
}

type githubReviewComment struct {
	ID          int64      `json:"id,omitempty"`
	Body        string     `json:"body,omitempty"`
	CreatedAt   string     `json:"created_at,omitempty"`
	UpdatedAt   string     `json:"updated_at,omitempty"`
	HTMLURL     string     `json:"html_url,omitempty"`
	Path        string     `json:"path,omitempty"`
	InReplyToID int64      `json:"in_reply_to_id,omitempty"`
	User        githubUser `json:"user"`
}

type githubIssuePayload struct {
	Action       string              `json:"action,omitempty"`
	Comment      githubIssueComment  `json:"comment"`
	Installation *githubInstallation `json:"installation,omitempty"`
	Issue        struct {
		Number      int64 `json:"number,omitempty"`
		PullRequest *struct {
			URL string `json:"url,omitempty"`
		} `json:"pull_request,omitempty"`
	} `json:"issue"`
	Repository githubRepository `json:"repository"`
	Sender     githubUser       `json:"sender"`
}

type githubReviewPayload struct {
	Action       string              `json:"action,omitempty"`
	Comment      githubReviewComment `json:"comment"`
	Installation *githubInstallation `json:"installation,omitempty"`
	PullRequest  struct {
		Number int64 `json:"number,omitempty"`
	} `json:"pull_request"`
	Repository githubRepository `json:"repository"`
	Sender     githubUser       `json:"sender"`
}

type githubInstallation struct {
	ID int64 `json:"id,omitempty"`
}

type githubMappedInbound struct {
	Envelope       bridgepkg.InboundMessageEnvelope
	InstallationID int64
}

type githubThreadRef struct {
	Owner           string
	Repo            string
	Number          int64
	Type            string
	ReviewCommentID int64
}

type githubRemoteCommentRef struct {
	Kind      string
	CommentID int64
}

func newGitHubProvider(stderr io.Writer) (*githubProvider, error) {
	if stderr == nil {
		stderr = io.Discard
	}

	provider := &githubProvider{
		stderr:            stderr,
		env:               markerEnvFromProcess(),
		now:               func() time.Time { return time.Now().UTC() },
		routes:            make(map[string]resolvedInstanceConfig),
		deliveries:        make(map[string]deliveryState),
		reportedStatus:    make(map[string]bridgepkg.BridgeStatus),
		installationCache: make(map[string]int64),
		apiClients:        make(map[string]githubAPI),
		rateLimiter:       bridgesdk.NewFixedWindowRateLimiter(300, time.Minute),
		inFlightLimiter:   bridgesdk.NewInFlightLimiter(48),
		initReady:         make(chan struct{}),
		stopCh:            make(chan struct{}),
	}
	provider.apiFactory = func(cfg resolvedInstanceConfig) githubAPI {
		provider.mu.Lock()
		defer provider.mu.Unlock()
		if client, ok := provider.apiClients[cfg.instanceID]; ok {
			return client
		}
		client := &githubClient{
			cfg: cfg,
			httpClient: &http.Client{
				Timeout: 10 * time.Second,
			},
			now: func() time.Time { return provider.now() },
		}
		provider.apiClients[cfg.instanceID] = client
		return client
	}

	sdkRuntime, err := bridgesdk.NewRuntime(bridgesdk.RuntimeConfig{
		ExtensionInfo: subprocess.InitializeExtensionInfo{
			Name:    "github",
			Version: "0.1.0",
			SDKName: "bridgesdk",
		},
		Initialize:  provider.handleInitialize,
		Deliver:     provider.handleBridgesDeliver,
		HealthCheck: func(context.Context, *bridgesdk.Session) error { return provider.healthCheck() },
		Shutdown:    provider.handleShutdown,
		Now:         func() time.Time { return provider.now() },
	})
	if err != nil {
		return nil, err
	}
	provider.sdk = sdkRuntime
	return provider, nil
}

func (p *githubProvider) serve(stdin io.Reader, stdout io.Writer) error {
	p.reportSideEffectError(
		"write start marker",
		appendMarkerLine(p.env.startsPath, fmt.Sprintf("pid=%d", os.Getpid())),
	)
	return p.sdk.Serve(context.Background(), stdin, stdout)
}

func (p *githubProvider) handleInitialize(_ context.Context, session *bridgesdk.Session) error {
	p.mu.Lock()
	p.session = session
	p.mu.Unlock()

	marker := initializeMarker{
		Request:  session.InitializeRequest(),
		Response: session.InitializeResponse(),
	}
	p.reportSideEffectError("write initialize marker", writeJSONFile(p.env.handshakePath, marker))
	p.clearLastError()

	p.wg.Go(func() {
		p.afterInitialize(session)
	})

	return nil
}

func (p *githubProvider) afterInitialize(session *bridgesdk.Session) {
	defer p.markInitializationReady()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	listed, err := p.syncOwnedInstances(ctx, session)
	ownershipErr := err
	fetched := make([]bridgepkg.BridgeInstance, 0, len(listed))
	if ownershipErr == nil {
		for _, managed := range listed {
			instance, getErr := p.getOwnedInstance(ctx, session, managed.Instance.ID)
			if getErr != nil {
				ownershipErr = getErr
				break
			}
			fetched = append(fetched, *instance)
		}
	}
	if len(listed) == 0 {
		listed = session.Cache().List()
	}

	ownership := ownershipMarker{
		Listed:  managedInstancesToInstances(listed),
		Fetched: fetched,
	}
	if ownershipErr != nil {
		ownership.Error = ownershipErr.Error()
	}
	p.reportSideEffectError("write ownership marker", writeJSONFile(p.env.ownershipPath, ownership))

	configs := p.reconcileInstanceConfigs(ctx, session, listed)
	for idx := range configs {
		cfg := configs[idx]
		status := cfg.initialStatus
		degradation := cfg.initialDegradation
		if status == "" {
			status = bridgepkg.BridgeStatusReady
		}
		if err := p.reportState(
			ctx,
			session,
			cfg.instanceID,
			status,
			degradation,
		); err != nil &&
			ownershipErr == nil {
			ownershipErr = err
		}
	}

	if ownershipErr != nil {
		p.setLastError(ownershipErr)
	} else {
		p.clearLastError()
	}
}

func (p *githubProvider) handleBridgesDeliver(
	ctx context.Context,
	session *bridgesdk.Session,
	request bridgepkg.DeliveryRequest,
) (bridgepkg.DeliveryAck, error) {
	marker := deliveryMarker{
		PID:     os.Getpid(),
		Request: request,
	}

	cfg, err := p.waitForInstanceConfig(ctx, strings.TrimSpace(request.Event.BridgeInstanceID))
	if err != nil {
		marker.Error = err.Error()
		p.reportSideEffectError("write failed delivery marker", appendJSONLine(p.env.deliveryPath, marker))
		p.setLastError(err)
		return bridgepkg.DeliveryAck{}, err
	}
	if shouldCrashOnce(p.env.crashOncePath) {
		p.reportSideEffectError("write pre-crash delivery marker", appendJSONLine(p.env.deliveryPath, marker))
		p.reportSideEffectError("write crash marker", writeJSONFile(p.env.crashOncePath, map[string]any{
			"crashed":            true,
			"pid":                os.Getpid(),
			"delivery_id":        strings.TrimSpace(request.Event.DeliveryID),
			"bridge_instance_id": cfg.instanceID,
		}))
		os.Exit(23)
	}

	installationID, err := p.resolveDeliveryInstallationID(&cfg, request)
	if err != nil {
		marker.Error = err.Error()
		p.reportSideEffectError("write failed delivery marker", appendJSONLine(p.env.deliveryPath, marker))
		p.setLastError(err)
		return bridgepkg.DeliveryAck{}, err
	}

	api := p.apiFactory(cfg)
	ack, state, err := executeGitHubDelivery(
		ctx,
		api,
		&cfg,
		request,
		p.deliveryState(cfg.instanceID, request.Event.DeliveryID),
		installationID,
	)
	if err != nil {
		marker.Error = err.Error()
		p.reportSideEffectError("write failed delivery marker", appendJSONLine(p.env.deliveryPath, marker))
		classified := bridgesdk.ClassifyError(err)
		_, _, reportErr := session.ReportClassifiedError(ctx, cfg.instanceID, classified)
		if reportErr != nil {
			p.setLastError(reportErr)
		} else {
			p.setLastError(err)
		}
		return bridgepkg.DeliveryAck{}, err
	}

	p.storeDeliveryState(cfg.instanceID, request.Event.DeliveryID, request.Event, state)
	p.reportReadyIfNeeded(ctx, session, cfg.instanceID)

	marker.Ack = &ack
	p.reportSideEffectError("write delivery marker", appendJSONLine(p.env.deliveryPath, marker))
	p.clearLastError()
	return ack, nil
}

func (p *githubProvider) healthCheck() error {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if strings.TrimSpace(p.lastError) == "" {
		return nil
	}
	return errors.New(strings.TrimSpace(p.lastError))
}

func (p *githubProvider) handleShutdown(
	_ context.Context,
	_ *bridgesdk.Session,
	request subprocess.ShutdownRequest,
) error {
	p.stop()

	shutdownCtx := context.Background()
	if request.DeadlineMS > 0 {
		var cancel context.CancelFunc
		shutdownCtx, cancel = context.WithTimeout(
			context.Background(),
			time.Duration(request.DeadlineMS)*time.Millisecond,
		)
		defer cancel()
	}

	p.mu.Lock()
	server := p.server
	p.mu.Unlock()
	if server != nil {
		if err := server.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
			p.reportSideEffectError("shutdown github webhook server", err)
			p.setLastError(err)
		}
	}

	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-shutdownCtx.Done():
	}

	p.reportSideEffectError(
		"write shutdown marker",
		appendMarkerLine(p.env.shutdownPath, fmt.Sprintf("pid=%d", os.Getpid())),
	)
	return nil
}

func (p *githubProvider) stop() {
	p.stopOnce.Do(func() {
		close(p.stopCh)
	})
}

func (p *githubProvider) syncOwnedInstances(
	ctx context.Context,
	session *bridgesdk.Session,
) ([]subprocess.InitializeBridgeManagedInstance, error) {
	var result []subprocess.InitializeBridgeManagedInstance
	err := p.retryHostCall(ctx, func(callCtx context.Context) error {
		items, callErr := session.SyncInstances(callCtx)
		if callErr == nil {
			result = items
		}
		return callErr
	})
	return result, err
}

func (p *githubProvider) getOwnedInstance(
	ctx context.Context,
	session *bridgesdk.Session,
	bridgeInstanceID string,
) (*bridgepkg.BridgeInstance, error) {
	var result *bridgepkg.BridgeInstance
	err := p.retryHostCall(ctx, func(callCtx context.Context) error {
		instance, callErr := session.HostAPI().GetBridgeInstance(callCtx, bridgeInstanceID)
		if callErr == nil {
			result = instance
		}
		return callErr
	})
	return result, err
}

func (p *githubProvider) reportState(
	ctx context.Context,
	session *bridgesdk.Session,
	bridgeInstanceID string,
	status bridgepkg.BridgeStatus,
	degradation *bridgepkg.BridgeDegradation,
) error {
	var result *bridgepkg.BridgeInstance
	err := p.retryHostCall(ctx, func(callCtx context.Context) error {
		instance, callErr := session.HostAPI().
			ReportBridgeInstanceState(callCtx, extensioncontract.BridgesInstancesReportStateParams{
				BridgeInstanceID: strings.TrimSpace(bridgeInstanceID),
				Status:           status,
				Degradation:      cloneDegradation(degradation),
			})
		if callErr == nil {
			result = instance
		}
		return callErr
	})
	if err != nil {
		p.reportSideEffectError("write failed state marker", appendJSONLine(p.env.statePath, stateMarker{
			BridgeInstanceID: strings.TrimSpace(bridgeInstanceID),
			Status:           status,
			Error:            err.Error(),
		}))
		return err
	}

	p.mu.Lock()
	p.reportedStatus[strings.TrimSpace(bridgeInstanceID)] = result.Status.Normalize()
	p.mu.Unlock()
	p.reportSideEffectError("write state marker", appendJSONLine(p.env.statePath, stateMarker{
		BridgeInstanceID: result.ID,
		Status:           result.Status,
		Instance:         *result,
	}))
	return nil
}

func (p *githubProvider) reportReadyIfNeeded(ctx context.Context, session *bridgesdk.Session, bridgeInstanceID string) {
	p.mu.RLock()
	status := p.reportedStatus[strings.TrimSpace(bridgeInstanceID)]
	p.mu.RUnlock()
	if status == bridgepkg.BridgeStatusReady {
		return
	}
	if err := p.reportState(ctx, session, bridgeInstanceID, bridgepkg.BridgeStatusReady, nil); err != nil {
		p.setLastError(err)
	}
}

func (p *githubProvider) ingestBridgeMessage(
	ctx context.Context,
	session *bridgesdk.Session,
	envelope bridgepkg.InboundMessageEnvelope,
) (*extensioncontract.BridgesMessagesIngestResult, error) {
	var result *extensioncontract.BridgesMessagesIngestResult
	err := p.retryHostCall(ctx, func(callCtx context.Context) error {
		ingestResult, callErr := session.HostAPI().IngestBridgeMessage(callCtx, envelope)
		if callErr == nil {
			result = ingestResult
		}
		return callErr
	})
	return result, err
}

func (p *githubProvider) retryHostCall(ctx context.Context, fn func(context.Context) error) error {
	if ctx == nil {
		ctx = context.Background()
	}

	delay := 10 * time.Millisecond
	var lastErr error
	for range 6 {
		err := fn(ctx)
		if err == nil {
			return nil
		}
		if !isNotInitializedRPCError(err) {
			return err
		}
		lastErr = err

		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return ctx.Err()
		case <-p.stopCh:
			if !timer.Stop() {
				<-timer.C
			}
			return err
		case <-timer.C:
		}

		if delay < 100*time.Millisecond {
			delay *= 2
			if delay > 100*time.Millisecond {
				delay = 100 * time.Millisecond
			}
		}
	}

	if lastErr != nil {
		return lastErr
	}
	return nil
}

func (p *githubProvider) reconcileInstanceConfigs(
	ctx context.Context,
	session *bridgesdk.Session,
	managed []subprocess.InitializeBridgeManagedInstance,
) []resolvedInstanceConfig {
	if len(managed) == 0 {
		p.mu.Lock()
		p.routes = make(map[string]resolvedInstanceConfig)
		p.installationCache = make(map[string]int64)
		p.apiClients = make(map[string]githubAPI)
		p.mu.Unlock()
		return nil
	}

	configs, requestedListen := p.collectGitHubConfigs(session, managed)
	p.applyGitHubListenErrors(configs, requestedListen)
	nextRoutes := buildGitHubRouteMap(configs)
	p.storeGitHubRoutes(nextRoutes, requestedListen)
	p.markInitializationReady()
	p.populateGitHubInitialState(ctx, configs, nextRoutes)
	p.storeGitHubFinalRoutes(nextRoutes)

	return configs
}

func (p *githubProvider) collectGitHubConfigs(
	session *bridgesdk.Session,
	managed []subprocess.InitializeBridgeManagedInstance,
) ([]resolvedInstanceConfig, string) {
	configs := make([]resolvedInstanceConfig, 0, len(managed))
	requestedListen := strings.TrimSpace(os.Getenv(githubListenAddrEnv))
	seenRepos := make(map[string]string, len(managed))

	for _, item := range managed {
		cfg := p.resolveInstanceConfig(session, item)
		requestedListen = applyGitHubListenConstraint(&cfg, requestedListen)
		applyGitHubRepoConflict(&cfg, seenRepos)
		configs = append(configs, cfg)
	}

	return configs, requestedListen
}

func applyGitHubListenConstraint(cfg *resolvedInstanceConfig, requestedListen string) string {
	if cfg == nil || cfg.listenAddr == "" {
		return requestedListen
	}
	if requestedListen == "" {
		return cfg.listenAddr
	}
	if requestedListen != cfg.listenAddr && cfg.configError == nil {
		cfg.configError = fmt.Errorf(
			"github: instance %q configured incompatible listen_addr %q (runtime uses %q)",
			cfg.instanceID,
			cfg.listenAddr,
			requestedListen,
		)
	}
	return requestedListen
}

func applyGitHubRepoConflict(cfg *resolvedInstanceConfig, seenRepos map[string]string) {
	if cfg == nil || cfg.repoFullName == "" {
		return
	}
	if owner, ok := seenRepos[cfg.repoFullName]; ok && cfg.configError == nil {
		cfg.configError = fmt.Errorf(
			"github: repository %q is already owned by %q and cannot also belong to %q",
			cfg.repoFullName,
			owner,
			cfg.instanceID,
		)
	}
	seenRepos[cfg.repoFullName] = cfg.instanceID
}

func (p *githubProvider) applyGitHubListenErrors(configs []resolvedInstanceConfig, requestedListen string) {
	if requestedListen == "" {
		for idx := range configs {
			if configs[idx].configError == nil {
				configs[idx].configError = errors.New("github: webhook listen address is required")
			}
		}
		return
	}
	if err := p.startServer(requestedListen); err != nil {
		for idx := range configs {
			if configs[idx].configError == nil {
				configs[idx].configError = err
			}
		}
	}
}

func buildGitHubRouteMap(configs []resolvedInstanceConfig) map[string]resolvedInstanceConfig {
	nextRoutes := make(map[string]resolvedInstanceConfig, len(configs))
	for idx := range configs {
		cfg := configs[idx]
		nextRoutes[cfg.instanceID] = cfg
	}
	return nextRoutes
}

func (p *githubProvider) storeGitHubRoutes(nextRoutes map[string]resolvedInstanceConfig, requestedListen string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.routes = nextRoutes
	p.listenAddr = requestedListen
	p.apiClients = make(map[string]githubAPI, len(nextRoutes))
}

func (p *githubProvider) populateGitHubInitialState(
	ctx context.Context,
	configs []resolvedInstanceConfig,
	nextRoutes map[string]resolvedInstanceConfig,
) {
	for idx := range configs {
		updated, status, degradation, err := p.determineInitialState(ctx, &configs[idx])
		if err != nil {
			p.setLastError(err)
		}
		updated.initialStatus = status
		updated.initialDegradation = degradation
		nextRoutes[updated.instanceID] = *updated
	}
}

func (p *githubProvider) storeGitHubFinalRoutes(nextRoutes map[string]resolvedInstanceConfig) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.routes = nextRoutes
}

func (p *githubProvider) resolveInstanceConfig(
	session *bridgesdk.Session,
	managed subprocess.InitializeBridgeManagedInstance,
) resolvedInstanceConfig {
	cfg := githubProviderConfig{}
	if len(managed.Instance.ProviderConfig) > 0 {
		if err := json.Unmarshal(managed.Instance.ProviderConfig, &cfg); err != nil {
			return resolvedInstanceConfig{
				managed:     managed,
				instanceID:  managed.Instance.ID,
				configError: fmt.Errorf("github: decode provider_config for %q: %w", managed.Instance.ID, err),
			}
		}
	}

	webhookSecret, _ := session.Cache().BoundSecretValue(managed.Instance.ID, "webhook_secret")
	token, _ := session.Cache().BoundSecretValue(managed.Instance.ID, "token")
	appID, _ := session.Cache().BoundSecretValue(managed.Instance.ID, "app_id")
	privateKey, _ := session.Cache().BoundSecretValue(managed.Instance.ID, "private_key")

	listenAddr := firstNonEmpty(cfg.Webhook.ListenAddr, strings.TrimSpace(os.Getenv(githubListenAddrEnv)))
	webhookPath := normalizeWebhookPath(firstNonEmpty(cfg.Webhook.Path, "/github"))
	apiBaseURL := normalizeURL(
		firstNonEmpty(cfg.APIBaseURL, strings.TrimSpace(os.Getenv(githubAPIBaseEnv)), githubDefaultAPIBaseURL),
	)
	mode := strings.ToLower(strings.TrimSpace(cfg.Mode))
	if mode == "" {
		switch {
		case strings.TrimSpace(token) != "" && strings.TrimSpace(appID) == "" && strings.TrimSpace(privateKey) == "":
			mode = githubModePAT
		case strings.TrimSpace(token) == "" && strings.TrimSpace(appID) != "" && strings.TrimSpace(privateKey) != "":
			mode = githubModeApp
		}
	}

	repoOwner, repoName, repoFullName, repoErr := normalizeGitHubRepository(
		cfg.Repository.Owner,
		cfg.Repository.Name,
		cfg.Repository.FullName,
	)
	resolved := resolvedInstanceConfig{
		managed:        managed,
		instanceID:     strings.TrimSpace(managed.Instance.ID),
		listenAddr:     listenAddr,
		webhookPath:    webhookPath,
		apiBaseURL:     apiBaseURL,
		mode:           mode,
		repoOwner:      repoOwner,
		repoName:       repoName,
		repoFullName:   repoFullName,
		installationID: cfg.InstallationID,
		webhookSecret:  strings.TrimSpace(webhookSecret),
		token:          strings.TrimSpace(token),
		appID:          strings.TrimSpace(appID),
		privateKey:     strings.TrimSpace(privateKey),
		botLogin:       strings.TrimSpace(cfg.BotLogin),
		dedup:          bridgesdk.NewDedupCache(5*time.Minute, 4000),
	}
	switch {
	case repoErr != nil:
		resolved.configError = repoErr
	case resolved.webhookPath == "":
		resolved.configError = errors.New("github: webhook path is required")
	case resolved.mode != githubModePAT && resolved.mode != githubModeApp:
		resolved.configError = fmt.Errorf("github: provider mode must be %q or %q", githubModePAT, githubModeApp)
	case resolved.installationID < 0:
		resolved.configError = errors.New("github: installation_id must be non-negative")
	}

	return resolved
}

func (p *githubProvider) determineInitialState(
	ctx context.Context,
	cfg *resolvedInstanceConfig,
) (*resolvedInstanceConfig, bridgepkg.BridgeStatus, *bridgepkg.BridgeDegradation, error) {
	if cfg == nil {
		err := errors.New("github: config is required")
		return nil, bridgepkg.BridgeStatusError, nil, err
	}
	if cfg.configError != nil {
		return cfg, bridgepkg.BridgeStatusDegraded, &bridgepkg.BridgeDegradation{
			Reason:  bridgepkg.BridgeDegradationReasonTenantConfigInvalid,
			Message: cfg.configError.Error(),
		}, cfg.configError
	}
	if strings.TrimSpace(cfg.webhookSecret) == "" {
		err := errors.New("github: webhook_secret secret binding is required")
		return cfg, bridgepkg.BridgeStatusAuthRequired, &bridgepkg.BridgeDegradation{
			Reason:  bridgepkg.BridgeDegradationReasonAuthFailed,
			Message: err.Error(),
		}, err
	}

	switch cfg.mode {
	case githubModePAT:
		if strings.TrimSpace(cfg.token) == "" {
			err := errors.New("github: token secret binding is required for PAT mode")
			return cfg, bridgepkg.BridgeStatusAuthRequired, &bridgepkg.BridgeDegradation{
				Reason:  bridgepkg.BridgeDegradationReasonAuthFailed,
				Message: err.Error(),
			}, err
		}
	case githubModeApp:
		if strings.TrimSpace(cfg.appID) == "" || strings.TrimSpace(cfg.privateKey) == "" {
			err := errors.New("github: app_id and private_key secret bindings are required for app mode")
			return cfg, bridgepkg.BridgeStatusAuthRequired, &bridgepkg.BridgeDegradation{
				Reason:  bridgepkg.BridgeDegradationReasonAuthFailed,
				Message: err.Error(),
			}, err
		}
		if err := validateGitHubAppCredentials(cfg); err != nil {
			return cfg, bridgepkg.BridgeStatusAuthRequired, &bridgepkg.BridgeDegradation{
				Reason:  bridgepkg.BridgeDegradationReasonAuthFailed,
				Message: err.Error(),
			}, err
		}
	}

	if cfg.mode == githubModeApp && cfg.installationID == 0 {
		return cfg, bridgepkg.BridgeStatusReady, nil, nil
	}

	viewer, err := p.apiFactory(*cfg).ValidateAuth(ctx, cfg.installationID)
	if err != nil {
		classified := bridgesdk.ClassifyError(err)
		recovery := classified.Recovery()
		status := recovery.Status
		if status == "" {
			status = bridgepkg.BridgeStatusError
		}
		if recovery.Degradation != nil {
			return cfg, status, recovery.Degradation, err
		}
		return cfg, status, &bridgepkg.BridgeDegradation{
			Reason:  bridgepkg.BridgeDegradationReasonProviderTimeout,
			Message: classified.Message,
		}, err
	}
	if strings.TrimSpace(cfg.botLogin) == "" && viewer != nil {
		cfg.botLogin = strings.TrimSpace(viewer.Login)
	}
	return cfg, bridgepkg.BridgeStatusReady, nil, nil
}

func (p *githubProvider) startServer(listenAddr string) error {
	p.mu.RLock()
	server := p.server
	currentListen := p.listenAddr
	p.mu.RUnlock()
	if server != nil {
		if currentListen != "" && currentListen != strings.TrimSpace(listenAddr) {
			return fmt.Errorf("github: runtime already listening on %q, cannot switch to %q", currentListen, listenAddr)
		}
		return nil
	}

	ln, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", strings.TrimSpace(listenAddr))
	if err != nil {
		return fmt.Errorf("github: listen %q: %w", listenAddr, err)
	}

	httpServer := &http.Server{
		Handler:           http.HandlerFunc(p.serveWebhookHTTP),
		ReadHeaderTimeout: githubWebhookReadHeaderTimeout,
		IdleTimeout:       githubWebhookIdleTimeout,
	}

	actualAddr := ln.Addr().String()
	p.mu.Lock()
	p.server = httpServer
	p.serverAddr = actualAddr
	p.listenAddr = strings.TrimSpace(listenAddr)
	p.mu.Unlock()

	p.reportSideEffectError(
		"write start marker",
		appendMarkerLine(p.env.startsPath, fmt.Sprintf("listen=%s", actualAddr)),
	)

	p.wg.Go(func() {
		if serveErr := httpServer.Serve(ln); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			p.setLastError(serveErr)
		}
	})
	return nil
}

func (p *githubProvider) serveWebhookHTTP(w http.ResponseWriter, r *http.Request) {
	candidates := p.configsForPath(r.URL.Path)
	if len(candidates) == 0 {
		http.NotFound(w, r)
		return
	}

	handler, err := bridgesdk.NewWebhookHandler(bridgesdk.WebhookGuardConfig{
		AllowedMethods:      []string{http.MethodPost},
		AllowedContentTypes: []string{"application/json"},
		MaxBodyBytes:        1 << 20,
		RateLimiter:         p.rateLimiter,
		InFlightLimiter:     p.inFlightLimiter,
		VerifySignature: func(ctx context.Context, req *http.Request, body []byte) error {
			return verifyGitHubWebhookSignature(ctx, req, body, candidates)
		},
		RequestKey: func(req *http.Request) string {
			return req.RemoteAddr + "|" + normalizeWebhookPath(req.URL.Path)
		},
		Now: func() time.Time { return p.now() },
	}, func(w http.ResponseWriter, r *http.Request, request bridgesdk.WebhookRequest) error {
		return p.handleWebhookRequest(w, r, candidates, request)
	})
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		p.setLastError(err)
		return
	}
	handler.ServeHTTP(w, r)
}

func (p *githubProvider) handleWebhookRequest(
	w http.ResponseWriter,
	r *http.Request,
	candidates []resolvedInstanceConfig,
	request bridgesdk.WebhookRequest,
) error {
	eventType := strings.ToLower(strings.TrimSpace(r.Header.Get("X-GitHub-Event")))
	switch eventType {
	case "ping":
		return writeWebhookText(w, "pong")
	case "issue_comment":
		return p.handleIssueCommentWebhook(w, r, candidates, request)
	case "pull_request_review_comment":
		return p.handleReviewCommentWebhook(w, r, candidates, request)
	default:
		return writeWebhookText(w, "ok")
	}
}

func (p *githubProvider) handleIssueCommentWebhook(
	w http.ResponseWriter,
	r *http.Request,
	candidates []resolvedInstanceConfig,
	request bridgesdk.WebhookRequest,
) error {
	payload := githubIssuePayload{}
	if err := json.Unmarshal(request.Body, &payload); err != nil {
		return &bridgesdk.HTTPError{StatusCode: http.StatusBadRequest, Message: "invalid github webhook payload"}
	}
	cfg, ok, err := selectGitHubIssueConfig(candidates, payload)
	if err != nil {
		return &bridgesdk.HTTPError{StatusCode: http.StatusBadRequest, Message: err.Error()}
	}
	if !ok {
		return writeWebhookText(w, "ignored")
	}
	if err := verifyGitHubWebhookSignature(r.Context(), r, request.Body, []resolvedInstanceConfig{cfg}); err != nil {
		return &bridgesdk.HTTPError{StatusCode: http.StatusUnauthorized, Message: "invalid github webhook signature"}
	}
	if strings.TrimSpace(payload.Action) != "created" {
		return writeWebhookText(w, "ok")
	}
	item, err := mapGitHubIssueComment(payload, cfg.managed, request.ReceivedAt)
	if err != nil {
		return &bridgesdk.HTTPError{StatusCode: http.StatusBadRequest, Message: err.Error()}
	}
	if item.InstallationID > 0 {
		p.storeInstallationID(cfg.repoFullName, item.InstallationID)
	}
	if cfg.dedup.Mark(item.Envelope.IdempotencyKey) || isGitHubSelfMessage(&cfg, payload.Sender) {
		return writeWebhookText(w, "ok")
	}
	if err := p.dispatchInboundEnvelope(context.Background(), cfg.instanceID, item.Envelope); err != nil {
		return &bridgesdk.HTTPError{StatusCode: http.StatusInternalServerError, Message: err.Error()}
	}
	return writeWebhookText(w, "ok")
}

func (p *githubProvider) handleReviewCommentWebhook(
	w http.ResponseWriter,
	r *http.Request,
	candidates []resolvedInstanceConfig,
	request bridgesdk.WebhookRequest,
) error {
	payload := githubReviewPayload{}
	if err := json.Unmarshal(request.Body, &payload); err != nil {
		return &bridgesdk.HTTPError{StatusCode: http.StatusBadRequest, Message: "invalid github webhook payload"}
	}
	cfg, ok, err := selectGitHubReviewConfig(candidates, payload)
	if err != nil {
		return &bridgesdk.HTTPError{StatusCode: http.StatusBadRequest, Message: err.Error()}
	}
	if !ok {
		return writeWebhookText(w, "ignored")
	}
	if err := verifyGitHubWebhookSignature(r.Context(), r, request.Body, []resolvedInstanceConfig{cfg}); err != nil {
		return &bridgesdk.HTTPError{StatusCode: http.StatusUnauthorized, Message: "invalid github webhook signature"}
	}
	if strings.TrimSpace(payload.Action) != "created" {
		return writeWebhookText(w, "ok")
	}
	item, err := mapGitHubReviewComment(payload, cfg.managed, request.ReceivedAt)
	if err != nil {
		return &bridgesdk.HTTPError{StatusCode: http.StatusBadRequest, Message: err.Error()}
	}
	if item.InstallationID > 0 {
		p.storeInstallationID(cfg.repoFullName, item.InstallationID)
	}
	if cfg.dedup.Mark(item.Envelope.IdempotencyKey) || isGitHubSelfMessage(&cfg, payload.Sender) {
		return writeWebhookText(w, "ok")
	}
	if err := p.dispatchInboundEnvelope(context.Background(), cfg.instanceID, item.Envelope); err != nil {
		return &bridgesdk.HTTPError{StatusCode: http.StatusInternalServerError, Message: err.Error()}
	}
	return writeWebhookText(w, "ok")
}

func (p *githubProvider) dispatchInboundEnvelope(
	ctx context.Context,
	bridgeInstanceID string,
	envelope bridgepkg.InboundMessageEnvelope,
) error {
	session := p.currentSession()
	if session == nil {
		return errors.New("github: runtime session is not initialized")
	}
	cfg, err := p.configForInstance(bridgeInstanceID)
	if err != nil {
		return err
	}
	result, err := p.ingestBridgeMessage(ctx, session, envelope)
	if err != nil {
		p.reportSideEffectError("write failed ingest marker", appendJSONLine(p.env.ingestPath, ingestMarker{
			Envelope: envelope,
			Error:    err.Error(),
		}))
		return err
	}
	p.reportSideEffectError("write ingest marker", appendJSONLine(p.env.ingestPath, ingestMarker{
		Envelope: envelope,
		Result:   *result,
	}))
	p.reportReadyIfNeeded(ctx, session, cfg.instanceID)
	return nil
}

func (p *githubProvider) configForInstance(instanceID string) (resolvedInstanceConfig, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	cfg, ok := p.routes[strings.TrimSpace(instanceID)]
	if !ok {
		return resolvedInstanceConfig{}, fmt.Errorf("github: unmanaged bridge instance %q", instanceID)
	}
	return cfg, nil
}

func (p *githubProvider) waitForInstanceConfig(ctx context.Context, instanceID string) (resolvedInstanceConfig, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	initReady := p.initializationReady()
	for {
		cfg, err := p.configForInstance(instanceID)
		if err == nil {
			return cfg, nil
		}
		if initReady != nil {
			select {
			case <-initReady:
				initReady = nil
				continue
			default:
			}
		}
		if initReady == nil {
			return resolvedInstanceConfig{}, err
		}

		timer := time.NewTimer(10 * time.Millisecond)
		select {
		case <-p.stopCh:
			if !timer.Stop() {
				<-timer.C
			}
			return resolvedInstanceConfig{}, err
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return resolvedInstanceConfig{}, ctx.Err()
		case <-initReady:
			if !timer.Stop() {
				<-timer.C
			}
			initReady = nil
		case <-timer.C:
		}
	}
}

func (p *githubProvider) configsForPath(path string) []resolvedInstanceConfig {
	normalizedPath := normalizeWebhookPath(path)
	p.mu.RLock()
	defer p.mu.RUnlock()

	configs := make([]resolvedInstanceConfig, 0, len(p.routes))
	for instanceID := range p.routes {
		cfg := p.routes[instanceID]
		if cfg.webhookPath == normalizedPath {
			configs = append(configs, cfg)
		}
	}
	return configs
}

func (p *githubProvider) currentSession() *bridgesdk.Session {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.session
}

func (p *githubProvider) deliveryState(instanceID string, deliveryID string) deliveryState {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.deliveries[deliveryStateKey(instanceID, deliveryID)]
}

func (p *githubProvider) storeDeliveryState(
	instanceID string,
	deliveryID string,
	event bridgepkg.DeliveryEvent,
	state deliveryState,
) {
	key := deliveryStateKey(instanceID, deliveryID)
	p.mu.Lock()
	defer p.mu.Unlock()
	if isTerminalGitHubDeliveryEvent(event) {
		delete(p.deliveries, key)
		return
	}
	p.deliveries[key] = state
}

func (p *githubProvider) initializationReady() <-chan struct{} {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.initReady
}

func (p *githubProvider) markInitializationReady() {
	p.initReadyOnce.Do(func() {
		p.mu.Lock()
		ch := p.initReady
		p.mu.Unlock()
		if ch != nil {
			close(ch)
		}
	})
}

func (p *githubProvider) storeInstallationID(repoFullName string, installationID int64) {
	if installationID <= 0 || strings.TrimSpace(repoFullName) == "" {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.installationCache[strings.ToLower(strings.TrimSpace(repoFullName))] = installationID
}

func (p *githubProvider) cachedInstallationID(repoFullName string) int64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.installationCache[strings.ToLower(strings.TrimSpace(repoFullName))]
}

func (p *githubProvider) resolveDeliveryInstallationID(
	cfg *resolvedInstanceConfig,
	request bridgepkg.DeliveryRequest,
) (int64, error) {
	if cfg == nil {
		return 0, errors.New("github: config is required")
	}
	if cfg.mode != githubModeApp {
		return 0, nil
	}
	if cfg.installationID > 0 {
		return cfg.installationID, nil
	}
	if installationID := installationIDFromMetadata(request.Event.ProviderMetadata); installationID > 0 {
		p.storeInstallationID(cfg.repoFullName, installationID)
		return installationID, nil
	}
	if request.Snapshot != nil {
		if installationID := installationIDFromMetadata(request.Snapshot.ProviderMetadata); installationID > 0 {
			p.storeInstallationID(cfg.repoFullName, installationID)
			return installationID, nil
		}
	}
	if installationID := p.cachedInstallationID(cfg.repoFullName); installationID > 0 {
		return installationID, nil
	}
	return 0, &bridgesdk.PermanentError{
		Err: fmt.Errorf("github: installation id is required for app-mode delivery on %q", cfg.repoFullName),
	}
}

func (p *githubProvider) setLastError(err error) {
	if err == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.lastError = err.Error()
}

func (p *githubProvider) clearLastError() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.lastError = ""
}

func (p *githubProvider) reportSideEffectError(action string, err error) {
	reportSideEffectError(p.stderr, action, err)
}

func executeGitHubDelivery(
	ctx context.Context,
	api githubAPI,
	cfg *resolvedInstanceConfig,
	request bridgepkg.DeliveryRequest,
	state deliveryState,
	installationID int64,
) (bridgepkg.DeliveryAck, deliveryState, error) {
	event := request.Event
	if event.Seq <= state.LastSeq {
		return bridgepkg.DeliveryAck{}, state, fmt.Errorf(
			"github: out-of-order delivery seq %d after %d",
			event.Seq,
			state.LastSeq,
		)
	}

	if isGitHubDeleteEvent(event) {
		return executeGitHubDelete(ctx, api, request, state, installationID)
	}

	target, err := resolveGitHubDeliveryTarget(cfg, event)
	if err != nil {
		return bridgepkg.DeliveryAck{}, state, err
	}
	if shouldPostGitHubMessage(event, state, request) {
		return executeGitHubCreate(ctx, api, event, target, state, installationID)
	}

	return executeGitHubUpdate(ctx, api, request, state, installationID)
}

func isGitHubDeleteEvent(event bridgepkg.DeliveryEvent) bool {
	return event.Operation.Normalize() == bridgepkg.DeliveryOperationDelete ||
		normalizeDeliveryEventType(event.EventType) == bridgepkg.DeliveryEventTypeDelete
}

func executeGitHubDelete(
	ctx context.Context,
	api githubAPI,
	request bridgepkg.DeliveryRequest,
	state deliveryState,
	installationID int64,
) (bridgepkg.DeliveryAck, deliveryState, error) {
	event := request.Event
	remoteID := gitHubRemoteIDFromRequest(request, state)
	ref, err := parseGitHubRemoteCommentRef(remoteID)
	if err != nil {
		return bridgepkg.DeliveryAck{}, state, err
	}
	if err := deleteGitHubComment(ctx, api, ref, installationID); err != nil {
		return bridgepkg.DeliveryAck{}, state, err
	}
	state.LastSeq = event.Seq
	state.ReplaceRemoteMessageID = remoteID
	ack := bridgepkg.DeliveryAck{
		DeliveryID:             event.DeliveryID,
		Seq:                    event.Seq,
		RemoteMessageID:        remoteID,
		ReplaceRemoteMessageID: remoteID,
	}
	return ack, state, ack.ValidateFor(event)
}

func deleteGitHubComment(ctx context.Context, api githubAPI, ref githubRemoteCommentRef, installationID int64) error {
	switch ref.Kind {
	case "review":
		return api.DeleteReviewComment(ctx, ref.CommentID, installationID)
	default:
		return api.DeleteIssueComment(ctx, ref.CommentID, installationID)
	}
}

func executeGitHubCreate(
	ctx context.Context,
	api githubAPI,
	event bridgepkg.DeliveryEvent,
	target githubThreadRef,
	state deliveryState,
	installationID int64,
) (bridgepkg.DeliveryAck, deliveryState, error) {
	remoteID, err := createGitHubComment(ctx, api, target, event.Content.Text, installationID)
	if err != nil {
		return bridgepkg.DeliveryAck{}, state, err
	}
	state.LastSeq = event.Seq
	state.RemoteMessageID = remoteID
	if event.Seq > 1 {
		state.ReplaceRemoteMessageID = remoteID
	}
	ack := bridgepkg.DeliveryAck{
		DeliveryID:             event.DeliveryID,
		Seq:                    event.Seq,
		RemoteMessageID:        state.RemoteMessageID,
		ReplaceRemoteMessageID: state.ReplaceRemoteMessageID,
	}
	return ack, state, ack.ValidateFor(event)
}

func createGitHubComment(
	ctx context.Context,
	api githubAPI,
	target githubThreadRef,
	body string,
	installationID int64,
) (string, error) {
	if target.ReviewCommentID > 0 {
		comment, err := api.CreateReviewCommentReply(
			ctx,
			target.Number,
			target.ReviewCommentID,
			body,
			installationID,
		)
		if err != nil {
			return "", err
		}
		return encodeGitHubRemoteCommentRef(githubRemoteCommentRef{Kind: "review", CommentID: comment.ID}), nil
	}
	comment, err := api.CreateIssueComment(ctx, target.Number, body, installationID)
	if err != nil {
		return "", err
	}
	return encodeGitHubRemoteCommentRef(githubRemoteCommentRef{Kind: "issue", CommentID: comment.ID}), nil
}

func executeGitHubUpdate(
	ctx context.Context,
	api githubAPI,
	request bridgepkg.DeliveryRequest,
	state deliveryState,
	installationID int64,
) (bridgepkg.DeliveryAck, deliveryState, error) {
	event := request.Event
	remoteID := gitHubRemoteIDFromRequest(request, state)
	ref, err := parseGitHubRemoteCommentRef(remoteID)
	if err != nil {
		return bridgepkg.DeliveryAck{}, state, err
	}
	updatedRemoteID, err := updateGitHubComment(ctx, api, ref, event.Content.Text, installationID)
	if err != nil {
		return bridgepkg.DeliveryAck{}, state, err
	}
	state.LastSeq = event.Seq
	state.RemoteMessageID = updatedRemoteID
	state.ReplaceRemoteMessageID = updatedRemoteID
	ack := bridgepkg.DeliveryAck{
		DeliveryID:             event.DeliveryID,
		Seq:                    event.Seq,
		RemoteMessageID:        updatedRemoteID,
		ReplaceRemoteMessageID: state.ReplaceRemoteMessageID,
	}
	return ack, state, ack.ValidateFor(event)
}

func updateGitHubComment(
	ctx context.Context,
	api githubAPI,
	ref githubRemoteCommentRef,
	body string,
	installationID int64,
) (string, error) {
	switch ref.Kind {
	case "review":
		comment, err := api.UpdateReviewComment(ctx, ref.CommentID, body, installationID)
		if err != nil {
			return "", err
		}
		return encodeGitHubRemoteCommentRef(githubRemoteCommentRef{Kind: "review", CommentID: comment.ID}), nil
	default:
		comment, err := api.UpdateIssueComment(ctx, ref.CommentID, body, installationID)
		if err != nil {
			return "", err
		}
		return encodeGitHubRemoteCommentRef(githubRemoteCommentRef{Kind: "issue", CommentID: comment.ID}), nil
	}
}

func gitHubRemoteIDFromRequest(request bridgepkg.DeliveryRequest, state deliveryState) string {
	remoteID := firstNonEmpty(referenceRemoteMessageID(request.Event.Reference), state.RemoteMessageID)
	if remoteID == "" && request.Snapshot != nil {
		return strings.TrimSpace(request.Snapshot.RemoteMessageID)
	}
	return remoteID
}

func shouldPostGitHubMessage(
	event bridgepkg.DeliveryEvent,
	state deliveryState,
	request bridgepkg.DeliveryRequest,
) bool {
	switch normalizeDeliveryEventType(event.EventType) {
	case bridgepkg.DeliveryEventTypeStart:
		return true
	case bridgepkg.DeliveryEventTypeResume:
		if request.Snapshot == nil {
			return strings.TrimSpace(state.RemoteMessageID) == ""
		}
		return strings.TrimSpace(request.Snapshot.RemoteMessageID) == ""
	default:
		return strings.TrimSpace(state.RemoteMessageID) == ""
	}
}

func mapGitHubIssueComment(
	payload githubIssuePayload,
	managed subprocess.InitializeBridgeManagedInstance,
	receivedAt time.Time,
) (githubMappedInbound, error) {
	threadType := "pr"
	if payload.Issue.PullRequest == nil {
		threadType = "issue"
	}
	threadID := encodeGitHubThreadID(githubThreadRef{
		Owner:  payload.Repository.Owner.Login,
		Repo:   payload.Repository.Name,
		Number: payload.Issue.Number,
		Type:   threadType,
	})
	envelope := bridgepkg.InboundMessageEnvelope{
		BridgeInstanceID:  managed.Instance.ID,
		Scope:             managed.Instance.Scope,
		WorkspaceID:       managed.Instance.WorkspaceID,
		GroupID:           strings.TrimSpace(payload.Repository.FullName),
		ThreadID:          threadID,
		PlatformMessageID: strconv.FormatInt(payload.Comment.ID, 10),
		ReceivedAt:        normalizeGitHubReceivedAt(receivedAt, payload.Comment.CreatedAt),
		Sender:            githubSender(payload.Comment.User),
		Content:           bridgepkg.MessageContent{Text: payload.Comment.Body},
		EventFamily:       bridgepkg.InboundEventFamilyMessage,
		IdempotencyKey:    fmt.Sprintf("github:%s:issue_comment:%d", managed.Instance.ID, payload.Comment.ID),
	}
	if metadata, err := json.Marshal(map[string]any{
		"source":          "issue_comment",
		"repository":      strings.TrimSpace(payload.Repository.FullName),
		"thread_type":     threadType,
		"issue_number":    payload.Issue.Number,
		"comment_id":      payload.Comment.ID,
		"installation_id": installationIDFromWebhook(payload.Installation),
	}); err == nil {
		envelope.ProviderMetadata = metadata
	}
	item := githubMappedInbound{
		Envelope:       envelope,
		InstallationID: installationIDFromWebhook(payload.Installation),
	}
	return item, item.Envelope.Validate()
}

func mapGitHubReviewComment(
	payload githubReviewPayload,
	managed subprocess.InitializeBridgeManagedInstance,
	receivedAt time.Time,
) (githubMappedInbound, error) {
	rootCommentID := payload.Comment.ID
	if payload.Comment.InReplyToID > 0 {
		rootCommentID = payload.Comment.InReplyToID
	}
	threadID := encodeGitHubThreadID(githubThreadRef{
		Owner:           payload.Repository.Owner.Login,
		Repo:            payload.Repository.Name,
		Number:          payload.PullRequest.Number,
		Type:            "pr",
		ReviewCommentID: rootCommentID,
	})
	envelope := bridgepkg.InboundMessageEnvelope{
		BridgeInstanceID:  managed.Instance.ID,
		Scope:             managed.Instance.Scope,
		WorkspaceID:       managed.Instance.WorkspaceID,
		GroupID:           strings.TrimSpace(payload.Repository.FullName),
		ThreadID:          threadID,
		PlatformMessageID: strconv.FormatInt(payload.Comment.ID, 10),
		ReceivedAt:        normalizeGitHubReceivedAt(receivedAt, payload.Comment.CreatedAt),
		Sender:            githubSender(payload.Comment.User),
		Content:           bridgepkg.MessageContent{Text: payload.Comment.Body},
		EventFamily:       bridgepkg.InboundEventFamilyMessage,
		IdempotencyKey:    fmt.Sprintf("github:%s:review_comment:%d", managed.Instance.ID, payload.Comment.ID),
	}
	if metadata, err := json.Marshal(map[string]any{
		"source":                 "pull_request_review_comment",
		"repository":             strings.TrimSpace(payload.Repository.FullName),
		"pull_number":            payload.PullRequest.Number,
		"comment_id":             payload.Comment.ID,
		"root_review_comment_id": rootCommentID,
		"review_comment_path":    strings.TrimSpace(payload.Comment.Path),
		"installation_id":        installationIDFromWebhook(payload.Installation),
	}); err == nil {
		envelope.ProviderMetadata = metadata
	}
	item := githubMappedInbound{
		Envelope:       envelope,
		InstallationID: installationIDFromWebhook(payload.Installation),
	}
	return item, item.Envelope.Validate()
}

func selectGitHubIssueConfig(
	candidates []resolvedInstanceConfig,
	payload githubIssuePayload,
) (resolvedInstanceConfig, bool, error) {
	return selectGitHubRoute(candidates, payload.Repository.FullName)
}

func selectGitHubReviewConfig(
	candidates []resolvedInstanceConfig,
	payload githubReviewPayload,
) (resolvedInstanceConfig, bool, error) {
	return selectGitHubRoute(candidates, payload.Repository.FullName)
}

func selectGitHubRoute(candidates []resolvedInstanceConfig, fullName string) (resolvedInstanceConfig, bool, error) {
	normalized := strings.ToLower(strings.TrimSpace(fullName))
	if normalized == "" {
		return resolvedInstanceConfig{}, false, errors.New("github: webhook repository full_name is required")
	}
	for idx := range candidates {
		cfg := candidates[idx]
		if strings.ToLower(strings.TrimSpace(cfg.repoFullName)) == normalized {
			return cfg, true, nil
		}
	}
	return resolvedInstanceConfig{}, false, nil
}

func resolveGitHubDeliveryTarget(cfg *resolvedInstanceConfig, event bridgepkg.DeliveryEvent) (githubThreadRef, error) {
	if cfg == nil {
		return githubThreadRef{}, errors.New("github: config is required")
	}
	threadID := firstNonEmpty(
		strings.TrimSpace(event.DeliveryTarget.ThreadID),
		strings.TrimSpace(event.RoutingKey.ThreadID),
	)
	if threadID == "" {
		return githubThreadRef{}, errors.New("github: delivery target requires thread_id")
	}
	target, err := decodeGitHubThreadID(threadID)
	if err != nil {
		return githubThreadRef{}, err
	}
	if !strings.EqualFold(strings.TrimSpace(cfg.repoOwner), strings.TrimSpace(target.Owner)) ||
		!strings.EqualFold(strings.TrimSpace(cfg.repoName), strings.TrimSpace(target.Repo)) {
		return githubThreadRef{}, fmt.Errorf(
			"github: delivery target repo %q/%q does not match instance repo %q",
			target.Owner,
			target.Repo,
			cfg.repoFullName,
		)
	}
	return target, nil
}

func verifyGitHubWebhookSignature(
	_ context.Context,
	req *http.Request,
	body []byte,
	candidates []resolvedInstanceConfig,
) error {
	if req == nil {
		return errors.New("github: webhook request is required")
	}
	signature := strings.TrimSpace(req.Header.Get("X-Hub-Signature-256"))
	if signature == "" {
		return errors.New("github: missing webhook signature")
	}
	parts := strings.SplitN(signature, "=", 2)
	if len(parts) != 2 || !strings.EqualFold(strings.TrimSpace(parts[0]), "sha256") {
		return errors.New("github: invalid webhook signature format")
	}
	expected := strings.ToLower(strings.TrimSpace(parts[1]))
	if expected == "" {
		return errors.New("github: invalid webhook signature")
	}

	seen := make(map[string]struct{}, len(candidates))
	for idx := range candidates {
		cfg := candidates[idx]
		secret := strings.TrimSpace(cfg.webhookSecret)
		if secret == "" {
			continue
		}
		if _, ok := seen[secret]; ok {
			continue
		}
		seen[secret] = struct{}{}
		mac := hmac.New(sha256.New, []byte(secret))
		_, _ = mac.Write(body)
		if hmac.Equal([]byte(expected), []byte(hex.EncodeToString(mac.Sum(nil)))) {
			return nil
		}
	}
	return errors.New("github: invalid webhook signature")
}

func isGitHubSelfMessage(cfg *resolvedInstanceConfig, sender githubUser) bool {
	if cfg == nil {
		return false
	}
	if strings.TrimSpace(cfg.botLogin) == "" {
		return false
	}
	return normalizeUsername(cfg.botLogin) == normalizeUsername(sender.Login)
}

func installationIDFromWebhook(installation *githubInstallation) int64 {
	if installation == nil {
		return 0
	}
	return installation.ID
}

func installationIDFromMetadata(raw json.RawMessage) int64 {
	if len(raw) == 0 {
		return 0
	}
	payload := struct {
		InstallationID int64 `json:"installation_id,omitempty"`
	}{}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return 0
	}
	return payload.InstallationID
}

func githubSender(user githubUser) bridgepkg.MessageSender {
	return bridgepkg.MessageSender{
		ID:          strconv.FormatInt(user.ID, 10),
		Username:    strings.TrimSpace(user.Login),
		DisplayName: strings.TrimSpace(user.Login),
	}
}

func normalizeGitHubReceivedAt(fallback time.Time, raw string) time.Time {
	if parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(raw)); err == nil {
		return parsed.UTC()
	}
	if fallback.IsZero() {
		return time.Now().UTC()
	}
	return fallback.UTC()
}

func encodeGitHubThreadID(ref githubThreadRef) string {
	owner := strings.TrimSpace(ref.Owner)
	repo := strings.TrimSpace(ref.Repo)
	threadType := strings.TrimSpace(ref.Type)
	if threadType == "" {
		threadType = "pr"
	}
	if threadType == "issue" {
		return fmt.Sprintf("github:%s/%s:issue:%d", owner, repo, ref.Number)
	}
	if ref.ReviewCommentID > 0 {
		return fmt.Sprintf("github:%s/%s:%d:rc:%d", owner, repo, ref.Number, ref.ReviewCommentID)
	}
	return fmt.Sprintf("github:%s/%s:%d", owner, repo, ref.Number)
}

func decodeGitHubThreadID(value string) (githubThreadRef, error) {
	trimmed := strings.TrimSpace(value)
	if matches := githubReviewThreadPattern.FindStringSubmatch(trimmed); len(matches) == 5 {
		number, err := strconv.ParseInt(matches[3], 10, 64)
		if err != nil {
			return githubThreadRef{}, fmt.Errorf("github: parse pull number: %w", err)
		}
		reviewCommentID, err := strconv.ParseInt(matches[4], 10, 64)
		if err != nil {
			return githubThreadRef{}, fmt.Errorf("github: parse review comment id: %w", err)
		}
		return githubThreadRef{
			Owner:           matches[1],
			Repo:            matches[2],
			Number:          number,
			Type:            "pr",
			ReviewCommentID: reviewCommentID,
		}, nil
	}
	if matches := githubIssueThreadPattern.FindStringSubmatch(trimmed); len(matches) == 4 {
		number, err := strconv.ParseInt(matches[3], 10, 64)
		if err != nil {
			return githubThreadRef{}, fmt.Errorf("github: parse issue number: %w", err)
		}
		return githubThreadRef{
			Owner:  matches[1],
			Repo:   matches[2],
			Number: number,
			Type:   "issue",
		}, nil
	}
	if matches := githubPRThreadPattern.FindStringSubmatch(trimmed); len(matches) == 4 {
		number, err := strconv.ParseInt(matches[3], 10, 64)
		if err != nil {
			return githubThreadRef{}, fmt.Errorf("github: parse pull number: %w", err)
		}
		return githubThreadRef{
			Owner:  matches[1],
			Repo:   matches[2],
			Number: number,
			Type:   "pr",
		}, nil
	}
	return githubThreadRef{}, fmt.Errorf("github: invalid thread id %q", trimmed)
}

func encodeGitHubRemoteCommentRef(ref githubRemoteCommentRef) string {
	kind := strings.TrimSpace(ref.Kind)
	if kind == "" {
		kind = "issue"
	}
	return fmt.Sprintf("%s:%d", kind, ref.CommentID)
}

func parseGitHubRemoteCommentRef(value string) (githubRemoteCommentRef, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return githubRemoteCommentRef{}, errors.New("github: remote message id is required")
	}
	parts := strings.SplitN(trimmed, ":", 2)
	if len(parts) != 2 {
		return githubRemoteCommentRef{}, fmt.Errorf("github: invalid remote message id %q", trimmed)
	}
	commentID, err := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
	if err != nil || commentID <= 0 {
		return githubRemoteCommentRef{}, fmt.Errorf("github: invalid remote message id %q", trimmed)
	}
	kind := strings.ToLower(strings.TrimSpace(parts[0]))
	switch kind {
	case "issue", "review":
		return githubRemoteCommentRef{Kind: kind, CommentID: commentID}, nil
	default:
		return githubRemoteCommentRef{}, fmt.Errorf("github: invalid remote message kind %q", kind)
	}
}

func referenceRemoteMessageID(reference *bridgepkg.DeliveryMessageReference) string {
	if reference == nil {
		return ""
	}
	return strings.TrimSpace(reference.RemoteMessageID)
}

func normalizeGitHubRepository(owner string, name string, fullName string) (string, string, string, error) {
	trimmedFull := strings.TrimSpace(fullName)
	if trimmedFull != "" {
		parts := strings.SplitN(trimmedFull, "/", 2)
		if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
			return "", "", "", fmt.Errorf("github: repository full_name %q must be owner/repo", trimmedFull)
		}
		return strings.TrimSpace(
				parts[0],
			), strings.TrimSpace(
				parts[1],
			), strings.TrimSpace(
				parts[0],
			) + "/" + strings.TrimSpace(
				parts[1],
			), nil
	}
	trimmedOwner := strings.TrimSpace(owner)
	trimmedName := strings.TrimSpace(name)
	if trimmedOwner == "" || trimmedName == "" {
		return "", "", "", errors.New("github: repository owner and name are required")
	}
	return trimmedOwner, trimmedName, trimmedOwner + "/" + trimmedName, nil
}

func deliveryStateKey(instanceID string, deliveryID string) string {
	return strings.TrimSpace(instanceID) + ":" + strings.TrimSpace(deliveryID)
}

func normalizeWebhookPath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return ""
	}
	parsed, err := url.Parse(trimmed)
	if err == nil && parsed != nil && parsed.Path != "" {
		trimmed = parsed.Path
	}
	if !strings.HasPrefix(trimmed, "/") {
		trimmed = "/" + trimmed
	}
	return strings.TrimRight(trimmed, "/")
}

func normalizeURL(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return trimmed
	}
	return strings.TrimRight(parsed.String(), "/")
}

func normalizeUsername(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func managedInstancesToInstances(items []subprocess.InitializeBridgeManagedInstance) []bridgepkg.BridgeInstance {
	result := make([]bridgepkg.BridgeInstance, 0, len(items))
	for _, item := range items {
		result = append(result, item.Instance)
	}
	return result
}

func writeWebhookText(w http.ResponseWriter, body string) error {
	w.WriteHeader(http.StatusOK)
	_, err := io.WriteString(w, body)
	return err
}

func parseRetryAfter(value string) time.Duration {
	seconds, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || seconds <= 0 {
		return 0
	}
	return time.Duration(seconds) * time.Second
}

func isNotInitializedRPCError(err error) bool {
	rpcErr := &subprocess.RPCError{}
	if !errors.As(err, &rpcErr) {
		return false
	}
	return rpcErr.Code == rpcCodeNotInitialized
}

func cloneDegradation(degradation *bridgepkg.BridgeDegradation) *bridgepkg.BridgeDegradation {
	if degradation == nil {
		return nil
	}
	cloned := *degradation
	return &cloned
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func normalizeDeliveryEventType(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func isTerminalGitHubDeliveryEvent(event bridgepkg.DeliveryEvent) bool {
	if event.Operation.Normalize() == bridgepkg.DeliveryOperationDelete {
		return true
	}
	switch normalizeDeliveryEventType(event.EventType) {
	case bridgepkg.DeliveryEventTypeFinal, bridgepkg.DeliveryEventTypeError, bridgepkg.DeliveryEventTypeDelete:
		return true
	default:
		return false
	}
}
