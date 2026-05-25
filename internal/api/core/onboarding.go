package core

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/compozy/agh/internal/api/contract"
	"github.com/gin-gonic/gin"
)

// onboardingCompletedAtKey stores the RFC3339 timestamp when first-run onboarding finished.
// Absence of the key means onboarding has not been completed.
const onboardingCompletedAtKey = "onboarding.completed_at"

// GetOnboardingStatus reports whether first-run onboarding has been completed.
func (h *BaseHandlers) GetOnboardingStatus(c *gin.Context) {
	payload, err := h.onboardingStatus(c.Request.Context())
	if err != nil {
		h.respondError(c, http.StatusServiceUnavailable, err)
		return
	}
	c.JSON(http.StatusOK, contract.OnboardingStatusResponse{Onboarding: payload})
}

// CompleteOnboarding marks first-run onboarding as completed and returns the new status.
func (h *BaseHandlers) CompleteOnboarding(c *gin.Context) {
	if h.Onboarding == nil {
		h.respondError(c, http.StatusServiceUnavailable, errors.New("api: onboarding store is not configured"))
		return
	}
	ctx := c.Request.Context()
	completedAt, found, err := h.Onboarding.GetAppMetadata(ctx, onboardingCompletedAtKey)
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, err)
		return
	}
	if !found {
		completedAt = h.Now().UTC().Format(time.RFC3339)
		if err := h.Onboarding.SetAppMetadata(ctx, onboardingCompletedAtKey, completedAt); err != nil {
			h.respondError(c, http.StatusInternalServerError, err)
			return
		}
	}
	c.JSON(http.StatusOK, contract.OnboardingStatusResponse{
		Onboarding: contract.OnboardingStatusPayload{Completed: true, CompletedAt: completedAt},
	})
}

// ResetOnboarding clears the onboarding completion flag so the wizard runs again.
func (h *BaseHandlers) ResetOnboarding(c *gin.Context) {
	if h.Onboarding == nil {
		h.respondError(c, http.StatusServiceUnavailable, errors.New("api: onboarding store is not configured"))
		return
	}
	if err := h.Onboarding.DeleteAppMetadata(c.Request.Context(), onboardingCompletedAtKey); err != nil {
		h.respondError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, contract.OnboardingStatusResponse{
		Onboarding: contract.OnboardingStatusPayload{Completed: false},
	})
}

func (h *BaseHandlers) onboardingStatus(ctx context.Context) (contract.OnboardingStatusPayload, error) {
	if h.Onboarding == nil {
		return contract.OnboardingStatusPayload{}, errors.New("api: onboarding store is not configured")
	}
	value, found, err := h.Onboarding.GetAppMetadata(ctx, onboardingCompletedAtKey)
	if err != nil {
		return contract.OnboardingStatusPayload{}, err
	}
	return contract.OnboardingStatusPayload{Completed: found, CompletedAt: value}, nil
}
