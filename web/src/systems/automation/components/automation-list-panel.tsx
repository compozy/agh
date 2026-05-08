import { AlertCircle, Clock3, Loader2, Zap } from "lucide-react";

import { Empty, Pill, SearchInput } from "@agh/ui";
import { cn } from "@/lib/utils";

import {
  automationScopeLabel,
  automationSourceLabel,
  automationStatusTone,
  describeSchedule,
  formatAutomationListSummary,
  formatPromptPreview,
  formatRelativeTime,
} from "../lib/automation-formatters";
import type {
  AutomationJob,
  AutomationKind,
  AutomationScopeFilter,
  AutomationTrigger,
} from "../types";

interface AutomationListPanelProps {
  activeWorkspaceName?: string;
  errorMessage?: string | null;
  isLoading?: boolean;
  jobs: AutomationJob[];
  kind: AutomationKind;
  onSearchChange: (query: string) => void;
  onSelect: (id: string) => void;
  scopeFilter: AutomationScopeFilter;
  searchQuery: string;
  selectedId: string | null;
  totalCount: number;
  triggers: AutomationTrigger[];
}

function sourceBadgeTone(source: AutomationJob["source"]): "accent" | "neutral" {
  return source === "dynamic" ? "accent" : "neutral";
}

function scopeBadgeTone(scope: AutomationJob["scope"]): "info" | "neutral" {
  return scope === "workspace" ? "info" : "neutral";
}

interface JobListItemProps {
  isSelected: boolean;
  job: AutomationJob;
  onSelect: () => void;
}

function JobListItem({ isSelected, job, onSelect }: JobListItemProps) {
  const enabledTone = automationStatusTone(job.enabled ? "enabled" : "disabled");

  return (
    <button
      aria-pressed={isSelected}
      className={cn(
        "relative flex w-full flex-col gap-2 border-b border-(--color-divider) px-4 py-3 text-left transition-colors",
        "hover:bg-(--color-hover) focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent",
        isSelected && "bg-(--color-surface)"
      )}
      data-state={isSelected ? "selected" : undefined}
      data-testid={`automation-item-${job.id}`}
      onClick={onSelect}
      type="button"
    >
      {isSelected ? (
        <span
          aria-hidden="true"
          className="absolute left-0 top-2 bottom-2 w-[3px] rounded-r bg-accent"
          data-testid="automation-active-indicator"
        />
      ) : null}

      <div className="flex items-start justify-between gap-3">
        <div className="flex min-w-0 items-center gap-2">
          <Pill.Dot tone={enabledTone} />
          <span className="truncate text-small-body font-medium text-(--color-text-primary)">
            {job.name}
          </span>
        </div>
        <span className="shrink-0 font-mono text-badge uppercase tracking-widest text-accent">
          {formatRelativeTime(job.next_run)}
        </span>
      </div>

      <p className="truncate text-xs text-(--color-text-secondary)">
        {describeSchedule(job.schedule)}
      </p>

      <div className="flex flex-wrap items-center gap-1.5">
        <Pill mono tone={sourceBadgeTone(job.source)}>
          {automationSourceLabel(job.source)}
        </Pill>
        <Pill mono tone={scopeBadgeTone(job.scope)}>
          {automationScopeLabel(job.scope)}
        </Pill>
      </div>
    </button>
  );
}

interface TriggerListItemProps {
  isSelected: boolean;
  onSelect: () => void;
  trigger: AutomationTrigger;
}

function TriggerListItem({ isSelected, onSelect, trigger }: TriggerListItemProps) {
  const enabledTone = automationStatusTone(trigger.enabled ? "enabled" : "disabled");

  return (
    <button
      aria-pressed={isSelected}
      className={cn(
        "relative flex w-full flex-col gap-2 border-b border-(--color-divider) px-4 py-3 text-left transition-colors",
        "hover:bg-(--color-hover) focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent",
        isSelected && "bg-(--color-surface)"
      )}
      data-state={isSelected ? "selected" : undefined}
      data-testid={`automation-item-${trigger.id}`}
      onClick={onSelect}
      type="button"
    >
      {isSelected ? (
        <span
          aria-hidden="true"
          className="absolute left-0 top-2 bottom-2 w-[3px] rounded-r bg-accent"
          data-testid="automation-active-indicator"
        />
      ) : null}

      <div className="flex items-start justify-between gap-3">
        <div className="flex min-w-0 items-center gap-2">
          <Pill.Dot tone={enabledTone} />
          <span className="truncate text-small-body font-medium text-(--color-text-primary)">
            {trigger.name}
          </span>
        </div>
        <Pill mono className="shrink-0 normal-case" tone="info">
          {trigger.event}
        </Pill>
      </div>

      <p className="line-clamp-2 text-xs text-(--color-text-secondary)">
        {formatPromptPreview(trigger.prompt)}
      </p>

      <div className="flex flex-wrap items-center gap-1.5">
        <Pill mono tone={sourceBadgeTone(trigger.source)}>
          {automationSourceLabel(trigger.source)}
        </Pill>
        <Pill mono tone={scopeBadgeTone(trigger.scope)}>
          {automationScopeLabel(trigger.scope)}
        </Pill>
      </div>
    </button>
  );
}

export function AutomationListPanel({
  activeWorkspaceName,
  errorMessage = null,
  isLoading = false,
  jobs,
  kind,
  onSearchChange,
  onSelect,
  scopeFilter,
  searchQuery,
  selectedId,
  totalCount,
  triggers,
}: AutomationListPanelProps) {
  const items = kind === "jobs" ? jobs : triggers;
  const isEmpty = items.length === 0;
  const EmptyIcon = kind === "jobs" ? Clock3 : Zap;
  const emptyTitle = kind === "jobs" ? "No jobs found" : "No triggers found";
  const summary = formatAutomationListSummary({
    activeWorkspaceName,
    kind,
    scopeFilter,
    searchQuery,
    totalCount,
    visibleCount: items.length,
  });

  return (
    <aside className="flex min-h-0 flex-1 flex-col" data-testid="automation-list-panel">
      <div className="space-y-2 border-b border-(--color-divider) p-3">
        <SearchInput
          data-testid="automation-search-input"
          onChange={onSearchChange}
          placeholder={kind === "jobs" ? "Search jobs…" : "Search triggers…"}
          value={searchQuery}
        />
        <p className="text-xs text-(--color-text-secondary)" data-testid="automation-list-summary">
          {summary}
        </p>
      </div>

      <div className="min-h-0 flex-1 overflow-y-auto">
        {isLoading && isEmpty ? (
          <div
            className="flex min-h-full items-center justify-center px-6 py-10"
            data-testid="automation-list-loading"
          >
            <Loader2
              aria-hidden="true"
              className="size-5 animate-spin text-(--color-text-tertiary)"
            />
          </div>
        ) : errorMessage && isEmpty ? (
          <div
            className="flex min-h-full items-center justify-center p-4"
            data-testid="automation-list-error"
          >
            <Empty
              className="max-w-sm"
              description={errorMessage}
              icon={AlertCircle}
              title={kind === "jobs" ? "Unable to load jobs" : "Unable to load triggers"}
            />
          </div>
        ) : isEmpty ? (
          <div
            className="flex min-h-full items-center justify-center p-4"
            data-testid="automation-list-empty"
          >
            <Empty
              className="max-w-sm"
              description={
                searchQuery.trim() !== ""
                  ? "Try a different search term or adjust the scope filter."
                  : kind === "jobs"
                    ? "Create your first job to dispatch prompts on a schedule."
                    : "Create your first trigger to react to daemon events and webhooks."
              }
              icon={EmptyIcon}
              title={emptyTitle}
            />
          </div>
        ) : kind === "jobs" ? (
          jobs.map(job => (
            <JobListItem
              isSelected={job.id === selectedId}
              job={job}
              key={job.id}
              onSelect={() => onSelect(job.id)}
            />
          ))
        ) : (
          triggers.map(trigger => (
            <TriggerListItem
              isSelected={trigger.id === selectedId}
              key={trigger.id}
              onSelect={() => onSelect(trigger.id)}
              trigger={trigger}
            />
          ))
        )}
      </div>
    </aside>
  );
}
