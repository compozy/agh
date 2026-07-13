package bridgesdk

import (
	"strings"
	"sync"
)

// DeliveryStateStore owns provider-specific delivery snapshots without deciding their lifecycle.
type DeliveryStateStore[S any] struct {
	mu     sync.RWMutex
	states map[string]S
}

// NewDeliveryStateStore creates an empty typed delivery-state store.
func NewDeliveryStateStore[S any]() *DeliveryStateStore[S] {
	return &DeliveryStateStore[S]{states: make(map[string]S)}
}

// Load returns one delivery state.
func (s *DeliveryStateStore[S]) Load(deliveryID string) (S, bool) {
	var zero S
	if s == nil {
		return zero, false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	state, ok := s.states[strings.TrimSpace(deliveryID)]
	return state, ok
}

// Store replaces one delivery state.
func (s *DeliveryStateStore[S]) Store(deliveryID string, state S) {
	if s == nil || strings.TrimSpace(deliveryID) == "" {
		return
	}
	s.mu.Lock()
	s.states[strings.TrimSpace(deliveryID)] = state
	s.mu.Unlock()
}

// Update mutates one delivery state atomically and reports whether it existed.
func (s *DeliveryStateStore[S]) Update(deliveryID string, update func(S) S) bool {
	if s == nil || update == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	deliveryID = strings.TrimSpace(deliveryID)
	state, ok := s.states[deliveryID]
	if !ok {
		return false
	}
	s.states[deliveryID] = update(state)
	return true
}

// Delete removes and returns one delivery state.
func (s *DeliveryStateStore[S]) Delete(deliveryID string) (S, bool) {
	var zero S
	if s == nil {
		return zero, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	deliveryID = strings.TrimSpace(deliveryID)
	state, ok := s.states[deliveryID]
	if !ok {
		return zero, false
	}
	delete(s.states, deliveryID)
	return state, true
}

// Drain removes and returns every delivery state.
func (s *DeliveryStateStore[S]) Drain() map[string]S {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	drained := s.states
	s.states = make(map[string]S)
	return drained
}

// TransformAll atomically replaces every retained state and returns the prior snapshot.
func (s *DeliveryStateStore[S]) TransformAll(transform func(string, S) S) map[string]S {
	if s == nil || transform == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	previous := make(map[string]S, len(s.states))
	for key, state := range s.states {
		previous[key] = state
		s.states[key] = transform(key, state)
	}
	return previous
}
