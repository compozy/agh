package daytona

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

const (
	defaultSSHPort            = "22"
	defaultSSHAccessExpiresIn = time.Hour
	defaultSSHKeepAlive       = 30 * time.Second
	defaultSSHDialTimeout     = 30 * time.Second
)

type sshAccess struct {
	Token     string
	IssuedAt  time.Time
	ExpiresAt time.Time
}

type sshTokenSource interface {
	FetchSSHAccess(ctx context.Context, apiURL string, sandboxID string, expiresIn time.Duration) (sshAccess, error)
}

type restSSHTokenSource struct {
	httpClient *http.Client
	apiKey     func() string
	now        func() time.Time
}

func newRESTSSHTokenSource(now func() time.Time) sshTokenSource {
	if now == nil {
		now = time.Now
	}
	return &restSSHTokenSource{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		apiKey:     func() string { return os.Getenv("DAYTONA_API_KEY") },
		now:        now,
	}
}

func (s *restSSHTokenSource) FetchSSHAccess(
	ctx context.Context,
	apiURL string,
	sandboxID string,
	expiresIn time.Duration,
) (sshAccess, error) {
	key := ""
	if s.apiKey != nil {
		key = s.apiKey()
	}
	if key == "" {
		return sshAccess{}, errors.New("environment/daytona: DAYTONA_API_KEY is required for SSH access")
	}
	endpoint, err := url.Parse(normalizeAPIURL(apiURL) + "/sandbox/" + url.PathEscape(sandboxID) + "/ssh-access")
	if err != nil {
		return sshAccess{}, fmt.Errorf("environment/daytona: build SSH access URL: %w", err)
	}
	query := endpoint.Query()
	query.Set("expiresInMinutes", strconv.Itoa(int(expiresIn.Minutes())))
	endpoint.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), http.NoBody)
	if err != nil {
		return sshAccess{}, fmt.Errorf("environment/daytona: build SSH access request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Accept", "application/json")

	client := s.httpClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return sshAccess{}, fmt.Errorf("environment/daytona: fetch SSH access token: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, 4096))
		if readErr != nil {
			return sshAccess{}, fmt.Errorf(
				"environment/daytona: fetch SSH access token status %d and read error body: %w",
				resp.StatusCode,
				readErr,
			)
		}
		return sshAccess{}, fmt.Errorf(
			"environment/daytona: fetch SSH access token status %d: %s",
			resp.StatusCode,
			string(body),
		)
	}

	var raw struct {
		Token          string `json:"token"`
		ExpiresAt      string `json:"expiresAt"`
		ExpiresAtSnake string `json:"expires_at"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return sshAccess{}, fmt.Errorf("environment/daytona: decode SSH access response: %w", err)
	}
	if raw.Token == "" {
		return sshAccess{}, errors.New("environment/daytona: SSH access response missing token")
	}
	now := s.now().UTC()
	expiresAt := now.Add(expiresIn)
	if raw.ExpiresAt != "" {
		if parsed, err := time.Parse(time.RFC3339, raw.ExpiresAt); err == nil {
			expiresAt = parsed.UTC()
		}
	}
	if raw.ExpiresAtSnake != "" {
		if parsed, err := time.Parse(time.RFC3339, raw.ExpiresAtSnake); err == nil {
			expiresAt = parsed.UTC()
		}
	}
	return sshAccess{Token: raw.Token, IssuedAt: now, ExpiresAt: expiresAt}, nil
}

type sshTokenManager struct {
	source    sshTokenSource
	now       func() time.Time
	expiresIn time.Duration
	mu        sync.Mutex
	tokens    map[string]sshAccess
}

func newSSHTokenManager(source sshTokenSource, now func() time.Time) *sshTokenManager {
	if now == nil {
		now = time.Now
	}
	return &sshTokenManager{
		source:    source,
		now:       now,
		expiresIn: defaultSSHAccessExpiresIn,
		tokens:    make(map[string]sshAccess),
	}
}

func (m *sshTokenManager) Ensure(
	ctx context.Context,
	apiURL string,
	sandboxID string,
	force bool,
) (sshAccess, error) {
	if m == nil || m.source == nil {
		return sshAccess{}, errors.New("environment/daytona: SSH token manager is not configured")
	}
	key := tokenCacheKey(apiURL, sandboxID)
	m.mu.Lock()
	cached, ok := m.tokens[key]
	if ok && !force && !m.shouldRefresh(cached) {
		m.mu.Unlock()
		return cached, nil
	}
	m.mu.Unlock()

	access, err := m.source.FetchSSHAccess(ctx, apiURL, sandboxID, m.expiresIn)
	if err != nil {
		return sshAccess{}, err
	}
	m.mu.Lock()
	m.tokens[key] = access
	m.mu.Unlock()
	return access, nil
}

func (m *sshTokenManager) shouldRefresh(access sshAccess) bool {
	if access.Token == "" || access.ExpiresAt.IsZero() {
		return true
	}
	now := m.now().UTC()
	if !now.Before(access.ExpiresAt) {
		return true
	}
	issuedAt := access.IssuedAt
	if issuedAt.IsZero() {
		issuedAt = access.ExpiresAt.Add(-m.expiresIn)
	}
	refreshAt := issuedAt.Add(access.ExpiresAt.Sub(issuedAt) / 2)
	return !now.Before(refreshAt)
}

func tokenCacheKey(apiURL string, sandboxID string) string {
	return normalizeAPIURL(apiURL) + "\x00" + sandboxID
}

type sshDialer func(
	ctx context.Context,
	network string,
	address string,
	config *ssh.ClientConfig,
) (*ssh.Client, error)

func defaultSSHDialer(
	ctx context.Context,
	network string,
	address string,
	config *ssh.ClientConfig,
) (*ssh.Client, error) {
	dialer := net.Dialer{Timeout: defaultSSHDialTimeout}
	conn, err := dialer.DialContext(ctx, network, address)
	if err != nil {
		return nil, err
	}
	clientConn, chans, reqs, err := ssh.NewClientConn(conn, address, config)
	if err != nil {
		if closeErr := conn.Close(); closeErr != nil {
			err = errors.Join(err, closeErr)
		}
		return nil, err
	}
	return ssh.NewClient(clientConn, chans, reqs), nil
}

type sshTransport struct {
	tokens          *sshTokenManager
	host            string
	port            string
	dial            sshDialer
	hostKeyCallback ssh.HostKeyCallback
	keepAlive       time.Duration
	now             func() time.Time
}

func newSSHTransport(tokens *sshTokenManager, opts ...func(*sshTransport)) *sshTransport {
	transport := &sshTransport{
		tokens:          tokens,
		host:            defaultSSHHost,
		port:            defaultSSHPort,
		dial:            defaultSSHDialer,
		hostKeyCallback: defaultHostKeyCallback(),
		keepAlive:       defaultSSHKeepAlive,
		now:             time.Now,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(transport)
		}
	}
	if transport.host == "" {
		transport.host = defaultSSHHost
	}
	if transport.port == "" {
		transport.port = defaultSSHPort
	}
	if transport.dial == nil {
		transport.dial = defaultSSHDialer
	}
	if transport.hostKeyCallback == nil {
		transport.hostKeyCallback = defaultHostKeyCallback()
	}
	if transport.keepAlive <= 0 {
		transport.keepAlive = defaultSSHKeepAlive
	}
	return transport
}

func (t *sshTransport) Dial(
	ctx context.Context,
	sandbox sandboxInfo,
	command string,
) (transportSession, error) {
	session, err := t.dialWithFreshness(ctx, sandbox, command, false)
	if err == nil {
		return session, nil
	}
	retry, retryErr := t.dialWithFreshness(ctx, sandbox, command, true)
	if retryErr != nil {
		return nil, errors.Join(err, retryErr)
	}
	return retry, nil
}

func (t *sshTransport) dialWithFreshness(
	ctx context.Context,
	sandbox sandboxInfo,
	command string,
	forceToken bool,
) (transportSession, error) {
	access, err := t.tokens.Ensure(ctx, sandbox.APIURL, sandbox.ID, forceToken)
	if err != nil {
		return nil, fmt.Errorf("environment/daytona: ensure SSH access for sandbox %q: %w", sandbox.ID, err)
	}
	config := &ssh.ClientConfig{
		User:            access.Token,
		Auth:            []ssh.AuthMethod{ssh.Password("")},
		HostKeyCallback: t.hostKeyCallback,
		Timeout:         defaultSSHDialTimeout,
	}
	address := net.JoinHostPort(normalizeSSHHost(firstNonEmpty(sandbox.SSHHost, t.host)), t.port)
	client, err := t.dial(ctx, "tcp", address, config)
	if err != nil {
		return nil, fmt.Errorf("environment/daytona: dial SSH sandbox %q: %w", sandbox.ID, err)
	}
	session, err := client.NewSession()
	if err != nil {
		if closeErr := client.Close(); closeErr != nil {
			err = errors.Join(err, closeErr)
		}
		return nil, fmt.Errorf("environment/daytona: create SSH session for sandbox %q: %w", sandbox.ID, err)
	}
	stdin, err := session.StdinPipe()
	if err != nil {
		closeSSH(client, session)
		return nil, fmt.Errorf("environment/daytona: open SSH stdin for sandbox %q: %w", sandbox.ID, err)
	}
	stdout, err := session.StdoutPipe()
	if err != nil {
		closeSSH(client, session)
		return nil, fmt.Errorf("environment/daytona: open SSH stdout for sandbox %q: %w", sandbox.ID, err)
	}
	var stderr bytes.Buffer
	session.Stderr = &stderr
	if err := session.Start(command); err != nil {
		closeSSH(client, session)
		return nil, fmt.Errorf("environment/daytona: start SSH command in sandbox %q: %w", sandbox.ID, err)
	}
	return newSSHSession(client, session, stdin, stdout, &stderr, t.keepAlive), nil
}

func closeSSH(client *ssh.Client, session *ssh.Session) {
	if session != nil {
		_ = session.Close()
	}
	if client != nil {
		_ = client.Close()
	}
}

type sshSession struct {
	client    *ssh.Client
	session   *ssh.Session
	stdin     io.WriteCloser
	stdout    io.Reader
	stderr    *bytes.Buffer
	closeOnce sync.Once
	done      chan struct{}
	waitErr   error
	cancel    context.CancelFunc
}

func newSSHSession(
	client *ssh.Client,
	session *ssh.Session,
	stdin io.WriteCloser,
	stdout io.Reader,
	stderr *bytes.Buffer,
	keepAlive time.Duration,
) *sshSession {
	ctx, cancel := context.WithCancel(context.Background())
	remote := &sshSession{
		client:  client,
		session: session,
		stdin:   stdin,
		stdout:  stdout,
		stderr:  stderr,
		done:    make(chan struct{}),
		cancel:  cancel,
	}
	go remote.keepAlive(ctx, keepAlive)
	go func() {
		remote.waitErr = session.Wait()
		cancel()
		close(remote.done)
	}()
	return remote
}

func (s *sshSession) Read(p []byte) (int, error) {
	return s.stdout.Read(p)
}

func (s *sshSession) Write(p []byte) (int, error) {
	return s.stdin.Write(p)
}

func (s *sshSession) CloseWrite() error {
	if s.stdin == nil {
		return nil
	}
	return s.stdin.Close()
}

func (s *sshSession) Close() error {
	var err error
	s.closeOnce.Do(func() {
		if s.cancel != nil {
			s.cancel()
		}
		err = errors.Join(s.CloseWrite(), s.session.Close(), s.client.Close())
	})
	return err
}

func (s *sshSession) Done() <-chan struct{} {
	return s.done
}

func (s *sshSession) Wait() error {
	<-s.done
	return s.waitErr
}

func (s *sshSession) Stop(ctx context.Context) error {
	if err := s.Close(); err != nil {
		return err
	}
	select {
	case <-s.done:
		return s.waitErr
	case <-ctx.Done():
		return fmt.Errorf("environment/daytona: stop SSH session: %w", ctx.Err())
	}
}

func (s *sshSession) Stderr() string {
	if s.stderr == nil {
		return ""
	}
	return s.stderr.String()
}

func (s *sshSession) keepAlive(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_, _, err := s.client.SendRequest("keepalive@openssh.com", true, nil)
			if err != nil {
				return
			}
		}
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func defaultHostKeyCallback() ssh.HostKeyCallback {
	home, err := os.UserHomeDir()
	if err != nil {
		return func(hostname string, _ net.Addr, _ ssh.PublicKey) error {
			return fmt.Errorf("environment/daytona: resolve home for SSH known_hosts %q: %w", hostname, err)
		}
	}
	callback, err := knownhosts.New(filepath.Join(home, ".ssh", "known_hosts"))
	if err != nil {
		return func(hostname string, _ net.Addr, _ ssh.PublicKey) error {
			return fmt.Errorf("environment/daytona: load SSH known_hosts for %q: %w", hostname, err)
		}
	}
	return callback
}
