package daemon

import (
	"context"
	"errors"

	looppkg "github.com/compozy/agh/internal/loop"
	goalpkg "github.com/compozy/agh/internal/loop/goal"
)

func (s *daemonLoopAPIService) findGoalReportTarget(
	ctx context.Context,
	workspaceID looppkg.WorkspaceID,
	sessionID string,
) (goalpkg.ToolReportTarget, bool, error) {
	store, err := s.goalToolStore()
	if err != nil {
		return goalpkg.ToolReportTarget{}, false, err
	}
	return store.FindGoalReportTarget(ctx, workspaceID, sessionID)
}

func (s *daemonLoopAPIService) resolveActiveGoalOriginAlias(
	ctx context.Context,
	workspaceID looppkg.WorkspaceID,
	sessionID string,
) (string, bool, error) {
	store, err := s.goalToolStore()
	if err != nil {
		return "", false, err
	}
	return store.ResolveActiveGoalOriginAlias(ctx, workspaceID, sessionID)
}

func (s *daemonLoopAPIService) recordGoalReport(
	ctx context.Context,
	request goalpkg.RecordToolReportRequest,
) (goalpkg.ReportIntent, error) {
	store, err := s.goalToolStore()
	if err != nil {
		return goalpkg.ReportIntent{}, err
	}
	return store.RecordGoalReport(ctx, request)
}

func (s *daemonLoopAPIService) goalToolStore() (goalpkg.ToolStore, error) {
	if s == nil || s.goalPersistence == nil {
		return nil, errors.New("daemon: Goal tool persistence is unavailable")
	}
	store, ok := s.goalPersistence.(goalpkg.ToolStore)
	if !ok {
		return nil, errors.New("daemon: Goal tool persistence is unavailable")
	}
	return store, nil
}
