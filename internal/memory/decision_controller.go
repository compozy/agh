package memory

import (
	"context"
	"errors"
	"sync"

	memcontract "github.com/compozy/agh/internal/memory/contract"
	"github.com/compozy/agh/internal/memory/controller"
)

// DecisionControllerFactory builds the effective write controller for one
// store view. Store clones share the live factory registration.
type DecisionControllerFactory func(controller.TargetIndex) memcontract.Controller

type decisionControllerFactoryState struct {
	mu      sync.RWMutex
	factory DecisionControllerFactory
}

// SetDecisionControllerFactory installs the daemon-owned live controller
// composition used by writes, batches, and dry-run decisions.
func (s *Store) SetDecisionControllerFactory(factory DecisionControllerFactory) {
	if s == nil {
		return
	}
	if s.decisionFactory == nil {
		s.decisionFactory = &decisionControllerFactoryState{}
	}
	s.decisionFactory.mu.Lock()
	s.decisionFactory.factory = factory
	s.decisionFactory.mu.Unlock()
}

// DecideCandidate computes a controller decision without applying it.
func (s *Store) DecideCandidate(
	ctx context.Context,
	candidate memcontract.Candidate,
) (memcontract.Decision, error) {
	if ctx == nil {
		return memcontract.Decision{}, errors.New("memory: decide candidate context is required")
	}
	if s == nil {
		return memcontract.Decision{}, errors.New("memory: store is required")
	}
	return s.newDecisionController().Decide(ctx, candidate)
}

func (s *Store) newDecisionController() memcontract.Controller {
	if s.decisionFactory != nil {
		s.decisionFactory.mu.RLock()
		factory := s.decisionFactory.factory
		s.decisionFactory.mu.RUnlock()
		if factory != nil {
			if built := factory(s); built != nil {
				return built
			}
		}
	}
	return controller.New(s)
}
