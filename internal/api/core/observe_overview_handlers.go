package core

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/compozy/agh/internal/api/contract"
	"github.com/compozy/agh/internal/observe"
	taskpkg "github.com/compozy/agh/internal/task"
)

const taskActionOverview = "overview"

// ObserveOverview returns the observer-backed home overview read model.
func (h *BaseHandlers) ObserveOverview(c *gin.Context) {
	observer, ok := h.requireTaskObserver(c)
	if !ok {
		return
	}

	actor, err := h.taskActorContext(c, taskActionOverview)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	usageWindow, err := parseOverviewUsageWindow(c.Query("usage_window"))
	if err != nil {
		h.respondError(c, http.StatusUnprocessableEntity, err)
		return
	}

	query := observe.OverviewQuery{
		TaskScope:       taskpkg.CatalogScopeGlobal,
		UsageWindowDays: usageWindow,
		Actor:           actor.Actor,
	}
	if workspace := strings.TrimSpace(c.Query("workspace")); workspace != "" {
		scope := taskpkg.CatalogScopeWorkspace
		if err := h.resolveTaskCatalogWorkspace(
			c.Request.Context(),
			workspace,
			&scope,
			&query.WorkspaceID,
		); err != nil {
			h.respondError(c, StatusForTaskError(err), err)
			return
		}
		query.TaskScope = scope
	}

	view, err := observer.QueryObserveOverview(c.Request.Context(), query)
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, contract.ObserveOverviewResponse{Overview: ObserveOverviewPayloadFromView(&view)})
}

func parseOverviewUsageWindow(raw string) (int, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return 0, nil
	}
	window, err := strconv.Atoi(trimmed)
	if err != nil {
		return 0, NewTaskValidationError(err)
	}
	switch window {
	case 7, 30, 90:
		return window, nil
	default:
		return 0, NewTaskValidationError(errOverviewUsageWindow)
	}
}
