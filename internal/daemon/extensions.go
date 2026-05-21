package daemon

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pedronauck/agh/internal/api/contract"
	"github.com/pedronauck/agh/internal/api/udsapi"
	aghconfig "github.com/pedronauck/agh/internal/config"
	eventspkg "github.com/pedronauck/agh/internal/events"
	extensionpkg "github.com/pedronauck/agh/internal/extension"
	registrypkg "github.com/pedronauck/agh/internal/registry"
	"github.com/pedronauck/agh/internal/store"
	taskpkg "github.com/pedronauck/agh/internal/task"
)

type daemonExtensionService struct {
	registry          *extensionpkg.Registry
	runtime           extensionRuntime
	hookBinds         hookBindingPublisher
	agentSkill        agentSkillPublisher
	toolMCP           toolMCPPublisher
	bundles           bundleResourcePublisher
	homePaths         aghconfig.HomePaths
	logger            *slog.Logger
	now               func() time.Time
	marketplace       aghconfig.ExtensionsMarketplaceConfig
	marketplaceLoader extensionMarketplaceSourceLoader
	eventWriter       store.EventSummaryStore
}

var _ udsapi.ExtensionService = (*daemonExtensionService)(nil)

type daemonExtensionServiceOption func(*daemonExtensionService)

func withDaemonExtensionMarketplace(
	cfg aghconfig.ExtensionsMarketplaceConfig,
	loader extensionMarketplaceSourceLoader,
) daemonExtensionServiceOption {
	return func(service *daemonExtensionService) {
		service.marketplace = cfg
		service.marketplaceLoader = loader
	}
}

func withDaemonExtensionEventWriter(writer store.EventSummaryStore) daemonExtensionServiceOption {
	return func(service *daemonExtensionService) {
		service.eventWriter = writer
	}
}

func newDaemonExtensionService(
	registry *extensionpkg.Registry,
	runtime extensionRuntime,
	hookBinds hookBindingPublisher,
	agentSkill agentSkillPublisher,
	toolMCP toolMCPPublisher,
	bundles bundleResourcePublisher,
	homePaths aghconfig.HomePaths,
	logger *slog.Logger,
	now func() time.Time,
	opts ...daemonExtensionServiceOption,
) udsapi.ExtensionService {
	if registry == nil {
		return nil
	}
	if logger == nil {
		logger = slog.Default()
	}
	if now == nil {
		now = func() time.Time {
			return time.Now().UTC()
		}
	}
	service := &daemonExtensionService{
		registry:   registry,
		runtime:    runtime,
		hookBinds:  hookBinds,
		agentSkill: agentSkill,
		toolMCP:    toolMCP,
		bundles:    bundles,
		homePaths:  homePaths,
		logger:     logger,
		now:        now,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(service)
		}
	}
	return service
}

func extensionEventSummaryStore(registry Registry) store.EventSummaryStore {
	writer, ok := registry.(store.EventSummaryStore)
	if !ok {
		return nil
	}
	return writer
}

func (s *daemonExtensionService) List(ctx context.Context) ([]contract.ExtensionPayload, error) {
	if err := s.checkReady(); err != nil {
		return nil, err
	}

	infos, err := s.registry.List()
	if err != nil {
		return nil, err
	}

	items := make([]contract.ExtensionPayload, 0, len(infos))
	for _, info := range infos {
		item, err := s.Status(ctx, info.Name)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *daemonExtensionService) SearchMarketplace(
	ctx context.Context,
	query string,
	source string,
	limit int,
) ([]contract.ExtensionMarketplaceEntry, error) {
	if err := s.checkReady(); err != nil {
		return nil, err
	}
	items, err := extensionpkg.SearchMarketplaceExtensions(
		ctx,
		s.marketplaceSourceLoader(),
		query,
		source,
		limit,
	)
	if err != nil {
		return nil, err
	}
	return extensionMarketplaceEntries(items), nil
}

func (s *daemonExtensionService) Install(
	ctx context.Context,
	req contract.InstallExtensionRequest,
	actor taskpkg.ActorContext,
) (contract.ExtensionPayload, error) {
	if err := s.checkReady(); err != nil {
		return contract.ExtensionPayload{}, err
	}
	if err := validateExtensionWriteActor(actor); err != nil {
		return contract.ExtensionPayload{}, err
	}
	installedBy := extensionInstalledBy(actor)

	if strings.TrimSpace(req.Slug) != "" || strings.TrimSpace(req.Path) == "" {
		info, err := extensionpkg.InstallMarketplaceManaged(
			ctx,
			s.homePaths,
			s.registry,
			s.marketplaceSourceLoader(),
			extensionpkg.MarketplaceInstallRequest{
				Slug:            req.Slug,
				SourceFilter:    req.Source,
				Version:         req.Version,
				Asset:           req.Asset,
				AllowUnverified: req.AllowUnverified,
				InstalledBy:     installedBy,
			},
		)
		if err != nil {
			return contract.ExtensionPayload{}, err
		}
		if err := s.reload(ctx); err != nil {
			return contract.ExtensionPayload{}, s.rollbackFailedInstall(ctx, info.Name, err)
		}
		item, err := s.Status(ctx, info.Name)
		if err != nil {
			return contract.ExtensionPayload{}, err
		}
		if err := s.recordExtensionEvent(ctx, eventspkg.ExtensionInstalled, actor, item); err != nil {
			return contract.ExtensionPayload{}, err
		}
		return item, nil
	}

	manifest, err := extensionpkg.LoadManifest(strings.TrimSpace(req.Path))
	if err != nil {
		return contract.ExtensionPayload{}, err
	}
	if !req.AllowUnverified {
		return contract.ExtensionPayload{}, extensionpkg.NewExtensionChecksumUnverifiedError(manifest.Name, req.Path)
	}
	provenance := extensionpkg.LocalPathProvenance(manifest, req.Path, req.Checksum, s.now(), req.AllowUnverified)
	provenance.InstalledBy = installedBy
	if err := extensionpkg.InstallLocalManaged(
		s.homePaths,
		s.registry,
		manifest,
		req.Path,
		req.Checksum,
		extensionpkg.WithInstallProvenance(provenance),
	); err != nil {
		return contract.ExtensionPayload{}, err
	}
	if err := s.reload(ctx); err != nil {
		return contract.ExtensionPayload{}, s.rollbackFailedInstall(ctx, manifest.Name, err)
	}
	item, err := s.Status(ctx, manifest.Name)
	if err != nil {
		return contract.ExtensionPayload{}, err
	}
	if err := s.recordExtensionEvent(ctx, eventspkg.ExtensionInstalled, actor, item); err != nil {
		return contract.ExtensionPayload{}, err
	}
	return item, nil
}

func (s *daemonExtensionService) Update(
	ctx context.Context,
	name string,
	req contract.UpdateExtensionRequest,
	actor taskpkg.ActorContext,
) (contract.ManagedExtensionUpdatePayload, error) {
	if err := s.checkReady(); err != nil {
		return contract.ManagedExtensionUpdatePayload{}, err
	}
	if err := validateExtensionWriteActor(actor); err != nil {
		return contract.ManagedExtensionUpdatePayload{}, err
	}
	items, err := extensionpkg.UpdateMarketplaceManaged(
		ctx,
		s.homePaths,
		s.registry,
		s.marketplaceSourceLoader(),
		extensionpkg.MarketplaceUpdateRequest{
			Names:           []string{name},
			Version:         req.Version,
			CheckOnly:       req.CheckOnly,
			AllowUnverified: req.AllowUnverified,
			InstalledBy:     extensionInstalledBy(actor),
		},
		s.reload,
	)
	if err != nil {
		return contract.ManagedExtensionUpdatePayload{}, err
	}
	if len(items) == 0 {
		return contract.ManagedExtensionUpdatePayload{}, extensionpkg.ErrExtensionNotFound
	}
	item := extensionUpdatePayload(items[0])
	if item.Status == extensionpkg.MarketplaceUpdateStatusUpdated {
		if err := s.recordExtensionUpdateEvent(ctx, actor, item); err != nil {
			return contract.ManagedExtensionUpdatePayload{}, err
		}
	}
	return item, nil
}

func (s *daemonExtensionService) Remove(
	ctx context.Context,
	name string,
	actor taskpkg.ActorContext,
) (contract.ManagedExtensionRemovePayload, error) {
	if err := s.checkReady(); err != nil {
		return contract.ManagedExtensionRemovePayload{}, err
	}
	if err := validateExtensionWriteActor(actor); err != nil {
		return contract.ManagedExtensionRemovePayload{}, err
	}
	removed, err := extensionpkg.RemoveManagedExtension(ctx, s.registry, name, s.reload)
	if err != nil {
		return contract.ManagedExtensionRemovePayload{}, err
	}
	item := contract.ManagedExtensionRemovePayload{
		Name:   removed.Name,
		Path:   removed.Path,
		Status: removed.Status,
	}
	if err := s.recordExtensionRemoveEvent(ctx, actor, item); err != nil {
		return contract.ManagedExtensionRemovePayload{}, err
	}
	return item, nil
}

func (s *daemonExtensionService) Provenance(
	ctx context.Context,
	name string,
) (contract.ExtensionProvenancePayload, error) {
	if err := s.checkReady(); err != nil {
		return contract.ExtensionProvenancePayload{}, err
	}
	payload, err := s.Status(ctx, name)
	if err != nil {
		return contract.ExtensionProvenancePayload{}, err
	}
	if payload.Provenance == nil {
		return contract.ExtensionProvenancePayload{}, extensionpkg.ErrExtensionNotFound
	}
	return *payload.Provenance, nil
}

func (s *daemonExtensionService) rollbackFailedInstall(
	ctx context.Context,
	name string,
	installErr error,
) error {
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return installErr
	}

	var rollbackErr error
	if err := s.registry.Uninstall(trimmedName); err != nil && !errors.Is(err, extensionpkg.ErrExtensionNotFound) {
		rollbackErr = errors.Join(
			rollbackErr,
			fmt.Errorf("daemon: rollback extension registry row %q: %w", trimmedName, err),
		)
	}

	managedPath := extensionpkg.ManagedInstallPath(s.homePaths, trimmedName)
	if err := os.RemoveAll(managedPath); err != nil {
		rollbackErr = errors.Join(rollbackErr, fmt.Errorf("daemon: rollback extension files %q: %w", managedPath, err))
	}

	if err := s.reload(ctx); err != nil {
		rollbackErr = errors.Join(rollbackErr, fmt.Errorf("daemon: reload after extension install rollback: %w", err))
	}

	return errors.Join(installErr, rollbackErr)
}

func (s *daemonExtensionService) Enable(
	ctx context.Context,
	name string,
	actor taskpkg.ActorContext,
) (contract.ExtensionPayload, error) {
	if err := s.checkReady(); err != nil {
		return contract.ExtensionPayload{}, err
	}
	if err := validateExtensionWriteActor(actor); err != nil {
		return contract.ExtensionPayload{}, err
	}
	if err := s.registry.Enable(name); err != nil {
		return contract.ExtensionPayload{}, err
	}
	if err := s.reload(ctx); err != nil {
		return contract.ExtensionPayload{}, err
	}
	item, err := s.Status(ctx, name)
	if err != nil {
		return contract.ExtensionPayload{}, err
	}
	if err := s.recordExtensionEvent(ctx, eventspkg.ExtensionEnabled, actor, item); err != nil {
		return contract.ExtensionPayload{}, err
	}
	return item, nil
}

func (s *daemonExtensionService) Disable(
	ctx context.Context,
	name string,
	actor taskpkg.ActorContext,
) (contract.ExtensionPayload, error) {
	if err := s.checkReady(); err != nil {
		return contract.ExtensionPayload{}, err
	}
	if err := validateExtensionWriteActor(actor); err != nil {
		return contract.ExtensionPayload{}, err
	}
	if err := s.registry.Disable(name); err != nil {
		return contract.ExtensionPayload{}, err
	}
	if err := s.reload(ctx); err != nil {
		return contract.ExtensionPayload{}, err
	}
	item, err := s.Status(ctx, name)
	if err != nil {
		return contract.ExtensionPayload{}, err
	}
	if err := s.recordExtensionEvent(ctx, eventspkg.ExtensionDisabled, actor, item); err != nil {
		return contract.ExtensionPayload{}, err
	}
	return item, nil
}

func (s *daemonExtensionService) Status(ctx context.Context, name string) (contract.ExtensionPayload, error) {
	if err := s.checkReady(); err != nil {
		return contract.ExtensionPayload{}, err
	}

	ext, err := s.lookup(ctx, name)
	if err != nil {
		return contract.ExtensionPayload{}, err
	}
	return s.payloadFromExtension(ext), nil
}

func (s *daemonExtensionService) reload(ctx context.Context) error {
	if s.runtime == nil {
		return nil
	}

	reloadErr := s.runtime.Reload(ctx)
	var syncErr error
	if s.agentSkill != nil {
		syncErr = errors.Join(syncErr, s.agentSkill.Sync(ctx))
	}
	if s.hookBinds != nil {
		syncErr = errors.Join(syncErr, s.hookBinds.Sync(ctx))
	}
	if s.toolMCP != nil {
		syncErr = errors.Join(syncErr, s.toolMCP.Sync(ctx))
	}
	if s.bundles != nil {
		syncErr = errors.Join(syncErr, s.bundles.Sync(ctx))
	}
	return errors.Join(reloadErr, syncErr)
}

func (s *daemonExtensionService) lookup(ctx context.Context, name string) (*extensionpkg.Extension, error) {
	return loadExtensionSnapshot(ctx, s.registry, s.runtime, s.logger, name)
}

func loadExtensionSnapshot(
	ctx context.Context,
	registry *extensionpkg.Registry,
	runtime extensionRuntime,
	logger *slog.Logger,
	name string,
) (*extensionpkg.Extension, error) {
	if registry == nil {
		return nil, errors.New("daemon: extension registry is required")
	}

	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return nil, errors.New("extension: extension name is required")
	}

	if runtime != nil {
		ext, err := runtime.Get(trimmed)
		if err == nil {
			populateExtensionManifest(ctx, logger, ext)
			return ext, nil
		}
		if !errors.Is(err, extensionpkg.ErrExtensionNotFound) {
			return nil, err
		}
	}

	info, err := registry.Get(trimmed)
	if err != nil {
		return nil, err
	}

	ext := &extensionpkg.Extension{
		Info: *info,
		Status: extensionpkg.ExtensionStatus{
			Name:    info.Name,
			Version: info.Version,
			Source:  info.Source,
			Enabled: info.Enabled,
		},
	}
	populateExtensionManifest(ctx, logger, ext)
	return ext, nil
}

func populateExtensionManifest(ctx context.Context, logger *slog.Logger, ext *extensionpkg.Extension) {
	if ext == nil || ext.Manifest != nil || strings.TrimSpace(ext.Info.ManifestPath) == "" {
		return
	}

	manifest, err := extensionpkg.LoadManifest(filepath.Dir(ext.Info.ManifestPath))
	if err != nil {
		if logger != nil {
			logger.Debug(
				"daemon: load extension manifest for status failed",
				"path",
				ext.Info.ManifestPath,
				"error",
				err,
			)
		}
		return
	}
	ext.Manifest = manifest
	if bundles, err := extensionpkg.LoadBundleSpecs(ctx, filepath.Dir(ext.Info.ManifestPath), manifest); err == nil {
		ext.Bundles = bundles
	} else if logger != nil {
		logger.Debug("daemon: load extension bundles for status failed", "path", ext.Info.ManifestPath, "error", err)
	}
}

func (s *daemonExtensionService) payloadFromExtension(ext *extensionpkg.Extension) contract.ExtensionPayload {
	return extensionpkg.DescribeExtension(ext, s.runtime != nil, s.now())
}

func (s *daemonExtensionService) marketplaceSourceLoader() extensionpkg.MarketplaceSourceLoader {
	return func(ctx context.Context) ([]registrypkg.Source, error) {
		loader := s.marketplaceLoader
		if loader == nil {
			loader = defaultDaemonExtensionMarketplaceSourceLoader
		}
		return loader(ctx, s.marketplace)
	}
}

func (s *daemonExtensionService) checkReady() error {
	if s == nil {
		return errors.New("daemon: extension service is required")
	}
	if s.registry == nil {
		return errors.New("daemon: extension registry is required")
	}
	return nil
}

func validateExtensionWriteActor(actor taskpkg.ActorContext) error {
	if err := actor.Validate(); err != nil {
		return err
	}
	if !actor.Authority.Write {
		return taskpkg.ErrPermissionDenied
	}
	return nil
}

func extensionInstalledBy(actor taskpkg.ActorContext) string {
	actorKind := strings.TrimSpace(string(actor.Actor.Kind.Normalize()))
	actorRef := strings.TrimSpace(actor.Actor.Ref)
	if actorKind == "" {
		return actorRef
	}
	if actorRef == "" {
		return actorKind
	}
	return actorKind + ":" + actorRef
}

func extensionMarketplaceEntries(values []registrypkg.Listing) []contract.ExtensionMarketplaceEntry {
	items := make([]contract.ExtensionMarketplaceEntry, 0, len(values))
	for _, value := range values {
		trust := contract.ExtensionTrustReportPayload{
			Decision:         extensionpkg.ExtensionTrustDecisionBlocked,
			RegistryTier:     extensionpkg.ExtensionRegistryTierUnverified,
			ChecksumVerified: false,
			AllowUnverified:  false,
		}
		items = append(items, contract.ExtensionMarketplaceEntry{
			Slug:        value.Slug,
			Name:        value.Name,
			Description: value.Description,
			Author:      value.Author,
			Version:     value.Version,
			Downloads:   value.Downloads,
			Source:      value.Source,
			Type:        string(value.Type),
			Trust:       &trust,
		})
	}
	return items
}

func extensionUpdatePayload(value extensionpkg.MarketplaceUpdateResult) contract.ManagedExtensionUpdatePayload {
	return contract.ManagedExtensionUpdatePayload{
		Name:           value.Name,
		Slug:           value.Slug,
		Registry:       value.Registry,
		CurrentVersion: value.CurrentVersion,
		LatestVersion:  value.LatestVersion,
		Path:           value.Path,
		Status:         value.Status,
	}
}

type extensionLifecycleEventPayload struct {
	ActorKind        string `json:"actor_kind,omitempty"`
	ActorID          string `json:"actor_id,omitempty"`
	OriginKind       string `json:"origin_kind,omitempty"`
	OriginRef        string `json:"origin_ref,omitempty"`
	Name             string `json:"name"`
	Slug             string `json:"slug,omitempty"`
	Version          string `json:"version,omitempty"`
	CurrentVersion   string `json:"current_version,omitempty"`
	LatestVersion    string `json:"latest_version,omitempty"`
	Status           string `json:"status,omitempty"`
	InstalledFrom    string `json:"installed_from,omitempty"`
	SourceURL        string `json:"source_url,omitempty"`
	ChecksumSHA256   string `json:"checksum_sha256,omitempty"`
	ChecksumVerified bool   `json:"checksum_verified"`
	RegistryTier     string `json:"registry_tier,omitempty"`
	AllowUnverified  bool   `json:"allow_unverified"`
}

func (s *daemonExtensionService) recordExtensionEvent(
	ctx context.Context,
	eventType string,
	actor taskpkg.ActorContext,
	item contract.ExtensionPayload,
) error {
	payload := extensionLifecycleEventPayload{}
	payload.Name = firstNonEmpty(payload.Name, item.Name)
	payload.Version = firstNonEmpty(payload.Version, item.Version)
	payload.Status = firstNonEmpty(payload.Status, item.State)
	if item.Provenance != nil {
		payload.Slug = firstNonEmpty(payload.Slug, item.Provenance.Slug)
		payload.InstalledFrom = firstNonEmpty(payload.InstalledFrom, item.Provenance.InstalledFrom)
		payload.SourceURL = firstNonEmpty(payload.SourceURL, item.Provenance.SourceURL)
		payload.ChecksumSHA256 = firstNonEmpty(payload.ChecksumSHA256, item.Provenance.ChecksumSHA256)
		payload.ChecksumVerified = item.Provenance.ChecksumVerified
		payload.RegistryTier = firstNonEmpty(payload.RegistryTier, item.Provenance.RegistryTier)
		payload.AllowUnverified = item.Provenance.AllowUnverified
	} else if item.Trust != nil {
		payload.ChecksumVerified = item.Trust.ChecksumVerified
		payload.RegistryTier = firstNonEmpty(payload.RegistryTier, item.Trust.RegistryTier)
		payload.AllowUnverified = item.Trust.AllowUnverified
	}
	return s.recordExtensionLifecycleEvent(ctx, eventType, actor, payload)
}

func (s *daemonExtensionService) recordExtensionUpdateEvent(
	ctx context.Context,
	actor taskpkg.ActorContext,
	item contract.ManagedExtensionUpdatePayload,
) error {
	return s.recordExtensionLifecycleEvent(ctx, eventspkg.ExtensionUpdated, actor, extensionLifecycleEventPayload{
		Name:           item.Name,
		Slug:           item.Slug,
		CurrentVersion: item.CurrentVersion,
		LatestVersion:  item.LatestVersion,
		Status:         item.Status,
	})
}

func (s *daemonExtensionService) recordExtensionRemoveEvent(
	ctx context.Context,
	actor taskpkg.ActorContext,
	item contract.ManagedExtensionRemovePayload,
) error {
	return s.recordExtensionLifecycleEvent(ctx, eventspkg.ExtensionRemoved, actor, extensionLifecycleEventPayload{
		Name:   item.Name,
		Status: item.Status,
	})
}

func (s *daemonExtensionService) recordExtensionLifecycleEvent(
	ctx context.Context,
	eventType string,
	actor taskpkg.ActorContext,
	payload extensionLifecycleEventPayload,
) error {
	if s.eventWriter == nil {
		return nil
	}
	if ctx == nil {
		return errors.New("daemon: extension event context is required")
	}
	payload.ActorKind = string(actor.Actor.Kind.Normalize())
	payload.ActorID = strings.TrimSpace(actor.Actor.Ref)
	payload.OriginKind = string(actor.Origin.Kind.Normalize())
	payload.OriginRef = strings.TrimSpace(actor.Origin.Ref)
	content, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("daemon: encode extension lifecycle event: %w", err)
	}
	if err := s.eventWriter.WriteEventSummary(context.WithoutCancel(ctx), store.EventSummary{
		Type:      eventType,
		Outcome:   string(eventspkg.OutcomeFor(eventType)),
		Content:   content,
		Summary:   extensionLifecycleEventSummary(eventType, payload),
		Timestamp: s.now().UTC(),
		EventCorrelation: store.EventCorrelation{
			ActorKind: payload.ActorKind,
			ActorID:   payload.ActorID,
		},
	}); err != nil {
		return fmt.Errorf("daemon: record extension lifecycle event: %w", err)
	}
	return nil
}

func extensionLifecycleEventSummary(eventType string, payload extensionLifecycleEventPayload) string {
	name := strings.TrimSpace(payload.Name)
	if name == "" {
		name = strings.TrimSpace(payload.Slug)
	}
	switch eventType {
	case eventspkg.ExtensionInstalled:
		return fmt.Sprintf("extension %s installed", name)
	case eventspkg.ExtensionUpdated:
		return fmt.Sprintf("extension %s updated", name)
	case eventspkg.ExtensionRemoved:
		return fmt.Sprintf("extension %s removed", name)
	case eventspkg.ExtensionEnabled:
		return fmt.Sprintf("extension %s enabled", name)
	case eventspkg.ExtensionDisabled:
		return fmt.Sprintf("extension %s disabled", name)
	default:
		return strings.TrimSpace(eventType)
	}
}
