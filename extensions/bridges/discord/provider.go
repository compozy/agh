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
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	bridgepkg "github.com/compozy/agh/internal/bridges"
	"github.com/compozy/agh/internal/bridgesdk"
	"github.com/compozy/agh/internal/subprocess"
)

const (
	providerMessageCreateValue      = "MESSAGE_CREATE"
	providerMessageReactionAddValue = "MESSAGE_REACTION_ADD"
	providerChannelIDKey            = "channel_id"
	providerChannelTypeKey          = "channel_type"
	providerDiscordKey              = "discord"
	providerGuildIDKey              = "guild_id"
	providerMessageIDKey            = "message_id"
	providerParentIDKey             = "parent_id"
	providerTypeKey                 = "type"
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
)

type discordProvider struct {
	sdk        *bridgesdk.Runtime
	lifecycle  *bridgesdk.ProviderLifecycle
	markers    *bridgesdk.AdapterMarkers
	http       *bridgesdk.ProviderHTTPServer
	stderr     io.Writer
	now        func() time.Time
	routes     *bridgesdk.RouteTable[resolvedInstanceConfig]
	deliveries *bridgesdk.DeliveryStateStore[deliveryState]

	mu                        sync.RWMutex
	lastError                 string
	listenAddr                string
	apiFactory                func(resolvedInstanceConfig) discordAPI
	progressDispatcherOptions []bridgesdk.ProgressDispatcherOption
}

type discordProviderConfig struct {
	ApplicationID string `json:"application_id,omitempty"`
	Webhook       struct {
		ListenAddr string `json:"listen_addr,omitempty"`
		Path       string `json:"path,omitempty"`
	} `json:"webhook"`
	Batching struct {
		DelayMS        int `json:"delay_ms,omitempty"`
		SplitDelayMS   int `json:"split_delay_ms,omitempty"`
		SplitThreshold int `json:"split_threshold,omitempty"`
	} `json:"batching"`
	DM struct {
		AllowUserIDs    []string `json:"allow_user_ids,omitempty"`
		AllowUsernames  []string `json:"allow_usernames,omitempty"`
		PairedUserIDs   []string `json:"paired_user_ids,omitempty"`
		PairedUsernames []string `json:"paired_usernames,omitempty"`
	} `json:"dm"`
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

//nolint:funlen // Construction keeps the provider's declarative runtime wiring visible in one place.
func newDiscordProvider(stderr io.Writer) (*discordProvider, error) {
	if stderr == nil {
		stderr = io.Discard
	}

	provider := &discordProvider{
		stderr:  stderr,
		markers: bridgesdk.NewAdapterMarkers(providerDiscordKey, stderr),
		now:     func() time.Time { return time.Now().UTC() },
		routes: bridgesdk.NewRouteTable(func(config resolvedInstanceConfig) []string {
			if config.configError != nil {
				return nil
			}
			return []string{config.webhookPath}
		}),
		deliveries: bridgesdk.NewDeliveryStateStore[deliveryState](),
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

	lifecycle, err := bridgesdk.NewProviderLifecycle(bridgesdk.ProviderLifecycleConfig{
		ProviderName: providerDiscordKey,
		Markers:      provider.markers,
		Reconcile: func(
			ctx context.Context,
			managed []subprocess.InitializeBridgeManagedInstance,
		) ([]bridgesdk.ProviderInitialState, error) {
			configs := provider.reconcileInstanceConfigs(ctx, provider.lifecycle.Session(), managed)
			states := make([]bridgesdk.ProviderInitialState, 0, len(configs))
			for idx := range configs {
				states = append(states, bridgesdk.ProviderInitialState{
					BridgeInstanceID: configs[idx].instanceID,
					Status:           configs[idx].initialStatus,
					Degradation:      configs[idx].initialDegradation,
				})
			}
			return states, nil
		},
		FinalizeInitialize: func(err error) {
			if err != nil {
				provider.setLastError(err)
			}
		},
		OnStop: provider.stopResources,
		ShutdownResources: func(ctx context.Context) error {
			if provider.http == nil {
				return nil
			}
			return provider.http.Shutdown(ctx)
		},
	})
	if err != nil {
		return nil, err
	}
	provider.lifecycle = lifecycle
	providerHTTP, err := bridgesdk.NewProviderHTTPServer(bridgesdk.ProviderHTTPConfig{
		ReadHeaderTimeout: discordWebhookReadHeaderTimeout,
		IdleTimeout:       discordWebhookIdleTimeout,
		Handler:           http.HandlerFunc(provider.serveWebhookHTTP),
		Go:                lifecycle.Go,
		OnError:           provider.setLastError,
	})
	if err != nil {
		return nil, err
	}
	provider.http = providerHTTP

	sdkRuntime, err := bridgesdk.NewRuntime(bridgesdk.RuntimeConfig{
		ExtensionInfo: subprocess.InitializeExtensionInfo{
			Name:    providerDiscordKey,
			Version: "0.1.0",
			SDKName: "bridgesdk",
		},
		Initialize:  lifecycle.Initialize,
		Deliver:     provider.handleBridgesDeliver,
		Progress:    provider.handleBridgesProgress,
		Check:       provider.handleBridgeCheck,
		HealthCheck: func(context.Context, *bridgesdk.Session) error { return provider.healthCheck() },
		Shutdown:    lifecycle.Shutdown,
		Now:         func() time.Time { return provider.now() },
	})
	if err != nil {
		return nil, err
	}
	provider.sdk = sdkRuntime
	return provider, nil
}

func (p *discordProvider) serve(stdin io.Reader, stdout io.Writer) error {
	return p.lifecycle.Serve(context.Background(), p.sdk, stdin, stdout)
}

func (p *discordProvider) handleInitialize(_ context.Context, session *bridgesdk.Session) error {
	return p.lifecycle.Initialize(context.Background(), session)
}

func (p *discordProvider) handleBridgesDeliver(
	ctx context.Context,
	session *bridgesdk.Session,
	request bridgepkg.DeliveryRequest,
) (bridgepkg.DeliveryAck, error) {
	marker := bridgesdk.DeliveryMarker{
		PID:     os.Getpid(),
		Request: request,
	}

	cfg, err := p.waitForInstanceConfig(strings.TrimSpace(request.Event.BridgeInstanceID), 500*time.Millisecond)
	if err != nil {
		marker.Error = err.Error()
		p.markers.RecordDelivery(marker)
		p.setLastError(err)
		return bridgepkg.DeliveryAck{}, err
	}

	if p.markers.ShouldCrashOnce() {
		p.markers.RecordDelivery(marker)
		p.markers.RecordCrash(map[string]any{
			"crashed":            true,
			"pid":                os.Getpid(),
			"delivery_id":        strings.TrimSpace(request.Event.DeliveryID),
			"bridge_instance_id": cfg.instanceID,
		})
		os.Exit(23)
	}

	state := p.deliveryState(cfg.instanceID, request.Event.DeliveryID)
	progress := state.Progress
	if progress != nil {
		if err := progress.Flush(ctx); err != nil {
			marker.Error = err.Error()
			p.markers.RecordDelivery(marker)
			p.reportDiscordDeliveryError(ctx, session, cfg.instanceID, err)
			return bridgepkg.DeliveryAck{}, err
		}
	}

	api := p.apiFactory(cfg)
	ack, state, err := executeDiscordDelivery(
		ctx,
		api,
		request,
		state,
	)
	if err != nil {
		marker.Error = err.Error()
		p.markers.RecordDelivery(marker)
		p.reportDiscordDeliveryError(ctx, session, cfg.instanceID, err)
		return bridgepkg.DeliveryAck{}, err
	}

	p.storeDeliveryState(cfg.instanceID, request.Event.DeliveryID, state)
	if err := p.lifecycle.Host().ReportReadyIfNeeded(ctx, session, cfg.instanceID); err != nil {
		p.setLastError(err)
	}
	var cleanupErr error
	if progress != nil {
		cleanupErr = progress.OnContent(ctx)
		if request.Event.Final {
			detached := p.detachDiscordProgressDispatcher(
				cfg.instanceID,
				request.Event.DeliveryID,
				progress,
			)
			if detached != nil {
				detached.Close()
			}
		}
	}

	marker.Ack = &ack
	p.markers.RecordDelivery(marker)
	if cleanupErr != nil {
		p.reportDiscordProgressError("clean up progress after text delivery", cleanupErr)
	} else {
		p.clearLastError()
	}
	return ack, nil
}

func (p *discordProvider) stopResources() {
	progressToClose := p.takeDiscordProgressDispatchers()
	batchersToClose := make(map[*bridgesdk.InboundBatcher]struct{})
	routes := p.routes.Snapshot()
	for instanceID := range routes {
		cfg := routes[instanceID]
		if cfg.batcher == nil {
			continue
		}
		batchersToClose[cfg.batcher] = struct{}{}
		p.routes.Update(instanceID, func(current resolvedInstanceConfig) resolvedInstanceConfig {
			current.batcher = nil
			return current
		})
	}
	closeDiscordInboundBatchers(batchersToClose)
	closeDiscordProgressDispatchers(progressToClose)
}

func (p *discordProvider) reconcileInstanceConfigs(
	ctx context.Context,
	session *bridgesdk.Session,
	managed []subprocess.InitializeBridgeManagedInstance,
) []resolvedInstanceConfig {
	batchersToClose := make(map[*bridgesdk.InboundBatcher]struct{})
	reconciler := bridgesdk.ManagedConfigReconciler[resolvedInstanceConfig]{
		Routes:   p.routes,
		Resolve:  p.resolveInstanceConfig,
		Prepare:  p.prepareDiscordConfigs,
		Finalize: p.finalizeDiscordConfigs,
		Identity: func(config resolvedInstanceConfig) string { return config.instanceID },
		Merge: func(prior resolvedInstanceConfig, next resolvedInstanceConfig) resolvedInstanceConfig {
			if prior.batcher != nil && prior.batcher != next.batcher {
				batchersToClose[prior.batcher] = struct{}{}
			}
			return next
		},
		OnRemoved: func(config resolvedInstanceConfig) error {
			if config.batcher != nil {
				batchersToClose[config.batcher] = struct{}{}
			}
			return nil
		},
		OnPublish: func() { closeDiscordInboundBatchers(batchersToClose) },
	}
	configs, err := reconciler.Reconcile(ctx, session, managed)
	if err != nil {
		p.setLastError(err)
		return nil
	}
	return configs
}

func (p *discordProvider) finalizeDiscordConfigs(
	ctx context.Context,
	_ *bridgesdk.Session,
	configs []resolvedInstanceConfig,
) ([]resolvedInstanceConfig, error) {
	for idx := range configs {
		status, degradation, probeErr := p.determineInitialState(ctx, &configs[idx])
		if probeErr != nil {
			p.setLastError(probeErr)
		}
		configs[idx].initialStatus = status
		configs[idx].initialDegradation = degradation
	}
	return configs, nil
}

func (p *discordProvider) prepareDiscordConfigs(
	_ context.Context,
	_ *bridgesdk.Session,
	configs []resolvedInstanceConfig,
) ([]resolvedInstanceConfig, error) {
	if len(configs) == 0 {
		return configs, nil
	}
	requestedListen := strings.TrimSpace(os.Getenv(discordListenAddrEnv))
	usedPaths := make(map[string]string, len(configs))

	for idx := range configs {
		requestedListen = applyDiscordListenConstraint(&configs[idx], requestedListen)
		applyDiscordWebhookPathConflict(&configs[idx], usedPaths)
	}
	p.applyDiscordListenErrors(configs, requestedListen)
	p.mu.Lock()
	p.listenAddr = requestedListen
	p.mu.Unlock()
	return configs, nil
}

func applyDiscordListenConstraint(cfg *resolvedInstanceConfig, requestedListen string) string {
	if cfg == nil || cfg.listenAddr == "" {
		return requestedListen
	}
	if requestedListen == "" {
		return cfg.listenAddr
	}
	if requestedListen != cfg.listenAddr && cfg.configError == nil {
		cfg.configError = fmt.Errorf(
			"discord: instance %q configured incompatible listen_addr %q (runtime uses %q)",
			cfg.instanceID,
			cfg.listenAddr,
			requestedListen,
		)
	}
	return requestedListen
}

func applyDiscordWebhookPathConflict(cfg *resolvedInstanceConfig, usedPaths map[string]string) {
	if cfg == nil || cfg.webhookPath == "" {
		return
	}
	if owner, ok := usedPaths[cfg.webhookPath]; ok && cfg.configError == nil {
		cfg.configError = fmt.Errorf(
			"discord: webhook path %q is shared by %q and %q",
			cfg.webhookPath,
			owner,
			cfg.instanceID,
		)
	}
	usedPaths[cfg.webhookPath] = cfg.instanceID
}

func (p *discordProvider) applyDiscordListenErrors(configs []resolvedInstanceConfig, requestedListen string) {
	if requestedListen == "" {
		for idx := range configs {
			if configs[idx].configError == nil {
				configs[idx].configError = errors.New("discord: webhook listen address is required")
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
	webhookPath := normalizeWebhookPath(
		firstNonEmpty(cfg.Webhook.Path, "/discord/"+strings.TrimSpace(managed.Instance.ID)),
	)
	apiBaseURL := normalizeURL(firstNonEmpty(
		strings.TrimSpace(os.Getenv(discordAPIBaseEnv)),
		discordDefaultAPIBaseURL,
	))

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
	cfg *resolvedInstanceConfig,
) (bridgepkg.BridgeStatus, *bridgepkg.BridgeDegradation, error) {
	if cfg == nil {
		return bridgepkg.BridgeStatusError, nil, errors.New("discord: config is required")
	}
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
	bot, err := p.apiFactory(*cfg).GetBotUser(ctx)
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
	if err := p.http.Start(listenAddr); err != nil {
		return fmt.Errorf("discord: %w", err)
	}
	p.markers.RecordListen(p.http.Address())
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
		return p.handleWebhookRequest(w, r, &cfg, request)
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
	cfg *resolvedInstanceConfig,
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
	cfg *resolvedInstanceConfig,
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
		if !cfg.dedup.Mark(mapped.Envelope.IdempotencyKey) &&
			allowDiscordDirectMessage(cfg, mapped.User, mapped.Direct) {
			p.dispatchAsyncInboundEnvelope(cfg.instanceID, mapped.Envelope)
		}
		return writeDiscordInteractionResponse(w, discordInteractionResponseTypeDeferredChannelMessage)
	case discordInteractionTypeMessageComponent:
		mapped, err := mapDiscordInteractionAction(interaction, cfg.managed, request.ReceivedAt)
		if err != nil {
			return &bridgesdk.HTTPError{StatusCode: http.StatusBadRequest, Message: err.Error()}
		}
		if !cfg.dedup.Mark(mapped.Envelope.IdempotencyKey) &&
			allowDiscordDirectMessage(cfg, mapped.User, mapped.Direct) {
			p.dispatchAsyncInboundEnvelope(cfg.instanceID, mapped.Envelope)
		}
		return writeDiscordInteractionResponse(w, discordInteractionResponseTypeDeferredUpdateMessage)
	default:
		return &bridgesdk.HTTPError{
			StatusCode: http.StatusBadRequest,
			Message:    fmt.Sprintf("unsupported discord interaction type %d", interaction.Type),
		}
	}
}

func (p *discordProvider) handleEventWebhook(
	w http.ResponseWriter,
	r *http.Request,
	cfg *resolvedInstanceConfig,
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
	case providerMessageCreateValue:
		return p.handleDiscordMessageWebhookEvent(ctx, w, cfg, request, envelope)
	case providerMessageReactionAddValue, "MESSAGE_REACTION_REMOVE":
		return p.handleDiscordReactionWebhookEvent(ctx, w, cfg, request, envelope)
	default:
		return writeWebhookNoContent(w)
	}
}

func (p *discordProvider) handleDiscordMessageWebhookEvent(
	ctx context.Context,
	w http.ResponseWriter,
	cfg *resolvedInstanceConfig,
	request bridgesdk.WebhookRequest,
	envelope discordWebhookEventEnvelope,
) error {
	var event discordMessageEvent
	if err := json.Unmarshal(envelope.Event.Data, &event); err != nil {
		return &bridgesdk.HTTPError{StatusCode: http.StatusBadRequest, Message: "invalid discord message event"}
	}
	mapped, ignored, err := mapDiscordMessageEvent(event, cfg.managed, request.ReceivedAt, envelope.Event.ID)
	if err != nil {
		return &bridgesdk.HTTPError{StatusCode: http.StatusBadRequest, Message: err.Error()}
	}
	if ignored {
		return writeWebhookNoContent(w)
	}
	return p.dispatchDiscordWebhookEnvelope(ctx, w, cfg, mapped, true)
}

func (p *discordProvider) handleDiscordReactionWebhookEvent(
	ctx context.Context,
	w http.ResponseWriter,
	cfg *resolvedInstanceConfig,
	request bridgesdk.WebhookRequest,
	envelope discordWebhookEventEnvelope,
) error {
	var event discordReactionEvent
	if err := json.Unmarshal(envelope.Event.Data, &event); err != nil {
		return &bridgesdk.HTTPError{StatusCode: http.StatusBadRequest, Message: "invalid discord reaction event"}
	}
	mapped, err := mapDiscordReactionEvent(
		event,
		cfg.managed,
		request.ReceivedAt,
		envelope.Event.ID,
		envelope.Event.Type,
	)
	if err != nil {
		return &bridgesdk.HTTPError{StatusCode: http.StatusBadRequest, Message: err.Error()}
	}
	return p.dispatchDiscordWebhookEnvelope(ctx, w, cfg, mapped, false)
}

func (p *discordProvider) dispatchDiscordWebhookEnvelope(
	ctx context.Context,
	w http.ResponseWriter,
	cfg *resolvedInstanceConfig,
	mapped discordMappedInbound,
	allowBatch bool,
) error {
	if cfg == nil {
		return &bridgesdk.HTTPError{StatusCode: http.StatusInternalServerError, Message: "discord: config is required"}
	}
	if cfg.dedup.Mark(mapped.Envelope.IdempotencyKey) {
		return writeWebhookNoContent(w)
	}
	if !allowDiscordDirectMessage(cfg, mapped.User, mapped.Direct) {
		return writeWebhookNoContent(w)
	}
	if allowBatch && cfg.batcher != nil {
		if err := cfg.batcher.Enqueue(mapped.Envelope); err != nil {
			return &bridgesdk.HTTPError{StatusCode: http.StatusInternalServerError, Message: err.Error()}
		}
		return writeWebhookNoContent(w)
	}
	if err := p.dispatchInboundEnvelope(ctx, cfg.instanceID, mapped.Envelope); err != nil {
		return &bridgesdk.HTTPError{StatusCode: http.StatusInternalServerError, Message: err.Error()}
	}
	return writeWebhookNoContent(w)
}

func (p *discordProvider) dispatchAsyncInboundEnvelope(instanceID string, envelope bridgepkg.InboundMessageEnvelope) {
	p.lifecycle.Go(func() {
		select {
		case <-p.lifecycle.StopChannel():
			return
		default:
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := p.dispatchInboundEnvelope(ctx, instanceID, envelope); err != nil {
			p.setLastError(err)
		}
	})
}

func (p *discordProvider) dispatchInboundBatch(
	ctx context.Context,
	bridgeInstanceID string,
	batch bridgesdk.InboundBatch,
) error {
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

func (p *discordProvider) dispatchInboundEnvelope(
	ctx context.Context,
	bridgeInstanceID string,
	envelope bridgepkg.InboundMessageEnvelope,
) error {
	session := p.currentSession()
	if session == nil {
		return errors.New("discord: runtime session is not initialized")
	}
	cfg, err := p.configForInstance(bridgeInstanceID)
	if err != nil {
		return err
	}

	_, err = p.lifecycle.Host().IngestBridgeMessage(ctx, session, envelope)
	if err != nil {
		return err
	}
	if err := p.lifecycle.Host().ReportReadyIfNeeded(ctx, session, cfg.instanceID); err != nil {
		p.setLastError(err)
	}
	return nil
}

func (p *discordProvider) configForInstance(instanceID string) (resolvedInstanceConfig, error) {
	cfg, ok := p.routes.Get(instanceID)
	if !ok {
		return resolvedInstanceConfig{}, fmt.Errorf("discord: delivery targeted unmanaged instance %q", instanceID)
	}
	return cfg, nil
}

func (p *discordProvider) waitForInstanceConfig(
	instanceID string,
	timeout time.Duration,
) (resolvedInstanceConfig, error) {
	if timeout <= 0 {
		return p.configForInstance(instanceID)
	}

	cfg, ok, err := p.routes.Wait(context.Background(), instanceID, timeout, p.lifecycle.StopChannel())
	if err != nil {
		return resolvedInstanceConfig{}, err
	}
	if !ok {
		return p.configForInstance(instanceID)
	}
	return cfg, nil
}

func (p *discordProvider) configForPath(path string) (resolvedInstanceConfig, bool) {
	configs := p.routes.ByPath(normalizeWebhookPath(path))
	if len(configs) == 0 {
		return resolvedInstanceConfig{}, false
	}
	return configs[0], true
}

func (p *discordProvider) currentSession() *bridgesdk.Session {
	return p.lifecycle.Session()
}

func closeDiscordInboundBatchers(batchers map[*bridgesdk.InboundBatcher]struct{}) {
	for batcher := range batchers {
		batcher.Close()
	}
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
