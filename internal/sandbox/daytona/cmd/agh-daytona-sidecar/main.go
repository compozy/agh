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
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

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
)

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
	mu     sync.Mutex
	cond   *sync.Cond
	chunks [][]byte
	closed bool
}

func newChunkQueue() *chunkQueue {
	q := &chunkQueue{}
	q.cond = sync.NewCond(&q.mu)
	return q
}

func (q *chunkQueue) Push(chunk []byte) {
	if len(chunk) == 0 {
		return
	}
	copied := append([]byte(nil), chunk...)
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed {
		return
	}
	q.chunks = append(q.chunks, copied)
	q.cond.Signal()
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
	return chunk, true
}

type managedProcess struct {
	id       string
	command  string
	cmd      *exec.Cmd
	cancel   context.CancelFunc
	stdin    io.WriteCloser
	stdout   *chunkQueue
	stderr   bytes.Buffer
	stderrMu sync.Mutex
	done     chan struct{}
	exitCode int
	stopOnce sync.Once
}

func newManagedProcess(command string) (*managedProcess, error) {
	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, "/bin/sh", "-lc", command)
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

func (p *managedProcess) captureStdout(stdout io.ReadCloser) {
	defer p.stdout.Close()
	defer stdout.Close()
	buf := make([]byte, 64*1024)
	for {
		n, err := stdout.Read(buf)
		if n > 0 {
			p.stdout.Push(buf[:n])
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
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
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
		if err := p.cmd.Process.Signal(os.Interrupt); err != nil && !errors.Is(err, os.ErrProcessDone) {
			stopErr = errors.Join(stopErr, err)
		}
		select {
		case <-p.done:
			if p.cancel != nil {
				p.cancel()
			}
			return
		case <-time.After(5 * time.Second):
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
	p.stderr.WriteString(text)
}

func (p *managedProcess) stderrText() string {
	p.stderrMu.Lock()
	defer p.stderrMu.Unlock()
	return p.stderr.String()
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

func main() {
	port := flag.Int("port", 0, "listen port")
	flag.Parse()
	if *port <= 0 {
		log.Fatal("port is required")
	}

	store := newProcessStore()
	upgrader := websocket.Upgrader{
		CheckOrigin: func(*http.Request) bool { return true },
	}

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
		process, found := store.Get(sessionID)
		if !found {
			http.Error(w, "session not found", http.StatusNotFound)
			return
		}
		switch {
		case r.Method == http.MethodDelete && suffix == "":
			if err := process.Stop(); err != nil {
				http.Error(w, fmt.Sprintf("stop session: %v", err), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && suffix == "/stream":
			handleStream(w, r, process, &upgrader)
		default:
			http.NotFound(w, r)
		}
	})

	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", *port),
		Handler:           mux,
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
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
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
