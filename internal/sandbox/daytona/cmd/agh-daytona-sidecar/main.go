package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/compozy/agh/internal/procutil"
	"github.com/gorilla/websocket"
)

const (
	version               = "agh-daytona-launcher-sidecar-v1"
	serverStdoutFrame     = 0x01
	serverStderrFrame     = 0x02
	serverExitFrame       = 0x03
	serverErrorFrame      = 0x04
	clientStdinFrame      = 0x01
	clientCloseStdinFrame = 0x02
	clientStopFrame       = 0x03
	stopTimeout           = 5 * time.Second
	stdoutBufferLimit     = 4 * 1024 * 1024
	stderrBufferLimit     = 1024 * 1024
	stderrTruncatedMarker = "\n[stderr truncated]\n"
)

var errOutputBufferExceeded = errors.New("sidecar output buffer exceeded")

type launchRequest struct {
	Command string `json:"command"`
}

type launchResponse struct {
	ID string `json:"id"`
}

type healthResponse struct {
	OK      bool   `json:"ok"`
	Version string `json:"version"`
}

type exitPayload struct {
	ExitCode int    `json:"exitCode"`
	Stderr   string `json:"stderr"`
}

type frameWriter func([]byte) error

type chunkQueue struct {
	mu            sync.Mutex
	cond          *sync.Cond
	chunks        [][]byte
	bufferedBytes int
	maxBytes      int
	closed        bool
}

func newChunkQueue() *chunkQueue {
	q := &chunkQueue{maxBytes: stdoutBufferLimit}
	q.cond = sync.NewCond(&q.mu)
	return q
}

func (q *chunkQueue) Push(chunk []byte) error {
	if len(chunk) == 0 {
		return nil
	}
	copied := append([]byte(nil), chunk...)
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed {
		return nil
	}
	if q.maxBytes > 0 && q.bufferedBytes+len(copied) > q.maxBytes {
		q.closed = true
		q.cond.Broadcast()
		return fmt.Errorf("%w: stdout buffer exceeds %d bytes", errOutputBufferExceeded, q.maxBytes)
	}
	q.chunks = append(q.chunks, copied)
	q.bufferedBytes += len(copied)
	q.cond.Signal()
	return nil
}

func (q *chunkQueue) Close() {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed {
		return
	}
	q.closed = true
	q.cond.Broadcast()
}

func (q *chunkQueue) Pop() ([]byte, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	for len(q.chunks) == 0 && !q.closed {
		q.cond.Wait()
	}
	if len(q.chunks) == 0 {
		return nil, false
	}
	chunk := q.chunks[0]
	q.chunks[0] = nil
	q.chunks = q.chunks[1:]
	q.bufferedBytes -= len(chunk)
	return chunk, true
}

type managedProcess struct {
	id              string
	command         string
	cmd             *exec.Cmd
	cancel          context.CancelFunc
	stdin           io.WriteCloser
	stdout          *chunkQueue
	stderr          bytes.Buffer
	stderrMu        sync.Mutex
	stderrTruncated bool
	done            chan struct{}
	exitCode        int
	stopOnce        sync.Once
	streamMu        sync.Mutex
	streamClaimed   bool
}

func newManagedProcess(command string) (*managedProcess, error) {
	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, "/bin/sh", "-lc", command)
	procutil.ConfigureCommandProcessGroup(cmd)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("open stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("open stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("open stderr pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		cancel()
		return nil, fmt.Errorf("start command: %w", err)
	}
	if err := procutil.RegisterCommandProcessGroup(cmd); err != nil {
		cancel()
		return nil, errors.Join(
			fmt.Errorf("register command process group: %w", err),
			cleanupStartedManagedCommand(cmd),
		)
	}
	process := &managedProcess{
		id:       randomID(),
		command:  command,
		cmd:      cmd,
		cancel:   cancel,
		stdin:    stdin,
		stdout:   newChunkQueue(),
		done:     make(chan struct{}),
		exitCode: -1,
	}
	go process.captureStdout(stdout)
	go process.captureStderr(stderr)
	go process.wait()
	return process, nil
}

func cleanupStartedManagedCommand(cmd *exec.Cmd) error {
	var errs []error
	if err := procutil.SignalCommandProcessGroup(cmd, syscall.SIGKILL); err != nil {
		errs = append(errs, fmt.Errorf("signal command process group: %w", err))
	}
	if err := cmd.Wait(); err != nil {
		errs = append(errs, fmt.Errorf("wait after cleanup: %w", err))
	}
	if err := procutil.KillCommandProcessGroupAndWait(cmd, stopTimeout); err != nil {
		errs = append(errs, fmt.Errorf("wait for command process group exit: %w", err))
	}
	return errors.Join(errs...)
}

func (p *managedProcess) captureStdout(stdout io.ReadCloser) {
	defer p.stdout.Close()
	defer stdout.Close()
	buf := make([]byte, 64*1024)
	for {
		n, err := stdout.Read(buf)
		if n > 0 {
			if pushErr := p.stdout.Push(buf[:n]); pushErr != nil {
				p.appendStderr(pushErr.Error() + "\n")
				if stopErr := p.Stop(); stopErr != nil {
					p.appendStderr(fmt.Sprintf("stop after stdout buffer overflow: %v\n", stopErr))
				}
				return
			}
		}
		if err != nil {
			if !errors.Is(err, io.EOF) {
				p.appendStderr(fmt.Sprintf("stdout read error: %v\n", err))
			}
			return
		}
	}
}

func (p *managedProcess) captureStderr(stderr io.ReadCloser) {
	defer stderr.Close()
	buf := make([]byte, 64*1024)
	for {
		n, err := stderr.Read(buf)
		if n > 0 {
			p.appendStderr(string(buf[:n]))
		}
		if err != nil {
			if !errors.Is(err, io.EOF) {
				p.appendStderr(fmt.Sprintf("stderr read error: %v\n", err))
			}
			return
		}
	}
}

func (p *managedProcess) wait() {
	defer close(p.done)
	if err := p.cmd.Wait(); err != nil {
		if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
			p.exitCode = exitErr.ExitCode()
			return
		}
		p.appendStderr(fmt.Sprintf("wait error: %v\n", err))
		p.exitCode = 1
		return
	}
	p.exitCode = 0
}

func (p *managedProcess) WriteStdin(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	if p.stdin == nil {
		return errors.New("stdin is closed")
	}
	if _, err := p.stdin.Write(data); err != nil {
		return err
	}
	return nil
}

func (p *managedProcess) CloseStdin() error {
	if p.stdin == nil {
		return nil
	}
	err := p.stdin.Close()
	p.stdin = nil
	return err
}

func (p *managedProcess) Stop() error {
	var stopErr error
	p.stopOnce.Do(func() {
		if err := p.CloseStdin(); err != nil {
			stopErr = errors.Join(stopErr, err)
		}
		if p.cmd.Process == nil {
			if p.cancel != nil {
				p.cancel()
			}
			return
		}
		if err := procutil.SignalCommandProcessGroup(p.cmd, syscall.SIGTERM); err != nil {
			stopErr = errors.Join(stopErr, err)
		}
		select {
		case <-p.done:
			if err := procutil.WaitForCommandProcessGroupExit(p.cmd, stopTimeout); err != nil {
				stopErr = errors.Join(stopErr, err)
			}
			if p.cancel != nil {
				p.cancel()
			}
			return
		case <-time.After(stopTimeout):
		}
		if err := procutil.KillCommandProcessGroupAndWait(p.cmd, stopTimeout); err != nil {
			stopErr = errors.Join(stopErr, err)
		}
		if p.cancel != nil {
			p.cancel()
		}
		<-p.done
	})
	return stopErr
}

func (p *managedProcess) appendStderr(text string) {
	if text == "" {
		return
	}
	p.stderrMu.Lock()
	defer p.stderrMu.Unlock()
	if p.stderr.Len() >= stderrBufferLimit {
		p.stderrTruncated = true
		return
	}
	remaining := stderrBufferLimit - p.stderr.Len()
	if len(text) > remaining {
		p.stderr.WriteString(text[:remaining])
		p.stderrTruncated = true
		return
	}
	p.stderr.WriteString(text)
}

func (p *managedProcess) stderrText() string {
	p.stderrMu.Lock()
	defer p.stderrMu.Unlock()
	text := p.stderr.String()
	if p.stderrTruncated {
		return text + stderrTruncatedMarker
	}
	return text
}

func (p *managedProcess) claimStream() bool {
	p.streamMu.Lock()
	defer p.streamMu.Unlock()
	if p.streamClaimed {
		return false
	}
	p.streamClaimed = true
	return true
}

func (p *managedProcess) releaseUnstartedStream() {
	p.streamMu.Lock()
	defer p.streamMu.Unlock()
	p.streamClaimed = false
}

type processStore struct {
	mu        sync.Mutex
	processes map[string]*managedProcess
}

func newProcessStore() *processStore {
	return &processStore{processes: make(map[string]*managedProcess)}
}

func (s *processStore) Put(process *managedProcess) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.processes[process.id] = process
}

func (s *processStore) Get(id string) (*managedProcess, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	process, ok := s.processes[id]
	return process, ok
}

func (s *processStore) Take(id string) (*managedProcess, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	process, ok := s.processes[id]
	if ok {
		delete(s.processes, id)
	}
	return process, ok
}

func sidecarListenAddr(port int) string {
	return fmt.Sprintf("127.0.0.1:%d", port)
}

func allowWebSocketOrigin(r *http.Request) bool {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		return true
	}
	parsed, err := url.Parse(origin)
	if err != nil {
		return false
	}
	return strings.EqualFold(parsed.Host, r.Host)
}

func newHandler(store *processStore, upgrader *websocket.Upgrader) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, healthResponse{OK: true, Version: version})
	})
	mux.HandleFunc("/v1/launch", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var request launchRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, fmt.Sprintf("decode launch request: %v", err), http.StatusBadRequest)
			return
		}
		process, err := newManagedProcess(request.Command)
		if err != nil {
			http.Error(w, fmt.Sprintf("launch command: %v", err), http.StatusBadRequest)
			return
		}
		store.Put(process)
		writeJSON(w, http.StatusCreated, launchResponse{ID: process.id})
	})
	mux.HandleFunc("/v1/sessions/", func(w http.ResponseWriter, r *http.Request) {
		sessionID, suffix, ok := splitSessionPath(r.URL.Path)
		if !ok {
			http.NotFound(w, r)
			return
		}
		switch {
		case r.Method == http.MethodDelete && suffix == "":
			process, found := store.Take(sessionID)
			if !found {
				http.Error(w, "session not found", http.StatusNotFound)
				return
			}
			if err := process.Stop(); err != nil {
				http.Error(w, fmt.Sprintf("stop session: %v", err), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && suffix == "/stream":
			process, found := store.Get(sessionID)
			if !found {
				http.Error(w, "session not found", http.StatusNotFound)
				return
			}
			handleStream(w, r, process, upgrader)
		default:
			http.NotFound(w, r)
		}
	})
	return mux
}

func main() {
	port := flag.Int("port", 0, "listen port")
	flag.Parse()
	if *port <= 0 {
		log.Fatal("port is required")
	}

	store := newProcessStore()
	upgrader := websocket.Upgrader{
		CheckOrigin: allowWebSocketOrigin,
	}

	server := &http.Server{
		Addr:              sidecarListenAddr(*port),
		Handler:           newHandler(store, &upgrader),
		ReadHeaderTimeout: 10 * time.Second,
	}
	log.Fatal(server.ListenAndServe())
}

func handleStream(
	w http.ResponseWriter,
	r *http.Request,
	process *managedProcess,
	upgrader *websocket.Upgrader,
) {
	if !process.claimStream() {
		http.Error(w, "session stream already attached", http.StatusConflict)
		return
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		process.releaseUnstartedStream()
		return
	}
	defer conn.Close()

	var writeMu sync.Mutex
	writeBinary := func(payload []byte) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		return conn.WriteMessage(websocket.BinaryMessage, payload)
	}
	go streamStdoutFrames(process, writeBinary)
	exitDone := make(chan struct{})
	go streamExitFrame(process, writeBinary, exitDone)

	for {
		_, payload, err := conn.ReadMessage()
		if err != nil {
			return
		}
		if shouldClose := handleClientFrame(process, payload, writeBinary); shouldClose {
			return
		}
		select {
		case <-exitDone:
			return
		default:
		}
	}
}

func streamStdoutFrames(process *managedProcess, writeBinary frameWriter) {
	for {
		chunk, ok := process.stdout.Pop()
		if !ok {
			return
		}
		if !writeServerFrame(writeBinary, serverStdoutFrame, chunk, "write sidecar stdout frame") {
			return
		}
	}
}

func streamExitFrame(process *managedProcess, writeBinary frameWriter, exitDone chan<- struct{}) {
	defer close(exitDone)
	<-process.done
	payload, err := json.Marshal(exitPayload{
		ExitCode: process.exitCode,
		Stderr:   process.stderrText(),
	})
	if err != nil {
		writeServerFrame(writeBinary, serverErrorFrame, []byte(err.Error()), "write sidecar error frame")
		return
	}
	writeServerFrame(writeBinary, serverExitFrame, payload, "write sidecar exit frame")
}

func handleClientFrame(process *managedProcess, payload []byte, writeBinary frameWriter) bool {
	if len(payload) == 0 {
		return false
	}
	var err error
	logMessage := ""
	switch payload[0] {
	case clientStdinFrame:
		err = process.WriteStdin(payload[1:])
		logMessage = "write sidecar stdin error frame"
	case clientCloseStdinFrame:
		err = process.CloseStdin()
		logMessage = "write sidecar close-stdin error frame"
	case clientStopFrame:
		err = process.Stop()
		logMessage = "write sidecar stop error frame"
	default:
		return false
	}
	if err == nil {
		return false
	}
	writeServerFrame(writeBinary, serverErrorFrame, []byte(err.Error()), logMessage)
	return true
}

func writeServerFrame(writeBinary frameWriter, frame byte, payload []byte, logMessage string) bool {
	framed := append([]byte{frame}, payload...)
	if err := writeBinary(framed); err != nil {
		log.Printf("%s: %v", logMessage, err)
		return false
	}
	return true
}

func splitSessionPath(raw string) (string, string, bool) {
	const prefix = "/v1/sessions/"
	if !strings.HasPrefix(raw, prefix) {
		return "", "", false
	}
	remainder := strings.TrimPrefix(raw, prefix)
	if remainder == "" {
		return "", "", false
	}
	parts := strings.SplitN(remainder, "/", 2)
	if len(parts) == 1 {
		return parts[0], "", true
	}
	return parts[0], "/" + parts[1], true
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
	w.WriteHeader(status)
	if _, err := w.Write(data); err != nil {
		log.Printf("write JSON response: %v", err)
	}
}

func randomID() string {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		panic(err)
	}
	return hex.EncodeToString(bytes[:])
}
