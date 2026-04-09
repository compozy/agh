package core

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pedronauck/agh/internal/api/contract"
)

// ListSkills returns skills for a workspace.
func (h *BaseHandlers) ListSkills(c *gin.Context) {
	if h.SkillsRegistry == nil {
		h.respondError(c, http.StatusServiceUnavailable, fmt.Errorf("%s: skills registry is not configured", h.transportName()))
		return
	}

	workspace := strings.TrimSpace(c.Query("workspace"))
	if workspace == "" {
		h.respondError(c, http.StatusBadRequest, fmt.Errorf("%w: workspace query parameter is required", ErrSkillValidation))
		return
	}

	resolved, err := h.Workspaces.Resolve(c.Request.Context(), workspace)
	if err != nil {
		h.respondError(c, StatusForWorkspaceError(err), err)
		return
	}

	skillList, err := h.SkillsRegistry.ForWorkspace(c.Request.Context(), resolved)
	if err != nil {
		h.respondError(c, StatusForSkillError(err), err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"skills": SkillPayloadsFromSkills(skillList)})
}

// GetSkill returns one skill by name.
func (h *BaseHandlers) GetSkill(c *gin.Context) {
	if h.SkillsRegistry == nil {
		h.respondError(c, http.StatusServiceUnavailable, fmt.Errorf("%s: skills registry is not configured", h.transportName()))
		return
	}

	name := strings.TrimSpace(c.Param("name"))
	if name == "" {
		h.respondError(c, http.StatusBadRequest, fmt.Errorf("%w: skill name is required", ErrSkillValidation))
		return
	}

	skill, ok := h.SkillsRegistry.Get(name)
	if !ok {
		h.respondError(c, http.StatusNotFound, fmt.Errorf("%w: %q", ErrSkillNotFound, name))
		return
	}

	c.JSON(http.StatusOK, gin.H{"skill": SkillPayloadFromSkill(skill)})
}

// EnableSkill enables a skill by name.
func (h *BaseHandlers) EnableSkill(c *gin.Context) {
	if h.SkillsRegistry == nil {
		h.respondError(c, http.StatusServiceUnavailable, fmt.Errorf("%s: skills registry is not configured", h.transportName()))
		return
	}

	name := strings.TrimSpace(c.Param("name"))
	if name == "" {
		h.respondError(c, http.StatusBadRequest, fmt.Errorf("%w: skill name is required", ErrSkillValidation))
		return
	}

	skill, ok := h.SkillsRegistry.Get(name)
	if !ok {
		h.respondError(c, http.StatusNotFound, fmt.Errorf("%w: %q", ErrSkillNotFound, name))
		return
	}

	if skill.Enabled {
		c.JSON(http.StatusOK, contract.SkillActionResponse{OK: true})
		return
	}

	h.Logger.Info("skills: enable skill", "name", name)
	c.JSON(http.StatusOK, contract.SkillActionResponse{OK: true})
}

// DisableSkill disables a skill by name.
func (h *BaseHandlers) DisableSkill(c *gin.Context) {
	if h.SkillsRegistry == nil {
		h.respondError(c, http.StatusServiceUnavailable, fmt.Errorf("%s: skills registry is not configured", h.transportName()))
		return
	}

	name := strings.TrimSpace(c.Param("name"))
	if name == "" {
		h.respondError(c, http.StatusBadRequest, fmt.Errorf("%w: skill name is required", ErrSkillValidation))
		return
	}

	skill, ok := h.SkillsRegistry.Get(name)
	if !ok {
		h.respondError(c, http.StatusNotFound, fmt.Errorf("%w: %q", ErrSkillNotFound, name))
		return
	}

	if !skill.Enabled {
		c.JSON(http.StatusOK, contract.SkillActionResponse{OK: true})
		return
	}

	h.Logger.Info("skills: disable skill", "name", name)
	c.JSON(http.StatusOK, contract.SkillActionResponse{OK: true})
}
