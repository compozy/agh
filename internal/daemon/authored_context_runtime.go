package daemon

import (
	"context"
	"errors"
	"log/slog"

	"github.com/pedronauck/agh/internal/acp"
	core "github.com/pedronauck/agh/internal/api/core"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/heartbeat"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/soul"
)

type authoredContextDeps struct {
	SoulAuthoring      core.SoulAuthoringService
	SoulRefresher      core.SoulRefresher
	HeartbeatAuthoring core.HeartbeatAuthoringService
	HeartbeatStatus    core.HeartbeatStatusService
	HeartbeatWake      core.HeartbeatWakeService
	SessionHealth      core.SessionHealthReader
	WakeEvents         core.HeartbeatWakeEventReader
}

type apiHeartbeatWakePrompter struct {
	ctx      context.Context
	sessions SessionManager
	logger   *slog.Logger
}

func authoredContextRuntimeDeps(ctx context.Context, state *bootState, sessions SessionManager) authoredContextDeps {
	var deps authoredContextDeps
	if state == nil {
		return deps
	}
	deps.SoulAuthoring = soulAuthoringServiceDependency(state.registry, state.logger)
	deps.SoulRefresher = soulRefresherDependency(sessions)
	deps.HeartbeatAuthoring = heartbeatAuthoringServiceDependency(state.registry, state.logger)
	deps.SessionHealth = sessionHealthReaderDependency(sessions)
	deps.HeartbeatStatus = heartbeatStatusServiceDependency(state.registry, sessions, state.logger)
	deps.HeartbeatWake = heartbeatWakeServiceDependency(
		ctx,
		state.registry,
		sessions,
		state.cfg.Agents.Heartbeat,
		state.logger,
	)
	deps.WakeEvents = heartbeatWakeEventReaderDependency(state.registry)
	return deps
}

func soulAuthoringServiceDependency(store any, logger *slog.Logger) core.SoulAuthoringService {
	authoringStore, ok := store.(soul.AuthoringStore)
	if !ok {
		return nil
	}
	service, err := soul.NewManagedSoulAuthoringService(authoringStore)
	if err != nil {
		logAuthoredContextDependencyError(logger, "daemon: create soul authoring service", err)
		return nil
	}
	return service
}

func soulRefresherDependency(sessions SessionManager) core.SoulRefresher {
	refresher, ok := sessions.(core.SoulRefresher)
	if !ok {
		return nil
	}
	return refresher
}

func heartbeatAuthoringServiceDependency(store any, logger *slog.Logger) core.HeartbeatAuthoringService {
	authoringStore, ok := store.(heartbeat.AuthoringStore)
	if !ok {
		return nil
	}
	service, err := heartbeat.NewManagedHeartbeatAuthoringService(authoringStore)
	if err != nil {
		logAuthoredContextDependencyError(logger, "daemon: create heartbeat authoring service", err)
		return nil
	}
	return service
}

func heartbeatStatusServiceDependency(
	store any,
	sessions SessionManager,
	logger *slog.Logger,
) core.HeartbeatStatusService {
	statusStore, ok := store.(heartbeat.StatusStore)
	if !ok {
		return nil
	}
	options := make([]heartbeat.StatusOption, 0, 1)
	if reader, ok := sessions.(heartbeat.SessionHealthReader); ok {
		options = append(options, heartbeat.WithHeartbeatStatusSessionHealthReader(reader))
	}
	service, err := heartbeat.NewManagedHeartbeatStatusService(statusStore, options...)
	if err != nil {
		logAuthoredContextDependencyError(logger, "daemon: create heartbeat status service", err)
		return nil
	}
	return service
}

func heartbeatWakeServiceDependency(
	ctx context.Context,
	store any,
	sessions SessionManager,
	config aghconfig.HeartbeatConfig,
	logger *slog.Logger,
) core.HeartbeatWakeService {
	wakeStore, ok := store.(heartbeat.WakeStore)
	if !ok {
		return nil
	}
	healthReader, ok := sessions.(heartbeat.SessionHealthReader)
	if !ok {
		return nil
	}
	service, err := heartbeat.NewManagedWakeService(
		wakeStore,
		healthReader,
		&apiHeartbeatWakePrompter{ctx: authoredContextLifecycle(ctx), sessions: sessions, logger: logger},
		config,
	)
	if err != nil {
		logAuthoredContextDependencyError(logger, "daemon: create heartbeat wake service", err)
		return nil
	}
	return service
}

func authoredContextLifecycle(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

func sessionHealthReaderDependency(sessions SessionManager) core.SessionHealthReader {
	reader, ok := sessions.(core.SessionHealthReader)
	if !ok {
		return nil
	}
	return reader
}

func heartbeatWakeEventReaderDependency(store any) core.HeartbeatWakeEventReader {
	reader, ok := store.(core.HeartbeatWakeEventReader)
	if !ok {
		return nil
	}
	return reader
}

func (p *apiHeartbeatWakePrompter) PromptHeartbeatWake(
	ctx context.Context,
	req heartbeat.SyntheticWakePromptRequest,
) (heartbeat.SyntheticWakePromptResult, error) {
	if p == nil || p.sessions == nil {
		return heartbeat.SyntheticWakePromptResult{}, errors.New("daemon: api heartbeat prompter requires sessions")
	}
	synthetic, ok := p.sessions.(schedulerSyntheticPrompter)
	if !ok {
		return heartbeat.SyntheticWakePromptResult{}, errors.New(
			"daemon: api heartbeat prompter requires synthetic prompt support",
		)
	}
	events, err := synthetic.PromptSynthetic(ctx, req.SessionID, session.SyntheticPromptOpts{
		Message: req.Message,
		TurnID:  req.TurnID,
		Metadata: acp.PromptSyntheticMeta{
			Reason:           heartbeat.SyntheticReasonHeartbeatWake,
			Summary:          req.Summary,
			WakeEventID:      req.WakeEventID,
			PolicySnapshotID: req.PolicySnapshotID,
			PolicyDigest:     req.PolicyDigest,
			ConfigDigest:     req.ConfigDigest,
		},
		SkipIfBusy: true,
	})
	if err != nil {
		if errors.Is(err, session.ErrPromptInProgress) {
			return heartbeat.SyntheticWakePromptResult{}, heartbeat.ErrSyntheticPromptBusy
		}
		return heartbeat.SyntheticWakePromptResult{}, err
	}
	p.drainEvents(req.SessionID, req.WakeEventID, events)
	return heartbeat.SyntheticWakePromptResult{SyntheticPromptID: req.TurnID}, nil
}

func (p *apiHeartbeatWakePrompter) drainEvents(sessionID string, wakeEventID string, events <-chan acp.AgentEvent) {
	if events == nil {
		return
	}
	drainCtx := context.Background()
	if p != nil && p.ctx != nil {
		drainCtx = p.ctx
	}
	go func() {
		for {
			select {
			case <-drainCtx.Done():
				return
			case event, ok := <-events:
				if !ok {
					return
				}
				if event.Type == acp.EventTypeError && p != nil && p.logger != nil {
					p.logger.Warn(
						"api.heartbeat_wake.agent_error",
						"session_id", sessionID,
						"wake_event_id", wakeEventID,
					)
				}
			}
		}
	}()
}

func logAuthoredContextDependencyError(logger *slog.Logger, message string, err error) {
	if logger == nil || err == nil {
		return
	}
	logger.Warn(message, "error", err)
}
