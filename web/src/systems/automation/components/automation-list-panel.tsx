import { Plus, Search } from "lucide-react";
import { useDeferredValue, useMemo } from "react";

import { cn } from "@/lib/utils";

import {
  automationSourceLabel,
  automationStatusTone,
  describeSchedule,
  describeTrigger,
  formatRelativeTime,
} from "../lib/automation-formatters";
import type {
  AutomationJob,
  AutomationKind,
  AutomationScopeFilter,
  AutomationTrigger,
} from "../types";

interface AutomationListPanelProps {
  jobs: AutomationJob[];
  kind: AutomationKind;
  onCreate: () => void;
  onSearchChange: (query: string) => void;
  onSelect: (id: string) => void;
  scopeFilter: AutomationScopeFilter;
  searchQuery: string;
  selectedId: string | null;
  triggers: AutomationTrigger[];
}

const SOURCE_ORDER = {
  config: 0,
  dynamic: 1,
} as const;

const TONE_CLASSES = {
  accent: "bg-[color:var(--color-accent-tint)] text-[color:var(--color-accent)]",
  success: "bg-[color:var(--color-success-tint)] text-[color:var(--color-success)]",
  warning: "bg-[color:var(--color-warning-tint)] text-[color:var(--color-warning)]",
  danger: "bg-[color:var(--color-danger-tint)] text-[color:var(--color-danger)]",
  neutral: "bg-[color:var(--color-neutral-tint)] text-[color:var(--color-text-tertiary)]",
} as const;

function scopeLabel(scope: "global" | "workspace") {
  return scope === "workspace" ? "WORKSPACE" : "GLOBAL";
}

function matchesJob(job: AutomationJob, query: string) {
  const haystack = [
    job.name,
    job.agent_name,
    job.prompt,
    job.scope,
    job.source,
    job.schedule?.mode,
    job.schedule?.expr,
    job.schedule?.interval,
    job.schedule?.time,
  ]
    .filter(Boolean)
    .join(" ")
    .toLowerCase();

  return haystack.includes(query);
}

function matchesTrigger(trigger: AutomationTrigger, query: string) {
  const haystack = [
    trigger.name,
    trigger.agent_name,
    trigger.prompt,
    trigger.scope,
    trigger.source,
    trigger.event,
    trigger.endpoint_slug,
    trigger.webhook_id,
    ...Object.entries(trigger.filter ?? {}).flat(),
  ]
    .filter(Boolean)
    .join(" ")
    .toLowerCase();

  return haystack.includes(query);
}

function AutomationBadge({
  children,
  tone,
}: {
  children: string;
  tone: keyof typeof TONE_CLASSES;
}) {
  return (
    <span
      className={cn(
        "inline-flex h-[18px] items-center rounded px-1.5 font-mono text-[9px] font-semibold uppercase tracking-[0.08em]",
        TONE_CLASSES[tone]
      )}
    >
      {children}
    </span>
  );
}

function JobListItem({
  isSelected,
  job,
  onSelect,
}: {
  isSelected: boolean;
  job: AutomationJob;
  onSelect: () => void;
}) {
  return (
    <button
      className={cn(
        "relative flex w-full flex-col gap-1 border-b border-[color:rgba(58,58,60,0.45)] px-3 py-2.5 text-left transition-colors",
        "hover:bg-[color:var(--color-surface)]",
        isSelected && "bg-[color:var(--color-surface)]"
      )}
      data-testid={`automation-item-${job.id}`}
      onClick={onSelect}
      type="button"
    >
      {isSelected ? (
        <span
          className="absolute left-0 top-1 bottom-1 w-[3px] rounded-r bg-[color:var(--color-accent)]"
          data-testid="automation-active-indicator"
        />
      ) : null}
      <div className="flex items-center gap-2">
        <span className="flex-1 truncate text-sm font-medium text-[color:var(--color-text-primary)]">
          {job.name}
        </span>
        <span className="text-[10px] text-[color:var(--color-text-tertiary)]">
          {formatRelativeTime(job.next_run)}
        </span>
      </div>
      <span className="truncate text-xs text-[color:var(--color-text-secondary)]">
        {describeSchedule(job.schedule)}
      </span>
      <div className="flex flex-wrap items-center gap-1.5">
        <AutomationBadge tone={automationStatusTone(job.enabled ? "enabled" : "disabled")}>
          {job.enabled ? "enabled" : "disabled"}
        </AutomationBadge>
        <AutomationBadge tone="neutral">{scopeLabel(job.scope)}</AutomationBadge>
        <AutomationBadge tone="neutral">{automationSourceLabel(job.source)}</AutomationBadge>
      </div>
    </button>
  );
}

function TriggerListItem({
  isSelected,
  onSelect,
  trigger,
}: {
  isSelected: boolean;
  onSelect: () => void;
  trigger: AutomationTrigger;
}) {
  return (
    <button
      className={cn(
        "relative flex w-full flex-col gap-1 border-b border-[color:rgba(58,58,60,0.45)] px-3 py-2.5 text-left transition-colors",
        "hover:bg-[color:var(--color-surface)]",
        isSelected && "bg-[color:var(--color-surface)]"
      )}
      data-testid={`automation-item-${trigger.id}`}
      onClick={onSelect}
      type="button"
    >
      {isSelected ? (
        <span
          className="absolute left-0 top-1 bottom-1 w-[3px] rounded-r bg-[color:var(--color-accent)]"
          data-testid="automation-active-indicator"
        />
      ) : null}
      <div className="flex items-center gap-2">
        <span className="flex-1 truncate text-sm font-medium text-[color:var(--color-text-primary)]">
          {trigger.name}
        </span>
        <span className="text-[10px] text-[color:var(--color-text-tertiary)]">{trigger.event}</span>
      </div>
      <span className="truncate text-xs text-[color:var(--color-text-secondary)]">
        {describeTrigger(trigger)}
      </span>
      <div className="flex flex-wrap items-center gap-1.5">
        <AutomationBadge tone={automationStatusTone(trigger.enabled ? "enabled" : "disabled")}>
          {trigger.enabled ? "enabled" : "disabled"}
        </AutomationBadge>
        <AutomationBadge tone="neutral">{scopeLabel(trigger.scope)}</AutomationBadge>
        <AutomationBadge tone="neutral">{automationSourceLabel(trigger.source)}</AutomationBadge>
      </div>
    </button>
  );
}

export function AutomationListPanel({
  jobs,
  kind,
  onCreate,
  onSearchChange,
  onSelect,
  scopeFilter,
  searchQuery,
  selectedId,
  triggers,
}: AutomationListPanelProps) {
  const deferredQuery = useDeferredValue(searchQuery);

  const items = useMemo(() => {
    const normalizedQuery = deferredQuery.trim().toLowerCase();
    const sourceItems = kind === "jobs" ? jobs : triggers;

    const filtered =
      normalizedQuery === ""
        ? sourceItems
        : sourceItems.filter(item =>
            kind === "jobs"
              ? matchesJob(item as AutomationJob, normalizedQuery)
              : matchesTrigger(item as AutomationTrigger, normalizedQuery)
          );

    return [...filtered].sort((left, right) => {
      const leftOrder = SOURCE_ORDER[left.source];
      const rightOrder = SOURCE_ORDER[right.source];
      if (leftOrder !== rightOrder) {
        return leftOrder - rightOrder;
      }

      return left.name.localeCompare(right.name);
    });
  }, [deferredQuery, jobs, kind, triggers]);

  return (
    <div
      className="flex w-[320px] shrink-0 flex-col border-r border-[color:var(--color-divider)] bg-[color:var(--color-surface-panel)]"
      data-testid="automation-list-panel"
    >
      <div className="space-y-3 border-b border-[color:var(--color-divider)] p-3">
        <div className="flex items-center gap-2 rounded-lg bg-[color:var(--color-surface)] px-3 py-2">
          <Search className="size-3.5 shrink-0 text-[color:var(--color-text-tertiary)]" />
          <input
            className="w-full bg-transparent text-sm text-[color:var(--color-text-primary)] outline-none placeholder:text-[color:var(--color-text-tertiary)]"
            data-testid="automation-search-input"
            onChange={event => onSearchChange(event.target.value)}
            placeholder={`Search ${kind}...`}
            type="text"
            value={searchQuery}
          />
        </div>
        <button
          className="inline-flex h-9 w-full items-center justify-center gap-2 rounded-lg bg-[color:var(--color-accent)] px-4 text-sm font-medium text-[color:var(--color-accent-ink)] transition-colors hover:bg-[color:var(--color-accent-hover)]"
          data-testid="create-automation-btn"
          onClick={onCreate}
          type="button"
        >
          <Plus className="size-4" />
          {kind === "jobs" ? "New job" : "New trigger"}
        </button>
        <p className="text-xs text-[color:var(--color-text-tertiary)]">
          Showing {scopeFilter === "all" ? "all scopes" : `${scopeFilter} scope`} for {kind}.
        </p>
      </div>

      <div className="flex-1 overflow-y-auto">
        {items.length === 0 ? (
          <div
            className="px-4 py-8 text-center text-sm text-[color:var(--color-text-secondary)]"
            data-testid="automation-list-empty"
          >
            No {kind} match the current filters.
          </div>
        ) : (
          items.map(item =>
            kind === "jobs" ? (
              <JobListItem
                key={item.id}
                isSelected={selectedId === item.id}
                job={item as AutomationJob}
                onSelect={() => onSelect(item.id)}
              />
            ) : (
              <TriggerListItem
                key={item.id}
                isSelected={selectedId === item.id}
                onSelect={() => onSelect(item.id)}
                trigger={item as AutomationTrigger}
              />
            )
          )
        )}
      </div>
    </div>
  );
}
