package core

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pedronauck/agh/internal/api/contract"
	"github.com/pedronauck/agh/internal/skills"
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

	skill, err := h.resolveSkill(c, name)
	if err != nil {
		h.respondError(c, StatusForSkillError(err), err)
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

	skill, err := h.resolveSkill(c, name)
	if err != nil {
		h.respondError(c, StatusForSkillError(err), err)
		return
	}

	if skill.Enabled {
		c.JSON(http.StatusOK, contract.SkillActionResponse{OK: true})
		return
	}

	if err := h.SkillsRegistry.SetEnabled(name, true); err != nil {
		h.respondError(c, http.StatusInternalServerError, fmt.Errorf("enable skill %q: %w", name, err))
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

	skill, err := h.resolveSkill(c, name)
	if err != nil {
		h.respondError(c, StatusForSkillError(err), err)
		return
	}

	if !skill.Enabled {
		c.JSON(http.StatusOK, contract.SkillActionResponse{OK: true})
		return
	}

	if err := h.SkillsRegistry.SetEnabled(name, false); err != nil {
		h.respondError(c, http.StatusInternalServerError, fmt.Errorf("disable skill %q: %w", name, err))
		return
	}

	h.Logger.Info("skills: disable skill", "name", name)
	c.JSON(http.StatusOK, contract.SkillActionResponse{OK: true})
}

func (h *BaseHandlers) resolveSkill(c *gin.Context, name string) (*skills.Skill, error) {
	workspace := strings.TrimSpace(c.Query("workspace"))
	if workspace == "" {
		skill, ok := h.SkillsRegistry.Get(name)
		if !ok {
			return nil, fmt.Errorf("%w: %q", ErrSkillNotFound, name)
		}
		return skill, nil
	}

	resolved, err := h.Workspaces.Resolve(c.Request.Context(), workspace)
	if err != nil {
		return nil, err
	}

	skillList, err := h.SkillsRegistry.ForWorkspace(c.Request.Context(), resolved)
	if err != nil {
		return nil, err
	}
	for _, skill := range skillList {
		if skill != nil && skill.Meta.Name == name {
			return skill, nil
		}
	}

	return nil, fmt.Errorf("%w: %q", ErrSkillNotFound, name)
}
