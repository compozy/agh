// Package github implements the GitHub Releases registry adapter.
package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pedronauck/agh/internal/registry"
)

const (
	defaultBaseURL          = "https://api.github.com"
	defaultRequestTimeout   = 30 * time.Second
	defaultInitialBackoff   = time.Second
	defaultMaxBackoff       = 30 * time.Second
	defaultMaxRetries       = 3
	maxErrorBodyBytes       = 64 << 10
	rateLimitWarnThreshold  = 10
	acceptJSON              = "application/vnd.github+json"
	acceptBinary            = "application/octet-stream"
	githubRepositoryBaseURL = "https://github.com"
)

// Option customizes a GitHub client.
type Option func(*Client)

// Client implements the GitHub Releases registry source.
type Client struct {
	baseURL        string
	httpClient     *http.Client
	sleep          func(context.Context, time.Duration) error
	initialBackoff time.Duration
	maxBackoff     time.Duration
	maxRetries     int
	logger         *slog.Logger
	token          string
	closeOnce      sync.Once
}

type repoSlug struct {
	owner string
	name  string
	full  string
}

type release struct {
	Name       string         `json:"name"`
	Body       string         `json:"body"`
	TagName    string         `json:"tag_name"`
	Draft      bool           `json:"draft"`
	Prerelease bool           `json:"prerelease"`
	TarballURL string         `json:"tarball_url"`
	Assets     []releaseAsset `json:"assets"`
	Author     releaseAuthor  `json:"author"`
}

type releaseAuthor struct {
	Login string `json:"login"`
}

type releaseAsset struct {
	URL                string `json:"url"`
	Name               string `json:"name"`
	ContentType        string `json:"content_type"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Digest             string `json:"digest"`
	Size               int64  `json:"size"`
	DownloadCount      int    `json:"download_count"`
}

type releaseSelection struct {
	asset      *releaseAsset
	useTarball bool
}

var _ registry.RegistrySource = (*Client)(nil)

// NewClient constructs a GitHub Releases registry client.
func NewClient(baseURL string, opts ...Option) *Client {
	client := &Client{
		baseURL:        strings.TrimSpace(baseURL),
		httpClient:     &http.Client{Timeout: defaultRequestTimeout},
		sleep:          sleepContext,
		initialBackoff: defaultInitialBackoff,
		maxBackoff:     defaultMaxBackoff,
		maxRetries:     defaultMaxRetries,
		logger:         slog.Default(),
		token:          strings.TrimSpace(os.Getenv("GITHUB_TOKEN")),
	}
	if client.baseURL == "" {
		client.baseURL = defaultBaseURL
	}

	for _, opt := range opts {
		if opt != nil {
			opt(client)
		}
	}

	if strings.TrimSpace(client.baseURL) == "" {
		client.baseURL = defaultBaseURL
	}
	client.baseURL = strings.TrimRight(client.baseURL, "/")

	if client.httpClient == nil {
		client.httpClient = &http.Client{Timeout: defaultRequestTimeout}
	}
	if client.sleep == nil {
		client.sleep = sleepContext
	}
	if client.initialBackoff <= 0 {
		client.initialBackoff = defaultInitialBackoff
	}
	if client.maxBackoff <= 0 {
		client.maxBackoff = defaultMaxBackoff
	}
	if client.maxRetries < 0 {
		client.maxRetries = 0
	}
	if client.logger == nil {
		client.logger = slog.Default()
	}

	return client
}

// WithHTTPClient overrides the underlying HTTP client.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(client *Client) {
		client.httpClient = httpClient
	}
}

// WithSleep overrides the retry sleep function.
func WithSleep(sleep func(context.Context, time.Duration) error) Option {
	return func(client *Client) {
		client.sleep = sleep
	}
}

// WithRetryPolicy overrides the retry policy used for retryable responses.
func WithRetryPolicy(initial, max time.Duration, retries int) Option {
	return func(client *Client) {
		client.initialBackoff = initial
		client.maxBackoff = max
		client.maxRetries = retries
	}
}

// WithLogger overrides the logger used for rate-limit warnings.
func WithLogger(logger *slog.Logger) Option {
	return func(client *Client) {
		client.logger = logger
	}
}

// WithToken overrides the GitHub token used for authenticated requests.
func WithToken(token string) Option {
	return func(client *Client) {
		client.token = strings.TrimSpace(token)
	}
}

// Name reports the registry source name.
func (c *Client) Name() string {
	return "github"
}

// Capabilities reports which registry operations GitHub supports.
func (c *Client) Capabilities() registry.SourceCaps {
	return registry.SourceCaps{Search: false}
}

// Search reports that the GitHub Releases adapter is slug-only.
func (c *Client) Search(context.Context, string, registry.SearchOpts) ([]registry.Listing, error) {
	return nil, registry.ErrNotSupported
}

// Info fetches metadata from the latest published release and page one of the
// releases listing.
func (c *Client) Info(ctx context.Context, slug string) (*registry.Detail, error) {
	repo, err := parseRepoSlug(slug)
	if err != nil {
		return nil, err
	}

	latest, err := c.fetchLatestRelease(ctx, repo)
	if err != nil {
		return nil, err
	}
	releases, err := c.fetchReleasePage(ctx, repo)
	if err != nil {
		return nil, err
	}

	return &registry.Detail{
		Listing: registry.Listing{
			Slug:        repo.full,
			Name:        firstNonEmpty(latest.Name, repo.name),
			Description: releaseDescription(latest),
			Author:      firstNonEmpty(latest.Author.Login, repo.owner),
			Version:     strings.TrimSpace(latest.TagName),
			Downloads:   releaseDownloadCount(latest),
			Source:      c.Name(),
		},
		Readme:     strings.TrimSpace(latest.Body),
		Repository: githubRepositoryBaseURL + "/" + repo.full,
		Versions:   releaseVersions(releases),
	}, nil
}

// Download fetches the selected release archive stream.
func (c *Client) Download(ctx context.Context, slug string, opts registry.DownloadOpts) (*registry.DownloadResult, error) {
	repo, err := parseRepoSlug(slug)
	if err != nil {
		return nil, err
	}

	release, err := c.fetchRequestedRelease(ctx, repo, strings.TrimSpace(opts.Version))
	if err != nil {
		return nil, err
	}

	selection, err := selectReleaseDownload(release, strings.TrimSpace(opts.Asset))
	if err != nil {
		return nil, fmt.Errorf("github: resolve release asset for %q: %w", repo.full, err)
	}

	var (
		response    *http.Response
		checksum    string
		contentSize int64 = -1
	)

	switch {
	case selection.asset != nil:
		response, err = c.doRequest(ctx, http.MethodGet, firstNonEmpty(selection.asset.URL, selection.asset.BrowserDownloadURL), acceptBinary, true)
		if err != nil {
			return nil, fmt.Errorf("github: download asset for %q: %w", repo.full, err)
		}
		checksum = strings.TrimSpace(selection.asset.Digest)
		if selection.asset.Size > 0 {
			contentSize = selection.asset.Size
		}
	case selection.useTarball:
		response, err = c.doRequest(ctx, http.MethodGet, strings.TrimSpace(release.TarballURL), acceptBinary, true)
		if err != nil {
			return nil, fmt.Errorf("github: download source archive for %q: %w", repo.full, err)
		}
	default:
		return nil, fmt.Errorf("github: no download candidate resolved for %q", repo.full)
	}

	contentType := strings.TrimSpace(response.Header.Get("Content-Type"))
	if err := validateDownloadContentType(contentType); err != nil {
		_ = response.Body.Close()
		return nil, err
	}

	if response.ContentLength > 0 {
		contentSize = response.ContentLength
	}

	return &registry.DownloadResult{
		Reader:      response.Body,
		Slug:        repo.full,
		Version:     strings.TrimSpace(release.TagName),
		ContentSize: contentSize,
		Checksum:    checksum,
		ContentType: contentType,
	}, nil
}

// Close releases any idle HTTP connections held by the client.
func (c *Client) Close() error {
	c.closeOnce.Do(func() {
		closeIdleConnections(c.httpClient)
	})
	return nil
}

func (c *Client) fetchLatestRelease(ctx context.Context, repo repoSlug) (*release, error) {
	endpoint := c.baseURL + "/repos/" + url.PathEscape(repo.owner) + "/" + url.PathEscape(repo.name) + "/releases/latest"
	response, err := c.doRequest(ctx, http.MethodGet, endpoint, acceptJSON, false)
	if err != nil {
		return nil, err
	}

	switch response.StatusCode {
	case http.StatusOK:
		var latest release
		if err := json.NewDecoder(response.Body).Decode(&latest); err != nil {
			return nil, joinErrors(
				fmt.Errorf("github: decode latest release for %q: %w", repo.full, err),
				closeResponseBody(response.Body, fmt.Sprintf("latest release response for %q", repo.full)),
			)
		}
		if err := closeResponseBody(response.Body, fmt.Sprintf("latest release response for %q", repo.full)); err != nil {
			return nil, err
		}
		return &latest, nil
	case http.StatusUnauthorized:
		closeErr := closeResponseBody(response.Body, fmt.Sprintf("latest release response for %q", repo.full))
		if closeErr != nil {
			return nil, joinErrors(privateRepositoryError(repo.full), closeErr)
		}
		return nil, privateRepositoryError(repo.full)
	case http.StatusNotFound:
		if err := closeResponseBody(response.Body, fmt.Sprintf("latest release response for %q", repo.full)); err != nil {
			return nil, err
		}
		releases, listErr := c.fetchReleasePage(ctx, repo)
		if listErr != nil {
			return nil, listErr
		}
		if len(releases) == 0 {
			return nil, fmt.Errorf("github: repository %q has no published releases", repo.full)
		}
		return nil, fmt.Errorf("github: latest release not found for %q", repo.full)
	default:
		return nil, responseError(response, "latest release", repo.full)
	}
}

func (c *Client) fetchRequestedRelease(ctx context.Context, repo repoSlug, version string) (*release, error) {
	if version == "" {
		return c.fetchLatestRelease(ctx, repo)
	}

	endpoint := c.baseURL + "/repos/" + url.PathEscape(repo.owner) + "/" + url.PathEscape(repo.name) + "/releases/tags/" + url.PathEscape(version)
	response, err := c.doRequest(ctx, http.MethodGet, endpoint, acceptJSON, false)
	if err != nil {
		return nil, err
	}

	switch response.StatusCode {
	case http.StatusOK:
		var result release
		if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
			return nil, joinErrors(
				fmt.Errorf("github: decode release %q for %q: %w", version, repo.full, err),
				closeResponseBody(response.Body, fmt.Sprintf("release response for %q at %q", repo.full, version)),
			)
		}
		if err := closeResponseBody(response.Body, fmt.Sprintf("release response for %q at %q", repo.full, version)); err != nil {
			return nil, err
		}
		if result.Draft || result.Prerelease {
			return nil, fmt.Errorf("github: release %q for %q is not a published full release", version, repo.full)
		}
		return &result, nil
	case http.StatusUnauthorized:
		closeErr := closeResponseBody(response.Body, fmt.Sprintf("release response for %q at %q", repo.full, version))
		if closeErr != nil {
			return nil, joinErrors(privateRepositoryError(repo.full), closeErr)
		}
		return nil, privateRepositoryError(repo.full)
	case http.StatusNotFound:
		if err := closeResponseBody(response.Body, fmt.Sprintf("release response for %q at %q", repo.full, version)); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("github: release %q not found for repository %q", version, repo.full)
	default:
		return nil, responseError(response, "release lookup", repo.full)
	}
}

func (c *Client) fetchReleasePage(ctx context.Context, repo repoSlug) ([]release, error) {
	endpoint := c.baseURL + "/repos/" + url.PathEscape(repo.owner) + "/" + url.PathEscape(repo.name) + "/releases?per_page=30&page=1"
	response, err := c.doRequest(ctx, http.MethodGet, endpoint, acceptJSON, false)
	if err != nil {
		return nil, err
	}

	switch response.StatusCode {
	case http.StatusOK:
		var releases []release
		if err := json.NewDecoder(response.Body).Decode(&releases); err != nil {
			return nil, joinErrors(
				fmt.Errorf("github: decode releases page for %q: %w", repo.full, err),
				closeResponseBody(response.Body, fmt.Sprintf("releases page response for %q", repo.full)),
			)
		}
		if err := closeResponseBody(response.Body, fmt.Sprintf("releases page response for %q", repo.full)); err != nil {
			return nil, err
		}
		return filterPublishedReleases(releases), nil
	case http.StatusUnauthorized:
		closeErr := closeResponseBody(response.Body, fmt.Sprintf("releases page response for %q", repo.full))
		if closeErr != nil {
			return nil, joinErrors(privateRepositoryError(repo.full), closeErr)
		}
		return nil, privateRepositoryError(repo.full)
	case http.StatusNotFound:
		if err := closeResponseBody(response.Body, fmt.Sprintf("releases page response for %q", repo.full)); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("github: repository %q not found", repo.full)
	default:
		return nil, responseError(response, "releases page", repo.full)
	}
}

func (c *Client) doRequest(ctx context.Context, method string, rawURL string, accept string, binary bool) (*http.Response, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("github: request aborted: %w", err)
	}
	if strings.TrimSpace(rawURL) == "" {
		return nil, errors.New("github: request URL is required")
	}

	backoff := c.initialBackoff
	var lastErr error

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		request, err := http.NewRequestWithContext(ctx, method, rawURL, nil)
		if err != nil {
			return nil, fmt.Errorf("github: create request %q: %w", rawURL, err)
		}
		request.Header.Set("Accept", accept)
		if token := strings.TrimSpace(c.token); token != "" {
			request.Header.Set("Authorization", "Bearer "+token)
		}

		response, err := c.httpClient.Do(request)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || ctx.Err() != nil {
				return nil, fmt.Errorf("github: request failed: %w", err)
			}

			lastErr = fmt.Errorf("github: request failed: %w", err)
			if attempt == c.maxRetries {
				return nil, lastErr
			}

			if err := c.sleep(ctx, backoff); err != nil {
				return nil, fmt.Errorf("github: retry wait aborted: %w", err)
			}
			backoff = nextBackoff(backoff, c.maxBackoff)
			continue
		}

		if err := c.checkRateLimit(response); err != nil {
			return nil, err
		}

		retryable := response.StatusCode == http.StatusTooManyRequests || response.StatusCode >= http.StatusInternalServerError
		if retryable && attempt < c.maxRetries {
			_ = response.Body.Close()
			if err := c.sleep(ctx, backoff); err != nil {
				return nil, fmt.Errorf("github: retry wait aborted: %w", err)
			}
			backoff = nextBackoff(backoff, c.maxBackoff)
			continue
		}

		if binary {
			return response, nil
		}
		if response.StatusCode >= http.StatusOK && response.StatusCode < http.StatusMultipleChoices {
			return response, nil
		}
		return response, nil
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("github: request failed: %s", rawURL)
	}
	return nil, lastErr
}

func (c *Client) checkRateLimit(response *http.Response) error {
	if response == nil {
		return nil
	}
	remainingValue := strings.TrimSpace(response.Header.Get("X-RateLimit-Remaining"))
	if remainingValue == "" {
		return nil
	}
	remaining, err := strconv.Atoi(remainingValue)
	if err != nil {
		return nil
	}
	if remaining == 0 {
		_ = response.Body.Close()
		return errors.New("github: rate limit exceeded; set GITHUB_TOKEN for higher limits")
	}
	if remaining < rateLimitWarnThreshold && c.logger != nil {
		c.logger.Warn("github: rate limit running low", "remaining", remaining, "url", response.Request.URL.String())
	}
	return nil
}

func parseRepoSlug(slug string) (repoSlug, error) {
	trimmed := strings.TrimSpace(slug)
	parts := strings.Split(trimmed, "/")
	if len(parts) != 2 {
		return repoSlug{}, fmt.Errorf("github: slug %q must be in owner/repo format", trimmed)
	}

	owner := strings.TrimSpace(parts[0])
	name := strings.TrimSpace(parts[1])
	if owner == "" || name == "" {
		return repoSlug{}, fmt.Errorf("github: slug %q must be in owner/repo format", trimmed)
	}

	return repoSlug{
		owner: owner,
		name:  name,
		full:  owner + "/" + name,
	}, nil
}

func filterPublishedReleases(releases []release) []release {
	filtered := make([]release, 0, len(releases))
	for _, release := range releases {
		if release.Draft || release.Prerelease {
			continue
		}
		filtered = append(filtered, release)
	}
	return filtered
}

func releaseVersions(releases []release) []string {
	versions := make([]string, 0, len(releases))
	for _, release := range releases {
		if tag := strings.TrimSpace(release.TagName); tag != "" {
			versions = append(versions, tag)
		}
	}
	return versions
}

func releaseDownloadCount(release *release) int {
	if release == nil {
		return 0
	}
	total := 0
	for _, asset := range release.Assets {
		total += asset.DownloadCount
	}
	return total
}

func releaseDescription(release *release) string {
	if release == nil {
		return ""
	}
	if name := strings.TrimSpace(release.Name); name != "" {
		return name
	}
	body := strings.TrimSpace(release.Body)
	if body == "" {
		return ""
	}
	line, _, _ := strings.Cut(body, "\n")
	return strings.TrimSpace(line)
}

func selectReleaseDownload(release *release, requestedAsset string) (releaseSelection, error) {
	if release == nil {
		return releaseSelection{}, errors.New("release metadata is required")
	}

	candidates := make([]releaseAsset, 0, len(release.Assets))
	for _, asset := range release.Assets {
		if strings.HasSuffix(strings.ToLower(strings.TrimSpace(asset.Name)), ".tar.gz") {
			candidates = append(candidates, asset)
		}
	}

	if requestedAsset != "" {
		for _, asset := range release.Assets {
			if strings.TrimSpace(asset.Name) != requestedAsset {
				continue
			}
			if !strings.HasSuffix(strings.ToLower(strings.TrimSpace(asset.Name)), ".tar.gz") {
				return releaseSelection{}, fmt.Errorf("asset %q is not a .tar.gz archive", requestedAsset)
			}
			selected := asset
			return releaseSelection{asset: &selected}, nil
		}
		return releaseSelection{}, fmt.Errorf("asset %q not found; available assets: %s", requestedAsset, strings.Join(assetNames(release.Assets), ", "))
	}

	switch len(candidates) {
	case 0:
		if strings.TrimSpace(release.TarballURL) == "" {
			return releaseSelection{}, errors.New("release has no .tar.gz assets and no source archive fallback")
		}
		return releaseSelection{useTarball: true}, nil
	case 1:
		selected := candidates[0]
		return releaseSelection{asset: &selected}, nil
	default:
		return releaseSelection{}, fmt.Errorf("multiple .tar.gz assets found: %s; specify one with --asset", strings.Join(assetNames(candidates), ", "))
	}
}

func assetNames[T interface{ GetName() string }](assets []T) []string {
	names := make([]string, 0, len(assets))
	for _, asset := range assets {
		names = append(names, asset.GetName())
	}
	slices.Sort(names)
	return names
}

func (a releaseAsset) GetName() string {
	return strings.TrimSpace(a.Name)
}

func validateDownloadContentType(contentType string) error {
	trimmed := strings.TrimSpace(contentType)
	if trimmed == "" {
		return errors.New("github: download missing Content-Type header")
	}

	mediaType, _, err := mime.ParseMediaType(trimmed)
	if err != nil {
		return fmt.Errorf("github: parse Content-Type %q: %w", trimmed, err)
	}

	switch mediaType {
	case "application/gzip", "application/x-gzip", "application/octet-stream":
		return nil
	default:
		return fmt.Errorf("github: unexpected download content type %q", trimmed)
	}
}

func privateRepositoryError(slug string) error {
	return fmt.Errorf("github: repository %q requires authentication; set GITHUB_TOKEN to access private repositories", slug)
}

func responseError(response *http.Response, operation string, slug string) error {
	message := readErrorMessage(response.Body)
	if message == "" {
		return fmt.Errorf("github: %s request failed for %q: %s", operation, slug, response.Status)
	}
	return fmt.Errorf("github: %s request failed for %q: %s: %s", operation, slug, response.Status, message)
}

func readErrorMessage(body io.ReadCloser) string {
	payload, err := io.ReadAll(io.LimitReader(body, maxErrorBodyBytes))
	closeErr := body.Close()
	if err != nil || closeErr != nil {
		return ""
	}

	trimmed := strings.TrimSpace(string(payload))
	if trimmed == "" {
		return ""
	}

	var envelope struct {
		Message string `json:"message"`
		Error   string `json:"error"`
	}
	if err := json.Unmarshal(payload, &envelope); err == nil {
		for _, candidate := range []string{envelope.Message, envelope.Error} {
			if candidate = strings.TrimSpace(candidate); candidate != "" {
				return candidate
			}
		}
	}

	return trimmed
}

func closeResponseBody(body io.Closer, context string) error {
	if body == nil {
		return nil
	}
	if err := body.Close(); err != nil {
		return fmt.Errorf("github: close %s: %w", context, err)
	}
	return nil
}

func joinErrors(errs ...error) error {
	filtered := make([]error, 0, len(errs))
	for _, err := range errs {
		if err != nil {
			filtered = append(filtered, err)
		}
	}
	return errors.Join(filtered...)
}

func sleepContext(ctx context.Context, wait time.Duration) error {
	if wait <= 0 {
		return nil
	}

	timer := time.NewTimer(wait)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func nextBackoff(current time.Duration, max time.Duration) time.Duration {
	if current <= 0 {
		return defaultInitialBackoff
	}
	if max <= 0 {
		max = defaultMaxBackoff
	}

	next := current * 2
	if next > max {
		return max
	}
	return next
}

func closeIdleConnections(httpClient *http.Client) {
	if httpClient == nil {
		return
	}
	if httpClient.Transport == nil {
		if transport, ok := http.DefaultTransport.(interface{ CloseIdleConnections() }); ok {
			transport.CloseIdleConnections()
		}
		return
	}
	if transport, ok := httpClient.Transport.(interface{ CloseIdleConnections() }); ok {
		transport.CloseIdleConnections()
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
