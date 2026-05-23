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

	"github.com/compozy/agh/internal/registry"
)

const (
	clientApplicationXGzipPath = "application/x-gzip"
)

const (
	clientApplicationGzipPath = "application/gzip"
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

var _ registry.Source = (*Client)(nil)

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
func WithRetryPolicy(initial, maxDelay time.Duration, retries int) Option {
	return func(client *Client) {
		client.initialBackoff = initial
		client.maxBackoff = maxDelay
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
func (c *Client) Download(
	ctx context.Context,
	slug string,
	opts registry.DownloadOpts,
) (*registry.DownloadResult, error) {
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

	response, checksum, contentSize, err := c.openDownloadResponse(ctx, repo, release, selection)
	if err != nil {
		return nil, err
	}

	reader, contentType, contentSize, err := finalizeDownloadResponse(
		response,
		repo.full,
		contentSize,
		normalizeArchiveSizeLimit(opts.MaxArchiveSize),
	)
	if err != nil {
		return nil, err
	}

	return &registry.DownloadResult{
		Reader:      reader,
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
	endpoint := c.baseURL + "/repos/" + url.PathEscape(
		repo.owner,
	) + "/" + url.PathEscape(
		repo.name,
	) + "/releases/latest"
	response, err := c.doRequest(ctx, endpoint, acceptJSON)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = response.Body.Close()
	}()

	switch response.StatusCode {
	case http.StatusOK:
		payload, err := io.ReadAll(response.Body)
		closeErr := closeResponseBody(response.Body, fmt.Sprintf("latest release response for %q", repo.full))
		if err != nil {
			return nil, joinErrors(
				fmt.Errorf("github: read latest release for %q: %w", repo.full, err),
				closeErr,
			)
		}
		if closeErr != nil {
			return nil, closeErr
		}

		var latest release
		if err := json.Unmarshal(payload, &latest); err != nil {
			return nil, fmt.Errorf("github: decode latest release for %q: %w", repo.full, err)
		}
		return &latest, nil
	case http.StatusUnauthorized:
		closeErr := closeResponseBody(
			response.Body,
			fmt.Sprintf("latest release response for %q", repo.full),
		)
		if closeErr != nil {
			return nil, joinErrors(privateRepositoryError(repo.full), closeErr)
		}
		return nil, privateRepositoryError(repo.full)
	case http.StatusNotFound:
		if err := closeResponseBody(
			response.Body,
			fmt.Sprintf("latest release response for %q", repo.full),
		); err != nil {
			return nil, err
		}
		releases, listErr := c.fetchReleasePage(ctx, repo)
		if listErr != nil {
			return nil, listErr
		}
		if len(releases) == 0 {
			return nil, fmt.Errorf("github: repository %q has no published releases", repo.full)
		}
		latest := releases[0]
		return &latest, nil
	default:
		err := responseError(response, "latest release", repo.full)
		return nil, joinErrors(
			err,
			closeResponseBody(response.Body, fmt.Sprintf("latest release response for %q", repo.full)),
		)
	}
}

func (c *Client) fetchRequestedRelease(ctx context.Context, repo repoSlug, version string) (*release, error) {
	if version == "" {
		return c.fetchLatestRelease(ctx, repo)
	}

	endpoint := c.baseURL + "/repos/" + url.PathEscape(
		repo.owner,
	) + "/" + url.PathEscape(
		repo.name,
	) + "/releases/tags/" + url.PathEscape(
		version,
	)
	response, err := c.doRequest(ctx, endpoint, acceptJSON)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = response.Body.Close()
	}()

	switch response.StatusCode {
	case http.StatusOK:
		payload, err := io.ReadAll(response.Body)
		closeErr := closeResponseBody(response.Body, fmt.Sprintf("release response for %q at %q", repo.full, version))
		if err != nil {
			return nil, joinErrors(
				fmt.Errorf("github: read release %q for %q: %w", version, repo.full, err),
				closeErr,
			)
		}
		if closeErr != nil {
			return nil, closeErr
		}

		var result release
		if err := json.Unmarshal(payload, &result); err != nil {
			return nil, fmt.Errorf("github: decode release %q for %q: %w", version, repo.full, err)
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
		if err := closeResponseBody(
			response.Body,
			fmt.Sprintf("release response for %q at %q", repo.full, version),
		); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("github: release %q not found for repository %q", version, repo.full)
	default:
		err := responseError(response, "release lookup", repo.full)
		return nil, joinErrors(
			err,
			closeResponseBody(response.Body, fmt.Sprintf("release response for %q at %q", repo.full, version)),
		)
	}
}

func (c *Client) fetchReleasePage(ctx context.Context, repo repoSlug) ([]release, error) {
	endpoint := c.baseURL + "/repos/" + url.PathEscape(
		repo.owner,
	) + "/" + url.PathEscape(
		repo.name,
	) + "/releases?per_page=30&page=1"
	response, err := c.doRequest(ctx, endpoint, acceptJSON)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = response.Body.Close()
	}()

	switch response.StatusCode {
	case http.StatusOK:
		payload, err := io.ReadAll(response.Body)
		closeErr := closeResponseBody(response.Body, fmt.Sprintf("releases page response for %q", repo.full))
		if err != nil {
			return nil, joinErrors(
				fmt.Errorf("github: read releases page for %q: %w", repo.full, err),
				closeErr,
			)
		}
		if closeErr != nil {
			return nil, closeErr
		}

		var releases []release
		if err := json.Unmarshal(payload, &releases); err != nil {
			return nil, fmt.Errorf("github: decode releases page for %q: %w", repo.full, err)
		}
		return filterPublishedReleases(releases), nil
	case http.StatusUnauthorized:
		closeErr := closeResponseBody(response.Body, fmt.Sprintf("releases page response for %q", repo.full))
		if closeErr != nil {
			return nil, joinErrors(privateRepositoryError(repo.full), closeErr)
		}
		return nil, privateRepositoryError(repo.full)
	case http.StatusNotFound:
		if err := closeResponseBody(
			response.Body,
			fmt.Sprintf("releases page response for %q", repo.full),
		); err != nil {
			return nil, err
		}
		return nil, repositoryNotFoundError(repo.full)
	default:
		err := responseError(response, "releases page", repo.full)
		return nil, joinErrors(
			err,
			closeResponseBody(response.Body, fmt.Sprintf("releases page response for %q", repo.full)),
		)
	}
}

func (c *Client) doRequest(
	ctx context.Context,
	rawURL string,
	accept string,
) (*http.Response, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("github: request aborted: %w", err)
	}
	if strings.TrimSpace(rawURL) == "" {
		return nil, errors.New("github: request URL is required")
	}

	backoff := c.initialBackoff
	var lastErr error

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		request, err := c.newRequest(ctx, rawURL, accept)
		if err != nil {
			return nil, err
		}

		response, err := c.httpClient.Do(request)
		if err != nil {
			lastErr, err = c.handleRequestError(ctx, err)
			if err != nil || attempt == c.maxRetries {
				if err != nil {
					return nil, err
				}
				return nil, lastErr
			}
			backoff, err = c.waitForRetry(ctx, backoff)
			if err != nil {
				return nil, err
			}
			continue
		}

		if err := c.checkRateLimit(response); err != nil {
			return nil, err
		}

		if shouldRetryStatus(response.StatusCode) && attempt < c.maxRetries {
			c.prepareRetryResponse(response, attempt)
			backoff, err = c.waitForRetry(ctx, backoff)
			if err != nil {
				return nil, err
			}
			continue
		}

		return response, nil
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("github: request failed: %s", rawURL)
	}
	return nil, lastErr
}

func (c *Client) openDownloadResponse(
	ctx context.Context,
	repo repoSlug,
	release *release,
	selection releaseSelection,
) (*http.Response, string, int64, error) {
	switch {
	case selection.asset != nil:
		response, err := c.doRequest(
			ctx,
			firstNonEmpty(selection.asset.URL, selection.asset.BrowserDownloadURL),
			acceptBinary,
		)
		if err != nil {
			return nil, "", -1, fmt.Errorf("github: download asset for %q: %w", repo.full, err)
		}
		size := int64(-1)
		if selection.asset.Size > 0 {
			size = selection.asset.Size
		}
		return response, strings.TrimSpace(selection.asset.Digest), size, nil
	case selection.useTarball:
		response, err := c.doRequest(ctx, strings.TrimSpace(release.TarballURL), acceptBinary)
		if err != nil {
			return nil, "", -1, fmt.Errorf("github: download source archive for %q: %w", repo.full, err)
		}
		return response, "", -1, nil
	default:
		return nil, "", -1, fmt.Errorf("github: no download candidate resolved for %q", repo.full)
	}
}

func finalizeDownloadResponse(
	response *http.Response,
	slug string,
	contentSize int64,
	maxArchiveSize int64,
) (_ io.ReadCloser, contentType string, finalSize int64, err error) {
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		err := responseError(response, "download", slug)
		return nil, "", 0, joinErrors(
			err,
			closeResponseBody(response.Body, fmt.Sprintf("download response for %q", slug)),
		)
	}

	contentType = strings.TrimSpace(response.Header.Get("Content-Type"))
	if err := validateDownloadContentType(contentType); err != nil {
		return nil, "", 0, joinErrors(
			err,
			closeResponseBody(response.Body, fmt.Sprintf("download response for %q", slug)),
		)
	}

	if response.ContentLength > 0 {
		if response.ContentLength > maxArchiveSize {
			return nil, "", 0, joinErrors(
				fmt.Errorf(
					"github: download for %q: %w: size=%d limit=%d",
					slug,
					registry.ErrArchiveTooLargeCompressed,
					response.ContentLength,
					maxArchiveSize,
				),
				closeResponseBody(response.Body, fmt.Sprintf("download response for %q", slug)),
			)
		}
		contentSize = response.ContentLength
	}

	reader, written, err := spoolDownloadResponse(response.Body, slug, maxArchiveSize)
	closeErr := closeResponseBody(response.Body, fmt.Sprintf("download response for %q", slug))
	if err != nil {
		return nil, "", 0, joinErrors(fmt.Errorf("github: spool download for %q: %w", slug, err), closeErr)
	}
	if closeErr != nil {
		return nil, "", 0, closeErr
	}
	if contentSize <= 0 {
		contentSize = written
	}
	response.Body = http.NoBody

	return reader, contentType, contentSize, nil
}

func (c *Client) newRequest(ctx context.Context, rawURL string, accept string) (*http.Request, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("github: create request %q: %w", rawURL, err)
	}
	request.Header.Set("Accept", accept)
	if token := strings.TrimSpace(c.token); token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	return request, nil
}

func (c *Client) handleRequestError(ctx context.Context, err error) (error, error) {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || ctx.Err() != nil {
		return nil, fmt.Errorf("github: request failed: %w", err)
	}
	return fmt.Errorf("github: request failed: %w", err), nil
}

func (c *Client) waitForRetry(ctx context.Context, backoff time.Duration) (time.Duration, error) {
	if err := c.sleep(ctx, backoff); err != nil {
		return 0, fmt.Errorf("github: retry wait aborted: %w", err)
	}
	return nextBackoff(backoff, c.maxBackoff), nil
}

func shouldRetryStatus(statusCode int) bool {
	return statusCode == http.StatusTooManyRequests || statusCode >= http.StatusInternalServerError
}

func (c *Client) prepareRetryResponse(response *http.Response, attempt int) {
	closeErr := closeResponseBody(
		response.Body,
		fmt.Sprintf("retry response for %s", requestURLString(response)),
	)
	if closeErr != nil && c.logger != nil {
		c.logger.Debug(
			"github: close response body before retry",
			"error",
			closeErr,
			"url",
			requestURLString(response),
			"attempt",
			attempt+1,
		)
	}
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
		if c.logger != nil {
			c.logger.Debug(
				"github: invalid X-RateLimit-Remaining header",
				"value",
				remainingValue,
				"error",
				err,
				"url",
				requestURLString(response),
			)
		}
		return nil
	}
	if remaining == 0 {
		if response.StatusCode == http.StatusForbidden || response.StatusCode == http.StatusTooManyRequests {
			rateLimitErr := errors.New("github: rate limit exceeded; set GITHUB_TOKEN for higher limits")
			return joinErrors(
				rateLimitErr,
				closeResponseBody(response.Body, fmt.Sprintf("rate limit response for %s", requestURLString(response))),
			)
		}
		if c.logger != nil {
			c.logger.Warn(
				"github: rate limit exhausted after successful response",
				"status", response.StatusCode,
				"url", requestURLString(response),
			)
		}
	}
	if remaining < rateLimitWarnThreshold && c.logger != nil {
		c.logger.Warn("github: rate limit running low", "remaining", remaining, "url", requestURLString(response))
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
		return releaseSelection{}, fmt.Errorf(
			"asset %q not found; available assets: %s",
			requestedAsset,
			strings.Join(assetNames(release.Assets), ", "),
		)
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
		return releaseSelection{}, fmt.Errorf(
			"multiple .tar.gz assets found: %s; specify one with --asset",
			strings.Join(assetNames(candidates), ", "),
		)
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
	case clientApplicationGzipPath, clientApplicationXGzipPath, "application/octet-stream":
		return nil
	default:
		return fmt.Errorf("github: unexpected download content type %q", trimmed)
	}
}

func privateRepositoryError(slug string) error {
	return fmt.Errorf(
		"github: repository %q requires authentication; set GITHUB_TOKEN to access private repositories",
		slug,
	)
}

func repositoryNotFoundError(slug string) error {
	return fmt.Errorf("github: repository %q not found: %w", slug, registry.NewPackageNotFoundError(slug))
}

func responseError(response *http.Response, operation string, slug string) error {
	message := readErrorMessage(response.Body)
	if message == "" {
		return fmt.Errorf("github: %s request failed for %q: %s", operation, slug, response.Status)
	}
	return fmt.Errorf("github: %s request failed for %q: %s: %s", operation, slug, response.Status, message)
}

func readErrorMessage(body io.Reader) string {
	payload, err := io.ReadAll(io.LimitReader(body, maxErrorBodyBytes))
	if err != nil {
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

func nextBackoff(current time.Duration, maxDelay time.Duration) time.Duration {
	if current <= 0 {
		return defaultInitialBackoff
	}
	if maxDelay <= 0 {
		maxDelay = defaultMaxBackoff
	}

	next := current * 2
	if next > maxDelay {
		return maxDelay
	}
	return next
}

func spoolDownloadResponse(body io.Reader, slug string, maxBytes int64) (_ io.ReadCloser, size int64, err error) {
	file, err := os.CreateTemp("", "agh-github-download-*")
	if err != nil {
		return nil, 0, fmt.Errorf("create temp download file for %q: %w", slug, err)
	}
	defer func() {
		if err != nil {
			_ = os.Remove(file.Name())
		}
	}()

	limit := normalizeArchiveSizeLimit(maxBytes)
	written, err := io.Copy(file, io.LimitReader(body, limit+1))
	if err != nil {
		closeErr := file.Close()
		return nil, 0, joinErrors(
			fmt.Errorf("write temp download file for %q: %w", slug, err),
			closeErr,
		)
	}
	if written > limit {
		closeErr := file.Close()
		return nil, written, joinErrors(
			fmt.Errorf(
				"%w: github download for %q exceeds compressed archive limit %d",
				registry.ErrArchiveTooLargeCompressed,
				slug,
				limit,
			),
			closeErr,
		)
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		closeErr := file.Close()
		return nil, 0, joinErrors(
			fmt.Errorf("rewind temp download file for %q: %w", slug, err),
			closeErr,
		)
	}

	return &tempFileReadCloser{File: file, path: file.Name()}, written, nil
}

func normalizeArchiveSizeLimit(limit int64) int64 {
	if limit > 0 {
		return limit
	}
	return registry.DefaultMaxArchiveSize
}

type tempFileReadCloser struct {
	*os.File
	path string
}

func (r *tempFileReadCloser) Close() error {
	if r == nil {
		return nil
	}
	closeErr := r.File.Close()
	removeErr := os.Remove(r.path)
	if removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
		return errors.Join(closeErr, fmt.Errorf("remove temp download file %q: %w", r.path, removeErr))
	}
	return closeErr
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

func requestURLString(response *http.Response) string {
	if response == nil || response.Request == nil || response.Request.URL == nil {
		return ""
	}
	return response.Request.URL.String()
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
