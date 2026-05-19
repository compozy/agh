package main

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	bridgepkg "github.com/pedronauck/agh/internal/bridges"
	"github.com/pedronauck/agh/internal/bridgesdk"
	extensioncontract "github.com/pedronauck/agh/internal/extension/contract"
	"github.com/pedronauck/agh/internal/subprocess"
)

const (
	providerAudioKey    = "audio"
	providerImageKey    = "image"
	providerMessagesKey = "messages"
	providerStickerKey  = "sticker"
	providerTextKey     = "text"
	providerWhatsappKey = "whatsapp"
)

const (
	whatsappListenAddrEnv = "AGH_BRIDGE_WHATSAPP_LISTEN_ADDR"
	whatsappAPIBaseEnv    = "AGH_BRIDGE_WHATSAPP_API_BASE_URL"

	whatsappDefaultAPIBaseURL        = "https://graph.facebook.com"
	whatsappDefaultAPIVersion        = "v21.0"
	whatsappMessageLimit             = 4096
	whatsappWebhookReadHeaderTimeout = 10 * time.Second
	whatsappWebhookIdleTimeout       = 30 * time.Second

	whatsappSignatureHeader = "X-Hub-Signature-256"

	rpcCodeNotInitialized = -32003
)

var (
	errWhatsAppInstanceConfigUnavailable = errors.New(
		"whatsapp: delivery targeted unmanaged instance",
	)
	errWhatsAppInstanceConfigInvalid = errors.New(
		"whatsapp: bridge instance configuration invalid",
	)
)

type whatsappProvider struct {
	sdk     *bridgesdk.Runtime
	stderr  io.Writer
	env     markerEnv
	now     func() time.Time
	session *bridgesdk.Session

	mu             sync.RWMutex
	lastError      string
	lastErrorSeq   uint64
	instanceErrors map[string]string
	server         *http.Server
	serverAddr     string
	listenAddr     string
	routes         map[string]resolvedInstanceConfig
	deliveries     map[string]deliveryState
	reportedStatus map[string]bridgepkg.BridgeStatus
	apiFactory     func(resolvedInstanceConfig) whatsappAPI

	stopCh   chan struct{}
	stopOnce sync.Once
	wg       sync.WaitGroup
}

type deliveryState struct {
	LastSeq                int64
	RemoteMessageID        string
	ReplaceRemoteMessageID string
}

type whatsappProviderConfig struct {
	APIBaseURL    string `json:"api_base_url,omitempty"`
	APIVersion    string `json:"api_version,omitempty"`
	PhoneNumberID string `json:"phone_number_id,omitempty"`
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
	managed            *subprocess.InitializeBridgeManagedInstance
	instanceID         string
	listenAddr         string
	webhookPath        string
	apiBaseURL         string
	apiVersion         string
	phoneNumberID      string
	accessToken        string
	appSecret          string
	verifyToken        string
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

type whatsappWebhookPayload struct {
	Object string                 `json:"object,omitempty"`
	Entry  []whatsappWebhookEntry `json:"entry,omitempty"`
}

type whatsappWebhookEntry struct {
	ID      string                  `json:"id,omitempty"`
	Changes []whatsappWebhookChange `json:"changes,omitempty"`
}

type whatsappWebhookChange struct {
	Field string               `json:"field,omitempty"`
	Value whatsappWebhookValue `json:"value"`
}

type whatsappWebhookValue struct {
	MessagingProduct string                   `json:"messaging_product,omitempty"`
	Metadata         whatsappMetadata         `json:"metadata"`
	Contacts         []whatsappContact        `json:"contacts,omitempty"`
	Messages         []whatsappInboundMessage `json:"messages,omitempty"`
	Statuses         []whatsappDeliveryStatus `json:"statuses,omitempty"`
}

type whatsappMetadata struct {
	DisplayPhoneNumber string `json:"display_phone_number,omitempty"`
	PhoneNumberID      string `json:"phone_number_id,omitempty"`
}

type whatsappContact struct {
	Profile struct {
		Name string `json:"name,omitempty"`
	} `json:"profile"`
	WaID string `json:"wa_id,omitempty"`
}

type whatsappInboundMessage struct {
	ID        string `json:"id,omitempty"`
	From      string `json:"from,omitempty"`
	Timestamp string `json:"timestamp,omitempty"`
	Type      string `json:"type,omitempty"`
	Context   *struct {
		From string `json:"from,omitempty"`
		ID   string `json:"id,omitempty"`
	} `json:"context,omitempty"`
	Text *struct {
		Body string `json:"body,omitempty"`
	} `json:"text,omitempty"`
	Image *struct {
		ID       string `json:"id,omitempty"`
		MIMEType string `json:"mime_type,omitempty"`
		Caption  string `json:"caption,omitempty"`
		SHA256   string `json:"sha256,omitempty"`
	} `json:"image,omitempty"`
	Document *struct {
		ID       string `json:"id,omitempty"`
		MIMEType string `json:"mime_type,omitempty"`
		Caption  string `json:"caption,omitempty"`
		Filename string `json:"filename,omitempty"`
		SHA256   string `json:"sha256,omitempty"`
	} `json:"document,omitempty"`
	Audio *struct {
		ID       string `json:"id,omitempty"`
		MIMEType string `json:"mime_type,omitempty"`
		Voice    bool   `json:"voice,omitempty"`
		SHA256   string `json:"sha256,omitempty"`
	} `json:"audio,omitempty"`
	Video *struct {
		ID       string `json:"id,omitempty"`
		MIMEType string `json:"mime_type,omitempty"`
		Caption  string `json:"caption,omitempty"`
		SHA256   string `json:"sha256,omitempty"`
	} `json:"video,omitempty"`
	Sticker *struct {
		ID       string `json:"id,omitempty"`
		MIMEType string `json:"mime_type,omitempty"`
		Animated bool   `json:"animated,omitempty"`
		SHA256   string `json:"sha256,omitempty"`
	} `json:"sticker,omitempty"`
	Location *struct {
		Latitude  float64 `json:"latitude,omitempty"`
		Longitude float64 `json:"longitude,omitempty"`
		Name      string  `json:"name,omitempty"`
		Address   string  `json:"address,omitempty"`
		URL       string  `json:"url,omitempty"`
	} `json:"location,omitempty"`
}

type whatsappDeliveryStatus struct {
	ID          string `json:"id,omitempty"`
	RecipientID string `json:"recipient_id,omitempty"`
	Status      string `json:"status,omitempty"`
	Timestamp   string `json:"timestamp,omitempty"`
}

type whatsappPhoneNumber struct {
	ID string `json:"id,omitempty"`
}

type whatsappGraphAPIErrorEnvelope struct {
	Error whatsappGraphAPIError `json:"error"`
}

type whatsappGraphAPIError struct {
	Message      string `json:"message,omitempty"`
	Type         string `json:"type,omitempty"`
	Code         int    `json:"code,omitempty"`
	ErrorSubcode int    `json:"error_subcode,omitempty"`
}

type whatsappVerifyChallenge string

type whatsappSendMessageRequest struct {
	MessagingProduct string `json:"messaging_product"`
	RecipientType    string `json:"recipient_type,omitempty"`
	To               string `json:"to"`
	Type             string `json:"type"`
	Text             struct {
		Body       string `json:"body"`
		PreviewURL bool   `json:"preview_url"`
	} `json:"text"`
}

type whatsappSendMessageResponse struct {
	Messages []struct {
		ID string `json:"id,omitempty"`
	} `json:"messages,omitempty"`
}

type whatsappAPI interface {
	GetPhoneNumber(context.Context, string) (*whatsappPhoneNumber, error)
	SendTextMessage(
		context.Context,
		string,
		whatsappSendMessageRequest,
	) (*whatsappSendMessageResponse, error)
}

type whatsappGraphClient struct {
	baseURL     string
	apiVersion  string
	accessToken string
	httpClient  *http.Client
}

func newWhatsAppProvider(stderr io.Writer) (*whatsappProvider, error) {
	if stderr == nil {
		stderr = io.Discard
	}

	provider := &whatsappProvider{
		stderr:         stderr,
		env:            markerEnvFromProcess(),
		now:            func() time.Time { return time.Now().UTC() },
		routes:         make(map[string]resolvedInstanceConfig),
		deliveries:     make(map[string]deliveryState),
		reportedStatus: make(map[string]bridgepkg.BridgeStatus),
		instanceErrors: make(map[string]string),
		stopCh:         make(chan struct{}),
	}
	provider.apiFactory = func(cfg resolvedInstanceConfig) whatsappAPI {
		return &whatsappGraphClient{
			baseURL:     cfg.apiBaseURL,
			apiVersion:  cfg.apiVersion,
			accessToken: cfg.accessToken,
			httpClient: &http.Client{
				Timeout: 10 * time.Second,
			},
		}
	}

	sdkRuntime, err := bridgesdk.NewRuntime(bridgesdk.RuntimeConfig{
		ExtensionInfo: subprocess.InitializeExtensionInfo{
			Name:    providerWhatsappKey,
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

func (p *whatsappProvider) serve(stdin io.Reader, stdout io.Writer) error {
	p.reportSideEffectError(
		"write start marker",
		appendMarkerLine(p.env.startsPath, fmt.Sprintf("pid=%d", os.Getpid())),
	)
	return p.sdk.Serve(context.Background(), stdin, stdout)
}

func (p *whatsappProvider) handleInitialize(_ context.Context, session *bridgesdk.Session) error {
	p.mu.Lock()
	p.session = session
	p.mu.Unlock()

	marker := initializeMarker{
		Request:  session.InitializeRequest(),
		Response: session.InitializeResponse(),
	}
	p.reportSideEffectError("write initialize marker", writeJSONFile(p.env.handshakePath, marker))
	p.clearLastError()
	globalErrorSeq := p.currentGlobalErrorSeq()

	p.wg.Go(func() {
		p.afterInitialize(session, globalErrorSeq)
	})

	return nil
}

func (p *whatsappProvider) afterInitialize(session *bridgesdk.Session, globalErrorSeq uint64) {
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
	for _, cfg := range configs {
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
		p.clearGlobalErrorIfUnchanged(globalErrorSeq)
	}
}

func (p *whatsappProvider) handleBridgesDeliver(
	ctx context.Context,
	session *bridgesdk.Session,
	request bridgepkg.DeliveryRequest,
) (bridgepkg.DeliveryAck, error) {
	marker := deliveryMarker{
		PID:     os.Getpid(),
		Request: request,
	}

	cfg, err := p.waitForInstanceConfig(
		strings.TrimSpace(request.Event.BridgeInstanceID),
		500*time.Millisecond,
	)
	if err != nil {
		marker.Error = err.Error()
		p.reportSideEffectError(
			"write failed delivery marker",
			appendJSONLine(p.env.deliveryPath, marker),
		)
		p.setInstanceError(request.Event.BridgeInstanceID, err)
		return bridgepkg.DeliveryAck{}, err
	}

	if shouldCrashOnce(p.env.crashOncePath) {
		p.reportSideEffectError(
			"write pre-crash delivery marker",
			appendJSONLine(p.env.deliveryPath, marker),
		)
		p.reportSideEffectError(
			"write crash marker",
			writeJSONFile(p.env.crashOncePath, map[string]any{
				"crashed":            true,
				"pid":                os.Getpid(),
				"delivery_id":        strings.TrimSpace(request.Event.DeliveryID),
				"bridge_instance_id": cfg.instanceID,
			}),
		)
		os.Exit(23)
	}

	api := p.apiFactory(cfg)
	ack, state, err := executeWhatsAppDelivery(
		ctx,
		api,
		cfg,
		request,
		p.deliveryState(cfg.instanceID, request.Event.DeliveryID),
	)
	if err != nil {
		marker.Error = err.Error()
		p.reportSideEffectError(
			"write failed delivery marker",
			appendJSONLine(p.env.deliveryPath, marker),
		)
		classified := bridgesdk.ClassifyError(err)
		_, _, reportErr := session.ReportClassifiedError(ctx, cfg.instanceID, classified)
		if reportErr != nil {
			p.setInstanceError(cfg.instanceID, reportErr)
		} else {
			p.setInstanceError(cfg.instanceID, err)
		}
		return bridgepkg.DeliveryAck{}, err
	}

	p.storeDeliveryState(cfg.instanceID, request.Event.DeliveryID, state)
	if err := p.reportReadyIfNeeded(ctx, session, cfg.instanceID); err != nil {
		p.setInstanceError(cfg.instanceID, err)
	} else {
		p.clearInstanceError(cfg.instanceID)
	}

	marker.Ack = &ack
	p.reportSideEffectError("write delivery marker", appendJSONLine(p.env.deliveryPath, marker))
	return ack, nil
}

func (p *whatsappProvider) healthCheck() error {
	p.mu.RLock()
	globalError := strings.TrimSpace(p.lastError)
	instanceErrors := make(map[string]string, len(p.instanceErrors))
	for instanceID, message := range p.instanceErrors {
		if strings.TrimSpace(message) == "" {
			continue
		}
		instanceErrors[instanceID] = strings.TrimSpace(message)
	}
	p.mu.RUnlock()
	if globalError != "" {
		return errors.New(globalError)
	}
	if len(instanceErrors) == 0 {
		return nil
	}
	instanceIDs := make([]string, 0, len(instanceErrors))
	for instanceID := range instanceErrors {
		instanceIDs = append(instanceIDs, instanceID)
	}
	sort.Strings(instanceIDs)
	return fmt.Errorf("%s: %s", instanceIDs[0], instanceErrors[instanceIDs[0]])
}

func (p *whatsappProvider) handleShutdown(
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
	var shutdownErr error
	if server != nil {
		if err := server.Shutdown(shutdownCtx); err != nil &&
			!errors.Is(err, http.ErrServerClosed) {
			shutdownErr = fmt.Errorf("whatsapp: shutdown webhook server: %w", err)
			p.setLastError(shutdownErr)
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
	return shutdownErr
}

func (p *whatsappProvider) stop() {
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

func (p *whatsappProvider) syncOwnedInstances(
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

func (p *whatsappProvider) getOwnedInstance(
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

func (p *whatsappProvider) reportState(
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
		p.reportSideEffectError(
			"write failed state marker",
			appendJSONLine(p.env.statePath, stateMarker{
				BridgeInstanceID: strings.TrimSpace(bridgeInstanceID),
				Status:           status,
				Error:            err.Error(),
			}),
		)
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

func (p *whatsappProvider) reportReadyIfNeeded(
	ctx context.Context,
	session *bridgesdk.Session,
	bridgeInstanceID string,
) error {
	bridgeInstanceID = strings.TrimSpace(bridgeInstanceID)
	p.mu.RLock()
	status := p.reportedStatus[bridgeInstanceID]
	p.mu.RUnlock()
	if status == bridgepkg.BridgeStatusReady {
		return nil
	}
	return p.reportState(ctx, session, bridgeInstanceID, bridgepkg.BridgeStatusReady, nil)
}

func (p *whatsappProvider) ingestBridgeMessage(
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

func (p *whatsappProvider) retryHostCall(
	ctx context.Context,
	fn func(context.Context) error,
) error {
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

func (p *whatsappProvider) reconcileInstanceConfigs(
	ctx context.Context,
	session *bridgesdk.Session,
	managed []subprocess.InitializeBridgeManagedInstance,
) []resolvedInstanceConfig {
	if len(managed) == 0 {
		p.mu.Lock()
		p.routes = make(map[string]resolvedInstanceConfig)
		p.mu.Unlock()
		return nil
	}

	configs, requestedListen := p.resolveWhatsAppManagedConfigs(session, managed)
	applyWhatsAppListenErrors(configs, requestedListen, p.startServer)
	p.swapWhatsAppRoutes(configs, requestedListen)
	p.populateWhatsAppInitialStates(ctx, configs)
	return configs
}

func (p *whatsappProvider) resolveInstanceConfig(
	session *bridgesdk.Session,
	managed subprocess.InitializeBridgeManagedInstance,
) resolvedInstanceConfig {
	cfg, err := decodeWhatsAppProviderConfig(managed)
	if err != nil {
		return resolvedInstanceConfig{
			managed:    &managed,
			instanceID: managed.Instance.ID,
			configError: fmt.Errorf(
				"whatsapp: decode provider_config for %q: %w",
				managed.Instance.ID,
				err,
			),
		}
	}

	resolved := buildWhatsAppResolvedInstance(session, managed, cfg)
	validateWhatsAppResolvedConfig(&resolved)
	if resolved.configError != nil {
		return resolved
	}
	configureWhatsAppBatcher(p, cfg, &resolved)
	return resolved
}

func (p *whatsappProvider) determineInitialState(
	ctx context.Context,
	cfg resolvedInstanceConfig,
) (bridgepkg.BridgeStatus, *bridgepkg.BridgeDegradation, error) {
	if cfg.configError != nil {
		return bridgepkg.BridgeStatusDegraded, &bridgepkg.BridgeDegradation{
			Reason:  bridgepkg.BridgeDegradationReasonTenantConfigInvalid,
			Message: cfg.configError.Error(),
		}, cfg.configError
	}
	if strings.TrimSpace(cfg.accessToken) == "" {
		err := errors.New("whatsapp: access_token secret binding is required")
		return bridgepkg.BridgeStatusAuthRequired, &bridgepkg.BridgeDegradation{
			Reason:  bridgepkg.BridgeDegradationReasonAuthFailed,
			Message: err.Error(),
		}, err
	}
	if strings.TrimSpace(cfg.appSecret) == "" {
		err := errors.New("whatsapp: app_secret secret binding is required")
		return bridgepkg.BridgeStatusAuthRequired, &bridgepkg.BridgeDegradation{
			Reason:  bridgepkg.BridgeDegradationReasonAuthFailed,
			Message: err.Error(),
		}, err
	}
	if strings.TrimSpace(cfg.verifyToken) == "" {
		err := errors.New("whatsapp: verify_token secret binding is required")
		return bridgepkg.BridgeStatusAuthRequired, &bridgepkg.BridgeDegradation{
			Reason:  bridgepkg.BridgeDegradationReasonAuthFailed,
			Message: err.Error(),
		}, err
	}

	_, err := p.apiFactory(cfg).GetPhoneNumber(ctx, cfg.phoneNumberID)
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
	return bridgepkg.BridgeStatusReady, nil, nil
}

func (p *whatsappProvider) startServer(listenAddr string) error {
	p.mu.RLock()
	server := p.server
	currentListen := p.listenAddr
	p.mu.RUnlock()
	if server != nil {
		if currentListen != "" && currentListen != strings.TrimSpace(listenAddr) {
			return fmt.Errorf(
				"whatsapp: runtime already listening on %q, cannot switch to %q",
				currentListen,
				listenAddr,
			)
		}
		return nil
	}

	ln, err := listenWhatsAppWebhook(strings.TrimSpace(listenAddr))
	if err != nil {
		return fmt.Errorf("whatsapp: listen %q: %w", listenAddr, err)
	}

	httpServer := &http.Server{
		Handler:           http.HandlerFunc(p.serveWebhookHTTP),
		ReadHeaderTimeout: whatsappWebhookReadHeaderTimeout,
		IdleTimeout:       whatsappWebhookIdleTimeout,
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
		if serveErr := httpServer.Serve(ln); serveErr != nil &&
			!errors.Is(serveErr, http.ErrServerClosed) {
			p.setLastError(serveErr)
		}
	})

	return nil
}

func (p *whatsappProvider) serveWebhookHTTP(w http.ResponseWriter, r *http.Request) {
	cfg, ok := p.configForPath(r.URL.Path)
	if !ok {
		http.NotFound(w, r)
		return
	}

	if r.Method == http.MethodGet {
		p.handleVerifyChallenge(w, r, cfg)
		return
	}

	handler, err := bridgesdk.NewWebhookHandler(bridgesdk.WebhookGuardConfig{
		AllowedMethods:      []string{http.MethodPost},
		AllowedContentTypes: []string{"application/json"},
		MaxBodyBytes:        1 << 20,
		RateLimiter:         cfg.rateLimiter,
		InFlightLimiter:     cfg.inFlightLimiter,
		VerifySignature: func(ctx context.Context, req *http.Request, body []byte) error {
			return verifyWhatsAppSignature(ctx, req, body, cfg.appSecret)
		},
		RequestKey: func(req *http.Request) string {
			return req.RemoteAddr + "|" + cfg.instanceID
		},
		Now: func() time.Time { return p.now() },
	}, func(w http.ResponseWriter, r *http.Request, request bridgesdk.WebhookRequest) error {
		return p.handleWebhookRequest(w, r, cfg, request)
	})
	if err != nil {
		http.Error(
			w,
			http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError,
		)
		p.setInstanceError(cfg.instanceID, err)
		return
	}
	handler.ServeHTTP(w, r)
}

func (p *whatsappProvider) handleVerifyChallenge(
	w http.ResponseWriter,
	r *http.Request,
	cfg resolvedInstanceConfig,
) {
	mode := strings.TrimSpace(r.URL.Query().Get("hub.mode"))
	token := strings.TrimSpace(r.URL.Query().Get("hub.verify_token"))
	challenge, err := parseWhatsAppVerifyChallenge(r.URL.Query().Get("hub.challenge"))
	if mode == "subscribe" && token == strings.TrimSpace(cfg.verifyToken) {
		if err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			p.setInstanceError(cfg.instanceID, err)
			return
		}
		if err := writeWhatsAppVerifyChallenge(w, challenge); err != nil {
			p.setInstanceError(cfg.instanceID, err)
		}
		return
	}
	http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
}

func (p *whatsappProvider) handleWebhookRequest(
	w http.ResponseWriter,
	_ *http.Request,
	cfg resolvedInstanceConfig,
	request bridgesdk.WebhookRequest,
) error {
	payload := whatsappWebhookPayload{}
	if err := json.Unmarshal(request.Body, &payload); err != nil {
		return &bridgesdk.HTTPError{
			StatusCode: http.StatusBadRequest,
			Message:    "invalid whatsapp webhook payload",
		}
	}

	var dispatchErr error
	for _, entry := range payload.Entry {
		for _, change := range entry.Changes {
			if strings.TrimSpace(change.Field) != providerMessagesKey {
				continue
			}
			if !matchesPhoneNumberID(cfg, change.Value.Metadata.PhoneNumberID) {
				continue
			}
			contacts := contactsByWaID(change.Value.Contacts)
			for _, message := range change.Value.Messages {
				envelope, err := mapWhatsAppInboundMessage(
					message,
					contacts[strings.TrimSpace(message.From)],
					cfg.managed,
					request.ReceivedAt,
					cfg.phoneNumberID,
				)
				if err != nil {
					return &bridgesdk.HTTPError{
						StatusCode: http.StatusBadRequest,
						Message:    err.Error(),
					}
				}
				if cfg.dedup.Mark(envelope.IdempotencyKey) {
					continue
				}
				if !allowWhatsAppDirectMessage(cfg, envelope.Sender) {
					continue
				}
				if cfg.batcher != nil {
					if err := cfg.batcher.Enqueue(envelope); err != nil {
						return &bridgesdk.HTTPError{
							StatusCode: http.StatusInternalServerError,
							Message:    err.Error(),
						}
					}
					continue
				}
				if err := p.dispatchInboundEnvelope(context.Background(), cfg.instanceID, envelope); err != nil {
					dispatchErr = err
					break
				}
			}
			if dispatchErr != nil {
				break
			}
		}
		if dispatchErr != nil {
			break
		}
	}
	if dispatchErr != nil {
		return &bridgesdk.HTTPError{
			StatusCode: http.StatusInternalServerError,
			Message:    dispatchErr.Error(),
		}
	}

	return writeWhatsAppWebhookOK(w)
}

func (p *whatsappProvider) dispatchInboundBatch(
	ctx context.Context,
	bridgeInstanceID string,
	batch bridgesdk.InboundBatch,
) error {
	if len(batch.Items) == 0 {
		return nil
	}
	if !canMergeWhatsAppInboundBatch(batch.Items) {
		for _, item := range batch.Items {
			if err := p.dispatchInboundEnvelope(ctx, bridgeInstanceID, item); err != nil {
				return err
			}
		}
		return nil
	}
	merged := batch.Items[0]
	if len(batch.Items) > 1 {
		parts := make([]string, 0, len(batch.Items))
		for _, item := range batch.Items {
			if strings.TrimSpace(item.Content.Text) != "" {
				parts = append(parts, item.Content.Text)
			}
		}
		merged.Content.Text = strings.Join(parts, "\n")
		merged.IdempotencyKey = fmt.Sprintf("%s:batch:%d", merged.IdempotencyKey, len(batch.Items))
	}
	return p.dispatchInboundEnvelope(ctx, bridgeInstanceID, merged)
}

func (p *whatsappProvider) dispatchInboundEnvelope(
	ctx context.Context,
	bridgeInstanceID string,
	envelope bridgepkg.InboundMessageEnvelope,
) error {
	session := p.currentSession()
	if session == nil {
		return errors.New("whatsapp: runtime session is not initialized")
	}
	cfg, err := p.configForInstance(bridgeInstanceID)
	if err != nil {
		p.setInstanceError(bridgeInstanceID, err)
		return err
	}

	result, err := p.ingestBridgeMessage(ctx, session, envelope)
	if err != nil {
		p.setInstanceError(cfg.instanceID, err)
		p.reportSideEffectError(
			"write failed ingest marker",
			appendJSONLine(p.env.ingestPath, ingestMarker{
				Envelope: envelope,
				Error:    err.Error(),
			}),
		)
		return err
	}
	p.reportSideEffectError("write ingest marker", appendJSONLine(p.env.ingestPath, ingestMarker{
		Envelope: envelope,
		Result:   *result,
	}))
	if err := p.reportReadyIfNeeded(ctx, session, cfg.instanceID); err != nil {
		p.setInstanceError(cfg.instanceID, err)
	} else {
		p.clearInstanceError(cfg.instanceID)
	}
	return nil
}

func canMergeWhatsAppInboundBatch(items []bridgepkg.InboundMessageEnvelope) bool {
	for _, item := range items {
		if len(item.Attachments) > 0 ||
			len(bytes.TrimSpace(item.ProviderMetadata)) > 0 ||
			item.Command != nil ||
			item.Action != nil ||
			item.Reaction != nil ||
			item.Conversation != nil {
			return false
		}
		family := item.EventFamily.Normalize()
		if family != "" && family != bridgepkg.InboundEventFamilyMessage {
			return false
		}
	}
	return true
}

func (p *whatsappProvider) configForInstance(instanceID string) (resolvedInstanceConfig, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	trimmedInstanceID := strings.TrimSpace(instanceID)
	cfg, ok := p.routes[trimmedInstanceID]
	if !ok {
		return resolvedInstanceConfig{}, fmt.Errorf(
			"%w: %q",
			errWhatsAppInstanceConfigUnavailable,
			trimmedInstanceID,
		)
	}
	if cfg.configError != nil {
		return resolvedInstanceConfig{}, fmt.Errorf(
			"%w: %q: %w",
			errWhatsAppInstanceConfigInvalid,
			trimmedInstanceID,
			cfg.configError,
		)
	}
	return cfg, nil
}

func (p *whatsappProvider) waitForInstanceConfig(
	instanceID string,
	timeout time.Duration,
) (resolvedInstanceConfig, error) {
	if timeout <= 0 {
		return p.configForInstance(instanceID)
	}

	deadline := time.Now().Add(timeout)
	for {
		cfg, err := p.configForInstance(instanceID)
		if err == nil {
			return cfg, nil
		}
		if !errors.Is(err, errWhatsAppInstanceConfigUnavailable) {
			return resolvedInstanceConfig{}, err
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

func (p *whatsappProvider) configForPath(path string) (resolvedInstanceConfig, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	normalizedPath := normalizeWebhookPath(path)
	matched := false
	selected := resolvedInstanceConfig{}
	for _, cfg := range p.routes {
		if cfg.webhookPath != normalizedPath {
			continue
		}
		if cfg.configError != nil || matched {
			return resolvedInstanceConfig{}, false
		}
		selected = cfg
		matched = true
	}
	return selected, matched
}

func (p *whatsappProvider) currentSession() *bridgesdk.Session {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.session
}

func (p *whatsappProvider) deliveryState(instanceID string, deliveryID string) deliveryState {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.deliveries[deliveryStateKey(instanceID, deliveryID)]
}

func (p *whatsappProvider) storeDeliveryState(
	instanceID string,
	deliveryID string,
	state deliveryState,
) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.deliveries[deliveryStateKey(instanceID, deliveryID)] = state
}

func (p *whatsappProvider) setLastError(err error) {
	p.setInstanceError("", err)
}

func (p *whatsappProvider) setInstanceError(instanceID string, err error) {
	if err == nil {
		return
	}
	instanceID = strings.TrimSpace(instanceID)
	p.mu.Lock()
	defer p.mu.Unlock()
	if instanceID != "" {
		if p.instanceErrors == nil {
			p.instanceErrors = make(map[string]string)
		}
		p.instanceErrors[instanceID] = err.Error()
		return
	}
	p.lastError = err.Error()
	p.lastErrorSeq++
}

func (p *whatsappProvider) clearLastError() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.lastError = ""
	p.lastErrorSeq++
	p.instanceErrors = make(map[string]string)
}

func (p *whatsappProvider) clearGlobalError() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.lastError = ""
	p.lastErrorSeq++
}

func (p *whatsappProvider) currentGlobalErrorSeq() uint64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.lastErrorSeq
}

func (p *whatsappProvider) clearGlobalErrorIfUnchanged(seq uint64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.lastErrorSeq != seq {
		return
	}
	p.lastError = ""
	p.lastErrorSeq++
}

func (p *whatsappProvider) clearInstanceError(instanceID string) {
	instanceID = strings.TrimSpace(instanceID)
	if instanceID == "" {
		p.clearGlobalError()
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.instanceErrors, instanceID)
}

func (p *whatsappProvider) reportSideEffectError(action string, err error) {
	reportSideEffectError(p.stderr, action, err)
}

func executeWhatsAppDelivery(
	ctx context.Context,
	api whatsappAPI,
	cfg resolvedInstanceConfig,
	request bridgepkg.DeliveryRequest,
	state deliveryState,
) (bridgepkg.DeliveryAck, deliveryState, error) {
	if err := request.Validate(); err != nil {
		return bridgepkg.DeliveryAck{}, state, err
	}

	event := request.Event
	if event.EventType != bridgepkg.DeliveryEventTypeResume && event.Seq <= state.LastSeq {
		return bridgepkg.DeliveryAck{}, state, fmt.Errorf(
			"whatsapp: out-of-order delivery seq %d after %d",
			event.Seq,
			state.LastSeq,
		)
	}
	if event.EventType == bridgepkg.DeliveryEventTypeResume && request.Snapshot != nil {
		state.LastSeq = request.Snapshot.LastAckedSeq
		state.RemoteMessageID = strings.TrimSpace(request.Snapshot.RemoteMessageID)
		state.ReplaceRemoteMessageID = strings.TrimSpace(request.Snapshot.ReplaceRemoteMessageID)
	}

	if isWhatsAppDeleteDelivery(event) {
		return bridgepkg.DeliveryAck{}, state, &bridgesdk.PermanentError{
			Err: errors.New("whatsapp: delete delivery is not supported by the Cloud API"),
		}
	}

	targetUserID, err := resolveDeliveryTarget(event)
	if err != nil {
		return bridgepkg.DeliveryAck{}, state, err
	}
	text := event.Content.Text
	if strings.TrimSpace(text) == "" {
		return bridgepkg.DeliveryAck{}, state, &bridgesdk.PermanentError{
			Err: errors.New("whatsapp: text delivery content is required"),
		}
	}

	// Resume requests with an already-acked remote message do not need to post
	// again; return the snapshot identity so the broker can continue.
	if ack, resumed, ok := resumeWhatsAppDelivery(event, request.Snapshot, state); ok {
		return ack, resumed, ack.ValidateFor(event)
	}

	remoteID, err := sendWhatsAppDeliveryChunks(ctx, api, cfg.phoneNumberID, targetUserID, text)
	if err != nil {
		return bridgepkg.DeliveryAck{}, state, err
	}
	replaceRemoteID := firstNonEmpty(
		state.RemoteMessageID,
		func() string {
			if request.Snapshot == nil {
				return ""
			}
			return strings.TrimSpace(request.Snapshot.RemoteMessageID)
		}(),
	)
	ack := bridgepkg.DeliveryAck{
		DeliveryID:      event.DeliveryID,
		Seq:             event.Seq,
		RemoteMessageID: remoteID,
	}
	if event.Seq > 1 || replaceRemoteID != "" {
		ack.ReplaceRemoteMessageID = replaceRemoteID
	}
	state.LastSeq = event.Seq
	state.ReplaceRemoteMessageID = replaceRemoteID
	state.RemoteMessageID = remoteID
	return ack, state, ack.ValidateFor(event)
}

func allowWhatsAppDirectMessage(cfg resolvedInstanceConfig, sender bridgepkg.MessageSender) bool {
	switch cfg.dmPolicy.Normalize() {
	case "", bridgepkg.BridgeDMPolicyOpen:
		return true
	case bridgepkg.BridgeDMPolicyAllowlist:
		return identityAllowed(cfg.allowUserIDs, cfg.allowUsernames, sender)
	case bridgepkg.BridgeDMPolicyPairing:
		if identityAllowed(cfg.pairedUserIDs, cfg.pairedUsernames, sender) {
			return true
		}
		return identityAllowed(cfg.allowUserIDs, cfg.allowUsernames, sender)
	default:
		return false
	}
}

func identityAllowed(
	ids map[string]struct{},
	usernames map[string]struct{},
	sender bridgepkg.MessageSender,
) bool {
	if len(ids) == 0 && len(usernames) == 0 {
		return false
	}
	if _, ok := ids[strings.TrimSpace(sender.ID)]; ok {
		return true
	}
	if _, ok := usernames[normalizeUsername(firstNonEmpty(sender.Username, sender.DisplayName))]; ok {
		return true
	}
	return false
}

func mapWhatsAppInboundMessage(
	message whatsappInboundMessage,
	contact *whatsappContact,
	managed *subprocess.InitializeBridgeManagedInstance,
	receivedAt time.Time,
	phoneNumberID string,
) (bridgepkg.InboundMessageEnvelope, error) {
	if managed == nil {
		return bridgepkg.InboundMessageEnvelope{}, errors.New(
			"whatsapp: managed bridge instance is required",
		)
	}
	if strings.TrimSpace(message.ID) == "" {
		return bridgepkg.InboundMessageEnvelope{}, errors.New(
			"whatsapp: inbound message id is required",
		)
	}
	if strings.TrimSpace(message.From) == "" {
		return bridgepkg.InboundMessageEnvelope{}, errors.New(
			"whatsapp: inbound sender id is required",
		)
	}
	text := extractWhatsAppTextContent(message)
	if text == "" {
		return bridgepkg.InboundMessageEnvelope{}, errors.New(
			"whatsapp: unsupported inbound message type",
		)
	}
	if receivedAt.IsZero() {
		receivedAt = time.Now().UTC()
	}
	if parsed := parseUnixTimestamp(message.Timestamp); !parsed.IsZero() {
		receivedAt = parsed
	}

	senderID, senderName, username := resolveWhatsAppSender(message, contact)

	envelope := bridgepkg.InboundMessageEnvelope{
		BridgeInstanceID:  managed.Instance.ID,
		Scope:             managed.Instance.Scope,
		WorkspaceID:       managed.Instance.WorkspaceID,
		PlatformMessageID: strings.TrimSpace(message.ID),
		ReceivedAt:        receivedAt,
		PeerID:            senderID,
		Sender: bridgepkg.MessageSender{
			ID:          senderID,
			Username:    username,
			DisplayName: senderName,
		},
		Content: bridgepkg.MessageContent{
			Text: text,
		},
		Attachments: normalizeWhatsAppAttachments(message),
		EventFamily: bridgepkg.InboundEventFamilyMessage,
		IdempotencyKey: fmt.Sprintf(
			"whatsapp:%s:%s",
			managed.Instance.ID,
			strings.TrimSpace(message.ID),
		),
	}

	metadata, err := buildWhatsAppInboundMetadata(message, phoneNumberID)
	if err == nil {
		envelope.ProviderMetadata = metadata
	}

	return envelope, envelope.Validate()
}

func resolveWhatsAppSender(
	message whatsappInboundMessage,
	contact *whatsappContact,
) (string, string, string) {
	senderID := strings.TrimSpace(message.From)
	senderName := senderID
	username := ""
	if contact != nil && strings.TrimSpace(contact.Profile.Name) != "" {
		senderName = strings.TrimSpace(contact.Profile.Name)
		username = normalizeUsername(contact.Profile.Name)
	}
	return senderID, senderName, username
}

func buildWhatsAppInboundMetadata(
	message whatsappInboundMessage,
	phoneNumberID string,
) ([]byte, error) {
	contextFrom, contextID := extractWhatsAppContext(message.Context)
	return json.Marshal(map[string]any{
		"context_from":    contextFrom,
		"context_id":      contextID,
		"message_id":      strings.TrimSpace(message.ID),
		"phone_number_id": strings.TrimSpace(phoneNumberID),
		"sender_wa_id":    strings.TrimSpace(message.From),
		"timestamp":       strings.TrimSpace(message.Timestamp),
		"type":            strings.TrimSpace(message.Type),
	})
}

func extractWhatsAppContext(contextRef *struct {
	From string `json:"from,omitempty"`
	ID   string `json:"id,omitempty"`
}) (string, string) {
	if contextRef == nil {
		return "", ""
	}
	return strings.TrimSpace(contextRef.From), strings.TrimSpace(contextRef.ID)
}

func normalizeWhatsAppAttachments(message whatsappInboundMessage) []bridgepkg.MessageAttachment {
	attachments := make([]bridgepkg.MessageAttachment, 0, 4)
	appendAttachment := func(id string, name string, mimeType string, url string) {
		attachment := bridgepkg.MessageAttachment{
			ID:       strings.TrimSpace(id),
			Name:     strings.TrimSpace(name),
			MIMEType: strings.TrimSpace(mimeType),
			URL:      strings.TrimSpace(url),
		}
		if attachment.ID == "" && attachment.Name == "" && attachment.URL == "" {
			return
		}
		attachments = append(attachments, attachment)
	}

	if message.Image != nil {
		appendAttachment(message.Image.ID, providerImageKey, message.Image.MIMEType, "")
	}
	if message.Document != nil {
		appendAttachment(
			message.Document.ID,
			firstNonEmpty(message.Document.Filename, "document"),
			message.Document.MIMEType,
			"",
		)
	}
	if message.Audio != nil {
		appendAttachment(message.Audio.ID, providerAudioKey, message.Audio.MIMEType, "")
	}
	if message.Video != nil {
		appendAttachment(message.Video.ID, "video", message.Video.MIMEType, "")
	}
	if message.Sticker != nil {
		appendAttachment(message.Sticker.ID, providerStickerKey, message.Sticker.MIMEType, "")
	}
	if message.Location != nil {
		name := firstNonEmpty(message.Location.Name, "location")
		url := strings.TrimSpace(message.Location.URL)
		if url == "" {
			url = fmt.Sprintf(
				"https://www.google.com/maps?q=%v,%v",
				message.Location.Latitude,
				message.Location.Longitude,
			)
		}
		appendAttachment("", name, "application/geo+json", url)
	}

	if len(attachments) == 0 {
		return nil
	}
	return attachments
}

func extractWhatsAppTextContent(message whatsappInboundMessage) string {
	switch strings.TrimSpace(message.Type) {
	case providerTextKey:
		if message.Text == nil {
			return ""
		}
		return strings.TrimSpace(message.Text.Body)
	case providerImageKey:
		if message.Image == nil {
			return ""
		}
		return firstNonEmpty(strings.TrimSpace(message.Image.Caption), "[Image]")
	case "document":
		if message.Document == nil {
			return ""
		}
		if caption := strings.TrimSpace(message.Document.Caption); caption != "" {
			return caption
		}
		return fmt.Sprintf("[Document: %s]", firstNonEmpty(message.Document.Filename, "file"))
	case providerAudioKey:
		return "[Audio message]"
	case "video":
		if message.Video == nil {
			return "[Video]"
		}
		return firstNonEmpty(strings.TrimSpace(message.Video.Caption), "[Video]")
	case providerStickerKey:
		return "[Sticker]"
	case "location":
		if message.Location == nil {
			return "[Location]"
		}
		parts := []string{
			firstNonEmpty(
				message.Location.Name,
				fmt.Sprintf("%v,%v", message.Location.Latitude, message.Location.Longitude),
			),
		}
		if address := strings.TrimSpace(message.Location.Address); address != "" {
			parts = append(parts, address)
		}
		return "[Location: " + strings.Join(parts, " - ") + "]"
	default:
		return ""
	}
}

func resolveDeliveryTarget(event bridgepkg.DeliveryEvent) (string, error) {
	userID := firstNonEmpty(
		strings.TrimSpace(event.DeliveryTarget.PeerID),
		strings.TrimSpace(event.RoutingKey.PeerID),
	)
	if userID == "" {
		return "", errors.New("whatsapp: delivery target requires peer_id")
	}
	return userID, nil
}

func verifyWhatsAppSignature(
	_ context.Context,
	req *http.Request,
	body []byte,
	appSecret string,
) error {
	secret := strings.TrimSpace(appSecret)
	if secret == "" {
		return errors.New("whatsapp: app secret is required")
	}
	if req == nil {
		return errors.New("whatsapp: webhook request is required")
	}

	signature := strings.TrimSpace(req.Header.Get(whatsappSignatureHeader))
	if signature == "" {
		return errors.New("whatsapp: missing webhook signature")
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(signature), []byte(expected)) {
		return errors.New("whatsapp: invalid webhook signature")
	}
	return nil
}

func writeWhatsAppWebhookOK(w http.ResponseWriter) error {
	if w == nil {
		return nil
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, err := io.WriteString(w, "ok")
	return err
}

func writeWhatsAppVerifyChallenge(w http.ResponseWriter, challenge whatsappVerifyChallenge) error {
	if w == nil {
		return nil
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, err := io.WriteString(w, template.HTMLEscapeString(string(challenge)))
	return err
}

func parseWhatsAppVerifyChallenge(value string) (whatsappVerifyChallenge, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", errors.New("whatsapp: hub.challenge is required")
	}
	if len(trimmed) > 256 {
		return "", errors.New("whatsapp: hub.challenge exceeds 256 characters")
	}
	for _, r := range trimmed {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '.', r == '-', r == '_':
		default:
			return "", errors.New("whatsapp: hub.challenge contains invalid characters")
		}
	}
	return whatsappVerifyChallenge(trimmed), nil
}

func listenWhatsAppWebhook(listenAddr string) (net.Listener, error) {
	var listenConfig net.ListenConfig
	return listenConfig.Listen(context.Background(), "tcp", strings.TrimSpace(listenAddr))
}

func (p *whatsappProvider) resolveWhatsAppManagedConfigs(
	session *bridgesdk.Session,
	managed []subprocess.InitializeBridgeManagedInstance,
) ([]resolvedInstanceConfig, string) {
	configs := make([]resolvedInstanceConfig, 0, len(managed))
	requestedListen := strings.TrimSpace(os.Getenv(whatsappListenAddrEnv))
	usedPaths := make(map[string]string, len(managed))

	for _, item := range managed {
		cfg := p.resolveInstanceConfig(session, item)
		requestedListen = updateWhatsAppRequestedListen(&cfg, requestedListen)
		markDuplicateWhatsAppWebhookPath(&cfg, usedPaths)
		configs = append(configs, cfg)
	}
	return configs, requestedListen
}

func updateWhatsAppRequestedListen(cfg *resolvedInstanceConfig, requestedListen string) string {
	if cfg == nil || cfg.listenAddr == "" {
		return requestedListen
	}
	if requestedListen == "" {
		return cfg.listenAddr
	}
	if requestedListen != cfg.listenAddr && cfg.configError == nil {
		cfg.configError = fmt.Errorf(
			"whatsapp: instance %q configured incompatible listen_addr %q (runtime uses %q)",
			cfg.instanceID,
			cfg.listenAddr,
			requestedListen,
		)
	}
	return requestedListen
}

func markDuplicateWhatsAppWebhookPath(cfg *resolvedInstanceConfig, usedPaths map[string]string) {
	if cfg == nil || cfg.webhookPath == "" {
		return
	}
	if owner, ok := usedPaths[cfg.webhookPath]; ok && cfg.configError == nil {
		cfg.configError = fmt.Errorf(
			"whatsapp: webhook path %q is shared by %q and %q",
			cfg.webhookPath,
			owner,
			cfg.instanceID,
		)
	}
	usedPaths[cfg.webhookPath] = cfg.instanceID
}

func applyWhatsAppListenErrors(
	configs []resolvedInstanceConfig,
	requestedListen string,
	startServer func(string) error,
) {
	if requestedListen == "" {
		for idx := range configs {
			if configs[idx].configError == nil {
				configs[idx].configError = errors.New(
					"whatsapp: webhook listen address is required",
				)
			}
		}
		return
	}

	if err := startServer(requestedListen); err != nil {
		for idx := range configs {
			if configs[idx].configError == nil {
				configs[idx].configError = err
			}
		}
	}
}

func (p *whatsappProvider) swapWhatsAppRoutes(
	configs []resolvedInstanceConfig,
	requestedListen string,
) {
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
}

func (p *whatsappProvider) populateWhatsAppInitialStates(
	ctx context.Context,
	configs []resolvedInstanceConfig,
) {
	for idx := range configs {
		status, degradation, err := p.determineInitialState(ctx, configs[idx])
		if err != nil {
			p.setInstanceError(configs[idx].instanceID, err)
		} else {
			p.clearInstanceError(configs[idx].instanceID)
		}
		configs[idx].initialStatus = status
		configs[idx].initialDegradation = degradation
	}
}

func decodeWhatsAppProviderConfig(
	managed subprocess.InitializeBridgeManagedInstance,
) (whatsappProviderConfig, error) {
	cfg := whatsappProviderConfig{}
	if len(managed.Instance.ProviderConfig) == 0 {
		return cfg, nil
	}
	if err := json.Unmarshal(managed.Instance.ProviderConfig, &cfg); err != nil {
		return whatsappProviderConfig{}, err
	}
	return cfg, nil
}

func buildWhatsAppResolvedInstance(
	session *bridgesdk.Session,
	managed subprocess.InitializeBridgeManagedInstance,
	cfg whatsappProviderConfig,
) resolvedInstanceConfig {
	accessToken, _ := session.Cache().BoundSecretValue(managed.Instance.ID, "access_token")
	appSecret, _ := session.Cache().BoundSecretValue(managed.Instance.ID, "app_secret")
	verifyToken, _ := session.Cache().BoundSecretValue(managed.Instance.ID, "verify_token")

	resolved := resolvedInstanceConfig{
		managed:    &managed,
		instanceID: strings.TrimSpace(managed.Instance.ID),
		listenAddr: firstNonEmpty(
			cfg.Webhook.ListenAddr,
			strings.TrimSpace(os.Getenv(whatsappListenAddrEnv)),
		),
		webhookPath: normalizeWebhookPath(
			firstNonEmpty(cfg.Webhook.Path, "/whatsapp/"+strings.TrimSpace(managed.Instance.ID)),
		),
		apiBaseURL: normalizeURL(
			firstNonEmpty(
				cfg.APIBaseURL,
				strings.TrimSpace(os.Getenv(whatsappAPIBaseEnv)),
				whatsappDefaultAPIBaseURL,
			),
		),
		apiVersion:      firstNonEmpty(cfg.APIVersion, whatsappDefaultAPIVersion),
		phoneNumberID:   strings.TrimSpace(cfg.PhoneNumberID),
		accessToken:     strings.TrimSpace(accessToken),
		appSecret:       strings.TrimSpace(appSecret),
		verifyToken:     strings.TrimSpace(verifyToken),
		dmPolicy:        managed.Instance.DMPolicy.Normalize(),
		allowUserIDs:    buildIdentitySet(cfg.DM.AllowUserIDs),
		allowUsernames:  buildIdentitySet(cfg.DM.AllowUsernames),
		pairedUserIDs:   buildIdentitySet(cfg.DM.PairedUserIDs),
		pairedUsernames: buildIdentitySet(cfg.DM.PairedUsernames),
		dedup:           bridgesdk.NewDedupCache(5*time.Minute, 4000),
		rateLimiter:     bridgesdk.NewFixedWindowRateLimiter(120, time.Minute),
		inFlightLimiter: bridgesdk.NewInFlightLimiter(24),
	}
	if resolved.dmPolicy == "" {
		resolved.dmPolicy = bridgepkg.BridgeDMPolicyOpen
	}
	return resolved
}

func validateWhatsAppResolvedConfig(resolved *resolvedInstanceConfig) {
	if resolved == nil {
		return
	}
	switch {
	case resolved.webhookPath == "":
		resolved.configError = errors.New("whatsapp: webhook path is required")
	case resolved.phoneNumberID == "":
		resolved.configError = errors.New("whatsapp: provider_config.phone_number_id is required")
	}
}

func configureWhatsAppBatcher(
	provider *whatsappProvider,
	cfg whatsappProviderConfig,
	resolved *resolvedInstanceConfig,
) {
	if resolved == nil || cfg.Batching.DelayMS <= 0 {
		return
	}

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
			return provider.dispatchInboundBatch(ctx, resolved.instanceID, batch)
		},
		Now: func() time.Time { return provider.now() },
	})
	if err != nil {
		resolved.configError = err
		return
	}
	resolved.batcher = batcher
}

func isWhatsAppDeleteDelivery(event bridgepkg.DeliveryEvent) bool {
	return event.Operation.Normalize() == bridgepkg.DeliveryOperationDelete ||
		normalizeDeliveryEventType(event.EventType) == bridgepkg.DeliveryEventTypeDelete
}

func resumeWhatsAppDelivery(
	event bridgepkg.DeliveryEvent,
	snapshot *bridgepkg.DeliverySnapshot,
	state deliveryState,
) (bridgepkg.DeliveryAck, deliveryState, bool) {
	if normalizeDeliveryEventType(event.EventType) != bridgepkg.DeliveryEventTypeResume ||
		snapshot == nil {
		return bridgepkg.DeliveryAck{}, state, false
	}
	remoteMessageID := strings.TrimSpace(snapshot.RemoteMessageID)
	if remoteMessageID == "" {
		return bridgepkg.DeliveryAck{}, state, false
	}

	ack := bridgepkg.DeliveryAck{
		DeliveryID:             event.DeliveryID,
		Seq:                    event.Seq,
		RemoteMessageID:        remoteMessageID,
		ReplaceRemoteMessageID: strings.TrimSpace(snapshot.ReplaceRemoteMessageID),
	}
	state.LastSeq = event.Seq
	state.RemoteMessageID = ack.RemoteMessageID
	state.ReplaceRemoteMessageID = ack.ReplaceRemoteMessageID
	return ack, state, true
}

func sendWhatsAppDeliveryChunks(
	ctx context.Context,
	api whatsappAPI,
	phoneNumberID string,
	targetUserID string,
	text string,
) (string, error) {
	var remoteID string
	for _, chunk := range splitMessage(text) {
		req := whatsappSendMessageRequest{
			MessagingProduct: providerWhatsappKey,
			RecipientType:    "individual",
			To:               targetUserID,
			Type:             providerTextKey,
		}
		req.Text.Body = chunk
		req.Text.PreviewURL = false

		response, err := api.SendTextMessage(ctx, phoneNumberID, req)
		if err != nil {
			return "", err
		}
		remoteID, err = lastWhatsAppResponseMessageID(response)
		if err != nil {
			return "", err
		}
	}
	return remoteID, nil
}

func lastWhatsAppResponseMessageID(response *whatsappSendMessageResponse) (string, error) {
	if response == nil || len(response.Messages) == 0 {
		return "", &bridgesdk.TransientError{
			Err: errors.New("whatsapp: send message response omitted a message id"),
		}
	}
	messageID := strings.TrimSpace(response.Messages[len(response.Messages)-1].ID)
	if messageID == "" {
		return "", &bridgesdk.TransientError{
			Err: errors.New("whatsapp: send message response omitted a message id"),
		}
	}
	return messageID, nil
}

func (c *whatsappGraphClient) GetPhoneNumber(
	ctx context.Context,
	phoneNumberID string,
) (*whatsappPhoneNumber, error) {
	var result whatsappPhoneNumber
	if err := c.call(ctx, http.MethodGet, "/"+strings.TrimSpace(phoneNumberID), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *whatsappGraphClient) SendTextMessage(
	ctx context.Context,
	phoneNumberID string,
	payload whatsappSendMessageRequest,
) (*whatsappSendMessageResponse, error) {
	var result whatsappSendMessageResponse
	if err := c.call(
		ctx,
		http.MethodPost,
		"/"+strings.TrimSpace(phoneNumberID)+"/messages",
		payload,
		&result,
	); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *whatsappGraphClient) call(
	ctx context.Context,
	method string,
	path string,
	payload any,
	out any,
) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if c == nil {
		return errors.New("whatsapp: graph api client is required")
	}
	if c.httpClient == nil {
		c.httpClient = &http.Client{Timeout: 10 * time.Second}
	}

	var body io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("whatsapp: marshal %s payload: %w", strings.TrimSpace(path), err)
		}
		body = bytes.NewReader(raw)
	}

	endpoint := strings.TrimRight(
		strings.TrimSpace(c.baseURL),
		"/",
	) + "/" + strings.Trim(
		strings.TrimSpace(c.apiVersion),
		"/",
	) + path
	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return fmt.Errorf("whatsapp: build %s request: %w", strings.TrimSpace(path), err)
	}
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(c.accessToken))
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

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("whatsapp: read %s response: %w", strings.TrimSpace(path), err)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return classifyWhatsAppHTTPError(resp.StatusCode, resp.Header.Get("Retry-After"), raw)
	}
	if out == nil || len(bytes.TrimSpace(raw)) == 0 {
		return nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("whatsapp: decode %s response: %w", strings.TrimSpace(path), err)
	}
	return nil
}

func classifyWhatsAppHTTPError(statusCode int, retryAfterHeader string, raw []byte) error {
	retryAfter := parseRetryAfter(retryAfterHeader)
	envelope := whatsappGraphAPIErrorEnvelope{}
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &envelope); err != nil {
			envelope.Error.Message = strings.TrimSpace(string(raw))
		}
	}

	message := strings.TrimSpace(envelope.Error.Message)
	if message == "" {
		message = fmt.Sprintf("whatsapp graph api request failed with status %d", statusCode)
	}
	code := envelope.Error.Code
	subcode := envelope.Error.ErrorSubcode

	switch {
	case statusCode == http.StatusUnauthorized, statusCode == http.StatusForbidden, code == 190:
		return &bridgesdk.AuthError{
			Err: &bridgesdk.HTTPError{
				StatusCode: statusCode,
				Message:    message,
				RetryAfter: retryAfter,
			},
		}
	case statusCode == http.StatusTooManyRequests,
		code == 4,
		code == 80007,
		code == 130429,
		subcode == 2494010:
		return &bridgesdk.RateLimitError{
			Err: &bridgesdk.HTTPError{
				StatusCode: maxInt(statusCode, http.StatusTooManyRequests),
				Message:    message,
				RetryAfter: retryAfter,
			},
			RetryAfter: retryAfter,
		}
	case statusCode == http.StatusRequestTimeout, statusCode == http.StatusGatewayTimeout:
		return &bridgesdk.HTTPError{
			StatusCode: statusCode,
			Message:    message,
			RetryAfter: retryAfter,
		}
	case statusCode >= http.StatusInternalServerError:
		return &bridgesdk.TransientError{
			Err: &bridgesdk.HTTPError{
				StatusCode: statusCode,
				Message:    message,
				RetryAfter: retryAfter,
			},
		}
	default:
		return &bridgesdk.HTTPError{
			StatusCode: statusCode,
			Message:    message,
			RetryAfter: retryAfter,
		}
	}
}

func matchesPhoneNumberID(cfg resolvedInstanceConfig, incoming string) bool {
	return strings.TrimSpace(incoming) == "" ||
		strings.TrimSpace(incoming) == strings.TrimSpace(cfg.phoneNumberID)
}

func contactsByWaID(items []whatsappContact) map[string]*whatsappContact {
	if len(items) == 0 {
		return nil
	}
	result := make(map[string]*whatsappContact, len(items))
	for idx := range items {
		item := items[idx]
		waID := strings.TrimSpace(item.WaID)
		if waID == "" {
			continue
		}
		copyItem := item
		result[waID] = &copyItem
	}
	return result
}

func parseUnixTimestamp(value string) time.Time {
	seconds, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil || seconds <= 0 {
		return time.Time{}
	}
	return time.Unix(seconds, 0).UTC()
}

func splitMessage(text string) []string {
	if len(text) <= whatsappMessageLimit {
		return []string{text}
	}

	chunks := make([]string, 0, (len(text)/whatsappMessageLimit)+1)
	remaining := text
	for len(remaining) > whatsappMessageLimit {
		breakIndex := whatsappMessageBreakIndex(remaining)
		chunks = append(chunks, remaining[:breakIndex])
		remaining = remaining[breakIndex:]
	}
	chunks = append(chunks, remaining)
	return chunks
}

func whatsappMessageBreakIndex(text string) int {
	limit := whatsappMessageLimit
	if len(text) <= limit {
		return len(text)
	}
	safeLimit := limit
	for safeLimit > 0 && !utf8.RuneStart(text[safeLimit]) {
		safeLimit--
	}
	if safeLimit == 0 {
		safeLimit = limit
	}
	slice := text[:safeLimit]
	if breakIndex := strings.LastIndex(slice, "\n\n"); breakIndex >= limit/2 {
		return breakIndex + len("\n\n")
	}
	if breakIndex := strings.LastIndex(slice, "\n"); breakIndex >= limit/2 {
		return breakIndex + len("\n")
	}
	return safeLimit
}

func deliveryStateKey(instanceID string, deliveryID string) string {
	return strings.TrimSpace(instanceID) + ":" + strings.TrimSpace(deliveryID)
}

func normalizeWebhookPath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return ""
	}
	if !strings.HasPrefix(trimmed, "/") {
		trimmed = "/" + trimmed
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

func buildIdentitySet(values []string) map[string]struct{} {
	if len(values) == 0 {
		return nil
	}
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		trimmed := normalizeUsername(value)
		if trimmed == "" {
			trimmed = strings.TrimSpace(value)
		}
		if trimmed == "" {
			continue
		}
		result[trimmed] = struct{}{}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func normalizeUsername(value string) string {
	return strings.ToLower(strings.TrimSpace(strings.TrimPrefix(value, "@")))
}

func managedInstancesToInstances(
	items []subprocess.InitializeBridgeManagedInstance,
) []bridgepkg.BridgeInstance {
	instances := make([]bridgepkg.BridgeInstance, 0, len(items))
	for _, item := range items {
		instances = append(instances, item.Instance)
	}
	return instances
}

func normalizeDeliveryEventType(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func parseRetryAfter(value string) time.Duration {
	seconds, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || seconds <= 0 {
		return 0
	}
	return time.Duration(seconds) * time.Second
}

func isNotInitializedRPCError(err error) bool {
	if err == nil {
		return false
	}
	var rpcErr *subprocess.RPCError
	if !errors.As(err, &rpcErr) {
		return false
	}
	return rpcErr.Code == rpcCodeNotInitialized ||
		strings.EqualFold(strings.TrimSpace(rpcErr.Message), "Not initialized")
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
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func maxInt(values ...int) int {
	result := 0
	for _, value := range values {
		if value > result {
			result = value
		}
	}
	return result
}
