package core

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pedronauck/agh/internal/api/contract"
	"github.com/pedronauck/agh/internal/skills"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
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

	skill, _, err := h.resolveSkill(c, name)
	if err != nil {
		h.respondError(c, StatusForSkillError(err), err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"skill": SkillPayloadFromSkill(skill)})
}

// GetSkillContent returns the explicit body for one skill.
func (h *BaseHandlers) GetSkillContent(c *gin.Context) {
	if h.SkillsRegistry == nil {
		h.respondError(c, http.StatusServiceUnavailable, fmt.Errorf("%s: skills registry is not configured", h.transportName()))
		return
	}

	name := strings.TrimSpace(c.Param("name"))
	if name == "" {
		h.respondError(c, http.StatusBadRequest, fmt.Errorf("%w: skill name is required", ErrSkillValidation))
		return
	}

	skill, _, err := h.resolveSkill(c, name)
	if err != nil {
		h.respondError(c, StatusForSkillError(err), err)
		return
	}

	content, err := h.SkillsRegistry.LoadContent(c.Request.Context(), skill)
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, fmt.Errorf("load skill content %q: %w", name, err))
		return
	}

	c.JSON(http.StatusOK, contract.SkillContentResponse{Content: content})
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

	skill, resolved, err := h.resolveSkill(c, name)
	if err != nil {
		h.respondError(c, StatusForSkillError(err), err)
		return
	}

	if skill.Enabled {
		c.JSON(http.StatusOK, contract.SkillActionResponse{OK: true})
		return
	}

	if err := h.SkillsRegistry.SetEnabled(name, resolved, true); err != nil {
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

	skill, resolved, err := h.resolveSkill(c, name)
	if err != nil {
		h.respondError(c, StatusForSkillError(err), err)
		return
	}

	if !skill.Enabled {
		c.JSON(http.StatusOK, contract.SkillActionResponse{OK: true})
		return
	}

	if err := h.SkillsRegistry.SetEnabled(name, resolved, false); err != nil {
		h.respondError(c, http.StatusInternalServerError, fmt.Errorf("disable skill %q: %w", name, err))
		return
	}

	h.Logger.Info("skills: disable skill", "name", name)
	c.JSON(http.StatusOK, contract.SkillActionResponse{OK: true})
}

func (h *BaseHandlers) resolveSkill(c *gin.Context, name string) (*skills.Skill, *workspacepkg.ResolvedWorkspace, error) {
	workspace := strings.TrimSpace(c.Query("workspace"))
	if workspace == "" {
		skill, ok := h.SkillsRegistry.Get(name)
		if !ok {
			return nil, nil, fmt.Errorf("%w: %q", ErrSkillNotFound, name)
		}
		return skill, nil, nil
	}

	resolved, err := h.Workspaces.Resolve(c.Request.Context(), workspace)
	if err != nil {
		return nil, nil, err
	}

	skillList, err := h.SkillsRegistry.ForWorkspace(c.Request.Context(), resolved)
	if err != nil {
		return nil, nil, err
	}
	for _, skill := range skillList {
		if skill != nil && skill.Meta.Name == name {
			resolvedCopy := resolved
			return skill, &resolvedCopy, nil
		}
	}

	return nil, nil, fmt.Errorf("%w: %q", ErrSkillNotFound, name)
}
