package globaldb

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	automation "github.com/compozy/agh/internal/automation/model"
)

const automationJobRichSelectSQL = `SELECT
	id, scope, name, agent_name, workspace_id, prompt, schedule, task,
	enabled, retry, fire_limit, source, target_kind, loop_workspace_id,
	loop_name, loop_inputs, loop_input_mapping, created_at, updated_at
	FROM automation_jobs`

const automationTriggerRichSelectSQL = `SELECT
	id, scope, name, agent_name, workspace_id, prompt, event, filter,
	enabled, retry, fire_limit, source, webhook_id, endpoint_slug,
	webhook_secret_ref, target_kind, loop_workspace_id, loop_name,
	loop_inputs, loop_input_mapping, created_at, updated_at
	FROM automation_triggers`

func upsertAutomationJobCatalog(ctx context.Context, exec sqlExecutor, job automation.Job) error {
	search := automationJobCatalogSearch(job)
	loopName := automationJobCatalogLoopName(job)
	if _, err := exec.ExecContext(ctx, `INSERT INTO automation_job_catalog_entries (
		job_id, scope, workspace_id, source, source_rank, name, loop_name, enabled,
		search_name, search_agent_name, search_prompt, search_scope, search_source,
		search_schedule_mode, search_schedule_expr, search_schedule_interval, search_schedule_time
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(job_id) DO UPDATE SET
		scope = excluded.scope,
		workspace_id = excluded.workspace_id,
		source = excluded.source,
		source_rank = excluded.source_rank,
		name = excluded.name,
		loop_name = excluded.loop_name,
		enabled = excluded.enabled,
		search_name = excluded.search_name,
		search_agent_name = excluded.search_agent_name,
		search_prompt = excluded.search_prompt,
		search_scope = excluded.search_scope,
		search_source = excluded.search_source,
		search_schedule_mode = excluded.search_schedule_mode,
		search_schedule_expr = excluded.search_schedule_expr,
		search_schedule_interval = excluded.search_schedule_interval,
		search_schedule_time = excluded.search_schedule_time`,
		job.ID,
		job.Scope,
		strings.TrimSpace(job.WorkspaceID),
		job.Source,
		automation.ListSourceRank(job.Source),
		job.Name,
		loopName,
		job.Enabled,
		search.name,
		search.agentName,
		search.prompt,
		search.scope,
		search.source,
		search.scheduleMode,
		search.scheduleExpr,
		search.scheduleInterval,
		search.scheduleTime,
	); err != nil {
		return fmt.Errorf("store: upsert automation job catalog %q: %w", job.ID, err)
	}
	return nil
}

func upsertAutomationTriggerCatalog(ctx context.Context, exec sqlExecutor, trigger automation.Trigger) error {
	search := automationTriggerCatalogSearch(trigger)
	loopName := automationTriggerCatalogLoopName(trigger)
	if _, err := exec.ExecContext(ctx, `INSERT INTO automation_trigger_catalog_entries (
		trigger_id, scope, workspace_id, event, source, source_rank, name, loop_name, enabled,
		search_name, search_agent_name, search_prompt, search_scope, search_source,
		search_event, search_endpoint_slug, search_webhook_id
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(trigger_id) DO UPDATE SET
		scope = excluded.scope,
		workspace_id = excluded.workspace_id,
		event = excluded.event,
		source = excluded.source,
		source_rank = excluded.source_rank,
		name = excluded.name,
		loop_name = excluded.loop_name,
		enabled = excluded.enabled,
		search_name = excluded.search_name,
		search_agent_name = excluded.search_agent_name,
		search_prompt = excluded.search_prompt,
		search_scope = excluded.search_scope,
		search_source = excluded.search_source,
		search_event = excluded.search_event,
		search_endpoint_slug = excluded.search_endpoint_slug,
		search_webhook_id = excluded.search_webhook_id`,
		trigger.ID,
		trigger.Scope,
		strings.TrimSpace(trigger.WorkspaceID),
		trigger.Event,
		trigger.Source,
		automation.ListSourceRank(trigger.Source),
		trigger.Name,
		loopName,
		trigger.Enabled,
		search.name,
		search.agentName,
		search.prompt,
		search.scope,
		search.source,
		search.event,
		search.endpointSlug,
		search.webhookID,
	); err != nil {
		return fmt.Errorf("store: upsert automation trigger catalog %q: %w", trigger.ID, err)
	}
	if _, err := exec.ExecContext(
		ctx,
		`DELETE FROM automation_trigger_catalog_filter_terms WHERE trigger_id = ?`,
		trigger.ID,
	); err != nil {
		return fmt.Errorf("store: replace automation trigger catalog terms %q: %w", trigger.ID, err)
	}
	termsJSON, err := json.Marshal(automationTriggerFilterTerms(trigger.Filter))
	if err != nil {
		return fmt.Errorf("store: encode automation trigger catalog terms %q: %w", trigger.ID, err)
	}
	if _, err := exec.ExecContext(ctx, `INSERT INTO automation_trigger_catalog_filter_terms (trigger_id, value)
		SELECT ?, CAST(value AS TEXT) FROM json_each(?)`, trigger.ID, string(termsJSON)); err != nil {
		return fmt.Errorf("store: insert automation trigger catalog terms %q: %w", trigger.ID, err)
	}
	return nil
}

type automationJobCatalogSearchFields struct {
	name             string
	agentName        string
	prompt           string
	scope            string
	source           string
	scheduleMode     string
	scheduleExpr     string
	scheduleInterval string
	scheduleTime     string
}

func automationJobCatalogSearch(job automation.Job) automationJobCatalogSearchFields {
	search := automationJobCatalogSearchFields{
		name:      strings.ToLower(job.Name),
		agentName: strings.ToLower(job.AgentName),
		prompt:    strings.ToLower(job.Prompt),
		scope:     strings.ToLower(string(job.Scope)),
		source:    strings.ToLower(string(job.Source)),
	}
	if job.Schedule != nil {
		search.scheduleMode = strings.ToLower(string(job.Schedule.Mode))
		search.scheduleExpr = strings.ToLower(job.Schedule.Expr)
		search.scheduleInterval = strings.ToLower(job.Schedule.Interval)
		search.scheduleTime = strings.ToLower(job.Schedule.Time)
	}
	return search
}

type automationTriggerCatalogSearchFields struct {
	name         string
	agentName    string
	prompt       string
	scope        string
	source       string
	event        string
	endpointSlug string
	webhookID    string
}

func automationTriggerCatalogSearch(trigger automation.Trigger) automationTriggerCatalogSearchFields {
	return automationTriggerCatalogSearchFields{
		name:         strings.ToLower(trigger.Name),
		agentName:    strings.ToLower(trigger.AgentName),
		prompt:       strings.ToLower(trigger.Prompt),
		scope:        strings.ToLower(string(trigger.Scope)),
		source:       strings.ToLower(string(trigger.Source)),
		event:        strings.ToLower(trigger.Event),
		endpointSlug: strings.ToLower(trigger.EndpointSlug),
		webhookID:    strings.ToLower(trigger.WebhookID),
	}
}

func automationTriggerFilterTerms(filter map[string]string) []string {
	terms := make(map[string]struct{}, len(filter)*2)
	for key, value := range filter {
		terms[strings.ToLower(key)] = struct{}{}
		terms[strings.ToLower(value)] = struct{}{}
	}
	result := make([]string, 0, len(terms))
	for term := range terms {
		result = append(result, term)
	}
	slices.Sort(result)
	return result
}

func automationJobCatalogLoopName(job automation.Job) string {
	if !job.IsLoopTarget() || job.LoopTarget == nil {
		return ""
	}
	return strings.TrimSpace(job.LoopTarget.LoopName)
}

func automationTriggerCatalogLoopName(trigger automation.Trigger) string {
	if !trigger.IsLoopTarget() || trigger.LoopTarget == nil {
		return ""
	}
	return strings.TrimSpace(trigger.LoopTarget.LoopName)
}
