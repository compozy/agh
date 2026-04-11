import { Loader2, Play, Trash2 } from "lucide-react";

import { cn } from "@/lib/utils";

import { AutomationJobForm } from "./automation-job-form";
import { AutomationRunHistory } from "./automation-run-history";
import { AutomationTriggerForm } from "./automation-trigger-form";
import {
  automationSourceLabel,
  automationStatusTone,
  describeFireLimit,
  describeRetry,
  describeSchedule,
  describeTrigger,
  formatDateTime,
  formatRelativeTime,
} from "../lib/automation-formatters";
import type {
  AutomationJob,
  AutomationRun,
  AutomationTrigger,
  CreateAutomationJobRequest,
  CreateAutomationTriggerRequest,
} from "../types";

type AutomationEditorState =
  | {
      draft: CreateAutomationJobRequest;
      isPending: boolean;
      kind: "jobs";
      mode: "create" | "edit";
      onCancel: () => void;
      onChange: (draft: CreateAutomationJobRequest) => void;
      onSubmit: () => void;
    }
  | {
      draft: CreateAutomationTriggerRequest;
      isPending: boolean;
      kind: "triggers";
      mode: "create" | "edit";
      onCancel: () => void;
      onChange: (draft: CreateAutomationTriggerRequest) => void;
      onSubmit: () => void;
    };

interface AutomationDetailPanelProps {
  activeWorkspaceId?: string | null;
  editor: AutomationEditorState | null;
  error: Error | null;
  isDeleting: boolean;
  isLoading: boolean;
  isTogglePending: boolean;
  isTriggerPending: boolean;
  item: AutomationJob | AutomationTrigger | undefined;
  kind: "jobs" | "triggers";
  onDelete: () => void;
  onEdit: () => void;
  onToggleEnabled: (enabled: boolean) => void;
  onTriggerNow: () => void;
  runs: AutomationRun[];
  runsError: Error | null;
  runsLoading: boolean;
}

const TONE_CLASSES = {
  accent: "bg-[color:var(--color-accent-tint)] text-[color:var(--color-accent)]",
  success: "bg-[color:var(--color-success-tint)] text-[color:var(--color-success)]",
  warning: "bg-[color:var(--color-warning-tint)] text-[color:var(--color-warning)]",
  danger: "bg-[color:var(--color-danger-tint)] text-[color:var(--color-danger)]",
  neutral: "bg-[color:var(--color-neutral-tint)] text-[color:var(--color-text-tertiary)]",
} as const;

function MetadataRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center justify-between gap-3 rounded-lg bg-[color:var(--color-surface)] px-3 py-2.5">
      <span className="text-xs text-[color:var(--color-text-tertiary)]">{label}</span>
      <span className="text-sm font-medium text-[color:var(--color-text-primary)]">{value}</span>
    </div>
  );
}

export function AutomationDetailPanel({
  activeWorkspaceId,
  editor,
  error,
  isDeleting,
  isLoading,
  isTogglePending,
  isTriggerPending,
  item,
  kind,
  onDelete,
  onEdit,
  onToggleEnabled,
  onTriggerNow,
  runs,
  runsError,
  runsLoading,
}: AutomationDetailPanelProps) {
  if (editor) {
    return editor.kind === "jobs" ? (
      <AutomationJobForm
        activeWorkspaceId={activeWorkspaceId}
        draft={editor.draft}
        isPending={editor.isPending}
        mode={editor.mode}
        onCancel={editor.onCancel}
        onChange={editor.onChange}
        onSubmit={editor.onSubmit}
      />
    ) : (
      <AutomationTriggerForm
        activeWorkspaceId={activeWorkspaceId}
        draft={editor.draft}
        isPending={editor.isPending}
        mode={editor.mode}
        onCancel={editor.onCancel}
        onChange={editor.onChange}
        onSubmit={editor.onSubmit}
      />
    );
  }

  if (isLoading) {
    return (
      <div
        className="flex flex-1 items-center justify-center"
        data-testid="automation-detail-loading"
      >
        <Loader2 className="size-5 animate-spin text-[color:var(--color-text-tertiary)]" />
      </div>
    );
  }

  if (error) {
    return (
      <div
        className="flex flex-1 items-center justify-center px-6 text-sm text-[color:var(--color-danger)]"
        data-testid="automation-detail-error"
      >
        Failed to load automation details
      </div>
    );
  }

  if (!item) {
    return (
      <div
        className="flex flex-1 items-center justify-center px-6 text-sm text-[color:var(--color-text-secondary)]"
        data-testid="automation-detail-empty"
      >
        Select an automation item to inspect its configuration and run history.
      </div>
    );
  }

  const isJob = kind === "jobs";
  const isDynamic = item.source === "dynamic";
  const statusTone = automationStatusTone(item.enabled ? "enabled" : "disabled");

  return (
    <div className="flex flex-1 flex-col overflow-y-auto p-6" data-testid="automation-detail-panel">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div className="space-y-2">
          <div className="flex flex-wrap items-center gap-2">
            <h2 className="text-base font-semibold text-[color:var(--color-text-primary)]">
              {item.name}
            </h2>
            <span
              className={cn(
                "inline-flex h-[22px] items-center rounded-md px-2 font-mono text-[10px] font-semibold uppercase tracking-[0.08em]",
                TONE_CLASSES[statusTone]
              )}
            >
              {item.enabled ? "enabled" : "disabled"}
            </span>
            <span className="inline-flex h-[22px] items-center rounded-md bg-[color:var(--color-neutral-tint)] px-2 font-mono text-[10px] font-semibold uppercase tracking-[0.08em] text-[color:var(--color-text-tertiary)]">
              {automationSourceLabel(item.source)}
            </span>
          </div>
          <p className="text-sm text-[color:var(--color-text-secondary)]">
            {isJob
              ? describeSchedule((item as AutomationJob).schedule)
              : describeTrigger(item as AutomationTrigger)}
          </p>
          <div className="flex flex-wrap items-center gap-3 text-xs text-[color:var(--color-text-tertiary)]">
            <span>Agent {item.agent_name}</span>
            <span>Scope {item.scope}</span>
            <span>Updated {formatDateTime(item.updated_at)}</span>
          </div>
        </div>

        <div className="flex flex-wrap items-center gap-2">
          <button
            className="inline-flex h-9 items-center rounded-lg border border-[color:var(--color-divider)] px-4 text-sm font-medium text-[color:var(--color-text-primary)] transition-colors hover:bg-[color:var(--color-hover)] disabled:opacity-50"
            data-testid="toggle-automation-btn"
            disabled={isTogglePending}
            onClick={() => onToggleEnabled(!item.enabled)}
            type="button"
          >
            {isTogglePending ? "Saving..." : item.enabled ? "Disable" : "Enable"}
          </button>
          {isDynamic ? (
            <button
              className="inline-flex h-9 items-center rounded-lg border border-[color:var(--color-divider)] px-4 text-sm font-medium text-[color:var(--color-text-primary)] transition-colors hover:bg-[color:var(--color-hover)]"
              data-testid="edit-automation-btn"
              onClick={onEdit}
              type="button"
            >
              Edit
            </button>
          ) : null}
          {isJob ? (
            <button
              className="inline-flex h-9 items-center gap-2 rounded-lg bg-[color:var(--color-accent)] px-4 text-sm font-medium text-[color:var(--color-accent-ink)] transition-colors hover:bg-[color:var(--color-accent-hover)] disabled:opacity-50"
              data-testid="trigger-job-btn"
              disabled={isTriggerPending}
              onClick={onTriggerNow}
              type="button"
            >
              <Play className="size-4" />
              {isTriggerPending ? "Queuing..." : "Run now"}
            </button>
          ) : null}
          {isDynamic ? (
            <button
              className="inline-flex h-9 items-center gap-2 rounded-lg border border-[color:var(--color-divider)] px-4 text-sm font-medium text-[color:var(--color-danger)] transition-colors hover:bg-[color:var(--color-hover)] disabled:opacity-50"
              data-testid="delete-automation-btn"
              disabled={isDeleting}
              onClick={onDelete}
              type="button"
            >
              <Trash2 className="size-4" />
              {isDeleting ? "Deleting..." : "Delete"}
            </button>
          ) : null}
        </div>
      </div>

      {!isDynamic ? (
        <div className="mt-4 rounded-xl border border-dashed border-[color:var(--color-divider)] px-4 py-3 text-sm text-[color:var(--color-text-secondary)]">
          Config-sourced automation can only toggle enabled state from the UI. Definition changes
          stay in configuration files.
        </div>
      ) : null}

      <div className="mt-6 grid gap-6 xl:grid-cols-[minmax(0,1.2fr)_minmax(320px,0.8fr)]">
        <section className="space-y-6">
          <section className="space-y-3 rounded-xl border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] p-4">
            <div className="space-y-1">
              <h3 className="font-mono text-[10px] font-semibold uppercase tracking-[0.1em] text-[color:var(--color-text-label)]">
                Prompt
              </h3>
              <p className="text-sm text-[color:var(--color-text-secondary)]">
                The exact prompt payload that will be sent to the agent session.
              </p>
            </div>
            <pre className="whitespace-pre-wrap rounded-lg bg-[color:var(--color-surface-panel)] p-4 font-mono text-xs leading-relaxed text-[color:var(--color-text-secondary)]">
              {item.prompt}
            </pre>
          </section>

          <AutomationRunHistory
            error={runsError}
            isLoading={runsLoading}
            runs={runs}
            title={isJob ? "Job runs" : "Trigger runs"}
          />
        </section>

        <section className="space-y-3 rounded-xl border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] p-4">
          <div className="space-y-1">
            <h3 className="font-mono text-[10px] font-semibold uppercase tracking-[0.1em] text-[color:var(--color-text-label)]">
              Metadata
            </h3>
            <p className="text-sm text-[color:var(--color-text-secondary)]">
              Operational state, retry posture, and scope binding for this automation.
            </p>
          </div>
          <div className="space-y-2">
            <MetadataRow label="Type" value={isJob ? "Job" : "Trigger"} />
            <MetadataRow label="Scope" value={item.scope} />
            <MetadataRow label="Source" value={item.source} />
            <MetadataRow label="Retry" value={describeRetry(item.retry)} />
            <MetadataRow label="Fire limit" value={describeFireLimit(item.fire_limit)} />
            <MetadataRow label="Created" value={formatDateTime(item.created_at)} />
            {item.workspace_id ? <MetadataRow label="Workspace" value={item.workspace_id} /> : null}
            {isJob ? (
              <>
                <MetadataRow
                  label="Next run"
                  value={formatRelativeTime((item as AutomationJob).next_run)}
                />
                <MetadataRow
                  label="Schedule"
                  value={describeSchedule((item as AutomationJob).schedule)}
                />
              </>
            ) : (
              <>
                <MetadataRow label="Event" value={(item as AutomationTrigger).event} />
                {(item as AutomationTrigger).endpoint_slug ? (
                  <MetadataRow
                    label="Endpoint"
                    value={(item as AutomationTrigger).endpoint_slug ?? ""}
                  />
                ) : null}
                {(item as AutomationTrigger).webhook_id ? (
                  <MetadataRow
                    label="Webhook id"
                    value={(item as AutomationTrigger).webhook_id ?? ""}
                  />
                ) : null}
              </>
            )}
          </div>
        </section>
      </div>
    </div>
  );
}
