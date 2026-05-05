package update

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// NewManager binds the update flow to the current AGH runtime.
func NewManager(cfg Config) (*Manager, error) {
	executablePath := cfg.ExecutablePath
	if executablePath == nil {
		executablePath = os.Executable
	}
	resolveSymlinks := cfg.ResolveSymlinks
	if resolveSymlinks == nil {
		resolveSymlinks = filepath.EvalSymlinks
	}
	getenv := cfg.Getenv
	if getenv == nil {
		getenv = os.Getenv
	}
	now := cfg.Now
	if now == nil {
		now = func() time.Time {
			return time.Now().UTC()
		}
	}
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultHTTPTimeout}
	}
	lookPath := cfg.LookPath
	if lookPath == nil {
		lookPath = exec.LookPath
	}
	runCommand := cfg.RunCommand
	if runCommand == nil {
		runCommand = defaultRunCommand
	}

	resolvedExecutable, err := executablePath()
	if err != nil {
		return nil, fmt.Errorf("update: resolve executable path: %w", err)
	}
	trimmedExecutable := strings.TrimSpace(resolvedExecutable)
	if trimmedExecutable == "" {
		return nil, errors.New("update: executable path is required")
	}
	resolvedExecutable = trimmedExecutable
	symlinkResolved, err := resolveSymlinks(resolvedExecutable)
	if err == nil && strings.TrimSpace(symlinkResolved) != "" {
		resolvedExecutable = strings.TrimSpace(symlinkResolved)
	}

	manager := &Manager{
		homePaths:      cfg.HomePaths,
		currentVersion: strings.TrimSpace(cfg.CurrentVersion),
		executablePath: resolvedExecutable,
		getenv:         getenv,
		now:            now,
		httpClient:     httpClient,
		runtimeOS:      strings.TrimSpace(cfg.RuntimeOS),
		runtimeArch:    strings.TrimSpace(cfg.RuntimeArch),
		lookPath:       lookPath,
		runCommand:     runCommand,
		binaryApplier:  cfg.BinaryApplier,
	}
	if manager.runtimeOS == "" {
		manager.runtimeOS = runtime.GOOS
	}
	if manager.runtimeArch == "" {
		manager.runtimeArch = runtime.GOARCH
	}
	if manager.binaryApplier == nil {
		manager.binaryApplier = selfBinaryApplier{}
	}
	if cfg.BundleVerifier != nil {
		manager.bundleVerifier = cfg.BundleVerifier
	} else {
		manager.bundleVerifier = sigstoreBundleVerifier{cachePath: manager.sigstoreCachePath()}
	}
	if strings.TrimSpace(manager.homePaths.HomeDir) == "" {
		return nil, errors.New("update: AGH home directory is required")
	}
	return manager, nil
}

// Check returns the current update state and the latest release metadata when available.
func (m *Manager) Check(ctx context.Context, opts CheckOptions) (State, *Release, error) {
	install := m.detectInstall(ctx)

	var (
		latest    *Release
		checkedAt *time.Time
	)
	if cached, err := readCache(m.cachePath()); err == nil {
		latest = &Release{
			Version:    strings.TrimSpace(cached.LatestVersion),
			ReleaseURL: strings.TrimSpace(cached.ReleaseURL),
		}
		checked := cached.CheckedAt.UTC()
		checkedAt = &checked
	} else if err != nil && !errors.Is(err, ErrNoCachedRelease) {
		return State{}, nil, err
	}

	needRefresh := opts.ForceRefresh || checkedAt == nil
	if !needRefresh && checkedAt != nil {
		needRefresh = m.now().UTC().Sub(checkedAt.UTC()) >= cacheTTL
	}

	if needRefresh {
		freshRelease, err := m.fetchLatestRelease(ctx)
		if err != nil {
			state := m.composeState(install, latest, checkedAt)
			state.LastError = err.Error()
			state.Message = "Failed to check for a newer stable AGH release."
			if latest != nil && opts.AllowCachedOnFailure {
				return state, latest, nil
			}
			state.Status = StatusFailed
			return state, nil, err
		}

		latest = freshRelease
		checked := m.now().UTC()
		checkedAt = &checked
		if err := writeCache(m.cachePath(), cacheEntry{
			LatestVersion: latest.Version,
			ReleaseURL:    latest.ReleaseURL,
			CheckedAt:     checked,
		}); err != nil {
			return State{}, nil, err
		}
	}

	return m.composeState(install, latest, checkedAt), latest, nil
}

// ApplyRelease downloads, verifies, extracts, and swaps in the supplied release.
func (m *Manager) ApplyRelease(ctx context.Context, release *Release) (AppliedBinary, error) {
	if release == nil {
		return AppliedBinary{}, errors.New("update: release metadata is required")
	}

	assets, err := m.resolveReleaseAssets(release)
	if err != nil {
		return AppliedBinary{}, err
	}
	currentInfo, err := os.Stat(m.executablePath)
	if err != nil {
		return AppliedBinary{}, fmt.Errorf("update: stat current executable %q: %w", m.executablePath, err)
	}

	tmpDir, err := os.MkdirTemp("", "agh-update-*")
	if err != nil {
		return AppliedBinary{}, fmt.Errorf("update: create temp directory: %w", err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	downloaded, err := m.downloadReleaseArtifacts(ctx, tmpDir, assets)
	if err != nil {
		return AppliedBinary{}, err
	}
	if err := m.verifyReleaseArtifacts(ctx, downloaded, assets.archive.Name); err != nil {
		return AppliedBinary{}, err
	}

	binaryPath, binaryMode, err := extractBinaryFromTarGz(
		downloaded.archivePath,
		tmpDir,
		m.archiveBinaryName(),
	)
	if err != nil {
		return AppliedBinary{}, err
	}
	if binaryMode == 0 {
		binaryMode = currentInfo.Mode().Perm()
	}

	backupPath := siblingBackupPath(m.executablePath, m.now().UTC())
	if err := m.binaryApplier.ApplyBinary(binaryPath, m.executablePath, backupPath, binaryMode); err != nil {
		return AppliedBinary{}, err
	}

	return AppliedBinary{
		TargetPath: m.executablePath,
		BackupPath: backupPath,
		Version:    strings.TrimSpace(release.Version),
	}, nil
}

type releaseAssets struct {
	archive   ReleaseAsset
	checksums ReleaseAsset
	bundle    ReleaseAsset
}

type downloadedReleaseArtifacts struct {
	archivePath   string
	checksumsPath string
	bundlePath    string
}

func (m *Manager) resolveReleaseAssets(release *Release) (releaseAssets, error) {
	archiveName, err := archiveAssetName(m.runtimeOS, m.runtimeArch)
	if err != nil {
		return releaseAssets{}, err
	}

	archiveAsset, ok := release.findAsset(archiveName)
	if !ok {
		return releaseAssets{}, fmt.Errorf(
			"update: release %q does not publish %s",
			release.Version,
			archiveName,
		)
	}
	checksumsAsset, ok := release.findAsset(checksumsAssetName)
	if !ok {
		return releaseAssets{}, fmt.Errorf(
			"update: release %q is missing %s",
			release.Version,
			checksumsAssetName,
		)
	}
	bundleAsset, ok := release.findAsset(checksumsBundleAssetName)
	if !ok {
		return releaseAssets{}, fmt.Errorf(
			"update: release %q is missing %s",
			release.Version,
			checksumsBundleAssetName,
		)
	}

	return releaseAssets{
		archive:   archiveAsset,
		checksums: checksumsAsset,
		bundle:    bundleAsset,
	}, nil
}

func (m *Manager) downloadReleaseArtifacts(
	ctx context.Context,
	tmpDir string,
	assets releaseAssets,
) (downloadedReleaseArtifacts, error) {
	downloaded := downloadedReleaseArtifacts{
		archivePath:   filepath.Join(tmpDir, assets.archive.Name),
		checksumsPath: filepath.Join(tmpDir, assets.checksums.Name),
		bundlePath:    filepath.Join(tmpDir, assets.bundle.Name),
	}

	if err := m.downloadFile(ctx, assets.archive.DownloadURL, downloaded.archivePath); err != nil {
		return downloadedReleaseArtifacts{}, err
	}
	if err := m.downloadFile(ctx, assets.checksums.DownloadURL, downloaded.checksumsPath); err != nil {
		return downloadedReleaseArtifacts{}, err
	}
	if err := m.downloadFile(ctx, assets.bundle.DownloadURL, downloaded.bundlePath); err != nil {
		return downloadedReleaseArtifacts{}, err
	}

	return downloaded, nil
}

func (m *Manager) verifyReleaseArtifacts(
	ctx context.Context,
	downloaded downloadedReleaseArtifacts,
	archiveName string,
) error {
	if err := m.bundleVerifier.VerifyChecksums(ctx, downloaded.checksumsPath, downloaded.bundlePath); err != nil {
		return err
	}

	expectedChecksum, err := checksumForAsset(downloaded.checksumsPath, archiveName)
	if err != nil {
		return err
	}
	return verifySHA256(downloaded.archivePath, expectedChecksum)
}

// Restore rolls back an applied binary swap using the preserved sibling backup.
func (m *Manager) Restore(applied AppliedBinary) error {
	if strings.TrimSpace(applied.BackupPath) == "" || strings.TrimSpace(applied.TargetPath) == "" {
		return errors.New("update: applied binary rollback paths are required")
	}
	currentInfo, err := os.Stat(applied.TargetPath)
	if err != nil {
		return fmt.Errorf("update: stat updated executable %q: %w", applied.TargetPath, err)
	}
	return m.binaryApplier.RestoreBinary(applied.BackupPath, applied.TargetPath, currentInfo.Mode().Perm())
}

// Finalize removes the preserved sibling backup after the full update flow succeeds.
func (m *Manager) Finalize(applied AppliedBinary) error {
	if strings.TrimSpace(applied.BackupPath) == "" {
		return nil
	}
	if err := os.Remove(applied.BackupPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("update: remove backup binary %q: %w", applied.BackupPath, err)
	}
	return nil
}

func (m *Manager) composeState(install installInfo, latest *Release, checkedAt *time.Time) State {
	state := State{
		Managed:        install.Managed,
		InstallMethod:  strings.TrimSpace(install.Method),
		CurrentVersion: strings.TrimSpace(m.currentVersion),
		CheckedAt:      checkedAt,
	}
	if latest != nil {
		state.LatestVersion = strings.TrimSpace(latest.Version)
		state.ReleaseURL = strings.TrimSpace(latest.ReleaseURL)
	}
	if state.InstallMethod == "" {
		state.InstallMethod = string(InstallMethodUnknown)
	}

	switch {
	case isDevVersion(state.CurrentVersion):
		state.Status = StatusUnsupported
		state.Message = "AGH self-update is unavailable for dev builds."
		state.Recommendation = "Install a tagged AGH release binary or rebuild from source."
		return state
	case latest == nil || strings.TrimSpace(latest.Version) == "":
		state.Status = StatusFailed
		state.Message = "The latest stable AGH release metadata is unavailable."
		return state
	}

	comparison, err := compareVersions(state.CurrentVersion, latest.Version)
	if err != nil {
		state.Status = StatusUnsupported
		state.Message = "The running AGH version cannot be compared against published releases."
		state.LastError = err.Error()
		return state
	}

	state.Available = comparison < 0
	supportedPlatform := supportsDirectBinarySelfUpdate(m.runtimeOS, m.runtimeArch)
	state.Supported = !state.Managed &&
		state.InstallMethod == string(InstallMethodDirectBinary) &&
		supportedPlatform

	switch {
	case state.Managed && state.Available:
		state.Status = StatusDeferred
		state.Message = "AGH is managed by an external package manager; no local update was performed."
		state.Recommendation = updateRecommendation(state.InstallMethod, state.ReleaseURL)
	case state.Managed:
		state.Status = StatusCurrent
		state.Message = "AGH is already on the latest stable release. Managed installs stay on the package manager path."
		state.Recommendation = updateRecommendation(state.InstallMethod, state.ReleaseURL)
	case !state.Supported && state.Available:
		state.Status = StatusUnsupported
		state.Message = "A newer stable AGH release is available, but this install method does not support in-place updates."
		state.Recommendation = updateRecommendation(state.InstallMethod, state.ReleaseURL)
	case !state.Supported:
		state.Status = StatusUnsupported
		state.Message = "This AGH install method does not support in-place self-update."
		state.Recommendation = updateRecommendation(state.InstallMethod, state.ReleaseURL)
	case state.Available:
		state.Status = StatusAvailable
		state.Message = "A newer stable AGH release is available."
		state.Recommendation = "Run `agh update`."
	default:
		state.Status = StatusCurrent
		state.Message = "AGH is already on the latest stable release."
	}

	if state.InstallMethod == string(InstallMethodDirectBinary) && !supportedPlatform {
		state.Recommendation = manualDirectBinaryRecommendation(state.ReleaseURL, m.runtimeOS)
	}
	return state
}

func (m *Manager) archiveBinaryName() string {
	if m.runtimeOS == runtimeOSWindows {
		return aghWindowsBinaryName
	}
	return aghBinaryName
}

func supportsDirectBinarySelfUpdate(runtimeOS string, runtimeArch string) bool {
	if runtimeOS != runtimeOSDarwin && runtimeOS != runtimeOSLinux {
		return false
	}
	return runtimeArch == runtimeArchAMD64 || runtimeArch == runtimeArchARM64
}

func updateRecommendation(installMethod string, releaseURL string) string {
	switch installMethod {
	case string(InstallMethodHomebrew):
		return "Use `brew upgrade --cask agh`."
	case string(InstallMethodAPT):
		return "Use `sudo apt update && sudo apt upgrade agh`."
	case string(InstallMethodDNF):
		return "Use `sudo dnf upgrade agh`."
	case string(InstallMethodRPM):
		return "Upgrade the installed RPM package through your system package tooling."
	case string(InstallMethodScoop):
		return "Use `scoop update agh`."
	case string(InstallMethodGoInstall):
		return "Use `go install " + goInstallModulePath + "@latest`."
	case string(InstallMethodDirectBinary):
		return manualDirectBinaryRecommendation(releaseURL, "")
	default:
		if strings.TrimSpace(installMethod) == "" {
			return ""
		}
		return "Use the package manager or installer that manages this AGH binary instead of mutating it in place."
	}
}

func manualDirectBinaryRecommendation(releaseURL string, runtimeOS string) string {
	if runtimeOS == runtimeOSWindows {
		if strings.TrimSpace(releaseURL) == "" {
			return "Download the latest AGH Windows release archive and replace `agh.exe` manually."
		}
		return "Download the latest AGH Windows release archive from " + releaseURL + " and replace `agh.exe` manually."
	}
	if strings.TrimSpace(releaseURL) == "" {
		return "Download the latest AGH release archive and replace the binary manually."
	}
	return "Download the latest AGH release archive from " + releaseURL + " and replace the binary manually."
}

func defaultRunCommand(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func checksumForAsset(checksumsPath string, assetName string) (string, error) {
	data, err := os.ReadFile(checksumsPath)
	if err != nil {
		return "", fmt.Errorf("update: read checksum catalog %q: %w", checksumsPath, err)
	}
	for line := range strings.SplitSeq(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		filename := strings.TrimPrefix(strings.TrimSpace(fields[1]), "*")
		if filename == assetName {
			return strings.TrimSpace(fields[0]), nil
		}
	}
	return "", fmt.Errorf("update: checksum catalog does not contain %s", assetName)
}

func verifySHA256(path string, expectedHex string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("update: open archive %q: %w", path, err)
	}
	defer func() {
		_ = file.Close()
	}()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return fmt.Errorf("update: hash archive %q: %w", path, err)
	}

	actual := hex.EncodeToString(hash.Sum(nil))
	expected := strings.ToLower(strings.TrimSpace(expectedHex))
	if actual != expected {
		return fmt.Errorf("update: checksum mismatch for %s: expected %s, got %s", path, expected, actual)
	}
	return nil
}

func extractBinaryFromTarGz(archivePath string, tempDir string, binaryName string) (string, os.FileMode, error) {
	file, err := os.Open(archivePath)
	if err != nil {
		return "", 0, fmt.Errorf("update: open archive %q: %w", archivePath, err)
	}
	defer func() {
		_ = file.Close()
	}()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return "", 0, fmt.Errorf("update: open gzip archive %q: %w", archivePath, err)
	}
	defer func() {
		_ = gzipReader.Close()
	}()

	tarReader := tar.NewReader(gzipReader)
	for {
		header, err := tarReader.Next()
		switch {
		case errors.Is(err, io.EOF):
			return "", 0, fmt.Errorf("update: archive %q did not contain %s", archivePath, binaryName)
		case err != nil:
			return "", 0, fmt.Errorf("update: read archive %q: %w", archivePath, err)
		case header == nil:
			continue
		case !isRegularArchiveFile(header):
			continue
		case filepath.Base(header.Name) != binaryName:
			continue
		}

		targetPath := filepath.Join(tempDir, binaryName)
		target, err := os.OpenFile(targetPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
		if err != nil {
			return "", 0, fmt.Errorf("update: create extracted binary %q: %w", targetPath, err)
		}
		if err := copyArchiveEntry(target, tarReader, header.Size); err != nil {
			_ = target.Close()
			return "", 0, fmt.Errorf("update: extract binary %q: %w", targetPath, err)
		}
		if err := target.Close(); err != nil {
			return "", 0, fmt.Errorf("update: close extracted binary %q: %w", targetPath, err)
		}

		mode := header.FileInfo().Mode().Perm()
		if mode != 0 {
			if err := os.Chmod(targetPath, mode); err != nil {
				return "", 0, fmt.Errorf("update: chmod extracted binary %q: %w", targetPath, err)
			}
		}
		return targetPath, mode, nil
	}
}

func isRegularArchiveFile(header *tar.Header) bool {
	return header != nil && (header.Typeflag == 0 || header.Typeflag == tar.TypeReg)
}

func copyArchiveEntry(target *os.File, reader io.Reader, size int64) error {
	if size <= 0 {
		return errors.New("archive entry size must be positive")
	}
	if size > maxExtractedBinaryBytes {
		return fmt.Errorf(
			"archive entry size %d exceeds the allowed limit of %d bytes",
			size,
			maxExtractedBinaryBytes,
		)
	}

	limitedReader := io.LimitReader(reader, size)
	copied, err := io.Copy(target, limitedReader)
	if err != nil {
		return err
	}
	if copied != size {
		return fmt.Errorf("archive entry truncated: copied %d of %d bytes", copied, size)
	}
	return nil
}

func siblingBackupPath(targetPath string, now time.Time) string {
	filename := filepath.Base(targetPath)
	return filepath.Join(
		filepath.Dir(targetPath),
		fmt.Sprintf(".%s.agh-backup-%d", filename, now.UnixNano()),
	)
}
