package clawhub

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/skills/marketplace"
)

var _ marketplace.Registry = (*Client)(nil)

func TestNewClientDefaults(t *testing.T) {
	t.Parallel()

	client := NewClient("")

	if client.baseURL != defaultBaseURL {
		t.Fatalf("baseURL = %q, want %q", client.baseURL, defaultBaseURL)
	}
	if client.httpClient == nil {
		t.Fatal("httpClient = nil, want default client")
	}
	if client.httpClient.Timeout != defaultRequestTimeout {
		t.Fatalf("httpClient.Timeout = %s, want %s", client.httpClient.Timeout, defaultRequestTimeout)
	}
	if client.initialBackoff != defaultInitialBackoff {
		t.Fatalf("initialBackoff = %s, want %s", client.initialBackoff, defaultInitialBackoff)
	}
	if client.maxBackoff != defaultMaxBackoff {
		t.Fatalf("maxBackoff = %s, want %s", client.maxBackoff, defaultMaxBackoff)
	}
	if client.maxRetries != defaultMaxRetries {
		t.Fatalf("maxRetries = %d, want %d", client.maxRetries, defaultMaxRetries)
	}
	if client.sleep == nil {
		t.Fatal("sleep = nil, want default sleeper")
	}
}

func TestClientSearchParsesListingsAndLimit(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			t.Fatalf("request.Method = %q, want %q", request.Method, http.MethodGet)
		}
		if request.URL.Path != "/api/v1/skills" {
			t.Fatalf("request.URL.Path = %q, want %q", request.URL.Path, "/api/v1/skills")
		}
		if got := request.URL.Query().Get("q"); got != "agent" {
			t.Fatalf("query q = %q, want %q", got, "agent")
		}
		if got := request.URL.Query().Get("limit"); got != "7" {
			t.Fatalf("query limit = %q, want %q", got, "7")
		}

		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"skills":[{"slug":"@agh/review","name":"Review","description":"Review code","author":"agh","version":"1.2.0","downloads":42}]}`))
	}))
	defer server.Close()

	client := NewClient(server.URL)

	listings, err := client.Search(context.Background(), "agent", marketplace.SearchOpts{Limit: 7})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(listings) != 1 {
		t.Fatalf("len(Search()) = %d, want 1", len(listings))
	}

	got := listings[0]
	if got.Slug != "@agh/review" || got.Name != "Review" || got.Author != "agh" || got.Version != "1.2.0" || got.Downloads != 42 {
		t.Fatalf("Search() listing = %#v", got)
	}
}

func TestClientSearchEmptyResultsReturnsEmptySlice(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"skills":[]}`))
	}))
	defer server.Close()

	client := NewClient(server.URL)

	listings, err := client.Search(context.Background(), "missing", marketplace.SearchOpts{})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if listings == nil {
		t.Fatal("Search() = nil, want empty slice")
	}
	if len(listings) != 0 {
		t.Fatalf("len(Search()) = %d, want 0", len(listings))
	}
}

func TestClientInfoParsesSkillDetail(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/v1/skills/@agh%2Freview" {
			t.Fatalf("request.URL.Path = %q, want %q", request.URL.Path, "/api/v1/skills/@agh%2Freview")
		}

		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{
			"slug":"@agh/review",
			"name":"Review",
			"description":"Review code",
			"author":"agh",
			"version":"1.2.0",
			"downloads":42,
			"readme":"# Review",
			"mcp_servers":["github"],
			"tags":["quality","code"]
		}`))
	}))
	defer server.Close()

	client := NewClient(server.URL)

	detail, err := client.Info(context.Background(), "@agh/review")
	if err != nil {
		t.Fatalf("Info() error = %v", err)
	}
	if detail == nil {
		t.Fatal("Info() = nil, want detail")
	}
	if detail.Slug != "@agh/review" || detail.Readme != "# Review" || !slices.Equal(detail.MCPServers, []string{"github"}) || !slices.Equal(detail.Tags, []string{"quality", "code"}) {
		t.Fatalf("Info() detail = %#v", detail)
	}
}

func TestClientInfoUnknownSlugReturnsNotFoundError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		http.Error(writer, `{"error":"missing skill"}`, http.StatusNotFound)
	}))
	defer server.Close()

	client := NewClient(server.URL, WithRetryPolicy(time.Millisecond, time.Millisecond, 0))

	_, err := client.Info(context.Background(), "@agh/missing")
	if err == nil {
		t.Fatal("Info() error = nil, want not found")
	}
	if !strings.Contains(err.Error(), "skill not found") {
		t.Fatalf("Info() error = %v, want skill not found", err)
	}
	if !strings.Contains(err.Error(), "@agh/missing") {
		t.Fatalf("Info() error = %v, want slug", err)
	}
}

func TestClientDownloadReturnsArchiveStream(t *testing.T) {
	t.Parallel()

	archive := mustTarGz(t, map[string]string{
		"review/SKILL.md": "---\nname: review\n---\n",
	})

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/v1/skills/@agh%2Freview/download" {
			t.Fatalf("request.URL.Path = %q, want %q", request.URL.Path, "/api/v1/skills/@agh%2Freview/download")
		}

		writer.Header().Set("Content-Type", "application/gzip")
		writer.Header().Set("X-Skill-Version", "1.2.0")
		_, _ = writer.Write(archive)
	}))
	defer server.Close()

	client := NewClient(server.URL)

	result, err := client.Download(context.Background(), "@agh/review")
	if err != nil {
		t.Fatalf("Download() error = %v", err)
	}
	if result == nil {
		t.Fatal("Download() = nil, want archive")
	}
	t.Cleanup(func() {
		if err := result.Data.Close(); err != nil {
			t.Errorf("result.Data.Close() error = %v", err)
		}
	})

	if result.Slug != "@agh/review" || result.Version != "1.2.0" {
		t.Fatalf("Download() archive = %#v, want slug/version", result)
	}

	files := readTarGz(t, result.Data)
	if got := files["review/SKILL.md"]; got != "---\nname: review\n---\n" {
		t.Fatalf("downloaded file = %q, want SKILL.md contents", got)
	}
}

func TestClientDownloadUnknownSlugReturnsNotFoundError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		http.Error(writer, `{"error":"missing skill"}`, http.StatusNotFound)
	}))
	defer server.Close()

	client := NewClient(server.URL, WithRetryPolicy(time.Millisecond, time.Millisecond, 0))

	_, err := client.Download(context.Background(), "@agh/missing")
	if err == nil {
		t.Fatal("Download() error = nil, want not found")
	}
	if !strings.Contains(err.Error(), "skill not found") {
		t.Fatalf("Download() error = %v, want skill not found", err)
	}
}

func TestClientRetriesHTTP500WithBackoff(t *testing.T) {
	t.Parallel()

	var attempts atomic.Int32
	var waits []time.Duration

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		current := attempts.Add(1)
		if current <= 3 {
			http.Error(writer, `{"error":"temporary failure"}`, http.StatusInternalServerError)
			return
		}

		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"skills":[{"slug":"@agh/review","name":"Review","description":"Review code","author":"agh","version":"1.2.0","downloads":42}]}`))
	}))
	defer server.Close()

	client := NewClient(
		server.URL,
		WithRetryPolicy(time.Millisecond, 4*time.Millisecond, 3),
		WithSleep(func(_ context.Context, wait time.Duration) error {
			waits = append(waits, wait)
			return nil
		}),
	)

	listings, err := client.Search(context.Background(), "review", marketplace.SearchOpts{})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(listings) != 1 {
		t.Fatalf("len(Search()) = %d, want 1", len(listings))
	}
	if got := attempts.Load(); got != 4 {
		t.Fatalf("attempts = %d, want 4 (initial + 3 retries)", got)
	}
	if !slices.Equal(waits, []time.Duration{time.Millisecond, 2 * time.Millisecond, 4 * time.Millisecond}) {
		t.Fatalf("waits = %#v, want [1ms 2ms 4ms]", waits)
	}
}

func TestClientRetryExhaustionReturnsFinalError(t *testing.T) {
	t.Parallel()

	var attempts atomic.Int32
	var waits []time.Duration

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		attempts.Add(1)
		http.Error(writer, `{"error":"still broken"}`, http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(
		server.URL,
		WithRetryPolicy(time.Millisecond, 2*time.Millisecond, 2),
		WithSleep(func(_ context.Context, wait time.Duration) error {
			waits = append(waits, wait)
			return nil
		}),
	)

	_, err := client.Search(context.Background(), "review", marketplace.SearchOpts{})
	if err == nil {
		t.Fatal("Search() error = nil, want final error")
	}
	if !strings.Contains(err.Error(), "500 Internal Server Error") {
		t.Fatalf("Search() error = %v, want final 500 error", err)
	}
	if got := attempts.Load(); got != 3 {
		t.Fatalf("attempts = %d, want 3", got)
	}
	if !slices.Equal(waits, []time.Duration{time.Millisecond, 2 * time.Millisecond}) {
		t.Fatalf("waits = %#v, want [1ms 2ms]", waits)
	}
}

func TestClientTimeoutReturnsDeadlineExceeded(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		time.Sleep(100 * time.Millisecond)
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"skills":[]}`))
	}))
	defer server.Close()

	client := NewClient(
		server.URL,
		WithHTTPClient(&http.Client{Timeout: 20 * time.Millisecond}),
		WithRetryPolicy(time.Millisecond, time.Millisecond, 0),
	)

	_, err := client.Search(context.Background(), "slow", marketplace.SearchOpts{})
	if err == nil {
		t.Fatal("Search() error = nil, want timeout")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Search() error = %v, want context deadline exceeded", err)
	}
}

func TestClientContextCancellationAbortsPromptly(t *testing.T) {
	t.Parallel()

	started := make(chan struct{})

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		close(started)
		<-request.Context().Done()
	}))
	defer server.Close()

	client := NewClient(server.URL, WithRetryPolicy(time.Millisecond, time.Millisecond, 0))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		<-started
		cancel()
		close(done)
	}()

	start := time.Now()
	_, err := client.Search(ctx, "cancel", marketplace.SearchOpts{})
	<-done
	if err == nil {
		t.Fatal("Search() error = nil, want cancellation")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Search() error = %v, want context canceled", err)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("Search() took %s after cancellation, want prompt abort", elapsed)
	}
}

func TestClientSearchRejectsEmptyQuery(t *testing.T) {
	t.Parallel()

	client := NewClient("")

	_, err := client.Search(context.Background(), "   ", marketplace.SearchOpts{})
	if err == nil {
		t.Fatal("Search() error = nil, want validation error")
	}
	if !strings.Contains(err.Error(), "query is required") {
		t.Fatalf("Search() error = %v, want query validation", err)
	}
}

func TestClientInfoRejectsEmptySlug(t *testing.T) {
	t.Parallel()

	client := NewClient("")

	_, err := client.Info(context.Background(), "   ")
	if err == nil {
		t.Fatal("Info() error = nil, want validation error")
	}
	if !strings.Contains(err.Error(), "skill slug is required") {
		t.Fatalf("Info() error = %v, want slug validation", err)
	}
}

func TestClientDownloadRejectsEmptySlug(t *testing.T) {
	t.Parallel()

	client := NewClient("")

	_, err := client.Download(context.Background(), "   ")
	if err == nil {
		t.Fatal("Download() error = nil, want validation error")
	}
	if !strings.Contains(err.Error(), "skill slug is required") {
		t.Fatalf("Download() error = %v, want slug validation", err)
	}
}

func TestDecodeListingsSupportsDirectArray(t *testing.T) {
	t.Parallel()

	listings, err := decodeListings(strings.NewReader(`[{"slug":"@agh/review","name":"Review","description":"Review code","author":"agh","version":"1.0.0","downloads":1}]`))
	if err != nil {
		t.Fatalf("decodeListings() error = %v", err)
	}
	if len(listings) != 1 || listings[0].Slug != "@agh/review" {
		t.Fatalf("decodeListings() = %#v, want one direct-array listing", listings)
	}
}

func TestDecodeListingsRejectsInvalidJSON(t *testing.T) {
	t.Parallel()

	_, err := decodeListings(strings.NewReader(`{"skills":`))
	if err == nil {
		t.Fatal("decodeListings() error = nil, want invalid json")
	}
}

func TestNormalizeBaseURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "empty uses default", raw: "", want: defaultBaseURL},
		{name: "root adds api version", raw: "https://clawhub.ai", want: "https://clawhub.ai/api/v1"},
		{name: "keeps explicit api path", raw: "https://clawhub.ai/custom", want: "https://clawhub.ai/custom"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := normalizeBaseURL(tt.raw); got != tt.want {
				t.Fatalf("normalizeBaseURL(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}

func TestResponseErrorUsesJSONAndPlainTextMessages(t *testing.T) {
	t.Parallel()

	notFound := responseError(newHTTPResponse(http.StatusNotFound, `{"error":"missing skill"}`), "info", "@agh/missing")
	if !strings.Contains(notFound.Error(), "skill not found") || !strings.Contains(notFound.Error(), "missing skill") {
		t.Fatalf("responseError(notFound) = %v, want not-found message", notFound)
	}

	internal := responseError(newHTTPResponse(http.StatusInternalServerError, "boom"), "search", "")
	if !strings.Contains(internal.Error(), "boom") {
		t.Fatalf("responseError(internal) = %v, want body text", internal)
	}
}

func TestReadErrorMessageHandlesEmptyAndStructuredBodies(t *testing.T) {
	t.Parallel()

	if got := readErrorMessage(io.NopCloser(strings.NewReader(""))); got != "" {
		t.Fatalf("readErrorMessage(empty) = %q, want empty", got)
	}
	if got := readErrorMessage(io.NopCloser(strings.NewReader(`{"message":"bad request"}`))); got != "bad request" {
		t.Fatalf("readErrorMessage(json) = %q, want %q", got, "bad request")
	}
}

func TestSleepContextReturnsOnCancelAndZeroWait(t *testing.T) {
	t.Parallel()

	if err := sleepContext(context.Background(), 0); err != nil {
		t.Fatalf("sleepContext(zero) error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := sleepContext(ctx, time.Second); !errors.Is(err, context.Canceled) {
		t.Fatalf("sleepContext(cancelled) error = %v, want context canceled", err)
	}
}

func TestNextBackoffRespectsDefaultsAndMax(t *testing.T) {
	t.Parallel()

	if got := nextBackoff(0, 0); got != defaultInitialBackoff {
		t.Fatalf("nextBackoff(0,0) = %s, want %s", got, defaultInitialBackoff)
	}
	if got := nextBackoff(2*time.Second, 3*time.Second); got != 3*time.Second {
		t.Fatalf("nextBackoff(2s,3s) = %s, want 3s", got)
	}
}

func TestDoRequestRejectsCanceledContextBeforeHTTP(t *testing.T) {
	t.Parallel()

	client := NewClient("")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.doRequest(ctx, http.MethodGet, "/skills", nil, "search", "")
	if err == nil {
		t.Fatal("doRequest() error = nil, want canceled context")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("doRequest() error = %v, want context canceled", err)
	}
}

func TestDoRequestRejectsInvalidBaseURL(t *testing.T) {
	t.Parallel()

	client := NewClient("://bad url")

	_, err := client.doRequest(context.Background(), http.MethodGet, "/skills", nil, "search", "")
	if err == nil {
		t.Fatal("doRequest() error = nil, want invalid base url")
	}
	if !strings.Contains(err.Error(), "build search request URL") {
		t.Fatalf("doRequest() error = %v, want URL build context", err)
	}
}

func mustTarGz(t *testing.T, files map[string]string) []byte {
	t.Helper()

	var buffer bytes.Buffer
	gzipWriter := gzip.NewWriter(&buffer)
	tarWriter := tar.NewWriter(gzipWriter)

	for name, content := range files {
		header := &tar.Header{
			Name: name,
			Mode: 0o644,
			Size: int64(len(content)),
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			t.Fatalf("WriteHeader(%q) error = %v", name, err)
		}
		if _, err := tarWriter.Write([]byte(content)); err != nil {
			t.Fatalf("Write(%q) error = %v", name, err)
		}
	}

	if err := tarWriter.Close(); err != nil {
		t.Fatalf("tarWriter.Close() error = %v", err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatalf("gzipWriter.Close() error = %v", err)
	}

	return buffer.Bytes()
}

func newHTTPResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Status:     http.StatusText(statusCode),
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

func readTarGz(t *testing.T, reader io.Reader) map[string]string {
	t.Helper()

	gzipReader, err := gzip.NewReader(reader)
	if err != nil {
		t.Fatalf("gzip.NewReader() error = %v", err)
	}
	t.Cleanup(func() {
		if err := gzipReader.Close(); err != nil {
			t.Errorf("gzipReader.Close() error = %v", err)
		}
	})

	tarReader := tar.NewReader(gzipReader)
	files := make(map[string]string)

	for {
		header, err := tarReader.Next()
		if errors.Is(err, io.EOF) {
			return files
		}
		if err != nil {
			t.Fatalf("tarReader.Next() error = %v", err)
		}

		payload, err := io.ReadAll(tarReader)
		if err != nil {
			t.Fatalf("io.ReadAll(%q) error = %v", header.Name, err)
		}
		files[header.Name] = string(payload)
	}
}
