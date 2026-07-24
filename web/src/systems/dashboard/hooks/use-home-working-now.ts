import { useSessions } from "@/systems/session";
import { useTaskDashboard } from "@/systems/tasks";

import type { HomeScope } from "../lib/home-scope";
import type { HomeSurfaceStatus } from "../types";

const WORKING_NOW_SESSION_LIMIT = 6;

export interface HomeRunCardModel {
  key: string;
  kind: "session" | "task_run";
  agentName: string;
  title: string;
  subtitle: string;
  /** Elapsed seconds at snapshot time; the card ticks forward from `baseAtMs`. */
  elapsedBaseSeconds: number;
  baseAtMs: number;
  sessionLink?: { agentName: string; sessionId: string };
  runLink?: { taskId: string; runId: string };
}

export interface HomeWorkingNowModel {
  cards: HomeRunCardModel[];
  total: number;
  status: HomeSurfaceStatus;
}

function elapsedSecondsFromIso(iso: string | null | undefined, nowMs: number): number {
  if (!iso) {
    return 0;
  }
  const parsed = Date.parse(iso);
  if (Number.isNaN(parsed)) {
    return 0;
  }
  return Math.max(0, Math.floor((nowMs - parsed) / 1000));
}

export function useHomeWorkingNow(scope: HomeScope): HomeWorkingNowModel {
  const sessionsQuery = useSessions(scope.workspaceParam || null, {
    filters: {
      state: "active",
      type: "user",
      sort: "last_activity",
      limit: WORKING_NOW_SESSION_LIMIT,
    },
  });
  const dashboardQuery = useTaskDashboard({
    scope: scope.taskScope.scope,
    workspace: scope.taskScope.workspace,
  });

  const nowMs = Date.now();
  const cards: HomeRunCardModel[] = [];

  for (const session of sessionsQuery.data ?? []) {
    const activity = session.activity;
    cards.push({
      key: `session:${session.id}`,
      kind: "session",
      agentName: session.agent_name,
      title: session.name?.trim() ? session.name : session.agent_name,
      subtitle: activity?.current_tool?.trim()
        ? `Running ${activity.current_tool}`
        : "Working on the current turn",
      elapsedBaseSeconds: elapsedSecondsFromIso(
        activity?.turn_started_at ?? session.created_at,
        nowMs
      ),
      baseAtMs: nowMs,
      sessionLink: { agentName: session.agent_name, sessionId: session.id },
    });
  }

  const activeRuns = dashboardQuery.data?.active_runs;
  for (const run of activeRuns?.items ?? []) {
    if (run.session_id && cards.some(card => card.key === `session:${run.session_id}`)) {
      continue;
    }
    cards.push({
      key: `run:${run.run_id}`,
      kind: "task_run",
      agentName: run.task_owner?.ref ?? "",
      title: run.task_title,
      subtitle: `Run ${run.run_status}`,
      elapsedBaseSeconds: Math.max(0, Math.floor(run.age_ms / 1000)),
      baseAtMs: nowMs,
      runLink: { taskId: run.task_id, runId: run.run_id },
    });
  }

  const sessionTotal = sessionsQuery.total ?? cards.filter(card => card.kind === "session").length;
  const runTotal = activeRuns?.total ?? 0;
  const isLoading = sessionsQuery.isLoading || dashboardQuery.isLoading;
  const isError = sessionsQuery.isError && dashboardQuery.isError;

  let status: HomeSurfaceStatus = "ready";
  if (cards.length === 0 && isLoading) {
    status = "loading";
  } else if (cards.length === 0 && isError) {
    status = "error";
  }

  return { cards, total: sessionTotal + runTotal, status };
}
