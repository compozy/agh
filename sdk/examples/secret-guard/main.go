package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	extensionprotocol "github.com/pedronauck/agh/internal/extension/protocol"
	hookspkg "github.com/pedronauck/agh/internal/hooks"
	"github.com/pedronauck/agh/internal/subprocess"
)

const (
	goHandshakeEnv = "AGH_SECRET_GUARD_HANDSHAKE_PATH"
	goHostCallEnv  = "AGH_SECRET_GUARD_HOST_CALL_PATH"
	goStartsEnv    = "AGH_SECRET_GUARD_STARTS_PATH"
	goCrashOnceEnv = "AGH_SECRET_GUARD_CRASH_ONCE_PATH"
	goShutdownEnv  = "AGH_SECRET_GUARD_SHUTDOWN_PATH"
)

var secretPatterns = []string{
	"sk-",
	"AKIA",
	"ghp_",
	"-----BEGIN RSA",
}

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	if len(args) > 0 {
		switch strings.TrimSpace(args[0]) {
		case "hook":
			if len(args) < 2 {
				return errors.New("secret-guard: hook name is required")
			}
			return runHook(strings.TrimSpace(args[1]), stdin, stdout)
		case "--hook":
			if len(args) < 2 {
				return errors.New("secret-guard: hook name is required")
			}
			return runHook(strings.TrimSpace(args[1]), stdin, stdout)
		case "serve":
			return runServe(stdin, stdout, stderr)
		}
	}

	return runServe(stdin, stdout, stderr)
}

func runHook(name string, stdin io.Reader, stdout io.Writer) error {
	switch strings.TrimSpace(name) {
	case "input_pre_submit":
		var payload hookspkg.InputPreSubmitPayload
		if err := json.NewDecoder(stdin).Decode(&payload); err != nil {
			return fmt.Errorf("secret-guard: decode input_pre_submit payload: %w", err)
		}
		patch := evaluateSecretGuard(payload.Message)
		return json.NewEncoder(stdout).Encode(patch)
	default:
		return fmt.Errorf("secret-guard: unsupported hook %q", name)
	}
}

func runServe(stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	appendMarkerLine(os.Getenv(goStartsEnv), fmt.Sprintf("pid=%d", os.Getpid()))

	peer := newRPCPeer(stdin, stdout)
	runtime := &secretGuardRuntime{
		stderr: stderr,
		peer:   peer,
	}

	peer.handle("initialize", runtime.handleInitialize)
	peer.handle("execute_hook", runtime.handleExecuteHook)
	peer.handle("health_check", runtime.handleHealthCheck)
	peer.handle("shutdown", runtime.handleShutdown)

	return peer.serve()
}

type secretGuardRuntime struct {
	stderr io.Writer
	peer   *rpcPeer

	mu          sync.RWMutex
	initialized bool
	session     runtimeSession
}

type runtimeSession struct {
	request  subprocess.InitializeRequest
	response subprocess.InitializeResponse
}

type executeHookParams struct {
	InvocationID string `json:"invocation_id"`
	Hook         struct {
		Name     string            `json:"name"`
		Event    string            `json:"event"`
		Mode     string            `json:"mode"`
		Required bool              `json:"required"`
		Timeout  int64             `json:"timeout_ms"`
		Source   string            `json:"source"`
		Metadata map[string]string `json:"metadata,omitempty"`
	} `json:"hook"`
	Payload json.RawMessage `json:"payload"`
}

type hostSessionSummary struct {
	ID string `json:"id"`
}

type rpcEnvelope struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      json.RawMessage  `json:"id,omitempty"`
	Method  string           `json:"method,omitempty"`
	Params  json.RawMessage  `json:"params,omitempty"`
	Result  any              `json:"result,omitempty"`
	Error   *rpcErrorPayload `json:"error,omitempty"`
}

type rpcErrorPayload struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type rpcPeer struct {
	scanner *bufio.Scanner
	encoder *json.Encoder

	writeMu sync.Mutex
	pending sync.Map
	wg      sync.WaitGroup
	nextID  int64

	handlers map[string]func(json.RawMessage) (any, error)
}

func newRPCPeer(stdin io.Reader, stdout io.Writer) *rpcPeer {
	scanner := bufio.NewScanner(stdin)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)

	encoder := json.NewEncoder(stdout)
	encoder.SetEscapeHTML(false)

	return &rpcPeer{
		scanner:  scanner,
		encoder:  encoder,
		handlers: make(map[string]func(json.RawMessage) (any, error)),
	}
}

func (p *rpcPeer) handle(method string, handler func(json.RawMessage) (any, error)) {
	p.handlers[strings.TrimSpace(method)] = handler
}

func (p *rpcPeer) serve() error {
	for p.scanner.Scan() {
		line := strings.TrimSpace(p.scanner.Text())
		if line == "" {
			continue
		}

		var envelope rpcEnvelope
		if err := json.Unmarshal([]byte(line), &envelope); err != nil {
			return fmt.Errorf("secret-guard: decode rpc frame: %w", err)
		}

		if strings.TrimSpace(envelope.Method) != "" {
			p.wg.Add(1)
			go func(env rpcEnvelope) {
				defer p.wg.Done()
				p.dispatchRequest(env)
			}(envelope)
			continue
		}

		idKey := string(bytesTrim(envelope.ID))
		if idKey == "" {
			continue
		}
		if pending, ok := p.pending.LoadAndDelete(idKey); ok {
			pending.(chan rpcEnvelope) <- envelope
		}
	}

	p.wg.Wait()
	if err := p.scanner.Err(); err != nil {
		return fmt.Errorf("secret-guard: read rpc frame: %w", err)
	}
	return nil
}

func (p *rpcPeer) dispatchRequest(envelope rpcEnvelope) {
	handler, ok := p.handlers[strings.TrimSpace(envelope.Method)]
	if !ok {
		_ = p.sendError(envelope.ID, rpcErrorPayload{Code: -32601, Message: "Method not found"})
		return
	}

	result, err := handler(envelope.Params)
	if err != nil {
		var rpcErr *runtimeRPCError
		if errors.As(err, &rpcErr) {
			_ = p.sendError(envelope.ID, rpcErrorPayload{Code: rpcErr.Code, Message: rpcErr.Message})
			return
		}
		_ = p.sendError(envelope.ID, rpcErrorPayload{Code: -32603, Message: err.Error()})
		return
	}

	_ = p.sendResult(envelope.ID, result)
}

func (p *rpcPeer) call(ctx context.Context, method string, params any, result any) error {
	idValue := fmt.Sprintf("secret-guard-%d", atomic.AddInt64(&p.nextID, 1))
	idBytes, err := json.Marshal(idValue)
	if err != nil {
		return err
	}

	responseCh := make(chan rpcEnvelope, 1)
	p.pending.Store(string(idBytes), responseCh)
	if err := p.writeFrame(rpcEnvelope{
		JSONRPC: "2.0",
		ID:      idBytes,
		Method:  method,
		Params:  mustRawJSON(params),
	}); err != nil {
		p.pending.Delete(string(idBytes))
		return err
	}

	select {
	case <-ctx.Done():
		p.pending.Delete(string(idBytes))
		return ctx.Err()
	case response := <-responseCh:
		if response.Error != nil {
			return &runtimeRPCError{Code: response.Error.Code, Message: response.Error.Message}
		}
		if result == nil {
			return nil
		}
		payload, err := json.Marshal(response.Result)
		if err != nil {
			return err
		}
		if len(payload) == 0 || string(payload) == "null" {
			return nil
		}
		if err := json.Unmarshal(payload, result); err != nil {
			return err
		}
		return nil
	}
}

func (p *rpcPeer) sendResult(id json.RawMessage, result any) error {
	return p.writeFrame(rpcEnvelope{
		JSONRPC: "2.0",
		ID:      append(json.RawMessage(nil), id...),
		Result:  result,
	})
}

func (p *rpcPeer) sendError(id json.RawMessage, rpcErr rpcErrorPayload) error {
	return p.writeFrame(rpcEnvelope{
		JSONRPC: "2.0",
		ID:      append(json.RawMessage(nil), id...),
		Error:   &rpcErr,
	})
}

func (p *rpcPeer) writeFrame(frame rpcEnvelope) error {
	p.writeMu.Lock()
	defer p.writeMu.Unlock()
	return p.encoder.Encode(frame)
}

func (r *secretGuardRuntime) handleInitialize(params json.RawMessage) (any, error) {
	var request subprocess.InitializeRequest
	if err := json.Unmarshal(params, &request); err != nil {
		return nil, fmt.Errorf("secret-guard: decode initialize request: %w", err)
	}

	response := subprocess.InitializeResponse{
		ProtocolVersion: request.ProtocolVersion,
		ExtensionInfo: subprocess.InitializeExtensionInfo{
			Name:    "secret-guard",
			Version: "0.1.0",
		},
		AcceptedCapabilities: subprocess.AcceptedCapabilities{
			Provides: append([]string(nil), request.Capabilities.Provides...),
			Actions:  append([]extensionprotocol.HostAPIMethod(nil), request.Capabilities.GrantedActions...),
			Security: append([]string(nil), request.Capabilities.GrantedSecurity...),
		},
		ImplementedMethods:  []string{"execute_hook", "health_check", "shutdown"},
		SupportedHookEvents: []string{string(hookspkg.HookInputPreSubmit)},
		Supports: subprocess.InitializeSupports{
			HealthCheck: true,
		},
	}

	r.mu.Lock()
	r.initialized = true
	r.session = runtimeSession{request: request, response: response}
	r.mu.Unlock()

	writeJSONFile(os.Getenv(goHandshakeEnv), map[string]any{
		"request":  request,
		"response": response,
		"pid":      os.Getpid(),
	})

	go r.afterInitialize()

	return response, nil
}

func (r *secretGuardRuntime) afterInitialize() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	summaries, err := r.sessionsListWithRetry(ctx)
	marker := map[string]any{}
	if err != nil {
		marker["error"] = err.Error()
	} else {
		marker["session_count"] = len(summaries)
		marker["sessions"] = summaries
	}
	marker["pid"] = os.Getpid()
	writeJSONFile(os.Getenv(goHostCallEnv), marker)

	crashMarker := strings.TrimSpace(os.Getenv(goCrashOnceEnv))
	if crashMarker != "" && !fileExists(crashMarker) {
		writeJSONFile(crashMarker, map[string]any{
			"crashed": true,
			"pid":     os.Getpid(),
		})
		go func() {
			time.Sleep(150 * time.Millisecond)
			os.Exit(23)
		}()
	}
}

func (r *secretGuardRuntime) sessionsListWithRetry(ctx context.Context) ([]hostSessionSummary, error) {
	var lastErr error
	for attempt := 0; attempt < 5; attempt++ {
		var sessions []hostSessionSummary
		err := r.peer.call(ctx, "sessions/list", map[string]any{}, &sessions)
		if err == nil {
			return sessions, nil
		}
		lastErr = err

		var rpcErr *runtimeRPCError
		if errors.As(err, &rpcErr) && rpcErr.Code == -32003 {
			time.Sleep(time.Duration(attempt+1) * 10 * time.Millisecond)
			continue
		}
		if ctx.Err() != nil {
			break
		}
		return nil, err
	}

	if lastErr == nil {
		lastErr = errors.New("secret-guard: sessions/list failed")
	}
	return nil, lastErr
}

func (r *secretGuardRuntime) handleExecuteHook(params json.RawMessage) (any, error) {
	var request executeHookParams
	if err := json.Unmarshal(params, &request); err != nil {
		return nil, fmt.Errorf("secret-guard: decode execute_hook params: %w", err)
	}

	switch hookspkg.HookEvent(strings.TrimSpace(request.Hook.Event)) {
	case hookspkg.HookInputPreSubmit:
		var payload hookspkg.InputPreSubmitPayload
		if err := json.Unmarshal(request.Payload, &payload); err != nil {
			return nil, fmt.Errorf("secret-guard: decode input.pre_submit payload: %w", err)
		}
		return evaluateSecretGuard(payload.Message), nil
	default:
		return map[string]any{}, nil
	}
}

func (r *secretGuardRuntime) handleHealthCheck(json.RawMessage) (any, error) {
	return subprocess.HealthCheckResponse{
		Healthy: true,
		Message: "",
	}, nil
}

func (r *secretGuardRuntime) handleShutdown(params json.RawMessage) (any, error) {
	var request subprocess.ShutdownRequest
	if len(bytesTrim(params)) > 0 && string(bytesTrim(params)) != "null" {
		if err := json.Unmarshal(params, &request); err != nil {
			return nil, fmt.Errorf("secret-guard: decode shutdown request: %w", err)
		}
	}

	appendMarkerLine(os.Getenv(goShutdownEnv), fmt.Sprintf("reason=%s", request.Reason))
	go func() {
		time.Sleep(5 * time.Millisecond)
		os.Exit(0)
	}()

	return subprocess.ShutdownResponse{Acknowledged: true}, nil
}

func evaluateSecretGuard(message string) hookspkg.InputPreSubmitPatch {
	for _, pattern := range secretPatterns {
		if strings.Contains(message, pattern) {
			reason := fmt.Sprintf("Message contains a potential secret (%s)", pattern)
			return hookspkg.InputPreSubmitPatch{
				ControlPatch: hookspkg.ControlPatch{
					Deny:       true,
					DenyReason: reason,
				},
			}
		}
	}
	return hookspkg.InputPreSubmitPatch{}
}

type runtimeRPCError struct {
	Code    int
	Message string
}

func (e *runtimeRPCError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

func mustRawJSON(value any) json.RawMessage {
	payload, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return payload
}

func bytesTrim(value []byte) []byte {
	return []byte(strings.TrimSpace(string(value)))
}

func writeJSONFile(path string, value any) {
	target := strings.TrimSpace(path)
	if target == "" {
		return
	}

	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return
	}

	payload, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return
	}
	payload = append(payload, '\n')
	_ = os.WriteFile(target, payload, 0o644)
}

func appendMarkerLine(path string, line string) {
	target := strings.TrimSpace(path)
	if target == "" {
		return
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return
	}
	file, err := os.OpenFile(target, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer func() {
		_ = file.Close()
	}()
	_, _ = fmt.Fprintln(file, strings.TrimSpace(line))
}

func fileExists(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}
