package daytona

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/kballard/go-shellquote"
)

const (
	launcherSidecarPort               = 40241
	launcherSidecarVersion            = "agh-daytona-launcher-sidecar-v1"
	launcherSidecarPath               = "/tmp/agh-daytona-launcher-sidecar-v1"
	launcherSidecarLogPath            = "/tmp/agh-daytona-launcher-sidecar-v1.log"
	sidecarHealthPath                 = "healthz"
	sidecarLaunchPath                 = "v1/launch"
	sidecarSessionStreamBasePath      = "v1/sessions"
	sidecarHealthTimeout              = 30 * time.Second
	sidecarHealthPollInterval         = 200 * time.Millisecond
	sidecarRequestTimeout             = 30 * time.Second
	sidecarCloseTimeout               = 5 * time.Second
	sidecarBuildTimeout               = 2 * time.Minute
	sidecarFrameClientStdin      byte = 0x01
	sidecarFrameClientCloseStdin      = 0x02
	sidecarFrameClientStop            = 0x03
	sidecarFrameServerStdout     byte = 0x01
	sidecarFrameServerStderr          = 0x02
	sidecarFrameServerExit            = 0x03
	sidecarFrameServerError           = 0x04
)

type sidecarTransport struct {
	logger             *slog.Logger
	newClient          sandboxClientFactory
	bootstrap          transport
	clientDialer       sshClientDialer
	httpClient         *http.Client
	healthTimeout      time.Duration
	healthPollInterval time.Duration
	closeTimeout       time.Duration
	binaryMu           sync.Mutex
	binaries           map[string][]byte
}

type sidecarEndpoint struct {
	base       *url.URL
	httpClient *http.Client
	wsDialer   *websocket.Dialer
	closeFn    func() error
}

type sidecarHealthResponse struct {
	OK      bool   `json:"ok"`
	Version string `json:"version"`
}

type sidecarLaunchRequest struct {
	Command string `json:"command"`
}

type sidecarLaunchResponse struct {
	ID string `json:"id"`
}

type sidecarExitPayload struct {
	ExitCode int    `json:"exitCode"`
	Stderr   string `json:"stderr"`
}

type deadlineConn struct {
	net.Conn
}

func (c deadlineConn) SetDeadline(time.Time) error {
	return nil
}

func (c deadlineConn) SetReadDeadline(time.Time) error {
	return nil
}

func (c deadlineConn) SetWriteDeadline(time.Time) error {
	return nil
}

func newSidecarTransport(
	logger *slog.Logger,
	newClient sandboxClientFactory,
	bootstrap transport,
) *sidecarTransport {
	if logger == nil {
		logger = slog.Default()
	}
	var clientDialer sshClientDialer
	if dialer, ok := bootstrap.(sshClientDialer); ok {
		clientDialer = dialer
	}
	return &sidecarTransport{
		logger:       logger,
		newClient:    newClient,
		bootstrap:    bootstrap,
		clientDialer: clientDialer,
		httpClient: &http.Client{
			Timeout: sidecarRequestTimeout,
		},
		healthTimeout:      sidecarHealthTimeout,
		healthPollInterval: sidecarHealthPollInterval,
		closeTimeout:       sidecarCloseTimeout,
		binaries:           make(map[string][]byte),
	}
}

func (t *sidecarTransport) Dial(
	ctx context.Context,
	sandbox sandboxInfo,
	command string,
) (transportSession, error) {
	if t == nil {
		return nil, errors.New("environment/daytona: launcher sidecar transport is required")
	}
	endpoint, err := t.ensureSidecar(ctx, sandbox)
	if err != nil {
		return nil, err
	}
	sessionID, err := t.launch(ctx, endpoint, command)
	if err != nil {
		return nil, err
	}
	return t.connect(ctx, endpoint, sessionID)
}

func (t *sidecarTransport) ensureSidecar(
	ctx context.Context,
	info sandboxInfo,
) (sidecarEndpoint, error) {
	sandbox, err := t.loadSandbox(ctx, info)
	if err != nil {
		return sidecarEndpoint{}, err
	}
	binary, err := t.sidecarBinary(ctx, info)
	if err != nil {
		return sidecarEndpoint{}, err
	}
	if err := sandbox.WriteFile(ctx, launcherSidecarPath, binary); err != nil {
		return sidecarEndpoint{}, fmt.Errorf("environment/daytona: upload launcher sidecar: %w", err)
	}
	endpoint, err := t.openTunnel(ctx, info)
	if err != nil {
		return sidecarEndpoint{}, err
	}
	healthy, err := t.health(ctx, endpoint)
	if err == nil && healthy {
		return endpoint, nil
	}
	if err := t.startSidecar(ctx, info); err != nil {
		return sidecarEndpoint{}, err
	}
	if err := t.waitForHealth(ctx, endpoint); err != nil {
		return sidecarEndpoint{}, err
	}
	return endpoint, nil
}

func (t *sidecarTransport) openTunnel(ctx context.Context, sandbox sandboxInfo) (sidecarEndpoint, error) {
	if t.clientDialer == nil {
		return sidecarEndpoint{}, errors.New("environment/daytona: launcher sidecar SSH tunnel is not configured")
	}
	client, err := t.clientDialer.DialClient(ctx, sandbox)
	if err != nil {
		return sidecarEndpoint{}, fmt.Errorf("environment/daytona: open launcher sidecar SSH tunnel: %w", err)
	}
	baseURL, err := url.Parse("http://sidecar")
	if err != nil {
		_ = client.Close()
		return sidecarEndpoint{}, fmt.Errorf("environment/daytona: parse launcher sidecar tunnel base URL: %w", err)
	}
	targetAddr := fmt.Sprintf("127.0.0.1:%d", launcherSidecarPort)
	httpTransport := &http.Transport{
		DialContext: func(context.Context, string, string) (net.Conn, error) {
			conn, err := client.Dial("tcp", targetAddr)
			if err != nil {
				return nil, err
			}
			return deadlineConn{Conn: conn}, nil
		},
	}
	httpClient := &http.Client{
		Transport: httpTransport,
		Timeout:   sidecarRequestTimeout,
	}
	wsDialer := &websocket.Dialer{
		NetDialContext: func(context.Context, string, string) (net.Conn, error) {
			conn, err := client.Dial("tcp", targetAddr)
			if err != nil {
				return nil, err
			}
			return deadlineConn{Conn: conn}, nil
		},
	}
	return sidecarEndpoint{
		base:       baseURL,
		httpClient: httpClient,
		wsDialer:   wsDialer,
		closeFn: func() error {
			httpTransport.CloseIdleConnections()
			return client.Close()
		},
	}, nil
}

func (t *sidecarTransport) sidecarBinary(ctx context.Context, sandbox sandboxInfo) ([]byte, error) {
	arch, err := t.remoteArch(ctx, sandbox)
	if err != nil {
		return nil, err
	}

	t.binaryMu.Lock()
	cached := append([]byte(nil), t.binaries[arch]...)
	t.binaryMu.Unlock()
	if len(cached) != 0 {
		return cached, nil
	}

	built, err := t.buildSidecarBinary(ctx, arch)
	if err != nil {
		return nil, err
	}
	t.binaryMu.Lock()
	t.binaries[arch] = append([]byte(nil), built...)
	t.binaryMu.Unlock()
	return built, nil
}

func (t *sidecarTransport) remoteArch(ctx context.Context, sandbox sandboxInfo) (string, error) {
	if t.bootstrap == nil {
		return "", errors.New("environment/daytona: launcher sidecar bootstrap transport is required")
	}
	session, err := t.bootstrap.Dial(ctx, sandbox, "uname -m")
	if err != nil {
		return "", fmt.Errorf("environment/daytona: detect sandbox architecture: %w", err)
	}
	defer func() {
		if closeErr := session.Close(); closeErr != nil {
			t.logger.Warn("environment/daytona: close arch probe session failed", "error", closeErr)
		}
	}()
	output, readErr := io.ReadAll(session)
	waitErr := session.Wait()
	if err := errors.Join(readErr, waitErr); err != nil {
		return "", fmt.Errorf("environment/daytona: detect sandbox architecture: %w stderr=%q", err, session.Stderr())
	}
	switch strings.TrimSpace(string(output)) {
	case "x86_64", "amd64":
		return "amd64", nil
	case "aarch64", "arm64":
		return "arm64", nil
	default:
		return "", fmt.Errorf(
			"environment/daytona: unsupported sandbox architecture %q",
			strings.TrimSpace(string(output)),
		)
	}
}

func (t *sidecarTransport) buildSidecarBinary(ctx context.Context, arch string) ([]byte, error) {
	goBinary, err := exec.LookPath("go")
	if err != nil {
		return nil, fmt.Errorf("environment/daytona: go toolchain is required to build launcher sidecar: %w", err)
	}
	buildCtx, cancel := context.WithTimeout(ctx, sidecarBuildTimeout)
	defer cancel()

	tmpFile, err := os.CreateTemp("", "agh-daytona-sidecar-*")
	if err != nil {
		return nil, fmt.Errorf("environment/daytona: create launcher sidecar temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	if err := tmpFile.Close(); err != nil {
		return nil, fmt.Errorf("environment/daytona: close launcher sidecar temp file: %w", err)
	}
	defer os.Remove(tmpPath)

	cmd := exec.CommandContext(buildCtx, goBinary, "build", "-o", tmpPath, t.sidecarSourceDir())
	cacheRoot := filepath.Join(os.TempDir(), "agh-daytona-sidecar-go")
	modCache := filepath.Join(cacheRoot, "mod")
	buildCache := filepath.Join(cacheRoot, "build")
	if err := os.MkdirAll(modCache, 0o755); err != nil {
		return nil, fmt.Errorf("environment/daytona: create launcher sidecar module cache: %w", err)
	}
	if err := os.MkdirAll(buildCache, 0o755); err != nil {
		return nil, fmt.Errorf("environment/daytona: create launcher sidecar build cache: %w", err)
	}
	cmd.Env = append(
		os.Environ(),
		"CGO_ENABLED=0",
		"GOOS=linux",
		"GOARCH="+arch,
		"GOMODCACHE="+modCache,
		"GOCACHE="+buildCache,
	)
	cmd.Dir = t.repoRootDir()
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf(
			"environment/daytona: build launcher sidecar for %s: %w: %s",
			arch,
			err,
			strings.TrimSpace(string(output)),
		)
	}
	binary, err := os.ReadFile(tmpPath)
	if err != nil {
		return nil, fmt.Errorf("environment/daytona: read launcher sidecar binary: %w", err)
	}
	return binary, nil
}

func (t *sidecarTransport) loadSandbox(ctx context.Context, info sandboxInfo) (sandbox, error) {
	if t.newClient == nil {
		return nil, errors.New("environment/daytona: launcher sidecar sandbox client is required")
	}
	client, err := t.newClient(clientConfig{APIURL: info.APIURL})
	if err != nil {
		return nil, fmt.Errorf("environment/daytona: launcher sidecar create client: %w", err)
	}
	sandbox, err := client.Get(ctx, info.ID)
	if err != nil {
		return nil, fmt.Errorf("environment/daytona: launcher sidecar load sandbox %q: %w", info.ID, err)
	}
	return sandbox, nil
}

func (t *sidecarTransport) repoRootDir() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "."
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", ".."))
}

func (t *sidecarTransport) sidecarSourceDir() string {
	return "./internal/environment/daytona/cmd/agh-daytona-sidecar"
}

func (t *sidecarTransport) health(ctx context.Context, endpoint sidecarEndpoint) (bool, error) {
	requestCtx, cancel := context.WithTimeout(ctx, sidecarRequestTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(
		requestCtx,
		http.MethodGet,
		endpoint.url(sidecarHealthPath),
		http.NoBody,
	)
	if err != nil {
		return false, fmt.Errorf("environment/daytona: build sidecar health request: %w", err)
	}
	client := endpoint.httpClient
	if client == nil {
		client = t.httpClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("environment/daytona: launcher sidecar health status %d", resp.StatusCode)
	}
	var payload sidecarHealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return false, fmt.Errorf("environment/daytona: decode launcher sidecar health: %w", err)
	}
	return payload.OK && payload.Version == launcherSidecarVersion, nil
}

func (t *sidecarTransport) waitForHealth(ctx context.Context, endpoint sidecarEndpoint) error {
	waitCtx, cancel := context.WithTimeout(ctx, t.healthTimeout)
	defer cancel()

	ticker := time.NewTicker(t.healthPollInterval)
	defer ticker.Stop()

	for {
		healthy, err := t.health(waitCtx, endpoint)
		if err == nil && healthy {
			return nil
		}
		select {
		case <-waitCtx.Done():
			if err != nil {
				return fmt.Errorf("environment/daytona: wait for launcher sidecar health: %w", err)
			}
			return fmt.Errorf(
				"environment/daytona: wait for launcher sidecar health: %w",
				waitCtx.Err(),
			)
		case <-ticker.C:
		}
	}
}

func (t *sidecarTransport) startSidecar(ctx context.Context, sandbox sandboxInfo) error {
	if t.bootstrap == nil {
		return errors.New("environment/daytona: launcher sidecar bootstrap transport is required")
	}
	session, err := t.bootstrap.Dial(ctx, sandbox, launcherSidecarStartCommand())
	if err != nil {
		return fmt.Errorf("environment/daytona: start launcher sidecar: %w", err)
	}
	defer func() {
		if closeErr := session.Close(); closeErr != nil {
			t.logger.Warn("environment/daytona: close launcher bootstrap session failed", "error", closeErr)
		}
	}()
	if err := session.Wait(); err != nil {
		stderr := strings.TrimSpace(session.Stderr())
		if stderr != "" {
			return fmt.Errorf("environment/daytona: start launcher sidecar: %w stderr=%q", err, stderr)
		}
		return fmt.Errorf("environment/daytona: start launcher sidecar: %w", err)
	}
	return nil
}

func (t *sidecarTransport) launch(
	ctx context.Context,
	endpoint sidecarEndpoint,
	command string,
) (string, error) {
	body, err := json.Marshal(sidecarLaunchRequest{Command: command})
	if err != nil {
		return "", fmt.Errorf("environment/daytona: marshal sidecar launch request: %w", err)
	}
	requestCtx, cancel := context.WithTimeout(ctx, sidecarRequestTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(
		requestCtx,
		http.MethodPost,
		endpoint.url(sidecarLaunchPath),
		bytes.NewReader(body),
	)
	if err != nil {
		return "", fmt.Errorf("environment/daytona: build sidecar launch request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	client := endpoint.httpClient
	if client == nil {
		client = t.httpClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("environment/daytona: launch command via sidecar: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		payload := readResponseSnippet(resp.Body)
		return "", fmt.Errorf(
			"environment/daytona: sidecar launch status %d: %s",
			resp.StatusCode,
			payload,
		)
	}
	var launched sidecarLaunchResponse
	if err := json.NewDecoder(resp.Body).Decode(&launched); err != nil {
		return "", fmt.Errorf("environment/daytona: decode sidecar launch response: %w", err)
	}
	if strings.TrimSpace(launched.ID) == "" {
		return "", errors.New("environment/daytona: sidecar launch response missing session id")
	}
	return strings.TrimSpace(launched.ID), nil
}

func (t *sidecarTransport) connect(
	ctx context.Context,
	endpoint sidecarEndpoint,
	sessionID string,
) (transportSession, error) {
	dialer := endpoint.wsDialer
	if dialer == nil {
		dialer = websocket.DefaultDialer
	}
	conn, resp, err := dialer.DialContext(
		ctx,
		endpoint.wsURL(sidecarSessionStreamBasePath, sessionID, "stream"),
		nil,
	)
	if err != nil {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
		return nil, fmt.Errorf("environment/daytona: connect launcher sidecar stream: %w", err)
	}
	httpClient := endpoint.httpClient
	if httpClient == nil {
		httpClient = t.httpClient
	}
	return newSidecarSession(conn, endpoint, sessionID, httpClient, t.closeTimeout), nil
}

func (e sidecarEndpoint) url(parts ...string) string {
	clone := *e.base
	joined := append([]string{strings.TrimSuffix(clone.Path, "/")}, parts...)
	clone.Path = path.Join(joined...)
	clone.RawPath = ""
	return clone.String()
}

func (e sidecarEndpoint) wsURL(parts ...string) string {
	u := e.url(parts...)
	parsed, err := url.Parse(u)
	if err != nil {
		return u
	}
	switch parsed.Scheme {
	case "https":
		parsed.Scheme = "wss"
	case "http":
		parsed.Scheme = "ws"
	}
	return parsed.String()
}

func launcherSidecarStartCommand() string {
	script := strings.Join([]string{
		"chmod 755 " + shellquote.Join(launcherSidecarPath),
		"&&",
		"nohup",
		shellquote.Join(launcherSidecarPath),
		"--port",
		fmt.Sprintf("%d", launcherSidecarPort),
		">" + shellquote.Join(launcherSidecarLogPath),
		"2>&1",
		"</dev/null",
		"&",
	}, " ")
	return strings.Join([]string{
		"sh", "-lc", shellquote.Join(script),
	}, " ")
}

type sidecarSession struct {
	conn         *websocket.Conn
	endpoint     sidecarEndpoint
	sessionID    string
	httpClient   *http.Client
	closeTimeout time.Duration
	stdoutReader *io.PipeReader
	stdoutWriter *io.PipeWriter
	done         chan struct{}
	writeMu      sync.Mutex
	closeWriteMu sync.Once
	closeSession sync.Once
	finishOnce   sync.Once
	stderrMu     sync.Mutex
	stderr       strings.Builder
	waitErr      error
}

func newSidecarSession(
	conn *websocket.Conn,
	endpoint sidecarEndpoint,
	sessionID string,
	httpClient *http.Client,
	closeTimeout time.Duration,
) *sidecarSession {
	stdoutReader, stdoutWriter := io.Pipe()
	session := &sidecarSession{
		conn:         conn,
		endpoint:     endpoint,
		sessionID:    sessionID,
		httpClient:   httpClient,
		closeTimeout: closeTimeout,
		stdoutReader: stdoutReader,
		stdoutWriter: stdoutWriter,
		done:         make(chan struct{}),
	}
	go session.readLoop()
	return session
}

func (s *sidecarSession) Read(p []byte) (int, error) {
	return s.stdoutReader.Read(p)
}

func (s *sidecarSession) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	payload := make([]byte, len(p)+1)
	payload[0] = sidecarFrameClientStdin
	copy(payload[1:], p)
	if err := s.writeFrame(payload); err != nil {
		return 0, err
	}
	return len(p), nil
}

func (s *sidecarSession) CloseWrite() error {
	var err error
	s.closeWriteMu.Do(func() {
		err = s.writeFrame([]byte{sidecarFrameClientCloseStdin})
	})
	return err
}

func (s *sidecarSession) Close() error {
	var err error
	s.closeSession.Do(func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), s.closeTimeout)
		defer cancel()
		err = s.requestStop(stopCtx)
		if s.endpoint.closeFn != nil {
			err = errors.Join(err, s.endpoint.closeFn())
		}
		err = errors.Join(err, s.conn.Close(), s.stdoutWriter.Close())
	})
	return err
}

func (s *sidecarSession) Done() <-chan struct{} {
	return s.done
}

func (s *sidecarSession) Wait() error {
	<-s.done
	return s.waitErr
}

func (s *sidecarSession) Stop(ctx context.Context) error {
	stopErr := s.requestStop(ctx)
	select {
	case <-s.done:
		return errors.Join(stopErr, s.waitErr)
	case <-ctx.Done():
		return errors.Join(stopErr, fmt.Errorf("environment/daytona: stop launcher sidecar session: %w", ctx.Err()))
	}
}

func (s *sidecarSession) Stderr() string {
	s.stderrMu.Lock()
	defer s.stderrMu.Unlock()
	return s.stderr.String()
}

func (s *sidecarSession) writeFrame(payload []byte) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	return s.conn.WriteMessage(websocket.BinaryMessage, payload)
}

func (s *sidecarSession) readLoop() {
	for {
		messageType, payload, err := s.conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				s.finish(s.waitErr)
				return
			}
			s.finish(fmt.Errorf("environment/daytona: launcher sidecar websocket read: %w", err))
			return
		}
		if messageType != websocket.BinaryMessage || len(payload) == 0 {
			continue
		}
		switch payload[0] {
		case sidecarFrameServerStdout:
			if _, err := s.stdoutWriter.Write(payload[1:]); err != nil && !errors.Is(err, io.ErrClosedPipe) {
				s.finish(fmt.Errorf("environment/daytona: forward launcher stdout: %w", err))
				return
			}
		case sidecarFrameServerStderr:
			s.appendStderr(payload[1:])
		case sidecarFrameServerExit:
			s.finish(s.exitError(payload[1:]))
			return
		case sidecarFrameServerError:
			s.finish(
				fmt.Errorf("environment/daytona: launcher sidecar error: %s", strings.TrimSpace(string(payload[1:]))),
			)
			return
		}
	}
}

func (s *sidecarSession) appendStderr(payload []byte) {
	if len(payload) == 0 {
		return
	}
	s.stderrMu.Lock()
	defer s.stderrMu.Unlock()
	s.stderr.WriteString(string(payload))
}

func (s *sidecarSession) exitError(payload []byte) error {
	var exit sidecarExitPayload
	if err := json.Unmarshal(payload, &exit); err != nil {
		return fmt.Errorf("environment/daytona: decode launcher exit payload: %w", err)
	}
	if strings.TrimSpace(exit.Stderr) != "" {
		s.stderrMu.Lock()
		s.stderr.Reset()
		s.stderr.WriteString(exit.Stderr)
		s.stderrMu.Unlock()
	}
	if exit.ExitCode == 0 {
		return nil
	}
	stderr := strings.TrimSpace(s.Stderr())
	if stderr != "" {
		return fmt.Errorf("environment/daytona: launcher exited with code %d: %s", exit.ExitCode, stderr)
	}
	return fmt.Errorf("environment/daytona: launcher exited with code %d", exit.ExitCode)
}

func (s *sidecarSession) finish(err error) {
	s.finishOnce.Do(func() {
		s.waitErr = err
		_ = s.stdoutWriter.Close()
		close(s.done)
	})
}

func (s *sidecarSession) requestStop(ctx context.Context) error {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodDelete,
		s.endpoint.url(sidecarSessionStreamBasePath, s.sessionID),
		http.NoBody,
	)
	if err != nil {
		return fmt.Errorf("environment/daytona: build sidecar stop request: %w", err)
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("environment/daytona: request sidecar stop: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body := readResponseSnippet(resp.Body)
		return fmt.Errorf(
			"environment/daytona: sidecar stop status %d: %s",
			resp.StatusCode,
			body,
		)
	}
	return nil
}

func readResponseSnippet(body io.Reader) string {
	if body == nil {
		return ""
	}
	payload, err := io.ReadAll(io.LimitReader(body, 4096))
	if err != nil {
		return fmt.Sprintf("read response body: %v", err)
	}
	return strings.TrimSpace(string(payload))
}
