import { Search } from "lucide-react";

import { Pill } from "@agh/ui";
import { cn } from "@/lib/utils";

import { pillVariantFromTone } from "@/lib/pill-variant";
import {
  automationScopeLabel,
  automationScopeTone,
  automationSemanticTone,
  automationSourceLabel,
  automationSourceTone,
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

function AutomationTag({
  children,
  tone,
}: {
  children: string;
  tone: "amber" | "danger" | "green" | "neutral" | "violet";
}) {
  return (
    <Pill className="border-none" variant={pillVariantFromTone(tone)}>
      {children}
    </Pill>
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
        "relative flex w-full flex-col gap-2 border-b border-[color:rgba(58,58,60,0.5)] px-4 py-3 text-left transition-colors",
        "hover:bg-[color:var(--color-surface)]",
        isSelected && "bg-[color:var(--color-surface)]"
      )}
      data-testid={`automation-item-${job.id}`}
      onClick={onSelect}
      type="button"
    >
      {isSelected ? (
        <span
          className="absolute left-0 top-2 bottom-2 w-[3px] rounded-r bg-[color:var(--color-accent)]"
          data-testid="automation-active-indicator"
        />
      ) : null}

      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <p className="truncate text-[0.95rem] font-medium text-[color:var(--color-text-primary)]">
            {job.name}
          </p>
          <p className="mt-0.5 truncate text-sm text-[color:var(--color-text-secondary)]">
            {describeSchedule(job.schedule)}
          </p>
        </div>
        <span className="shrink-0 font-mono text-[0.66rem] uppercase tracking-[0.1em] text-[color:var(--color-accent)]">
          {formatRelativeTime(job.next_run)}
        </span>
      </div>

      <div className="flex flex-wrap items-center gap-1.5">
        <AutomationTag tone={automationSemanticTone(job.enabled ? "enabled" : "disabled")}>
          {job.enabled ? "ENABLED" : "DISABLED"}
        </AutomationTag>
        <AutomationTag tone={automationScopeTone(job.scope)}>
          {automationScopeLabel(job.scope)}
        </AutomationTag>
        <AutomationTag tone={automationSourceTone(job.source)}>
          {automationSourceLabel(job.source)}
        </AutomationTag>
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
        "relative flex w-full flex-col gap-2 border-b border-[color:rgba(58,58,60,0.5)] px-4 py-3 text-left transition-colors",
        "hover:bg-[color:var(--color-surface)]",
        isSelected && "bg-[color:var(--color-surface)]"
      )}
      data-testid={`automation-item-${trigger.id}`}
      onClick={onSelect}
      type="button"
    >
      {isSelected ? (
        <span
          className="absolute left-0 top-2 bottom-2 w-[3px] rounded-r bg-[color:var(--color-accent)]"
          data-testid="automation-active-indicator"
        />
      ) : null}

      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <p className="truncate text-[0.95rem] font-medium text-[color:var(--color-text-primary)]">
            {trigger.name}
          </p>
          <p className="mt-0.5 truncate text-sm text-[color:var(--color-text-secondary)]">
            {formatPromptPreview(trigger.prompt)}
          </p>
        </div>
        <span className="shrink-0 font-mono text-[0.66rem] tracking-[0.06em] text-[color:var(--color-info)]">
          {trigger.event}
        </span>
      </div>

      <div className="flex flex-wrap items-center gap-1.5">
        <AutomationTag tone={automationSemanticTone(trigger.enabled ? "enabled" : "disabled")}>
          {trigger.enabled ? "ENABLED" : "DISABLED"}
        </AutomationTag>
        <AutomationTag tone={automationScopeTone(trigger.scope)}>
          {automationScopeLabel(trigger.scope)}
        </AutomationTag>
        <AutomationTag tone={automationSourceTone(trigger.source)}>
          {automationSourceLabel(trigger.source)}
        </AutomationTag>
      </div>
    </button>
  );
}

export function AutomationListPanel({
  activeWorkspaceName,
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
  const emptyLabel = kind === "jobs" ? "No jobs found" : "No triggers found";
  const summary = formatAutomationListSummary({
    activeWorkspaceName,
    kind,
    scopeFilter,
    searchQuery,
    totalCount,
    visibleCount: items.length,
  });

  return (
    <aside
      className="flex w-[320px] shrink-0 flex-col border-r border-[color:var(--color-divider)] bg-[color:var(--color-surface-panel)]"
      data-testid="automation-list-panel"
    >
      <div className="space-y-3 border-b border-[color:var(--color-divider)] px-3 py-4">
        <label className="flex h-9 items-center gap-2 rounded-lg border border-[color:var(--color-divider)] bg-[color:var(--color-surface-elevated)] px-3">
          <Search className="size-3.5 shrink-0 text-[color:var(--color-text-tertiary)]" />
          <span className="sr-only">Search automation items</span>
          <input
            className="min-w-0 flex-1 bg-transparent text-sm text-[color:var(--color-text-primary)] outline-none placeholder:text-[color:var(--color-text-tertiary)]"
            data-testid="automation-search-input"
            onChange={event => onSearchChange(event.target.value)}
            placeholder={kind === "jobs" ? "Search jobs..." : "Search triggers..."}
            type="text"
            value={searchQuery}
          />
        </label>
        <p
          className="text-sm text-[color:var(--color-text-secondary)]"
          data-testid="automation-list-summary"
        >
          {summary}
        </p>
      </div>

      <div className="flex-1 overflow-y-auto">
        {items.length === 0 ? (
          <div
            className="flex min-h-full items-center justify-center px-6 py-10 text-center text-sm text-[color:var(--color-text-secondary)]"
            data-testid="automation-list-empty"
          >
            {emptyLabel}
          </div>
        ) : (
          <>
            {kind === "jobs"
              ? jobs.map(job => (
                  <JobListItem
                    isSelected={job.id === selectedId}
                    job={job}
                    key={job.id}
                    onSelect={() => onSelect(job.id)}
                  />
                ))
              : triggers.map(trigger => (
                  <TriggerListItem
                    isSelected={trigger.id === selectedId}
                    key={trigger.id}
                    onSelect={() => onSelect(trigger.id)}
                    trigger={trigger}
                  />
                ))}
          </>
        )}
      </div>
    </aside>
  );
}
