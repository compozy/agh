package core

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/compozy/agh/internal/api/contract"
	"github.com/compozy/agh/internal/network/participation"
	"github.com/compozy/agh/internal/store"
	taskpkg "github.com/compozy/agh/internal/task"
)

func (h *BaseHandlers) taskRunNetworkPayload(
	ctx context.Context,
	run taskpkg.Run,
) (*contract.TaskRunNetworkPayload, error) {
	spec := run.NetworkSpecSnapshot()
	if spec.Mode != participation.ModeLive {
		return nil, nil
	}
	workspaceID := strings.TrimSpace(spec.WorkspaceID)
	channel := strings.TrimSpace(spec.ChannelID)
	if workspaceID == "" || channel == "" {
		return nil, errors.New("api: live task run requires workspace and channel identity")
	}
	if h.NetworkUsage == nil {
		return nil, errors.New("api: network usage store is unavailable")
	}
	report, err := h.NetworkUsage.GetNetworkUsage(ctx, store.NetworkUsageQuery{
		WorkspaceID: workspaceID,
		RunID:       strings.TrimSpace(run.ID),
	})
	if err != nil {
		return nil, fmt.Errorf("api: load task run network usage: %w", err)
	}
	return &contract.TaskRunNetworkPayload{
		Conversation: contract.TaskRunConversationRefPayload{
			WorkspaceID: workspaceID,
			Channel:     channel,
			Surface:     store.NetworkSurfaceThread,
			ThreadID:    agentChannelThreadID,
			StreamURL: "/api/task-runs/" + url.PathEscape(strings.TrimSpace(run.ID)) +
				"/conversation/stream",
		},
		Usage: NetworkUsageResponseFromReport(workspaceID, report),
	}, nil
}
