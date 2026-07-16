package core

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/compozy/agh/internal/api/contract"
	"github.com/compozy/agh/internal/store"
	"github.com/gin-gonic/gin"
)

// GetNetworkUsage returns workspace-scoped wake usage from the ledger.
func (h *BaseHandlers) GetNetworkUsage(c *gin.Context) {
	workspaceID, ok := h.requireRouteWorkspaceID(c)
	if !ok {
		return
	}
	if h.NetworkUsage == nil {
		h.respondError(c, http.StatusServiceUnavailable, errors.New("api: network usage store is unavailable"))
		return
	}
	report, err := h.NetworkUsage.GetNetworkUsage(c.Request.Context(), store.NetworkUsageQuery{
		WorkspaceID: workspaceID,
		Channel:     strings.TrimSpace(c.Query("channel")),
		RunID:       strings.TrimSpace(c.Query("run_id")),
		OwnerKey:    strings.TrimSpace(c.Query("owner_key")),
	})
	if err != nil {
		h.respondError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, NetworkUsageResponseFromReport(workspaceID, report))
}

// NetworkUsageResponseFromReport converts store usage into the public payload.
func NetworkUsageResponseFromReport(workspaceID string, report store.NetworkUsageReport) contract.NetworkUsageResponse {
	details := make([]contract.NetworkUsageDetailPayload, 0, len(report.Details))
	for _, detail := range report.Details {
		details = append(details, contract.NetworkUsageDetailPayload{
			WakeID:          detail.WakeID,
			TaskRunID:       detail.TaskRunID,
			OwnerKey:        detail.OwnerKey,
			WorkspaceID:     detail.WorkspaceID,
			Channel:         detail.Channel,
			RootID:          detail.RootID,
			Depth:           detail.Depth,
			State:           detail.State,
			UsageState:      detail.UsageState,
			ChargedWallTime: detail.ChargedWallTime.String(),
			InputTokens:     detail.InputTokens,
			OutputTokens:    detail.OutputTokens,
			ReservedAt:      detail.ReservedAt,
			SettledAt:       cloneOptionalTime(detail.SettledAt),
			Reason:          detail.Reason,
		})
	}
	response := contract.NetworkUsageResponse{
		WorkspaceID: workspaceID,
		Details:     details,
		Total: contract.NetworkUsageSummaryPayload{
			WakeCount:            report.Total.WakeCount,
			ReservedWakeCount:    report.Total.ReservedWakeCount,
			ActualWakeCount:      report.Total.ActualWakeCount,
			UnavailableWakeCount: report.Total.UnavailableWakeCount,
			ChargedWallTime:      report.Total.ChargedWallTime.String(),
			InputTokens:          report.Total.InputTokens,
			OutputTokens:         report.Total.OutputTokens,
		},
	}
	if report.Budget != nil {
		response.Budget = &contract.NetworkBudgetUsagePayload{
			OwnerKey:         report.Budget.OwnerKey,
			WakesUsed:        report.Budget.WakesUsed,
			WallTimeUsed:     report.Budget.WallTimeUsed.String(),
			InputTokensUsed:  report.Budget.InputTokensUsed,
			OutputTokensUsed: report.Budget.OutputTokensUsed,
			ExhaustedReason:  report.Budget.ExhaustedReason,
			UpdatedAt:        report.Budget.UpdatedAt,
		}
	}
	return response
}

func cloneOptionalTime(value *time.Time) *time.Time {
	if value == nil || value.IsZero() {
		return nil
	}
	cloned := value.UTC()
	return &cloned
}
