package daemon

import (
	"context"
	"errors"
	"fmt"
	"strings"

	looppkg "github.com/compozy/agh/internal/loop"
	"github.com/compozy/agh/internal/session"
	"github.com/compozy/agh/internal/store"
	taskpkg "github.com/compozy/agh/internal/task"
)

func (s *daemonLoopAPIService) startSessionGoal(
	ctx context.Context,
	workspaceID string,
	sessionID string,
	objective string,
	actor taskpkg.ActorContext,
) (session.GoalDispatchDecision, error) {
	contract, err := session.ParseGoalObjective(objective)
	if err != nil {
		return s.commandErrorDecision(ctx, workspaceID, sessionID, err)
	}
	origin, agent, maxTurns, decision, err := s.prepareSessionGoalDefinition(
		ctx,
		workspaceID,
		sessionID,
	)
	if err != nil || decision.Result != nil {
		return decision, err
	}
	definition := buildSessionGoalDefinition(contract, agent, maxTurns)
	run, err := s.aggregate.StartInline(
		ctx,
		looppkg.WorkspaceID(strings.TrimSpace(workspaceID)),
		definition,
		looppkg.Inputs{},
		origin,
		actor,
	)
	if err != nil {
		return s.commandErrorDecision(ctx, workspaceID, sessionID, err)
	}
	snapshot, err := s.requireSessionGoalSnapshot(ctx, workspaceID, sessionID, run.ID)
	if err != nil {
		return session.GoalDispatchDecision{}, err
	}
	return goalCommandSuccessDecision(session.GoalOutcomeStarted, snapshot, nil), nil
}

func (s *daemonLoopAPIService) replaceSessionGoal(
	ctx context.Context,
	workspaceID string,
	sessionID string,
	expectedRunID string,
	objective string,
	actor taskpkg.ActorContext,
) (session.GoalDispatchDecision, error) {
	contract, err := session.ParseGoalObjective(objective)
	if err != nil {
		return s.commandErrorDecision(ctx, workspaceID, sessionID, err)
	}
	origin, agent, maxTurns, decision, err := s.prepareSessionGoalDefinition(
		ctx,
		workspaceID,
		sessionID,
	)
	if err != nil || decision.Result != nil {
		return decision, err
	}
	definition := buildSessionGoalDefinition(contract, agent, maxTurns)
	replaced, err := s.aggregate.ReplaceInline(
		ctx,
		looppkg.RunID(strings.TrimSpace(expectedRunID)),
		looppkg.WorkspaceID(strings.TrimSpace(workspaceID)),
		definition,
		looppkg.Inputs{},
		origin,
		actor,
	)
	if err != nil {
		return s.commandErrorDecision(ctx, workspaceID, sessionID, err)
	}
	if replaced.Run == nil {
		return session.GoalDispatchDecision{}, errors.New("daemon: Goal replacement returned no Run")
	}
	snapshot, err := s.requireSessionGoalSnapshot(ctx, workspaceID, sessionID, replaced.Run.ID)
	if err != nil {
		return session.GoalDispatchDecision{}, err
	}
	replacedRunID := string(replaced.ReplacedRunID)
	return goalCommandSuccessDecision(session.GoalOutcomeReplaced, snapshot, &replacedRunID), nil
}

func (s *daemonLoopAPIService) prepareSessionGoalDefinition(
	ctx context.Context,
	workspaceID string,
	sessionID string,
) (
	looppkg.RunOrigin,
	string,
	int,
	session.GoalDispatchDecision,
	error,
) {
	origin, profile, reason, err := s.sessionGoalOrigin(ctx, workspaceID, sessionID)
	if err != nil {
		return looppkg.RunOrigin{}, "", 0, session.GoalDispatchDecision{}, err
	}
	if reason != "" {
		return looppkg.RunOrigin{}, "", 0, goalCommandErrorDecision(reason, nil), nil
	}
	cfg, err := resolveLoopServiceConfig(
		ctx,
		s.homePaths,
		s.workspaceResolver,
		looppkg.WorkspaceID(strings.TrimSpace(workspaceID)),
	)
	if err != nil {
		return looppkg.RunOrigin{}, "", 0, session.GoalDispatchDecision{}, err
	}
	return origin, profile.AgentName, cfg.Goals.MaxTurns, session.GoalDispatchDecision{}, nil
}

func (s *daemonLoopAPIService) sessionGoalOrigin(
	ctx context.Context,
	workspaceID string,
	sessionID string,
) (looppkg.RunOrigin, store.SessionCreationProfile, session.GoalReasonCode, error) {
	if s.sessionStatus == nil {
		return looppkg.RunOrigin{}, store.SessionCreationProfile{}, "", errors.New(
			"daemon: Goal origin session reader is unavailable",
		)
	}
	info, err := s.sessionStatus.Status(ctx, strings.TrimSpace(sessionID))
	if err != nil {
		if errors.Is(err, store.ErrSessionNotFound) {
			return looppkg.RunOrigin{}, store.SessionCreationProfile{},
				session.GoalReasonCode(looppkg.ReasonCodeGoalOriginInvalid), nil
		}
		return looppkg.RunOrigin{}, store.SessionCreationProfile{}, "", fmt.Errorf(
			"daemon: read Goal origin session %q: %w",
			sessionID,
			err,
		)
	}
	if info == nil || info.State != session.StateActive {
		return looppkg.RunOrigin{}, store.SessionCreationProfile{},
			session.GoalReasonCode(looppkg.ReasonCodeGoalOriginInvalid), nil
	}
	if strings.TrimSpace(info.WorkspaceID) != strings.TrimSpace(workspaceID) {
		return looppkg.RunOrigin{}, store.SessionCreationProfile{},
			session.GoalReasonCode(looppkg.ReasonCodeGoalOriginWorkspaceMismatch), nil
	}
	if s.creationStore == nil {
		return looppkg.RunOrigin{}, store.SessionCreationProfile{},
			session.GoalReasonCode(looppkg.ReasonCodeGoalOriginProfileUnavailable), nil
	}
	identity, err := s.creationStore.GetSessionCreationIdentity(ctx, info.ID)
	if err != nil {
		if errors.Is(err, store.ErrSessionNotFound) {
			return looppkg.RunOrigin{}, store.SessionCreationProfile{},
				session.GoalReasonCode(looppkg.ReasonCodeGoalOriginProfileUnavailable), nil
		}
		return looppkg.RunOrigin{}, store.SessionCreationProfile{}, "", err
	}
	if err := identity.Validate(); err != nil {
		return looppkg.RunOrigin{}, store.SessionCreationProfile{},
			session.GoalReasonCode(looppkg.ReasonCodeGoalOriginProfileUnavailable), nil
	}
	profile, err := s.creationStore.GetSessionCreationProfile(ctx, identity.CreationProfileRef)
	if err != nil {
		if errors.Is(err, store.ErrSessionNotFound) {
			return looppkg.RunOrigin{}, store.SessionCreationProfile{},
				session.GoalReasonCode(looppkg.ReasonCodeGoalOriginProfileUnavailable), nil
		}
		return looppkg.RunOrigin{}, store.SessionCreationProfile{}, "", err
	}
	if err := validateSessionGoalProfile(profile, identity, workspaceID); err != nil {
		return looppkg.RunOrigin{}, store.SessionCreationProfile{},
			session.GoalReasonCode(looppkg.ReasonCodeGoalOriginProfileUnavailable), nil
	}
	return looppkg.RunOrigin{
		Kind: looppkg.RunOriginSession, SessionID: info.ID,
		CreationProfileRef: identity.CreationProfileRef,
		PolicySpecDigest:   identity.PolicySpecDigest,
		CreationDigest:     identity.CreationDigest,
	}, profile, "", nil
}

func validateSessionGoalProfile(
	profile store.SessionCreationProfile,
	identity store.SessionCreationIdentity,
	workspaceID string,
) error {
	profile = store.NormalizeSessionCreationProfile(profile)
	if err := profile.Validate(); err != nil {
		return err
	}
	profileRef, err := profile.Ref()
	if err != nil {
		return err
	}
	policyDigest, err := profile.PolicySpecDigest()
	if err != nil {
		return err
	}
	if profileRef != strings.TrimSpace(identity.CreationProfileRef) ||
		policyDigest != strings.TrimSpace(identity.PolicySpecDigest) ||
		profile.WorkspaceID != strings.TrimSpace(workspaceID) {
		return fmt.Errorf("%w: Goal origin creation profile identity is inconsistent", looppkg.ErrValidation)
	}
	return nil
}

func (s *daemonLoopAPIService) requireSessionGoalSnapshot(
	ctx context.Context,
	workspaceID string,
	sessionID string,
	wantRunID looppkg.RunID,
) (*session.GoalSnapshot, error) {
	snapshot, err := s.GetSessionGoal(ctx, workspaceID, sessionID)
	if err != nil {
		return nil, err
	}
	if snapshot == nil || strings.TrimSpace(snapshot.RunID) != strings.TrimSpace(string(wantRunID)) {
		return nil, fmt.Errorf("%w: post-operation Goal snapshot is unavailable", looppkg.ErrTransitionConflict)
	}
	return snapshot, nil
}
