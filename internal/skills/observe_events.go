package skills

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	eventspkg "github.com/pedronauck/agh/internal/events"
	"github.com/pedronauck/agh/internal/store"
)

const (
	observeEventsAgentLocalValue = "agent-local"
)

type skillShadowContent struct {
	SkillName       string `json:"skill_name"`
	OldSource       string `json:"old_source"`
	NewSource       string `json:"new_source"`
	OldPath         string `json:"old_path,omitempty"`
	NewPath         string `json:"new_path,omitempty"`
	LayerPair       string `json:"layer_pair"`
	ShadowKind      string `json:"shadow_kind"`
	ResolutionScope string `json:"resolution_scope"`
	AgentName       string `json:"agent_name,omitempty"`
	WorkspaceID     string `json:"workspace_id,omitempty"`
}

type skillLoadFailedContent struct {
	AgentName   string `json:"agent_name"`
	Source      string `json:"source"`
	Path        string `json:"path,omitempty"`
	ErrorCode   string `json:"error_code"`
	ErrorDetail string `json:"error_detail,omitempty"`
}

type agentLocalLoadError struct {
	path   string
	code   string
	detail string
	err    error
}

func (e *agentLocalLoadError) Error() string {
	if e == nil {
		return ""
	}
	if e.err != nil {
		return e.err.Error()
	}
	return strings.TrimSpace(e.detail)
}

func (e *agentLocalLoadError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

func newAgentLocalLoadError(code string, path string, detail string, err error) error {
	return &agentLocalLoadError{
		path:   strings.TrimSpace(path),
		code:   strings.TrimSpace(code),
		detail: strings.TrimSpace(detail),
		err:    err,
	}
}

func agentLocalLoadErrorDetails(err error) (path string, code string, detail string) {
	typed := &agentLocalLoadError{}
	if !errors.As(err, &typed) || typed == nil {
		return "", "validation", strings.TrimSpace(err.Error())
	}
	return strings.TrimSpace(typed.path), strings.TrimSpace(typed.code), strings.TrimSpace(typed.detail)
}

func (r *Registry) buildSkillShadowSummaries(
	base map[string]*Skill,
	overlay map[string]*Skill,
	resolutionScope string,
	layerPair string,
	workspaceID string,
	agentName string,
) []store.EventSummary {
	if r == nil || len(base) == 0 || len(overlay) == 0 {
		return nil
	}

	names := make([]string, 0, len(overlay))
	for name, skill := range overlay {
		if skill == nil || base[name] == nil {
			continue
		}
		names = append(names, name)
	}
	slices.Sort(names)

	summaries := make([]store.EventSummary, 0, len(names))
	for _, name := range names {
		existing := base[name]
		next := overlay[name]
		if existing == nil || next == nil {
			continue
		}

		shadowKind := "logical"
		if strings.TrimSpace(existing.FilePath) != "" && strings.TrimSpace(next.FilePath) != "" {
			shadowKind = "logical_path"
		}
		pair := strings.TrimSpace(layerPair)
		if pair == "" {
			pair = skillSourceName(next.Source) + ">" + skillSourceName(existing.Source)
		}

		content, err := json.Marshal(skillShadowContent{
			SkillName:       strings.TrimSpace(next.Meta.Name),
			OldSource:       skillSourceName(existing.Source),
			NewSource:       skillSourceName(next.Source),
			OldPath:         strings.TrimSpace(existing.FilePath),
			NewPath:         strings.TrimSpace(next.FilePath),
			LayerPair:       pair,
			ShadowKind:      shadowKind,
			ResolutionScope: strings.TrimSpace(resolutionScope),
			AgentName:       strings.TrimSpace(agentName),
			WorkspaceID:     strings.TrimSpace(workspaceID),
		})
		if err != nil {
			r.logger.Warn("skills: marshal skill.shadowed content failed", "name", next.Meta.Name, "error", err)
			continue
		}

		summaries = append(summaries, store.EventSummary{
			WorkspaceID: strings.TrimSpace(workspaceID),
			AgentName:   strings.TrimSpace(agentName),
			Type:        eventspkg.SkillShadowed,
			Content:     content,
			Summary: fmt.Sprintf(
				"skill %s shadowed %s with %s",
				strings.TrimSpace(next.Meta.Name),
				skillSourceName(existing.Source),
				skillSourceName(next.Source),
			),
		})
	}
	return summaries
}

func (r *Registry) emitEventSummaries(ctx context.Context, summaries []store.EventSummary) {
	if r == nil || r.events == nil || len(summaries) == 0 {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	for _, summary := range summaries {
		if err := r.events.WriteEventSummary(ctx, summary); err != nil {
			r.logger.Warn("skills: write observe event failed", "type", summary.Type, "error", err)
		}
	}
}

func (r *Registry) emitSkillsLoadFailed(ctx context.Context, workspaceID string, agentName string, err error) {
	if r == nil || r.events == nil || err == nil {
		return
	}

	path, code, detail := agentLocalLoadErrorDetails(err)
	content, marshalErr := json.Marshal(skillLoadFailedContent{
		AgentName:   strings.TrimSpace(agentName),
		Source:      observeEventsAgentLocalValue,
		Path:        path,
		ErrorCode:   code,
		ErrorDetail: detail,
	})
	if marshalErr != nil {
		r.logger.Warn("skills: marshal skills.load_failed content failed", "agent_name", agentName, "error", marshalErr)
		return
	}

	if ctx == nil {
		ctx = context.Background()
	}
	if writeErr := r.events.WriteEventSummary(ctx, store.EventSummary{
		WorkspaceID: strings.TrimSpace(workspaceID),
		AgentName:   strings.TrimSpace(agentName),
		Type:        eventspkg.SkillLoadFailed,
		Content:     content,
		Summary:     fmt.Sprintf("agent-local skills load failed for %s", strings.TrimSpace(agentName)),
	}); writeErr != nil {
		r.logger.Warn("skills: write skills.load_failed failed", "agent_name", agentName, "error", writeErr)
	}
}
