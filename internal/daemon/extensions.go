package daemon

import (
	"context"
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
	extensionpkg "github.com/pedronauck/agh/internal/extension"
)

type daemonExtensionService struct {
	registry   *extensionpkg.Registry
	runtime    extensionRuntime
	hookBinds  hookBindingPublisher
	agentSkill agentSkillPublisher
	toolMCP    toolMCPPublisher
	bundles    bundleResourcePublisher
	homePaths  aghconfig.HomePaths
	logger     *slog.Logger
	now        func() time.Time
}

var _ udsapi.ExtensionService = (*daemonExtensionService)(nil)

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
	return &daemonExtensionService{
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

func (s *daemonExtensionService) Install(
	ctx context.Context,
	req contract.InstallExtensionRequest,
) (contract.ExtensionPayload, error) {
	if err := s.checkReady(); err != nil {
		return contract.ExtensionPayload{}, err
	}

	manifest, err := extensionpkg.LoadManifest(strings.TrimSpace(req.Path))
	if err != nil {
		return contract.ExtensionPayload{}, err
	}
	if err := extensionpkg.InstallLocalManaged(s.homePaths, s.registry, manifest, req.Path, req.Checksum); err != nil {
		return contract.ExtensionPayload{}, err
	}
	if err := s.reload(ctx); err != nil {
		return contract.ExtensionPayload{}, s.rollbackFailedInstall(ctx, manifest.Name, err)
	}
	return s.Status(ctx, manifest.Name)
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

func (s *daemonExtensionService) Enable(ctx context.Context, name string) (contract.ExtensionPayload, error) {
	if err := s.checkReady(); err != nil {
		return contract.ExtensionPayload{}, err
	}
	if err := s.registry.Enable(name); err != nil {
		return contract.ExtensionPayload{}, err
	}
	if err := s.reload(ctx); err != nil {
		return contract.ExtensionPayload{}, err
	}
	return s.Status(ctx, name)
}

func (s *daemonExtensionService) Disable(ctx context.Context, name string) (contract.ExtensionPayload, error) {
	if err := s.checkReady(); err != nil {
		return contract.ExtensionPayload{}, err
	}
	if err := s.registry.Disable(name); err != nil {
		return contract.ExtensionPayload{}, err
	}
	if err := s.reload(ctx); err != nil {
		return contract.ExtensionPayload{}, err
	}
	return s.Status(ctx, name)
}

func (s *daemonExtensionService) Status(_ context.Context, name string) (contract.ExtensionPayload, error) {
	if err := s.checkReady(); err != nil {
		return contract.ExtensionPayload{}, err
	}

	ext, err := s.lookup(name)
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

func (s *daemonExtensionService) lookup(name string) (*extensionpkg.Extension, error) {
	return loadExtensionSnapshot(s.registry, s.runtime, s.logger, name)
}

func loadExtensionSnapshot(
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
			populateExtensionManifest(logger, ext)
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
	populateExtensionManifest(logger, ext)
	return ext, nil
}

func populateExtensionManifest(logger *slog.Logger, ext *extensionpkg.Extension) {
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
	if bundles, err := extensionpkg.LoadBundleSpecs(filepath.Dir(ext.Info.ManifestPath), manifest); err == nil {
		ext.Bundles = bundles
	} else if logger != nil {
		logger.Debug("daemon: load extension bundles for status failed", "path", ext.Info.ManifestPath, "error", err)
	}
}

func (s *daemonExtensionService) payloadFromExtension(ext *extensionpkg.Extension) contract.ExtensionPayload {
	return extensionpkg.DescribeExtension(ext, s.runtime != nil, s.now())
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
