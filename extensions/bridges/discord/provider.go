package main

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
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
	discordListenAddrEnv = "AGH_BRIDGE_DISCORD_LISTEN_ADDR"
	discordAPIBaseEnv    = "AGH_BRIDGE_DISCORD_API_BASE_URL"

	discordDefaultAPIBaseURL        = "https://discord.com/api/v10"
	discordWebhookReadHeaderTimeout = 10 * time.Second
	discordWebhookIdleTimeout       = 2 * time.Minute

	discordInteractionTypePing               = 1
	discordInteractionTypeApplicationCommand = 2
	discordInteractionTypeMessageComponent   = 3

	discordInteractionResponseTypePong                   = 1
	discordInteractionResponseTypeDeferredChannelMessage = 5
	discordInteractionResponseTypeDeferredUpdateMessage  = 6
	discordWebhookEnvelopeTypePing                       = 0
	discordWebhookEnvelopeTypeEvent                      = 1
	discordChannelTypeDM                                 = 1
	discordChannelTypeGroupDM                            = 3
	discordChannelTypeAnnouncementThread                 = 10
	discordChannelTypePublicThread                       = 11
	discordChannelTypePrivateThread                      = 12
	discordApplicationCommandOptionTypeSubcommand        = 1
	discordApplicationCommandOptionTypeSubcommandGroup   = 2
	rpcCodeNotInitialized                                = -32003
)

type discordProvider struct {
	sdk     *bridgesdk.Runtime
	stderr  io.Writer
	env     markerEnv
	now     func() time.Time
	session *bridgesdk.Session

	mu             sync.RWMutex
	lastError      string
	server         *http.Server
	serverAddr     string
	listenAddr     string
	routes         map[string]resolvedInstanceConfig
	deliveries     map[string]deliveryState
	reportedStatus map[string]bridgepkg.BridgeStatus
	apiFactory     func(resolvedInstanceConfig) discordAPI

	stopCh   chan struct{}
	stopOnce sync.Once
	wg       sync.WaitGroup
}

type deliveryState struct {
	LastSeq                int64
	RemoteMessageID        string
	ReplaceRemoteMessageID string
}

type discordProviderConfig struct {
	APIBaseURL    string `json:"api_base_url,omitempty"`
	ApplicationID string `json:"application_id,omitempty"`
	Webhook       struct {
		ListenAddr string `json:"listen_addr,omitempty"`
		Path       string `json:"path,omitempty"`
	} `json:"webhook,omitempty"`
	Batching struct {
		DelayMS        int `json:"delay_ms,omitempty"`
		SplitDelayMS   int `json:"split_delay_ms,omitempty"`
		SplitThreshold int `json:"split_threshold,omitempty"`
	} `json:"batching,omitempty"`
	DM struct {
		AllowUserIDs    []string `json:"allow_user_ids,omitempty"`
		AllowUsernames  []string `json:"allow_usernames,omitempty"`
		PairedUserIDs   []string `json:"paired_user_ids,omitempty"`
		PairedUsernames []string `json:"paired_usernames,omitempty"`
	} `json:"dm,omitempty"`
}

type resolvedInstanceConfig struct {
	managed            subprocess.InitializeBridgeManagedInstance
	instanceID         string
	listenAddr         string
	webhookPath        string
	apiBaseURL         string
	applicationID      string
	botToken           string
	publicKey          string
	dmPolicy           bridgepkg.BridgeDMPolicy
	allowUserIDs       map[string]struct{}
	allowUsernames     map[string]struct{}
	pairedUserIDs      map[string]struct{}
	pairedUsernames    map[string]struct{}
	dedup              *bridgesdk.DedupCache
	rateLimiter        *bridgesdk.FixedWindowRateLimiter
	inFlightLimiter    *bridgesdk.InFlightLimiter
	batcher            *bridgesdk.InboundBatcher
	configError        error
	initialDegradation *bridgepkg.BridgeDegradation
	initialStatus      bridgepkg.BridgeStatus
}

type discordPayloadProbe struct {
	Token json.RawMessage `json:"token,omitempty"`
	Event json.RawMessage `json:"event,omitempty"`
}

type discordWebhookEventEnvelope struct {
	Type  int                  `json:"type"`
	Event *discordWebhookEvent `json:"event,omitempty"`
}

type discordWebhookEvent struct {
	ID        string          `json:"id,omitempty"`
	Type      string          `json:"type"`
	Timestamp string          `json:"timestamp,omitempty"`
	Data      json.RawMessage `json:"data,omitempty"`
}

type discordMessageEvent struct {
	ID          string              `json:"id"`
	ChannelID   string              `json:"channel_id"`
	GuildID     string              `json:"guild_id,omitempty"`
	ParentID    string              `json:"parent_id,omitempty"`
	ChannelType int                 `json:"channel_type,omitempty"`
	Content     string              `json:"content,omitempty"`
	Timestamp   string              `json:"timestamp,omitempty"`
	Author      discordUser         `json:"author"`
	Attachments []discordAttachment `json:"attachments,omitempty"`
}

type discordAttachment struct {
	ID          string `json:"id,omitempty"`
	Filename    string `json:"filename,omitempty"`
	ContentType string `json:"content_type,omitempty"`
	URL         string `json:"url,omitempty"`
}

type discordReactionEvent struct {
	ID          string                    `json:"id,omitempty"`
	ChannelID   string                    `json:"channel_id"`
	GuildID     string                    `json:"guild_id,omitempty"`
	ParentID    string                    `json:"parent_id,omitempty"`
	ChannelType int                       `json:"channel_type,omitempty"`
	MessageID   string                    `json:"message_id"`
	UserID      string                    `json:"user_id"`
	Member      *discordInteractionMember `json:"member,omitempty"`
	Emoji       discordEmoji              `json:"emoji"`
	Timestamp   string                    `json:"timestamp,omitempty"`
}

type discordInteraction struct {
	ID            string                     `json:"id"`
	ApplicationID string                     `json:"application_id,omitempty"`
	Type          int                        `json:"type"`
	Token         string                     `json:"token,omitempty"`
	GuildID       string                     `json:"guild_id,omitempty"`
	ChannelID     string                     `json:"channel_id,omitempty"`
	Channel       *discordInteractionChannel `json:"channel,omitempty"`
	Data          *discordInteractionData    `json:"data,omitempty"`
	Member        *discordInteractionMember  `json:"member,omitempty"`
	User          *discordUser               `json:"user,omitempty"`
	Message       *discordInteractionMessage `json:"message,omitempty"`
}

type discordInteractionChannel struct {
	ID       string `json:"id,omitempty"`
	Type     int    `json:"type,omitempty"`
	ParentID string `json:"parent_id,omitempty"`
}

type discordInteractionMember struct {
	User *discordUser `json:"user,omitempty"`
}

type discordInteractionMessage struct {
	ID string `json:"id,omitempty"`
}

type discordInteractionData struct {
	ID            string                     `json:"id,omitempty"`
	Name          string                     `json:"name,omitempty"`
	Type          int                        `json:"type,omitempty"`
	CustomID      string                     `json:"custom_id,omitempty"`
	ComponentType int                        `json:"component_type,omitempty"`
	Values        []string                   `json:"values,omitempty"`
	Options       []discordInteractionOption `json:"options,omitempty"`
}

type discordInteractionOption struct {
	Name    string                     `json:"name,omitempty"`
	Type    int                        `json:"type,omitempty"`
	Value   any                        `json:"value,omitempty"`
	Options []discordInteractionOption `json:"options,omitempty"`
}

type discordEmoji struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

type discordUser struct {
	ID         string `json:"id,omitempty"`
	Username   string `json:"username,omitempty"`
	GlobalName string `json:"global_name,omitempty"`
	Bot        bool   `json:"bot,omitempty"`
}

type discordMappedInbound struct {
	Envelope bridgepkg.InboundMessageEnvelope
	Direct   bool
	User     discordUserIdentity
}

type discordUserIdentity struct {
	ID          string
	Username    string
	DisplayName string
}

type discordAPI interface {
	GetBotUser(context.Context) (*discordBotIdentity, error)
	PostMessage(context.Context, discordPostMessageRequest) (*discordPostedMessage, error)
	UpdateMessage(context.Context, discordUpdateMessageRequest) error
	DeleteMessage(context.Context, discordDeleteMessageRequest) error
}

type discordBotIdentity struct {
	ID       string `json:"id,omitempty"`
	Username string `json:"username,omitempty"`
}

type discordPostedMessage struct {
	ID string `json:"id,omitempty"`
}

type discordPostMessageRequest struct {
	ChannelID string `json:"-"`
	Content   string `json:"content"`
}

type discordUpdateMessageRequest struct {
	ChannelID string `json:"-"`
	MessageID string `json:"-"`
	Content   string `json:"content"`
}

type discordDeleteMessageRequest struct {
	ChannelID string `json:"-"`
	MessageID string `json:"-"`
}

type discordBotClient struct {
	baseURL    string
	botToken   string
	httpClient *http.Client
}

func newDiscordProvider(stderr io.Writer) (*discordProvider, error) {
	if stderr == nil {
		stderr = io.Discard
	}

	provider := &discordProvider{
		stderr:         stderr,
		env:            markerEnvFromProcess(),
		now:            func() time.Time { return time.Now().UTC() },
		routes:         make(map[string]resolvedInstanceConfig),
		deliveries:     make(map[string]deliveryState),
		reportedStatus: make(map[string]bridgepkg.BridgeStatus),
		stopCh:         make(chan struct{}),
	}
	provider.apiFactory = func(cfg resolvedInstanceConfig) discordAPI {
		return &discordBotClient{
			baseURL:  cfg.apiBaseURL,
			botToken: cfg.botToken,
			httpClient: &http.Client{
				Timeout: 10 * time.Second,
			},
		}
	}

	sdkRuntime, err := bridgesdk.NewRuntime(bridgesdk.RuntimeConfig{
		ExtensionInfo: subprocess.InitializeExtensionInfo{
			Name:    "discord",
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

func (p *discordProvider) serve(stdin io.Reader, stdout io.Writer) error {
	p.reportSideEffectError("write start marker", appendMarkerLine(p.env.startsPath, fmt.Sprintf("pid=%d", os.Getpid())))
	return p.sdk.Serve(context.Background(), stdin, stdout)
}

func (p *discordProvider) handleInitialize(_ context.Context, session *bridgesdk.Session) error {
	p.mu.Lock()
	p.session = session
	p.mu.Unlock()

	marker := initializeMarker{
		Request:  session.InitializeRequest(),
		Response: session.InitializeResponse(),
	}
	p.reportSideEffectError("write initialize marker", writeJSONFile(p.env.handshakePath, marker))
	p.clearLastError()

	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		p.afterInitialize(session)
	}()

	return nil
}

func (p *discordProvider) afterInitialize(session *bridgesdk.Session) {
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

	configs, reconcileErr := p.reconcileInstanceConfigs(ctx, session, listed)
	if reconcileErr != nil && ownershipErr == nil {
		ownershipErr = reconcileErr
	}
	for _, cfg := range configs {
		status := cfg.initialStatus
		degradation := cfg.initialDegradation
		if status == "" {
			status = bridgepkg.BridgeStatusReady
		}
		if _, err := p.reportState(ctx, session, cfg.instanceID, status, degradation); err != nil && ownershipErr == nil {
			ownershipErr = err
		}
	}

	if ownershipErr != nil {
		p.setLastError(ownershipErr)
	} else {
		p.clearLastError()
	}
}

func (p *discordProvider) handleBridgesDeliver(
	ctx context.Context,
	session *bridgesdk.Session,
	request bridgepkg.DeliveryRequest,
) (bridgepkg.DeliveryAck, error) {
	marker := deliveryMarker{
		PID:     os.Getpid(),
		Request: request,
	}

	cfg, err := p.waitForInstanceConfig(strings.TrimSpace(request.Event.BridgeInstanceID), 500*time.Millisecond)
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

	api := p.apiFactory(cfg)
	ack, state, err := executeDiscordDelivery(ctx, api, request, p.deliveryState(cfg.instanceID, request.Event.DeliveryID))
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

	p.storeDeliveryState(cfg.instanceID, request.Event.DeliveryID, state)
	p.reportReadyIfNeeded(ctx, session, cfg.instanceID)

	marker.Ack = &ack
	p.reportSideEffectError("write delivery marker", appendJSONLine(p.env.deliveryPath, marker))
	p.clearLastError()
	return ack, nil
}

func (p *discordProvider) healthCheck() error {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if strings.TrimSpace(p.lastError) == "" {
		return nil
	}
	return errors.New(strings.TrimSpace(p.lastError))
}

func (p *discordProvider) handleShutdown(
	_ context.Context,
	_ *bridgesdk.Session,
	request subprocess.ShutdownRequest,
) error {
	p.stop()

	shutdownCtx := context.Background()
	if request.DeadlineMS > 0 {
		var cancel context.CancelFunc
		shutdownCtx, cancel = context.WithTimeout(context.Background(), time.Duration(request.DeadlineMS)*time.Millisecond)
		defer cancel()
	}

	p.mu.Lock()
	server := p.server
	p.mu.Unlock()
	if server != nil {
		_ = server.Shutdown(shutdownCtx)
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

	p.reportSideEffectError("write shutdown marker", appendMarkerLine(p.env.shutdownPath, fmt.Sprintf("pid=%d", os.Getpid())))
	return nil
}

func (p *discordProvider) stop() {
	p.stopOnce.Do(func() {
		close(p.stopCh)
		p.mu.Lock()
		defer p.mu.Unlock()
		for id, cfg := range p.routes {
			if cfg.batcher != nil {
				cfg.batcher.Close()
				cfg.batcher = nil
				p.routes[id] = cfg
			}
		}
	})
}

func (p *discordProvider) syncOwnedInstances(
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

func (p *discordProvider) getOwnedInstance(
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

func (p *discordProvider) reportState(
	ctx context.Context,
	session *bridgesdk.Session,
	bridgeInstanceID string,
	status bridgepkg.BridgeStatus,
	degradation *bridgepkg.BridgeDegradation,
) (*bridgepkg.BridgeInstance, error) {
	var result *bridgepkg.BridgeInstance
	err := p.retryHostCall(ctx, func(callCtx context.Context) error {
		instance, callErr := session.HostAPI().ReportBridgeInstanceState(callCtx, extensioncontract.BridgesInstancesReportStateParams{
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
		return nil, err
	}

	p.mu.Lock()
	p.reportedStatus[strings.TrimSpace(bridgeInstanceID)] = result.Status.Normalize()
	p.mu.Unlock()
	p.reportSideEffectError("write state marker", appendJSONLine(p.env.statePath, stateMarker{
		BridgeInstanceID: result.ID,
		Status:           result.Status,
		Instance:         *result,
	}))
	return result, nil
}

func (p *discordProvider) reportReadyIfNeeded(ctx context.Context, session *bridgesdk.Session, bridgeInstanceID string) {
	p.mu.RLock()
	status := p.reportedStatus[strings.TrimSpace(bridgeInstanceID)]
	p.mu.RUnlock()
	if status == bridgepkg.BridgeStatusReady {
		return
	}
	_, _ = p.reportState(ctx, session, bridgeInstanceID, bridgepkg.BridgeStatusReady, nil)
}

func (p *discordProvider) ingestBridgeMessage(
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

func (p *discordProvider) retryHostCall(ctx context.Context, fn func(context.Context) error) error {
	if ctx == nil {
		ctx = context.Background()
	}

	delay := 10 * time.Millisecond
	var lastErr error
	for attempt := 0; attempt < 6; attempt++ {
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

func (p *discordProvider) reconcileInstanceConfigs(
	ctx context.Context,
	session *bridgesdk.Session,
	managed []subprocess.InitializeBridgeManagedInstance,
) ([]resolvedInstanceConfig, error) {
	if len(managed) == 0 {
		p.mu.Lock()
		p.routes = make(map[string]resolvedInstanceConfig)
		p.mu.Unlock()
		return nil, nil
	}

	configs := make([]resolvedInstanceConfig, 0, len(managed))
	requestedListen := strings.TrimSpace(os.Getenv(discordListenAddrEnv))
	usedPaths := make(map[string]string, len(managed))

	for _, item := range managed {
		cfg := p.resolveInstanceConfig(session, item)
		if cfg.listenAddr != "" {
			if requestedListen == "" {
				requestedListen = cfg.listenAddr
			} else if requestedListen != cfg.listenAddr && cfg.configError == nil {
				cfg.configError = fmt.Errorf("discord: instance %q configured incompatible listen_addr %q (runtime uses %q)", cfg.instanceID, cfg.listenAddr, requestedListen)
			}
		}
		if owner, ok := usedPaths[cfg.webhookPath]; ok && cfg.webhookPath != "" && cfg.configError == nil {
			cfg.configError = fmt.Errorf("discord: webhook path %q is shared by %q and %q", cfg.webhookPath, owner, cfg.instanceID)
		}
		if cfg.webhookPath != "" {
			usedPaths[cfg.webhookPath] = cfg.instanceID
		}
		configs = append(configs, cfg)
	}

	if requestedListen == "" {
		for idx := range configs {
			if configs[idx].configError == nil {
				configs[idx].configError = errors.New("discord: webhook listen address is required")
			}
		}
	} else if err := p.startServer(requestedListen); err != nil {
		for idx := range configs {
			if configs[idx].configError == nil {
				configs[idx].configError = err
			}
		}
	}

	nextRoutes := make(map[string]resolvedInstanceConfig, len(configs))
	p.mu.Lock()
	existing := p.routes
	for _, cfg := range configs {
		if prior, ok := existing[cfg.instanceID]; ok && prior.batcher != nil && cfg.batcher == nil {
			prior.batcher.Close()
		}
		nextRoutes[cfg.instanceID] = cfg
	}
	for instanceID, prior := range existing {
		if _, ok := nextRoutes[instanceID]; ok {
			continue
		}
		if prior.batcher != nil {
			prior.batcher.Close()
		}
	}
	p.routes = nextRoutes
	p.listenAddr = requestedListen
	p.mu.Unlock()

	for idx := range configs {
		status, degradation, err := p.determineInitialState(ctx, configs[idx])
		if err != nil {
			p.setLastError(err)
		}
		configs[idx].initialStatus = status
		configs[idx].initialDegradation = degradation
	}

	return configs, nil
}

func (p *discordProvider) resolveInstanceConfig(
	session *bridgesdk.Session,
	managed subprocess.InitializeBridgeManagedInstance,
) resolvedInstanceConfig {
	cfg := discordProviderConfig{}
	if len(managed.Instance.ProviderConfig) > 0 {
		if err := json.Unmarshal(managed.Instance.ProviderConfig, &cfg); err != nil {
			return resolvedInstanceConfig{
				managed:     managed,
				instanceID:  managed.Instance.ID,
				configError: fmt.Errorf("discord: decode provider_config for %q: %w", managed.Instance.ID, err),
			}
		}
	}

	botToken, _ := session.Cache().BoundSecretValue(managed.Instance.ID, "bot_token")
	publicKey, _ := session.Cache().BoundSecretValue(managed.Instance.ID, "public_key")
	listenAddr := firstNonEmpty(cfg.Webhook.ListenAddr, strings.TrimSpace(os.Getenv(discordListenAddrEnv)))
	webhookPath := normalizeWebhookPath(firstNonEmpty(cfg.Webhook.Path, "/discord/"+strings.TrimSpace(managed.Instance.ID)))
	apiBaseURL := normalizeURL(firstNonEmpty(strings.TrimSpace(os.Getenv(discordAPIBaseEnv)), discordDefaultAPIBaseURL))

	resolved := resolvedInstanceConfig{
		managed:         managed,
		instanceID:      strings.TrimSpace(managed.Instance.ID),
		listenAddr:      listenAddr,
		webhookPath:     webhookPath,
		apiBaseURL:      apiBaseURL,
		applicationID:   strings.TrimSpace(cfg.ApplicationID),
		botToken:        strings.TrimSpace(botToken),
		publicKey:       strings.TrimSpace(publicKey),
		dmPolicy:        managed.Instance.DMPolicy.Normalize(),
		allowUserIDs:    buildDiscordIDSet(cfg.DM.AllowUserIDs),
		allowUsernames:  buildDiscordUsernameSet(cfg.DM.AllowUsernames),
		pairedUserIDs:   buildDiscordIDSet(cfg.DM.PairedUserIDs),
		pairedUsernames: buildDiscordUsernameSet(cfg.DM.PairedUsernames),
		dedup:           bridgesdk.NewDedupCache(5*time.Minute, 4000),
		rateLimiter:     bridgesdk.NewFixedWindowRateLimiter(200, time.Minute),
		inFlightLimiter: bridgesdk.NewInFlightLimiter(24),
	}
	if resolved.dmPolicy == "" {
		resolved.dmPolicy = bridgepkg.BridgeDMPolicyOpen
	}
	if resolved.webhookPath == "" {
		resolved.configError = errors.New("discord: webhook path is required")
		return resolved
	}

	if cfg.Batching.DelayMS > 0 {
		batcher, err := bridgesdk.NewInboundBatcher(bridgesdk.InboundBatcherConfig{
			Context: context.Background(),
			Delay:   time.Duration(cfg.Batching.DelayMS) * time.Millisecond,
			SplitDelay: func() time.Duration {
				if cfg.Batching.SplitDelayMS <= 0 {
					return time.Duration(cfg.Batching.DelayMS) * time.Millisecond
				}
				return time.Duration(cfg.Batching.SplitDelayMS) * time.Millisecond
			}(),
			SplitThreshold: cfg.Batching.SplitThreshold,
			Dispatch: func(ctx context.Context, batch bridgesdk.InboundBatch) error {
				return p.dispatchInboundBatch(ctx, resolved.instanceID, batch)
			},
			Now: func() time.Time { return p.now() },
		})
		if err != nil {
			resolved.configError = err
			return resolved
		}
		resolved.batcher = batcher
	}

	return resolved
}

func (p *discordProvider) determineInitialState(
	ctx context.Context,
	cfg resolvedInstanceConfig,
) (bridgepkg.BridgeStatus, *bridgepkg.BridgeDegradation, error) {
	if cfg.configError != nil {
		return bridgepkg.BridgeStatusDegraded, &bridgepkg.BridgeDegradation{
			Reason:  bridgepkg.BridgeDegradationReasonTenantConfigInvalid,
			Message: cfg.configError.Error(),
		}, cfg.configError
	}
	if strings.TrimSpace(cfg.botToken) == "" {
		err := errors.New("discord: bot_token secret binding is required")
		return bridgepkg.BridgeStatusAuthRequired, &bridgepkg.BridgeDegradation{
			Reason:  bridgepkg.BridgeDegradationReasonAuthFailed,
			Message: err.Error(),
		}, err
	}
	if _, err := decodeDiscordPublicKey(cfg.publicKey); err != nil {
		wrapped := fmt.Errorf("discord: public_key secret binding invalid: %w", err)
		return bridgepkg.BridgeStatusAuthRequired, &bridgepkg.BridgeDegradation{
			Reason:  bridgepkg.BridgeDegradationReasonAuthFailed,
			Message: wrapped.Error(),
		}, wrapped
	}
	bot, err := p.apiFactory(cfg).GetBotUser(ctx)
	if err != nil {
		classified := bridgesdk.ClassifyError(err)
		recovery := classified.Recovery()
		status := recovery.Status
		if status == "" {
			status = bridgepkg.BridgeStatusError
		}
		if recovery.Degradation != nil {
			return status, recovery.Degradation, err
		}
		return status, &bridgepkg.BridgeDegradation{
			Reason:  bridgepkg.BridgeDegradationReasonProviderTimeout,
			Message: classified.Message,
		}, err
	}
	if cfg.applicationID != "" && strings.TrimSpace(bot.ID) != "" && strings.TrimSpace(bot.ID) != cfg.applicationID {
		err := fmt.Errorf("discord: application_id %q does not match bot identity %q", cfg.applicationID, bot.ID)
		return bridgepkg.BridgeStatusDegraded, &bridgepkg.BridgeDegradation{
			Reason:  bridgepkg.BridgeDegradationReasonTenantConfigInvalid,
			Message: err.Error(),
		}, err
	}
	return bridgepkg.BridgeStatusReady, nil, nil
}

func (p *discordProvider) startServer(listenAddr string) error {
	p.mu.RLock()
	server := p.server
	currentListen := p.listenAddr
	p.mu.RUnlock()
	if server != nil {
		if currentListen != "" && currentListen != strings.TrimSpace(listenAddr) {
			return fmt.Errorf("discord: runtime already listening on %q, cannot switch to %q", currentListen, listenAddr)
		}
		return nil
	}

	ln, err := net.Listen("tcp", strings.TrimSpace(listenAddr))
	if err != nil {
		return fmt.Errorf("discord: listen %q: %w", listenAddr, err)
	}

	httpServer := &http.Server{
		Handler:           http.HandlerFunc(p.serveWebhookHTTP),
		ReadHeaderTimeout: discordWebhookReadHeaderTimeout,
		IdleTimeout:       discordWebhookIdleTimeout,
	}

	actualAddr := ln.Addr().String()
	p.mu.Lock()
	p.server = httpServer
	p.serverAddr = actualAddr
	p.listenAddr = strings.TrimSpace(listenAddr)
	p.mu.Unlock()

	p.reportSideEffectError("write start marker", appendMarkerLine(p.env.startsPath, fmt.Sprintf("listen=%s", actualAddr)))

	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		if serveErr := httpServer.Serve(ln); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			p.setLastError(serveErr)
		}
	}()

	return nil
}

func (p *discordProvider) serveWebhookHTTP(w http.ResponseWriter, r *http.Request) {
	cfg, ok := p.configForPath(r.URL.Path)
	if !ok {
		http.NotFound(w, r)
		return
	}

	handler, err := bridgesdk.NewWebhookHandler(bridgesdk.WebhookGuardConfig{
		AllowedMethods:      []string{http.MethodPost},
		AllowedContentTypes: []string{"application/json"},
		MaxBodyBytes:        1 << 20,
		RateLimiter:         cfg.rateLimiter,
		InFlightLimiter:     cfg.inFlightLimiter,
		VerifySignature: func(ctx context.Context, req *http.Request, body []byte) error {
			return verifyDiscordSignature(ctx, req, body, cfg.publicKey, p.now())
		},
		RequestKey: func(req *http.Request) string {
			return req.RemoteAddr + "|" + cfg.instanceID
		},
		Now: func() time.Time { return p.now() },
	}, func(w http.ResponseWriter, r *http.Request, request bridgesdk.WebhookRequest) error {
		return p.handleWebhookRequest(w, r, cfg, request)
	})
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		p.setLastError(err)
		return
	}
	handler.ServeHTTP(w, r)
}

func (p *discordProvider) handleWebhookRequest(
	w http.ResponseWriter,
	r *http.Request,
	cfg resolvedInstanceConfig,
	request bridgesdk.WebhookRequest,
) error {
	var probe discordPayloadProbe
	if err := json.Unmarshal(request.Body, &probe); err != nil {
		return &bridgesdk.HTTPError{StatusCode: http.StatusBadRequest, Message: "invalid discord webhook payload"}
	}
	if len(bytes.TrimSpace(probe.Token)) > 0 {
		return p.handleInteractionWebhook(w, cfg, request)
	}
	return p.handleEventWebhook(w, r, cfg, request)
}

func (p *discordProvider) handleInteractionWebhook(
	w http.ResponseWriter,
	cfg resolvedInstanceConfig,
	request bridgesdk.WebhookRequest,
) error {
	var interaction discordInteraction
	if err := json.Unmarshal(request.Body, &interaction); err != nil {
		return &bridgesdk.HTTPError{StatusCode: http.StatusBadRequest, Message: "invalid discord interaction payload"}
	}

	switch interaction.Type {
	case discordInteractionTypePing:
		return writeDiscordInteractionResponse(w, discordInteractionResponseTypePong)
	case discordInteractionTypeApplicationCommand:
		mapped, err := mapDiscordInteractionCommand(interaction, cfg.managed, request.ReceivedAt)
		if err != nil {
			return &bridgesdk.HTTPError{StatusCode: http.StatusBadRequest, Message: err.Error()}
		}
		if !cfg.dedup.Mark(mapped.Envelope.IdempotencyKey) && allowDiscordDirectMessage(cfg, mapped.User, mapped.Direct) {
			p.dispatchAsyncInboundEnvelope(cfg.instanceID, mapped.Envelope)
		}
		return writeDiscordInteractionResponse(w, discordInteractionResponseTypeDeferredChannelMessage)
	case discordInteractionTypeMessageComponent:
		mapped, err := mapDiscordInteractionAction(interaction, cfg.managed, request.ReceivedAt)
		if err != nil {
			return &bridgesdk.HTTPError{StatusCode: http.StatusBadRequest, Message: err.Error()}
		}
		if !cfg.dedup.Mark(mapped.Envelope.IdempotencyKey) && allowDiscordDirectMessage(cfg, mapped.User, mapped.Direct) {
			p.dispatchAsyncInboundEnvelope(cfg.instanceID, mapped.Envelope)
		}
		return writeDiscordInteractionResponse(w, discordInteractionResponseTypeDeferredUpdateMessage)
	default:
		return &bridgesdk.HTTPError{StatusCode: http.StatusBadRequest, Message: fmt.Sprintf("unsupported discord interaction type %d", interaction.Type)}
	}
}

func (p *discordProvider) handleEventWebhook(
	w http.ResponseWriter,
	r *http.Request,
	cfg resolvedInstanceConfig,
	request bridgesdk.WebhookRequest,
) error {
	var envelope discordWebhookEventEnvelope
	if err := json.Unmarshal(request.Body, &envelope); err != nil {
		return &bridgesdk.HTTPError{StatusCode: http.StatusBadRequest, Message: "invalid discord event payload"}
	}

	switch envelope.Type {
	case discordWebhookEnvelopeTypePing:
		return writeWebhookNoContent(w)
	case discordWebhookEnvelopeTypeEvent:
	default:
		return writeWebhookNoContent(w)
	}
	if envelope.Event == nil {
		return writeWebhookNoContent(w)
	}
	ctx := context.Background()
	if r != nil && r.Context() != nil {
		ctx = r.Context()
	}

	switch strings.TrimSpace(envelope.Event.Type) {
	case "MESSAGE_CREATE":
		var event discordMessageEvent
		if err := json.Unmarshal(envelope.Event.Data, &event); err != nil {
			return &bridgesdk.HTTPError{StatusCode: http.StatusBadRequest, Message: "invalid discord message event"}
		}
		mapped, ignored, err := mapDiscordMessageEvent(event, cfg.managed, parseDiscordReceivedAt(envelope.Event.Timestamp, request.ReceivedAt), strings.TrimSpace(envelope.Event.ID))
		if err != nil {
			return &bridgesdk.HTTPError{StatusCode: http.StatusBadRequest, Message: err.Error()}
		}
		if ignored || cfg.dedup.Mark(mapped.Envelope.IdempotencyKey) || !allowDiscordDirectMessage(cfg, mapped.User, mapped.Direct) {
			return writeWebhookNoContent(w)
		}
		if cfg.batcher != nil {
			if err := cfg.batcher.Enqueue(mapped.Envelope); err != nil {
				return &bridgesdk.HTTPError{StatusCode: http.StatusInternalServerError, Message: err.Error()}
			}
			return writeWebhookNoContent(w)
		}
		if err := p.dispatchInboundEnvelope(ctx, cfg.instanceID, mapped.Envelope); err != nil {
			return &bridgesdk.HTTPError{StatusCode: http.StatusInternalServerError, Message: err.Error()}
		}
		return writeWebhookNoContent(w)
	case "MESSAGE_REACTION_ADD", "MESSAGE_REACTION_REMOVE":
		var event discordReactionEvent
		if err := json.Unmarshal(envelope.Event.Data, &event); err != nil {
			return &bridgesdk.HTTPError{StatusCode: http.StatusBadRequest, Message: "invalid discord reaction event"}
		}
		mapped, err := mapDiscordReactionEvent(event, cfg.managed, parseDiscordReceivedAt(envelope.Event.Timestamp, request.ReceivedAt), strings.TrimSpace(envelope.Event.ID), strings.TrimSpace(envelope.Event.Type))
		if err != nil {
			return &bridgesdk.HTTPError{StatusCode: http.StatusBadRequest, Message: err.Error()}
		}
		if cfg.dedup.Mark(mapped.Envelope.IdempotencyKey) {
			return writeWebhookNoContent(w)
		}
		if err := p.dispatchInboundEnvelope(ctx, cfg.instanceID, mapped.Envelope); err != nil {
			return &bridgesdk.HTTPError{StatusCode: http.StatusInternalServerError, Message: err.Error()}
		}
		return writeWebhookNoContent(w)
	default:
		return writeWebhookNoContent(w)
	}
}

func (p *discordProvider) dispatchAsyncInboundEnvelope(instanceID string, envelope bridgepkg.InboundMessageEnvelope) {
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()

		select {
		case <-p.stopCh:
			return
		default:
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := p.dispatchInboundEnvelope(ctx, instanceID, envelope); err != nil {
			p.setLastError(err)
		}
	}()
}

func (p *discordProvider) dispatchInboundBatch(ctx context.Context, bridgeInstanceID string, batch bridgesdk.InboundBatch) error {
	if len(batch.Items) == 0 {
		return nil
	}
	merged := batch.Items[0]
	if len(batch.Items) > 1 {
		parts := make([]string, 0, len(batch.Items))
		for _, item := range batch.Items {
			if text := strings.TrimSpace(item.Content.Text); text != "" {
				parts = append(parts, text)
			}
		}
		merged.Content.Text = strings.Join(parts, "\n")
		merged.IdempotencyKey = fmt.Sprintf("%s:batch:%d", merged.IdempotencyKey, len(batch.Items))
	}
	return p.dispatchInboundEnvelope(ctx, bridgeInstanceID, merged)
}

func (p *discordProvider) dispatchInboundEnvelope(ctx context.Context, bridgeInstanceID string, envelope bridgepkg.InboundMessageEnvelope) error {
	session := p.currentSession()
	if session == nil {
		return errors.New("discord: runtime session is not initialized")
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

func (p *discordProvider) configForInstance(instanceID string) (resolvedInstanceConfig, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	cfg, ok := p.routes[strings.TrimSpace(instanceID)]
	if !ok {
		return resolvedInstanceConfig{}, fmt.Errorf("discord: delivery targeted unmanaged instance %q", instanceID)
	}
	return cfg, nil
}

func (p *discordProvider) waitForInstanceConfig(instanceID string, timeout time.Duration) (resolvedInstanceConfig, error) {
	if timeout <= 0 {
		return p.configForInstance(instanceID)
	}

	deadline := time.Now().Add(timeout)
	for {
		cfg, err := p.configForInstance(instanceID)
		if err == nil {
			return cfg, nil
		}
		if time.Now().After(deadline) {
			return resolvedInstanceConfig{}, err
		}

		timer := time.NewTimer(10 * time.Millisecond)
		select {
		case <-p.stopCh:
			if !timer.Stop() {
				<-timer.C
			}
			return resolvedInstanceConfig{}, err
		case <-timer.C:
		}
	}
}

func (p *discordProvider) configForPath(path string) (resolvedInstanceConfig, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	for _, cfg := range p.routes {
		if cfg.webhookPath == normalizeWebhookPath(path) {
			return cfg, true
		}
	}
	return resolvedInstanceConfig{}, false
}

func (p *discordProvider) currentSession() *bridgesdk.Session {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.session
}

func (p *discordProvider) deliveryState(instanceID string, deliveryID string) deliveryState {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.deliveries[deliveryStateKey(instanceID, deliveryID)]
}

func (p *discordProvider) storeDeliveryState(instanceID string, deliveryID string, state deliveryState) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.deliveries[deliveryStateKey(instanceID, deliveryID)] = state
}

func (p *discordProvider) setLastError(err error) {
	if err == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.lastError = err.Error()
}

func (p *discordProvider) clearLastError() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.lastError = ""
}

func (p *discordProvider) reportSideEffectError(action string, err error) {
	reportSideEffectError(p.stderr, action, err)
}

func executeDiscordDelivery(
	ctx context.Context,
	api discordAPI,
	request bridgepkg.DeliveryRequest,
	state deliveryState,
) (bridgepkg.DeliveryAck, deliveryState, error) {
	if err := request.Validate(); err != nil {
		return bridgepkg.DeliveryAck{}, state, err
	}

	event := request.Event
	if event.EventType != bridgepkg.DeliveryEventTypeResume && event.Seq <= state.LastSeq {
		return bridgepkg.DeliveryAck{}, state, fmt.Errorf("discord: out-of-order delivery seq %d after %d", event.Seq, state.LastSeq)
	}
	if event.EventType == bridgepkg.DeliveryEventTypeResume && request.Snapshot != nil {
		state.LastSeq = request.Snapshot.LastAckedSeq
		state.RemoteMessageID = strings.TrimSpace(request.Snapshot.RemoteMessageID)
		state.ReplaceRemoteMessageID = strings.TrimSpace(request.Snapshot.ReplaceRemoteMessageID)
	}

	channelID, err := resolveDiscordDeliveryChannelID(event)
	if err != nil {
		return bridgepkg.DeliveryAck{}, state, err
	}

	switch {
	case event.Operation.Normalize() == bridgepkg.DeliveryOperationDelete || normalizeDeliveryEventType(event.EventType) == bridgepkg.DeliveryEventTypeDelete:
		remoteID := firstNonEmpty(referenceRemoteMessageID(event.Reference), state.RemoteMessageID)
		if remoteID == "" && request.Snapshot != nil {
			remoteID = strings.TrimSpace(request.Snapshot.RemoteMessageID)
		}
		if remoteID == "" {
			return bridgepkg.DeliveryAck{}, state, errors.New("discord: delete delivery requires a remote message id")
		}
		targetChannelID, messageID, err := decodeRemoteMessageID(remoteID)
		if err != nil {
			return bridgepkg.DeliveryAck{}, state, err
		}
		if err := api.DeleteMessage(ctx, discordDeleteMessageRequest{
			ChannelID: targetChannelID,
			MessageID: messageID,
		}); err != nil {
			return bridgepkg.DeliveryAck{}, state, err
		}
		ack := bridgepkg.DeliveryAck{
			DeliveryID:             event.DeliveryID,
			Seq:                    event.Seq,
			RemoteMessageID:        remoteID,
			ReplaceRemoteMessageID: firstNonEmpty(state.RemoteMessageID, remoteID),
		}
		state.LastSeq = event.Seq
		state.RemoteMessageID = remoteID
		state.ReplaceRemoteMessageID = ack.ReplaceRemoteMessageID
		return ack, state, ack.ValidateFor(event)
	case shouldPostDiscordMessage(event, state, request):
		sent, err := api.PostMessage(ctx, discordPostMessageRequest{
			ChannelID: channelID,
			Content:   event.Content.Text,
		})
		if err != nil {
			return bridgepkg.DeliveryAck{}, state, err
		}
		remoteID := encodeRemoteMessageID(channelID, sent.ID)
		ack := bridgepkg.DeliveryAck{
			DeliveryID:      event.DeliveryID,
			Seq:             event.Seq,
			RemoteMessageID: remoteID,
		}
		state.LastSeq = event.Seq
		state.ReplaceRemoteMessageID = state.RemoteMessageID
		state.RemoteMessageID = remoteID
		if state.ReplaceRemoteMessageID != "" {
			ack.ReplaceRemoteMessageID = state.ReplaceRemoteMessageID
		}
		return ack, state, ack.ValidateFor(event)
	default:
		remoteID := state.RemoteMessageID
		if remoteID == "" && request.Snapshot != nil {
			remoteID = strings.TrimSpace(request.Snapshot.RemoteMessageID)
		}
		if remoteID == "" {
			return bridgepkg.DeliveryAck{}, state, errors.New("discord: edit delivery requires a remote message id")
		}
		targetChannelID, messageID, err := decodeRemoteMessageID(remoteID)
		if err != nil {
			return bridgepkg.DeliveryAck{}, state, err
		}
		if err := api.UpdateMessage(ctx, discordUpdateMessageRequest{
			ChannelID: targetChannelID,
			MessageID: messageID,
			Content:   event.Content.Text,
		}); err != nil {
			return bridgepkg.DeliveryAck{}, state, err
		}
		ack := bridgepkg.DeliveryAck{
			DeliveryID:             event.DeliveryID,
			Seq:                    event.Seq,
			RemoteMessageID:        remoteID,
			ReplaceRemoteMessageID: firstNonEmpty(state.RemoteMessageID, remoteID),
		}
		state.LastSeq = event.Seq
		state.RemoteMessageID = remoteID
		state.ReplaceRemoteMessageID = ack.ReplaceRemoteMessageID
		return ack, state, ack.ValidateFor(event)
	}
}

func shouldPostDiscordMessage(event bridgepkg.DeliveryEvent, state deliveryState, request bridgepkg.DeliveryRequest) bool {
	if normalizeDeliveryEventType(event.EventType) == bridgepkg.DeliveryEventTypeStart {
		return true
	}
	if normalizeDeliveryEventType(event.EventType) == bridgepkg.DeliveryEventTypeResume {
		if request.Snapshot == nil {
			return state.RemoteMessageID == ""
		}
		return strings.TrimSpace(request.Snapshot.RemoteMessageID) == ""
	}
	return strings.TrimSpace(state.RemoteMessageID) == ""
}

func mapDiscordMessageEvent(
	event discordMessageEvent,
	managed subprocess.InitializeBridgeManagedInstance,
	receivedAt time.Time,
	eventID string,
) (discordMappedInbound, bool, error) {
	if strings.TrimSpace(event.ChannelID) == "" || strings.TrimSpace(event.ID) == "" {
		return discordMappedInbound{}, false, errors.New("discord: message event requires channel_id and id")
	}
	if event.Author.Bot {
		return discordMappedInbound{}, true, nil
	}

	receivedAt = parseDiscordReceivedAt(event.Timestamp, receivedAt)
	user := discordUserIdentity{
		ID:          normalizeDiscordUserID(event.Author.ID),
		Username:    normalizeUsername(event.Author.Username),
		DisplayName: firstNonEmpty(strings.TrimSpace(event.Author.GlobalName), strings.TrimSpace(event.Author.Username), normalizeDiscordUserID(event.Author.ID)),
	}
	peerID, groupID, threadID, direct, err := discordRouteIdentity(event.GuildID, event.ChannelID, event.ParentID, event.ChannelType)
	if err != nil {
		return discordMappedInbound{}, false, err
	}

	envelope := bridgepkg.InboundMessageEnvelope{
		BridgeInstanceID:  managed.Instance.ID,
		Scope:             managed.Instance.Scope,
		WorkspaceID:       managed.Instance.WorkspaceID,
		PlatformMessageID: strings.TrimSpace(event.ID),
		ReceivedAt:        receivedAt,
		Sender: bridgepkg.MessageSender{
			ID:          user.ID,
			Username:    user.Username,
			DisplayName: user.DisplayName,
		},
		Content: bridgepkg.MessageContent{
			Text: strings.TrimSpace(event.Content),
		},
		Attachments:    normalizeDiscordAttachments(event.Attachments),
		EventFamily:    bridgepkg.InboundEventFamilyMessage,
		IdempotencyKey: firstNonEmpty(strings.TrimSpace(eventID), fmt.Sprintf("discord:%s:message:%s", managed.Instance.ID, strings.TrimSpace(event.ID))),
		PeerID:         peerID,
		GroupID:        groupID,
		ThreadID:       threadID,
	}
	metadata, err := json.Marshal(map[string]any{
		"channel_id":   strings.TrimSpace(event.ChannelID),
		"channel_type": event.ChannelType,
		"event_id":     strings.TrimSpace(eventID),
		"guild_id":     strings.TrimSpace(event.GuildID),
		"message_id":   strings.TrimSpace(event.ID),
		"parent_id":    strings.TrimSpace(event.ParentID),
	})
	if err == nil {
		envelope.ProviderMetadata = metadata
	}
	if err := envelope.Validate(); err != nil {
		return discordMappedInbound{}, false, err
	}
	return discordMappedInbound{Envelope: envelope, Direct: direct, User: user}, false, nil
}

func mapDiscordInteractionCommand(
	interaction discordInteraction,
	managed subprocess.InitializeBridgeManagedInstance,
	receivedAt time.Time,
) (discordMappedInbound, error) {
	if strings.TrimSpace(interaction.ID) == "" || interaction.Data == nil {
		return discordMappedInbound{}, errors.New("discord: command interaction requires id and data")
	}

	user, err := discordUserIdentityFromInteraction(interaction)
	if err != nil {
		return discordMappedInbound{}, err
	}
	peerID, groupID, threadID, direct, err := discordRouteIdentity(
		interaction.GuildID,
		firstNonEmpty(interaction.ChannelID, channelIDFromInteraction(interaction.Channel)),
		parentIDFromInteraction(interaction.Channel),
		channelTypeFromInteraction(interaction.Channel),
	)
	if err != nil {
		return discordMappedInbound{}, err
	}
	command, text := parseDiscordCommand(interaction.Data.Name, interaction.Data.Options)
	envelope := bridgepkg.InboundMessageEnvelope{
		BridgeInstanceID: managed.Instance.ID,
		Scope:            managed.Instance.Scope,
		WorkspaceID:      managed.Instance.WorkspaceID,
		ReceivedAt:       receivedAt,
		Sender: bridgepkg.MessageSender{
			ID:          user.ID,
			Username:    user.Username,
			DisplayName: user.DisplayName,
		},
		PeerID:      peerID,
		GroupID:     groupID,
		ThreadID:    threadID,
		EventFamily: bridgepkg.InboundEventFamilyCommand,
		Command: &bridgepkg.InboundCommand{
			Command:   command,
			Text:      text,
			TriggerID: strings.TrimSpace(interaction.Token),
		},
		IdempotencyKey: strings.TrimSpace(interaction.ID),
	}
	metadata, err := json.Marshal(map[string]any{
		"application_id": strings.TrimSpace(interaction.ApplicationID),
		"channel_id":     firstNonEmpty(interaction.ChannelID, channelIDFromInteraction(interaction.Channel)),
		"channel_type":   channelTypeFromInteraction(interaction.Channel),
		"guild_id":       strings.TrimSpace(interaction.GuildID),
		"interaction_id": strings.TrimSpace(interaction.ID),
		"kind":           "application_command",
	})
	if err == nil {
		envelope.ProviderMetadata = metadata
	}
	if err := envelope.Validate(); err != nil {
		return discordMappedInbound{}, err
	}
	return discordMappedInbound{Envelope: envelope, Direct: direct, User: user}, nil
}

func mapDiscordInteractionAction(
	interaction discordInteraction,
	managed subprocess.InitializeBridgeManagedInstance,
	receivedAt time.Time,
) (discordMappedInbound, error) {
	if strings.TrimSpace(interaction.ID) == "" || interaction.Data == nil {
		return discordMappedInbound{}, errors.New("discord: action interaction requires id and data")
	}
	if strings.TrimSpace(interaction.Data.CustomID) == "" {
		return discordMappedInbound{}, errors.New("discord: action interaction requires custom_id")
	}

	user, err := discordUserIdentityFromInteraction(interaction)
	if err != nil {
		return discordMappedInbound{}, err
	}
	peerID, groupID, threadID, direct, err := discordRouteIdentity(
		interaction.GuildID,
		firstNonEmpty(interaction.ChannelID, channelIDFromInteraction(interaction.Channel)),
		parentIDFromInteraction(interaction.Channel),
		channelTypeFromInteraction(interaction.Channel),
	)
	if err != nil {
		return discordMappedInbound{}, err
	}
	value := strings.TrimSpace(interaction.Data.CustomID)
	if len(interaction.Data.Values) > 0 && strings.TrimSpace(interaction.Data.Values[0]) != "" {
		value = strings.TrimSpace(interaction.Data.Values[0])
	}
	envelope := bridgepkg.InboundMessageEnvelope{
		BridgeInstanceID: managed.Instance.ID,
		Scope:            managed.Instance.Scope,
		WorkspaceID:      managed.Instance.WorkspaceID,
		ReceivedAt:       receivedAt,
		Sender: bridgepkg.MessageSender{
			ID:          user.ID,
			Username:    user.Username,
			DisplayName: user.DisplayName,
		},
		PeerID:      peerID,
		GroupID:     groupID,
		ThreadID:    threadID,
		EventFamily: bridgepkg.InboundEventFamilyAction,
		Action: &bridgepkg.InboundAction{
			ActionID:  strings.TrimSpace(interaction.Data.CustomID),
			MessageID: messageIDFromInteraction(interaction.Message),
			Value:     value,
			TriggerID: strings.TrimSpace(interaction.Token),
		},
		IdempotencyKey: strings.TrimSpace(interaction.ID),
	}
	metadata, err := json.Marshal(map[string]any{
		"application_id": strings.TrimSpace(interaction.ApplicationID),
		"channel_id":     firstNonEmpty(interaction.ChannelID, channelIDFromInteraction(interaction.Channel)),
		"channel_type":   channelTypeFromInteraction(interaction.Channel),
		"component_type": interaction.Data.ComponentType,
		"guild_id":       strings.TrimSpace(interaction.GuildID),
		"interaction_id": strings.TrimSpace(interaction.ID),
		"kind":           "message_component",
	})
	if err == nil {
		envelope.ProviderMetadata = metadata
	}
	if err := envelope.Validate(); err != nil {
		return discordMappedInbound{}, err
	}
	return discordMappedInbound{Envelope: envelope, Direct: direct, User: user}, nil
}

func mapDiscordReactionEvent(
	event discordReactionEvent,
	managed subprocess.InitializeBridgeManagedInstance,
	receivedAt time.Time,
	eventID string,
	eventType string,
) (discordMappedInbound, error) {
	if strings.TrimSpace(event.ChannelID) == "" || strings.TrimSpace(event.MessageID) == "" || strings.TrimSpace(event.UserID) == "" {
		return discordMappedInbound{}, errors.New("discord: reaction event requires channel_id, message_id, and user_id")
	}

	receivedAt = parseDiscordReceivedAt(event.Timestamp, receivedAt)
	user := discordUserIdentity{
		ID:          normalizeDiscordUserID(event.UserID),
		Username:    normalizeUsername(discordUsernameFromMember(event.Member)),
		DisplayName: firstNonEmpty(discordGlobalNameFromMember(event.Member), discordUsernameFromMember(event.Member), normalizeDiscordUserID(event.UserID)),
	}
	peerID, groupID, threadID, direct, err := discordRouteIdentity(event.GuildID, event.ChannelID, event.ParentID, event.ChannelType)
	if err != nil {
		return discordMappedInbound{}, err
	}
	envelope := bridgepkg.InboundMessageEnvelope{
		BridgeInstanceID: managed.Instance.ID,
		Scope:            managed.Instance.Scope,
		WorkspaceID:      managed.Instance.WorkspaceID,
		ReceivedAt:       receivedAt,
		Sender: bridgepkg.MessageSender{
			ID:          user.ID,
			Username:    user.Username,
			DisplayName: user.DisplayName,
		},
		PeerID:      peerID,
		GroupID:     groupID,
		ThreadID:    threadID,
		EventFamily: bridgepkg.InboundEventFamilyReaction,
		Reaction: &bridgepkg.InboundReaction{
			MessageID: strings.TrimSpace(event.MessageID),
			Emoji:     normalizeDiscordEmoji(event.Emoji),
			RawEmoji:  rawDiscordEmoji(event.Emoji),
			Added:     strings.TrimSpace(eventType) == "MESSAGE_REACTION_ADD",
		},
		IdempotencyKey: firstNonEmpty(strings.TrimSpace(eventID), fmt.Sprintf("discord:%s:reaction:%s:%s:%s:%s", managed.Instance.ID, strings.TrimSpace(event.ChannelID), strings.TrimSpace(event.MessageID), strings.TrimSpace(event.UserID), rawDiscordEmoji(event.Emoji))),
	}
	metadata, err := json.Marshal(map[string]any{
		"channel_id":   strings.TrimSpace(event.ChannelID),
		"channel_type": event.ChannelType,
		"event_id":     strings.TrimSpace(eventID),
		"event_type":   strings.TrimSpace(eventType),
		"guild_id":     strings.TrimSpace(event.GuildID),
		"parent_id":    strings.TrimSpace(event.ParentID),
	})
	if err == nil {
		envelope.ProviderMetadata = metadata
	}
	if err := envelope.Validate(); err != nil {
		return discordMappedInbound{}, err
	}
	return discordMappedInbound{Envelope: envelope, Direct: direct, User: user}, nil
}

func allowDiscordDirectMessage(cfg resolvedInstanceConfig, user discordUserIdentity, direct bool) bool {
	if !direct {
		return true
	}

	switch cfg.dmPolicy.Normalize() {
	case "", bridgepkg.BridgeDMPolicyOpen:
		return true
	case bridgepkg.BridgeDMPolicyAllowlist:
		return discordIdentityAllowed(cfg.allowUserIDs, cfg.allowUsernames, user)
	case bridgepkg.BridgeDMPolicyPairing:
		if discordIdentityAllowed(cfg.pairedUserIDs, cfg.pairedUsernames, user) {
			return true
		}
		return discordIdentityAllowed(cfg.allowUserIDs, cfg.allowUsernames, user)
	default:
		return false
	}
}

func discordIdentityAllowed(ids map[string]struct{}, usernames map[string]struct{}, user discordUserIdentity) bool {
	if len(ids) == 0 && len(usernames) == 0 {
		return false
	}
	if _, ok := ids[normalizeDiscordUserID(user.ID)]; ok {
		return true
	}
	if _, ok := usernames[normalizeUsername(user.Username)]; ok {
		return true
	}
	return false
}

func resolveDiscordDeliveryChannelID(event bridgepkg.DeliveryEvent) (string, error) {
	channelID := firstNonEmpty(
		strings.TrimSpace(event.DeliveryTarget.PeerID),
		strings.TrimSpace(event.DeliveryTarget.ThreadID),
		strings.TrimSpace(event.DeliveryTarget.GroupID),
		strings.TrimSpace(event.RoutingKey.PeerID),
		strings.TrimSpace(event.RoutingKey.ThreadID),
		strings.TrimSpace(event.RoutingKey.GroupID),
	)
	if channelID == "" {
		return "", errors.New("discord: delivery target requires peer_id, thread_id, or group_id")
	}
	return channelID, nil
}

func verifyDiscordSignature(_ context.Context, req *http.Request, body []byte, publicKey string, now time.Time) error {
	decodedPublicKey, err := decodeDiscordPublicKey(publicKey)
	if err != nil {
		return fmt.Errorf("discord: invalid public key: %w", err)
	}
	if req == nil {
		return errors.New("discord: webhook request is required")
	}

	timestamp := strings.TrimSpace(req.Header.Get("X-Signature-Timestamp"))
	signature := strings.TrimSpace(req.Header.Get("X-Signature-Ed25519"))
	if timestamp == "" || signature == "" {
		return errors.New("discord: missing signature headers")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if tsValue, err := strconv.ParseInt(timestamp, 10, 64); err == nil {
		if delta := now.Unix() - tsValue; delta > 300 || delta < -300 {
			return errors.New("discord: stale request timestamp")
		}
	}

	decodedSignature, err := hex.DecodeString(signature)
	if err != nil {
		return errors.New("discord: invalid signature encoding")
	}
	if len(decodedSignature) != ed25519.SignatureSize {
		return errors.New("discord: invalid signature length")
	}
	message := make([]byte, 0, len(timestamp)+len(body))
	message = append(message, timestamp...)
	message = append(message, body...)
	if !ed25519.Verify(decodedPublicKey, message, decodedSignature) {
		return errors.New("discord: invalid signature")
	}
	return nil
}

func decodeDiscordPublicKey(value string) (ed25519.PublicKey, error) {
	raw, err := hex.DecodeString(strings.TrimSpace(value))
	if err != nil {
		return nil, err
	}
	if len(raw) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("expected %d-byte public key, got %d", ed25519.PublicKeySize, len(raw))
	}
	return ed25519.PublicKey(raw), nil
}

func (c *discordBotClient) GetBotUser(ctx context.Context) (*discordBotIdentity, error) {
	var result discordBotIdentity
	if err := c.callJSON(ctx, http.MethodGet, "/users/@me", nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *discordBotClient) PostMessage(ctx context.Context, req discordPostMessageRequest) (*discordPostedMessage, error) {
	var result discordPostedMessage
	if err := c.callJSON(ctx, http.MethodPost, "/channels/"+strings.TrimSpace(req.ChannelID)+"/messages", req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *discordBotClient) UpdateMessage(ctx context.Context, req discordUpdateMessageRequest) error {
	return c.callJSON(ctx, http.MethodPatch, "/channels/"+strings.TrimSpace(req.ChannelID)+"/messages/"+strings.TrimSpace(req.MessageID), req, nil)
}

func (c *discordBotClient) DeleteMessage(ctx context.Context, req discordDeleteMessageRequest) error {
	return c.callJSON(ctx, http.MethodDelete, "/channels/"+strings.TrimSpace(req.ChannelID)+"/messages/"+strings.TrimSpace(req.MessageID), nil, nil)
}

func (c *discordBotClient) callJSON(ctx context.Context, method string, path string, payload any, out any) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if c == nil {
		return errors.New("discord: api client is required")
	}

	var body io.Reader
	if payload != nil {
		buf := &bytes.Buffer{}
		encoder := json.NewEncoder(buf)
		encoder.SetEscapeHTML(false)
		if err := encoder.Encode(payload); err != nil {
			return err
		}
		body = buf
	}

	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(c.baseURL, "/")+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bot "+strings.TrimSpace(c.botToken))
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		message := strings.TrimSpace(readResponseBody(resp.Body))
		httpErr := &bridgesdk.HTTPError{
			StatusCode: resp.StatusCode,
			Message:    firstNonEmpty(message, fmt.Sprintf("discord: http %d", resp.StatusCode)),
			RetryAfter: parseRetryAfter(resp.Header.Get("Retry-After")),
		}
		return httpErr
	}
	if out == nil || resp.StatusCode == http.StatusNoContent {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("discord: decode response: %w", err)
	}
	return nil
}

func discordUserIdentityFromInteraction(interaction discordInteraction) (discordUserIdentity, error) {
	if interaction.Member != nil && interaction.Member.User != nil {
		return discordUserIdentity{
			ID:          normalizeDiscordUserID(interaction.Member.User.ID),
			Username:    normalizeUsername(interaction.Member.User.Username),
			DisplayName: firstNonEmpty(strings.TrimSpace(interaction.Member.User.GlobalName), strings.TrimSpace(interaction.Member.User.Username), normalizeDiscordUserID(interaction.Member.User.ID)),
		}, nil
	}
	if interaction.User != nil {
		return discordUserIdentity{
			ID:          normalizeDiscordUserID(interaction.User.ID),
			Username:    normalizeUsername(interaction.User.Username),
			DisplayName: firstNonEmpty(strings.TrimSpace(interaction.User.GlobalName), strings.TrimSpace(interaction.User.Username), normalizeDiscordUserID(interaction.User.ID)),
		}, nil
	}
	return discordUserIdentity{}, errors.New("discord: interaction is missing a user")
}

func discordRouteIdentity(guildID string, channelID string, parentID string, channelType int) (string, string, string, bool, error) {
	channelID = strings.TrimSpace(channelID)
	parentID = strings.TrimSpace(parentID)
	if channelID == "" {
		return "", "", "", false, errors.New("discord: channel id is required")
	}

	if isDiscordThreadChannel(channelType) {
		groupID := firstNonEmpty(parentID, channelID)
		return "", groupID, channelID, false, nil
	}
	if strings.TrimSpace(guildID) == "" {
		if channelType == discordChannelTypeGroupDM {
			return "", channelID, "", false, nil
		}
		return channelID, "", "", true, nil
	}
	return "", channelID, "", false, nil
}

func isDiscordThreadChannel(channelType int) bool {
	switch channelType {
	case discordChannelTypeAnnouncementThread, discordChannelTypePublicThread, discordChannelTypePrivateThread:
		return true
	default:
		return false
	}
}

func channelIDFromInteraction(channel *discordInteractionChannel) string {
	if channel == nil {
		return ""
	}
	return strings.TrimSpace(channel.ID)
}

func parentIDFromInteraction(channel *discordInteractionChannel) string {
	if channel == nil {
		return ""
	}
	return strings.TrimSpace(channel.ParentID)
}

func channelTypeFromInteraction(channel *discordInteractionChannel) int {
	if channel == nil {
		return 0
	}
	return channel.Type
}

func messageIDFromInteraction(message *discordInteractionMessage) string {
	if message == nil {
		return ""
	}
	return strings.TrimSpace(message.ID)
}

func parseDiscordCommand(name string, options []discordInteractionOption) (string, string) {
	commandParts := []string{strings.TrimSpace(name)}
	if commandParts[0] == "" {
		commandParts[0] = "/"
	} else if !strings.HasPrefix(commandParts[0], "/") {
		commandParts[0] = "/" + commandParts[0]
	}

	valueParts := make([]string, 0)
	var walk func([]discordInteractionOption)
	walk = func(items []discordInteractionOption) {
		for _, item := range items {
			switch item.Type {
			case discordApplicationCommandOptionTypeSubcommand, discordApplicationCommandOptionTypeSubcommandGroup:
				if trimmed := strings.TrimSpace(item.Name); trimmed != "" {
					commandParts = append(commandParts, trimmed)
				}
				walk(item.Options)
			default:
				if item.Value == nil {
					continue
				}
				text := strings.TrimSpace(fmt.Sprint(item.Value))
				if text != "" {
					valueParts = append(valueParts, text)
				}
			}
		}
	}
	walk(options)

	return strings.Join(commandParts, " "), strings.Join(valueParts, " ")
}

func normalizeDiscordAttachments(items []discordAttachment) []bridgepkg.MessageAttachment {
	attachments := make([]bridgepkg.MessageAttachment, 0, len(items))
	for _, item := range items {
		attachment := bridgepkg.MessageAttachment{
			ID:       strings.TrimSpace(item.ID),
			Name:     strings.TrimSpace(item.Filename),
			MIMEType: strings.TrimSpace(item.ContentType),
			URL:      strings.TrimSpace(item.URL),
		}
		if attachment.ID == "" && attachment.Name == "" && attachment.URL == "" {
			continue
		}
		attachments = append(attachments, attachment)
	}
	return attachments
}

func normalizeDiscordEmoji(emoji discordEmoji) string {
	if strings.TrimSpace(emoji.Name) == "" {
		return ""
	}
	if strings.TrimSpace(emoji.ID) == "" {
		return ":" + strings.Trim(strings.TrimSpace(emoji.Name), ":") + ":"
	}
	return "<:" + strings.TrimSpace(emoji.Name) + ":" + strings.TrimSpace(emoji.ID) + ">"
}

func rawDiscordEmoji(emoji discordEmoji) string {
	if strings.TrimSpace(emoji.ID) == "" {
		return strings.TrimSpace(emoji.Name)
	}
	return strings.TrimSpace(emoji.Name) + ":" + strings.TrimSpace(emoji.ID)
}

func discordUsernameFromMember(member *discordInteractionMember) string {
	if member == nil || member.User == nil {
		return ""
	}
	return strings.TrimSpace(member.User.Username)
}

func discordGlobalNameFromMember(member *discordInteractionMember) string {
	if member == nil || member.User == nil {
		return ""
	}
	return strings.TrimSpace(member.User.GlobalName)
}

func normalizeDiscordUserID(value string) string {
	return strings.TrimSpace(value)
}

func buildDiscordIDSet(values []string) map[string]struct{} {
	if len(values) == 0 {
		return nil
	}
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		if normalized := normalizeDiscordUserID(value); normalized != "" {
			set[normalized] = struct{}{}
		}
	}
	if len(set) == 0 {
		return nil
	}
	return set
}

func buildDiscordUsernameSet(values []string) map[string]struct{} {
	if len(values) == 0 {
		return nil
	}
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		if normalized := normalizeUsername(value); normalized != "" {
			set[normalized] = struct{}{}
		}
	}
	if len(set) == 0 {
		return nil
	}
	return set
}

func normalizeUsername(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func parseDiscordReceivedAt(value string, fallback time.Time) time.Time {
	if strings.TrimSpace(value) == "" {
		if fallback.IsZero() {
			return time.Now().UTC()
		}
		return fallback
	}
	if parsed, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(value)); err == nil {
		return parsed.UTC()
	}
	if parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(value)); err == nil {
		return parsed.UTC()
	}
	if ts, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64); err == nil {
		return time.Unix(ts, 0).UTC()
	}
	if fallback.IsZero() {
		return time.Now().UTC()
	}
	return fallback
}

func writeDiscordInteractionResponse(w http.ResponseWriter, responseType int) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	return json.NewEncoder(w).Encode(map[string]int{"type": responseType})
}

func writeWebhookNoContent(w http.ResponseWriter) error {
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func managedInstancesToInstances(items []subprocess.InitializeBridgeManagedInstance) []bridgepkg.BridgeInstance {
	if len(items) == 0 {
		return nil
	}
	instances := make([]bridgepkg.BridgeInstance, 0, len(items))
	for _, item := range items {
		instances = append(instances, item.Instance)
	}
	return instances
}

func deliveryStateKey(instanceID string, deliveryID string) string {
	return strings.TrimSpace(instanceID) + ":" + strings.TrimSpace(deliveryID)
}

func encodeRemoteMessageID(channelID string, messageID string) string {
	return strings.TrimSpace(channelID) + ":" + strings.TrimSpace(messageID)
}

func decodeRemoteMessageID(value string) (string, string, error) {
	parts := strings.SplitN(strings.TrimSpace(value), ":", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return "", "", fmt.Errorf("discord: invalid remote message id %q", value)
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), nil
}

func referenceRemoteMessageID(reference *bridgepkg.DeliveryMessageReference) string {
	if reference == nil {
		return ""
	}
	return strings.TrimSpace(reference.RemoteMessageID)
}

func readResponseBody(reader io.Reader) string {
	if reader == nil {
		return ""
	}
	body, err := io.ReadAll(reader)
	if err != nil {
		return ""
	}
	return string(body)
}

func parseRetryAfter(value string) time.Duration {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0
	}
	seconds, err := strconv.Atoi(trimmed)
	if err != nil {
		return 0
	}
	if seconds <= 0 {
		return 0
	}
	return time.Duration(seconds) * time.Second
}

func normalizeWebhookPath(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	if !strings.HasPrefix(trimmed, "/") {
		return "/" + trimmed
	}
	return trimmed
}

func normalizeURL(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	return strings.TrimRight(trimmed, "/")
}

func normalizeDeliveryEventType(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func isNotInitializedRPCError(err error) bool {
	var rpcErr interface{ Code() int }
	if errors.As(err, &rpcErr) {
		return rpcErr.Code() == rpcCodeNotInitialized
	}
	return false
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
