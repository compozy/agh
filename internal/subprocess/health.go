package subprocess

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// HealthCheckResponse is the structured result of the health_check RPC.
type HealthCheckResponse struct {
	Healthy bool            `json:"healthy"`
	Message string          `json:"message,omitempty"`
	Details json.RawMessage `json:"details,omitempty"`
}

// HealthState captures the current health-monitor snapshot for the subprocess.
type HealthState struct {
	Healthy             bool
	Message             string
	Details             json.RawMessage
	LastCheckedAt       time.Time
	ConsecutiveFailures int
	LastError           string
}

type healthMonitor struct {
	mu     sync.RWMutex
	state  HealthState
	cancel context.CancelFunc
	done   chan struct{}
}

// HealthState returns the latest health-monitor snapshot.
func (p *Process) HealthState() HealthState {
	if p == nil {
		return HealthState{}
	}
	p.health.mu.RLock()
	defer p.health.mu.RUnlock()
	state := p.health.state
	if state.Details != nil {
		state.Details = append(json.RawMessage(nil), state.Details...)
	}
	return state
}

func (p *Process) maybeStartHealthMonitor(runtime InitializeRuntime, supports InitializeSupports) {
	if !supports.HealthCheck || runtime.HealthCheckIntervalMS <= 0 || runtime.HealthCheckTimeoutMS <= 0 {
		return
	}

	p.health.mu.Lock()
	if p.health.cancel != nil {
		p.health.mu.Unlock()
		return
	}

	ctx, cancel := context.WithCancel(p.lifecycleCtx)
	p.health.cancel = cancel
	p.health.done = make(chan struct{})
	p.health.state = HealthState{Healthy: true}
	p.health.mu.Unlock()

	interval := time.Duration(runtime.HealthCheckIntervalMS) * time.Millisecond
	timeout := time.Duration(runtime.HealthCheckTimeoutMS) * time.Millisecond
	go p.runHealthMonitor(ctx, interval, timeout)
}

func (p *Process) runHealthMonitor(ctx context.Context, interval, timeout time.Duration) {
	defer close(p.health.done)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.runHealthProbe(timeout)
		}
	}
}

func (p *Process) runHealthProbe(timeout time.Duration) {
	probeCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var response HealthCheckResponse
	err := p.Call(probeCtx, "health_check", struct{}{}, &response)
	if err != nil {
		p.recordHealthFailure(fmt.Errorf("health_check: %w", err))
		return
	}
	if !response.Healthy {
		p.markUnhealthy(response.Message, response.Details, "")
		return
	}
	p.recordHealthy(response)
}

func (p *Process) recordHealthy(response HealthCheckResponse) {
	p.health.mu.Lock()
	defer p.health.mu.Unlock()
	p.health.state = HealthState{
		Healthy:             true,
		Message:             response.Message,
		Details:             append(json.RawMessage(nil), response.Details...),
		LastCheckedAt:       time.Now().UTC(),
		ConsecutiveFailures: 0,
	}
}

func (p *Process) recordHealthFailure(err error) {
	p.health.mu.Lock()
	defer p.health.mu.Unlock()

	state := p.health.state
	state.LastCheckedAt = time.Now().UTC()
	state.ConsecutiveFailures++
	state.LastError = err.Error()
	if state.ConsecutiveFailures >= p.healthThreshold {
		state.Healthy = false
	}
	p.health.state = state
}

func (p *Process) markUnhealthy(message string, details json.RawMessage, lastError string) {
	p.health.mu.Lock()
	defer p.health.mu.Unlock()
	p.health.state = HealthState{
		Healthy:             false,
		Message:             message,
		Details:             append(json.RawMessage(nil), details...),
		LastCheckedAt:       time.Now().UTC(),
		ConsecutiveFailures: 1,
		LastError:           lastError,
	}
}

func (p *Process) stopHealthMonitor() {
	p.health.mu.Lock()
	cancel := p.health.cancel
	done := p.health.done
	p.health.cancel = nil
	p.health.done = nil
	p.health.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if done != nil {
		<-done
	}
}
