// Package clawhub implements the ClawHub registry adapter.
package clawhub

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/compozy/agh/internal/registry"
)

const (
	sourceName            = "clawhub"
	defaultBaseURL        = "https://clawhub.ai/api/v1"
	defaultRequestTimeout = 30 * time.Second
	defaultInitialBackoff = time.Second
	defaultMaxBackoff     = 30 * time.Second
	defaultMaxRetries     = 3
	maxErrorBodyBytes     = 64 << 10
	maxJSONResponseBytes  = 1 << 20
)

var errResponseTooLarge = errors.New("clawhub: response exceeds max size")

// Option customizes a ClawHub client.
type Option func(*Client)

// Client implements the ClawHub registry source.
type Client struct {
	baseURL        string
	httpClient     *http.Client
	sleep          func(context.Context, time.Duration) error
	initialBackoff time.Duration
	maxBackoff     time.Duration
	maxRetries     int
	closeOnce      sync.Once
}

var _ registry.Source = (*Client)(nil)

// NewClient constructs a ClawHub registry client.
func NewClient(baseURL string, opts ...Option) *Client {
	client := &Client{
		baseURL:        strings.TrimSpace(baseURL),
		httpClient:     &http.Client{Timeout: defaultRequestTimeout},
		sleep:          sleepContext,
		initialBackoff: defaultInitialBackoff,
		maxBackoff:     defaultMaxBackoff,
		maxRetries:     defaultMaxRetries,
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
	client.baseURL = normalizeBaseURL(client.baseURL)

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

// Name reports the registry source name.
func (c *Client) Name() string {
	return sourceName
}

// Capabilities reports which registry operations ClawHub supports.
func (c *Client) Capabilities() registry.SourceCaps {
	return registry.SourceCaps{Search: true}
}

// Search queries ClawHub for skill listings.
func (c *Client) Search(ctx context.Context, query string, opts registry.SearchOpts) ([]registry.Listing, error) {
	trimmedQuery := strings.TrimSpace(query)
	if trimmedQuery == "" {
		return nil, errors.New("clawhub: search query is required")
	}
	if opts.Type == registry.PackageTypeExtension {
		return []registry.Listing{}, nil
	}

	values := url.Values{}
	values.Set("q", trimmedQuery)
	if opts.Limit > 0 {
		values.Set("limit", fmt.Sprintf("%d", opts.Limit))
	}
	if opts.Offset > 0 {
		values.Set("offset", fmt.Sprintf("%d", opts.Offset))
	}

	response, err := c.doRequest(ctx, "/skills", values, "search", "")
	if err != nil {
		return nil, err
	}

	payload, err := readLimitedBody(response.Body, maxJSONResponseBytes)
	closeErr := response.Body.Close()
	if err != nil {
		return nil, joinErrors(
			fmt.Errorf("clawhub: read search response: %w", err),
			wrapSearchCloseError(closeErr),
		)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("clawhub: close search response: %w", closeErr)
	}

	listings, err := decodeListings(bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("clawhub: decode search response: %w", err)
	}
	for index := range listings {
		listings[index].Source = c.Name()
		listings[index].Type = registry.PackageTypeSkill
	}

	return listings, nil
}

// Info fetches the full metadata for one skill slug.
func (c *Client) Info(ctx context.Context, slug string) (*registry.Detail, error) {
	trimmedSlug := strings.TrimSpace(slug)
	if trimmedSlug == "" {
		return nil, errors.New("clawhub: skill slug is required")
	}

	response, err := c.doRequest(ctx, "/skills/"+url.PathEscape(trimmedSlug), nil, "info", trimmedSlug)
	if err != nil {
		return nil, err
	}

	payload, err := readLimitedBody(response.Body, maxJSONResponseBytes)
	closeErr := response.Body.Close()
	if err != nil {
		return nil, joinErrors(
			fmt.Errorf("clawhub: read info response for %q: %w", trimmedSlug, err),
			wrapCloseError(trimmedSlug, closeErr),
		)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("clawhub: close info response for %q: %w", trimmedSlug, closeErr)
	}

	var detail registry.Detail
	if err := json.Unmarshal(payload, &detail); err != nil {
		return nil, fmt.Errorf("clawhub: decode info response for %q: %w", trimmedSlug, err)
	}

	detail.Slug = firstNonEmpty(strings.TrimSpace(detail.Slug), trimmedSlug)
	detail.Source = c.Name()
	detail.Type = registry.PackageTypeSkill
	return &detail, nil
}

// Download fetches the archived skill package stream for one skill slug.
func (c *Client) Download(
	ctx context.Context,
	slug string,
	opts registry.DownloadOpts,
) (*registry.DownloadResult, error) {
	trimmedSlug := strings.TrimSpace(slug)
	if trimmedSlug == "" {
		return nil, errors.New("clawhub: skill slug is required")
	}

	requestPath := "/skills/" + url.PathEscape(trimmedSlug) + "/download"
	if version := strings.TrimSpace(opts.Version); version != "" {
		requestPath = "/skills/" + url.PathEscape(trimmedSlug) + "/versions/" + url.PathEscape(version) + "/archive"
	}

	response, err := c.doRequest(ctx, requestPath, nil, "download", trimmedSlug)
	if err != nil {
		return nil, err
	}
	maxArchiveSize := normalizeArchiveSizeLimit(opts.MaxArchiveSize)
	if response.ContentLength > maxArchiveSize {
		closeErr := response.Body.Close()
		return nil, joinErrors(
			fmt.Errorf(
				"clawhub: download for %q: %w: size=%d limit=%d",
				trimmedSlug,
				registry.ErrArchiveTooLargeCompressed,
				response.ContentLength,
				maxArchiveSize,
			),
			wrapCloseError(trimmedSlug, closeErr),
		)
	}
	downloadReader, downloadSize, spoolErr := spoolDownloadResponse(response.Body, trimmedSlug, maxArchiveSize)
	closeErr := response.Body.Close()
	if spoolErr != nil {
		return nil, joinErrors(
			fmt.Errorf("clawhub: spool download for %q: %w", trimmedSlug, spoolErr),
			wrapCloseError(trimmedSlug, closeErr),
		)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("clawhub: close download response for %q: %w", trimmedSlug, closeErr)
	}

	return &registry.DownloadResult{
		Reader:      downloadReader,
		Slug:        trimmedSlug,
		Version:     strings.TrimSpace(response.Header.Get("X-Skill-Version")),
		ContentSize: contentSize(firstPositiveInt64(response.ContentLength, downloadSize)),
		ContentType: strings.TrimSpace(response.Header.Get("Content-Type")),
	}, nil
}

// Close releases any idle HTTP connections held by the client.
func (c *Client) Close() error {
	c.closeOnce.Do(func() {
		closeIdleConnections(c.httpClient)
	})
	return nil
}

func (c *Client) doRequest(
	ctx context.Context,
	requestPath string,
	query url.Values,
	operation string,
	slug string,
) (*http.Response, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("clawhub: %s request aborted: %w", operation, err)
	}

	requestURL, err := c.buildURL(requestPath, query)
	if err != nil {
		return nil, fmt.Errorf("clawhub: build %s request URL: %w", operation, err)
	}

	backoff := c.initialBackoff
	var lastErr error

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, http.NoBody)
		if err != nil {
			return nil, fmt.Errorf("clawhub: create %s request: %w", operation, err)
		}
		request.Header.Set("Accept", "application/json, application/gzip, application/octet-stream")

		response, err := c.httpClient.Do(request)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || ctx.Err() != nil {
				return nil, fmt.Errorf("clawhub: %s request failed: %w", operation, err)
			}

			lastErr = fmt.Errorf("clawhub: %s request failed: %w", operation, err)
			if attempt == c.maxRetries {
				return nil, lastErr
			}

			if err := c.sleep(ctx, backoff); err != nil {
				return nil, fmt.Errorf("clawhub: %s retry wait aborted: %w", operation, err)
			}
			backoff = nextBackoff(backoff, c.maxBackoff)
			continue
		}

		if response.StatusCode >= http.StatusOK && response.StatusCode < http.StatusMultipleChoices {
			return response, nil
		}

		lastErr = responseError(response, operation, slug)
		retryable := response.StatusCode >= http.StatusInternalServerError
		if attempt == c.maxRetries || !retryable {
			return nil, lastErr
		}

		if err := c.sleep(ctx, backoff); err != nil {
			return nil, fmt.Errorf("clawhub: %s retry wait aborted: %w", operation, err)
		}
		backoff = nextBackoff(backoff, c.maxBackoff)
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("clawhub: %s request failed", operation)
	}

	return nil, lastErr
}

func (c *Client) buildURL(requestPath string, query url.Values) (string, error) {
	base, err := url.Parse(c.baseURL)
	if err != nil {
		return "", err
	}

	base.Path = strings.TrimRight(base.Path, "/") + "/" + strings.TrimLeft(requestPath, "/")
	if len(query) > 0 {
		base.RawQuery = query.Encode()
	}

	return base.String(), nil
}

func normalizeBaseURL(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return defaultBaseURL
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return strings.TrimRight(trimmed, "/")
	}

	path := strings.TrimRight(parsed.Path, "/")
	if path == "" {
		path = "/api/v1"
	}
	parsed.Path = path

	return strings.TrimRight(parsed.String(), "/")
}

func responseError(response *http.Response, operation string, slug string) error {
	message := readErrorMessage(response.Body)

	if response.StatusCode == http.StatusNotFound && slug != "" {
		notFound := registry.NewPackageNotFoundError(slug)
		if message == "" {
			return fmt.Errorf("clawhub: skill not found: %w", notFound)
		}
		return fmt.Errorf("clawhub: skill not found: %w: %s", notFound, message)
	}

	if message == "" {
		return fmt.Errorf("clawhub: %s request failed: %s", operation, response.Status)
	}

	return fmt.Errorf("clawhub: %s request failed: %s: %s", operation, response.Status, message)
}

func readErrorMessage(body io.ReadCloser) string {
	payload, err := io.ReadAll(io.LimitReader(body, maxErrorBodyBytes))
	closeErr := body.Close()
	if err != nil {
		return ""
	}
	if closeErr != nil {
		return ""
	}

	trimmed := strings.TrimSpace(string(payload))
	if trimmed == "" {
		return ""
	}

	var envelope struct {
		Error   string `json:"error"`
		Message string `json:"message"`
		Detail  string `json:"detail"`
	}
	if err := json.Unmarshal(payload, &envelope); err == nil {
		for _, candidate := range []string{envelope.Error, envelope.Message, envelope.Detail} {
			if candidate = strings.TrimSpace(candidate); candidate != "" {
				return candidate
			}
		}
	}

	return trimmed
}

func readLimitedBody(body io.Reader, maxBytes int64) ([]byte, error) {
	if maxBytes <= 0 {
		return io.ReadAll(body)
	}

	payload, err := io.ReadAll(io.LimitReader(body, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(payload)) > maxBytes {
		return nil, fmt.Errorf("%w: limit=%d", errResponseTooLarge, maxBytes)
	}
	return payload, nil
}

func decodeListings(body io.Reader) ([]registry.Listing, error) {
	payload, err := io.ReadAll(body)
	if err != nil {
		return nil, err
	}

	var direct []clawhubListing
	if err := json.Unmarshal(payload, &direct); err == nil {
		if direct == nil {
			return []registry.Listing{}, nil
		}
		return normalizeClawHubListings(direct), nil
	}

	var envelope struct {
		Skills  []clawhubListing `json:"skills"`
		Results []clawhubListing `json:"results"`
		Items   []clawhubListing `json:"items"`
	}
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return nil, err
	}

	switch {
	case envelope.Skills != nil:
		return normalizeClawHubListings(envelope.Skills), nil
	case envelope.Results != nil:
		return normalizeClawHubListings(envelope.Results), nil
	case envelope.Items != nil:
		return normalizeClawHubListings(envelope.Items), nil
	default:
		return []registry.Listing{}, nil
	}
}

type clawhubListing struct {
	registry.Listing
	DisplayName   string              `json:"displayName"`
	Summary       string              `json:"summary"`
	Tags          clawhubListingTags  `json:"tags"`
	Stats         clawhubListingStats `json:"stats"`
	LatestVersion struct {
		Version string `json:"version"`
	} `json:"latestVersion"`
}

type clawhubListingTags struct {
	Latest string `json:"latest"`
}

type clawhubListingStats struct {
	Downloads       int `json:"downloads"`
	InstallsAllTime int `json:"installsAllTime"`
	InstallsCurrent int `json:"installsCurrent"`
}

func normalizeClawHubListings(listings []clawhubListing) []registry.Listing {
	if listings == nil {
		return []registry.Listing{}
	}

	normalized := make([]registry.Listing, 0, len(listings))
	for _, listing := range listings {
		normalized = append(normalized, listing.registryListing())
	}
	return normalized
}

func (listing clawhubListing) registryListing() registry.Listing {
	result := listing.Listing
	result.Slug = strings.TrimSpace(result.Slug)
	result.Name = firstNonEmpty(result.Name, listingNameFromSlug(result.Slug), listing.DisplayName)
	result.Description = firstNonEmpty(result.Description, listing.Summary, listing.DisplayName)
	result.Author = firstNonEmpty(result.Author, listingAuthorFromSlug(result.Slug))
	result.Version = firstNonEmpty(result.Version, listing.Tags.Latest, listing.LatestVersion.Version)
	if result.Downloads <= 0 {
		result.Downloads = firstPositiveInt(
			listing.Stats.Downloads,
			listing.Stats.InstallsAllTime,
			listing.Stats.InstallsCurrent,
		)
	}
	return result
}

func listingNameFromSlug(slug string) string {
	trimmed := strings.TrimSpace(slug)
	if trimmed == "" {
		return ""
	}
	_, name, found := strings.Cut(strings.TrimPrefix(trimmed, "@"), "/")
	if found {
		return strings.TrimSpace(name)
	}
	return trimmed
}

func listingAuthorFromSlug(slug string) string {
	trimmed := strings.TrimSpace(slug)
	if !strings.HasPrefix(trimmed, "@") {
		return ""
	}
	author, _, found := strings.Cut(strings.TrimPrefix(trimmed, "@"), "/")
	if !found {
		return ""
	}
	return strings.TrimSpace(author)
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
	file, err := os.CreateTemp("", "agh-clawhub-download-*")
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
				"%w: clawhub download for %q exceeds compressed archive limit %d",
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

func wrapSearchCloseError(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("clawhub: close search response: %w", err)
}

func wrapCloseError(slug string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("clawhub: close download response for %q: %w", slug, err)
}

func firstPositiveInt64(values ...int64) int64 {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return -1
}

func firstPositiveInt(values ...int) int {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
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

func contentSize(length int64) int64 {
	if length > 0 {
		return length
	}
	return -1
}

func joinErrors(errs ...error) error {
	var compact []error
	for _, err := range errs {
		if err != nil {
			compact = append(compact, err)
		}
	}
	if len(compact) == 0 {
		return nil
	}
	return errors.Join(compact...)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
