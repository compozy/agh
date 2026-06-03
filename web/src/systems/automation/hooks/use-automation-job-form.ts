import type { FormEvent } from "react";
import { useMemo } from "react";

import {
  jobOutputMode,
  retryDraftForStrategy,
  setJobOutputMode,
  type JobOutputMode,
} from "../lib/automation-drafts";
import {
  compileCron,
  decodeCron,
  defaultAtLocal,
  fullCronModel,
  localInputToDate,
  parseCron,
  parseDuration,
  type CronFrequency,
  type CronModel,
} from "../lib/cron-engine";
import { buildJobPreview } from "../lib/job-preview";
import type { WorkspaceOption } from "../lib/trigger-preview";
import type {
  AutomationFireLimit,
  AutomationRetry,
  AutomationScheduleMode,
  AutomationScope,
  CreateAutomationJobRequest,
} from "../types";

type JobTask = NonNullable<CreateAutomationJobRequest["task"]>;
type JobOwner = NonNullable<JobTask["owner"]>;
type JobOwnerKind = JobOwner["kind"];

const DEFAULT_CRON_EXPR = "0 9 * * *";
const DEFAULT_EVERY_INTERVAL = "30m";

/** Weekday presets for the visual cron builder (0=Sun … 6=Sat). */
const WEEKDAY_PRESETS = {
  weekdays: [1, 2, 3, 4, 5],
  weekend: [0, 6],
  all: [0, 1, 2, 3, 4, 5, 6],
} as const;

export type WeekdayPreset = keyof typeof WEEKDAY_PRESETS;

export interface UseAutomationJobFormParams {
  activeWorkspaceId?: string | null;
  draft: CreateAutomationJobRequest;
  isPending: boolean;
  mode: "create" | "edit";
  onChange: (draft: CreateAutomationJobRequest) => void;
  onSubmit: () => void;
  workspaces?: ReadonlyArray<WorkspaceOption>;
  agents?: string[];
}

/** Read the current cron expression off the draft schedule (cron mode only). */
function scheduleExpr(draft: CreateAutomationJobRequest): string {
  return draft.schedule.expr ?? "";
}

/** True when the draft's active schedule is valid for its own mode. */
function scheduleValid(draft: CreateAutomationJobRequest, now: number): boolean {
  const schedule = draft.schedule;
  if (schedule.mode === "cron") {
    return parseCron(schedule.expr ?? "") !== null;
  }
  if (schedule.mode === "every") {
    return parseDuration(schedule.interval ?? "") !== null;
  }
  const date = localInputToDate(schedule.time ?? "");
  return date !== null && date.getTime() > now;
}

function computeCanSubmit(draft: CreateAutomationJobRequest, now: number): boolean {
  const baseValid =
    draft.name.trim() !== "" && (draft.scope === "global" || Boolean(draft.workspace_id));
  if (!baseValid) return false;
  if (!scheduleValid(draft, now)) return false;
  if (jobOutputMode(draft) === "task") {
    // Task mode: owner is optional, title/description fall back to the job — nothing more required.
    return true;
  }
  return draft.agent_name.trim() !== "" && draft.prompt.trim() !== "";
}

/** View-model for the Job form: derives preview/schedule model and owns every patch handler. */
export function useAutomationJobForm({
  activeWorkspaceId,
  draft,
  isPending,
  mode,
  onChange,
  onSubmit,
  workspaces,
  agents,
}: UseAutomationJobFormParams) {
  const output: JobOutputMode = jobOutputMode(draft);
  const retry = retryDraftForStrategy(draft.retry?.strategy ?? "none", draft.retry ?? undefined);

  const cronModel = useMemo<CronModel>(
    () => fullCronModel(decodeCron(scheduleExpr(draft)) ?? { frequency: "custom" }),
    [draft]
  );

  const resolvedWorkspaces = useMemo<WorkspaceOption[]>(() => {
    if (workspaces && workspaces.length > 0) return [...workspaces];
    return activeWorkspaceId ? [{ id: activeWorkspaceId, name: activeWorkspaceId }] : [];
  }, [workspaces, activeWorkspaceId]);

  const preview = useMemo(() => buildJobPreview(draft), [draft]);
  const canSubmit = computeCanSubmit(draft, Date.now());

  const patch = (next: Partial<CreateAutomationJobRequest>) => onChange({ ...draft, ...next });

  /** Patch the schedule object, preserving fields that belong to other modes. */
  const patchSchedule = (next: Partial<CreateAutomationJobRequest["schedule"]>) =>
    patch({ schedule: { ...draft.schedule, ...next } });

  /** Patch the task object (only meaningful in task mode). */
  const patchTask = (next: Partial<JobTask>) => {
    const current = draft.task ?? {};
    patch({ task: { ...current, ...next } });
  };

  /** Compile the builder model back into the draft's cron expression. */
  const applyCronModel = (model: CronModel) => {
    const compiled = compileCron(model);
    // `custom` keeps the raw user-entered expression; everything else recompiles.
    patchSchedule({ mode: "cron", expr: compiled ?? scheduleExpr(draft) });
  };

  const handleScopeChange = (scope: AutomationScope) => {
    if (scope === "global") {
      patch({ scope: "global", workspace_id: undefined });
      return;
    }
    const fallback = draft.workspace_id ?? activeWorkspaceId ?? resolvedWorkspaces[0]?.id;
    patch({ scope: "workspace", workspace_id: fallback ?? undefined });
  };

  const handleOutputChange = (next: JobOutputMode) => onChange(setJobOutputMode(draft, next));

  const handleOwnerKind = (kind: JobOwnerKind | "") => {
    if (kind === "") {
      patchTask({ owner: null });
      return;
    }
    const ref = draft.task?.owner?.ref ?? "";
    patchTask({ owner: { kind, ref } });
  };

  const handleOwnerRef = (ref: string) => {
    const owner = draft.task?.owner;
    if (!owner) return;
    patchTask({ owner: { ...owner, ref } });
  };

  const handleScheduleMode = (next: AutomationScheduleMode) => {
    if (next === "cron") {
      const expr = scheduleExpr(draft) || DEFAULT_CRON_EXPR;
      patch({ schedule: { mode: "cron", expr } });
      return;
    }
    if (next === "every") {
      const interval = draft.schedule.interval ?? DEFAULT_EVERY_INTERVAL;
      patch({ schedule: { mode: "every", interval } });
      return;
    }
    const time = draft.schedule.time ?? defaultAtLocal();
    patch({ schedule: { mode: "at", time } });
  };

  const handleCronFrequency = (frequency: CronFrequency) => {
    if (frequency === "custom") {
      patchSchedule({ mode: "cron", expr: scheduleExpr(draft) || DEFAULT_CRON_EXPR });
      return;
    }
    applyCronModel({ ...cronModel, frequency });
  };

  const handleToggleWeekday = (weekday: number) => {
    const current = cronModel.weekdays;
    const has = current.includes(weekday);
    // Never let the weekly set drop to empty — keep at least one day selected.
    const next = has
      ? current.length > 1
        ? current.filter(day => day !== weekday)
        : current
      : [...current, weekday].sort((a, b) => a - b);
    applyCronModel({ ...cronModel, frequency: "weekly", weekdays: next });
  };

  const handleRetryChange = (next: AutomationRetry) => {
    // Task mode locks retry to none — the materialized task owns its own retries.
    if (output === "task") return;
    patch({ retry: next });
  };

  const handleSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!canSubmit || isPending) return;
    onSubmit();
  };

  return {
    output,
    retry,
    cronModel,
    resolvedWorkspaces,
    agents: agents ?? [],
    preview,
    canSubmit,
    scheduleMode: draft.schedule.mode,
    reliabilityDefaultOpen:
      mode === "edit" ||
      retry.strategy === "backoff" ||
      draft.enabled === false ||
      output === "task",

    // identity & scope
    onName: (name: string) => patch({ name }),
    onScopeChange: handleScopeChange,
    onWorkspaceChange: (workspace_id: string) => patch({ workspace_id }),

    // output / agent / task
    onOutputChange: handleOutputChange,
    onAgentChange: (agent_name: string) => patch({ agent_name }),
    onPromptChange: (prompt: string) => patch({ prompt }),
    onTaskTitle: (title: string) => patchTask({ title }),
    onTaskDescription: (description: string) => patchTask({ description }),
    onTaskChannel: (network_channel: string) => patchTask({ network_channel }),
    onOwnerKind: handleOwnerKind,
    onOwnerRef: handleOwnerRef,

    // schedule mode
    onScheduleMode: handleScheduleMode,

    // cron builder
    onCronFrequency: handleCronFrequency,
    onCronExpr: (expr: string) => patchSchedule({ mode: "cron", expr }),
    onEveryMinutes: (everyMinutes: number) =>
      applyCronModel({ ...cronModel, frequency: "minutes", everyMinutes }),
    onHourlyMinute: (hourlyMinute: number) =>
      applyCronModel({ ...cronModel, frequency: "hourly", hourlyMinute }),
    onDailyTime: (hour: number, minute: number) =>
      applyCronModel({ ...cronModel, frequency: "daily", hour, minute }),
    onWeeklyTime: (hour: number, minute: number) =>
      applyCronModel({ ...cronModel, frequency: "weekly", hour, minute }),
    onMonthlyTime: (hour: number, minute: number) =>
      applyCronModel({ ...cronModel, frequency: "monthly", hour, minute }),
    onToggleWeekday: handleToggleWeekday,
    onWeekdayPreset: (preset: WeekdayPreset) =>
      applyCronModel({
        ...cronModel,
        frequency: "weekly",
        weekdays: [...WEEKDAY_PRESETS[preset]],
      }),
    onMonthDay: (monthDay: number) =>
      applyCronModel({ ...cronModel, frequency: "monthly", monthDay }),
    onCronPreset: (expr: string) => patchSchedule({ mode: "cron", expr }),

    // every / at
    onEveryInterval: (interval: string) => patchSchedule({ mode: "every", interval }),
    onEveryPreset: (interval: string) => patchSchedule({ mode: "every", interval }),
    onAtTime: (time: string) => patchSchedule({ mode: "at", time }),

    // reliability
    onRetryChange: handleRetryChange,
    onFireLimitChange: (next: AutomationFireLimit) => patch({ fire_limit: next }),
    onEnabledChange: (enabled: boolean) => patch({ enabled }),

    handleSubmit,
  };
}
