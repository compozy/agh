import { buildLocalNetworkParticipationFixture } from "@/test/network-participation-fixtures";

import type { GoalTurnTimelineItem } from "../../hooks/use-goal-turns";
import {
  applyLoopEventFrame,
  emptyLoopRunLiveState,
  type LoopRunLiveState,
} from "../../lib/loop-events";
import { findWatchNode, goalNodeIds, readLoopGraph } from "../../lib/loop-graph";
import { buildInputRows, humanizeStartOrigin, watchedSubject } from "../../lib/loop-run-about";
import { buildRunProgress, latestGateVerdict } from "../../lib/loop-run-progress";
import { buildNextNote, buildRunStory } from "../../lib/loop-run-story";
import {
  buildRunUsage,
  formatClockDuration,
  runElapsedSeconds,
  usageNote,
  usageSnapshotFacts,
} from "../../lib/loop-run-usage";
import { isTerminalLoopStatus } from "../../lib/loop-formatters";
import type { LoopRunPageBodyProps } from "../run-page/loop-run-page-body";
import type {
  LoopDefinition,
  LoopRunEventFrame,
  LoopRunGeneration,
  LoopRunRecord,
  LoopWatchEventsState,
} from "../../types";

/**
 * The `reviews-watch` story world mirroring the canonical prototypes
 * (`loop-run-detail.html` + states): a watch loop fixing review comments on PR
 * #128 in three fan-out groups. The data is fixture flavor; every derivation
 * below runs through the production libs, exactly like the page hook.
 */

const NOW = Date.now();

function minutesAgo(minutes: number): string {
  return new Date(NOW - minutes * 60_000).toISOString();
}

export const reviewsWatchDefinition = {
  apiVersion: "agh.loop/v1",
  kind: "Loop",
  meta: {
    name: "reviews-watch",
    version: 3,
    description: "Watches a PR and resolves every review comment.",
    catalog: { category: "watch" },
  },
  concurrency: "forbid",
  inputs: {
    pr: { type: "string", required: true, description: "The pull request to watch." },
    fixer: { type: "ref", ref: { kind: "agent" }, description: "The fix agent." },
  },
  contract: {
    goal: "Resolve every review comment on PR #128",
    definition_of_done:
      "Done when a fresh CodeRabbit review of the current changes reports zero unresolved comments.",
    stop_when: "review.unresolved == 0",
    iteration_cap: 0,
    budget: { tokens: 1_500_000, wall_clock_sec: 2_700, on_exceeded: "escalate" },
    no_progress: { window: 2, hash_fields: ["gate_verdict"] },
    terminal_states: ["done", "no-op", "blocked", "failed", "exhausted", "stalled"],
    verification: [
      {
        id: "all_issues_handled",
        type: "agent-judge",
        agent: "agent-judge",
        rubric: "Every comment has a decision; valid ones a fix with tests.",
      },
    ],
  },
  graph: {
    nodes: [
      {
        id: "watch_pr",
        class: "source",
        kind: "watch-events",
        watch: { poll: "30s", settle: "20s" },
        events: [{ kind: "review.completed" }],
      },
      { id: "fetch_issues", class: "action", kind: "run-agent" },
      {
        id: "split_batches",
        class: "control",
        kind: "fan-out",
        batch_size: 10,
        max_parallel: 1,
        max_fan_out: 64,
      },
      { id: "fix_batches", class: "action", kind: "run-agent" },
      {
        id: "check_all",
        class: "control",
        kind: "gate",
        verdict_policy: "revise_until_clean",
        criteria: [{ id: "all_issues_handled", type: "agent-judge" }],
      },
      { id: "resolve_threads", class: "action", kind: "run-agent" },
      { id: "push_changes", class: "action", kind: "run-agent" },
    ],
    edges: [
      { from: "watch_pr", to: "fetch_issues" },
      { from: "fetch_issues", to: "split_batches" },
      { from: "split_batches", to: "fix_batches" },
      { from: "fix_batches", to: "check_all" },
      { from: "check_all", to: "resolve_threads" },
      { from: "resolve_threads", to: "push_changes" },
    ],
  },
} as unknown as LoopDefinition;

export function reviewsWatchRun(overrides: Partial<LoopRunRecord> = {}): LoopRunRecord {
  return {
    id: "r-7c4e19",
    workspace_id: "ws_default",
    loop_name: "reviews-watch",
    status: "running",
    generation: 2,
    reattempt_strategy: "failed_only",
    created_at: minutesAgo(22),
    started_at: minutesAgo(22),
    last_progress_at: minutesAgo(4),
    started_by_kind: "user",
    started_by_ref: "operator",
    started_origin_kind: "webhook",
    started_origin_ref: "pr-opened",
    definition_version: 3,
    definition_digest: "sha256:4f9c2a1e8b",
    iteration_cap: 0,
    budget_tokens: 1_500_000,
    budget_wall_sec: 2_700,
    budget_on_exceeded: "escalate",
    tokens_used: 268_000,
    pause_requested: false,
    inputs: { pr: "128", fixer: "review-fixer" },
    resolved_network_participation: buildLocalNetworkParticipationFixture(),
    ...overrides,
  };
}

let frameSeq = 0;

function frame(
  kind: LoopRunEventFrame["kind"],
  minutes: number,
  payload: Record<string, unknown>
): LoopRunEventFrame {
  frameSeq += 1;
  return {
    id: `loopevt_${frameSeq}`,
    seq: frameSeq,
    kind,
    loop_run_id: "r-7c4e19",
    workspace_id: "ws_default",
    at: minutesAgo(minutes),
    payload,
  };
}

function nodePayload(
  nodeId: string,
  generation: number,
  extra: Record<string, unknown> = {}
): Record<string, unknown> {
  return { node_id: nodeId, generation, ...extra };
}

const REVISE_ISSUES = [
  { id: "issue_022", note: "no decision recorded in the group harvest" },
  { id: "issue_024", note: "triaged valid but no fix landed" },
];

/**
 * Round-1 history shared by every state: wake → fetch → groups → check.
 * `offset` shifts the whole round into the past so each scenario's later
 * frames stay chronologically after it (seq order == time order).
 */
function roundOneFrames(offset = 0): LoopRunEventFrame[] {
  return [
    frame("status_changed", offset + 22, {
      from: "watching",
      to: "running",
      status: "running",
      cause: "watch_events",
    }),
    frame(
      "node_succeeded",
      offset + 22,
      nodePayload("fetch_issues", 1, { task_id: "task_fetch", task_run_id: "tr_101" })
    ),
    frame(
      "node_succeeded",
      offset + 10,
      nodePayload("fix_batches", 1, { item_index: 1, task_id: "task_fix", task_run_id: "tr_102" })
    ),
    frame(
      "node_succeeded",
      offset + 10,
      nodePayload("fix_batches", 1, { item_index: 2, task_id: "task_fix", task_run_id: "tr_103" })
    ),
    frame(
      "node_failed",
      offset + 6,
      nodePayload("fix_batches", 1, {
        item_index: 3,
        task_id: "task_fix",
        task_run_id: "tr_104",
        output_ref: JSON.stringify({
          kind: "action_failure",
          code: "incomplete_batch",
          cause: "2 of its 4 comments weren't fully handled.",
          recovery: "Re-run the failed group.",
        }),
      })
    ),
    frame("gate_verdict", offset + 5, {
      node_id: "check_all",
      generation: 1,
      verdict: "revise",
      confidence: 0.91,
      criteria: [
        {
          id: "all_issues_handled",
          type: "agent-judge",
          status: "revise",
          note: "two open points",
        },
      ],
      blocking_issues: REVISE_ISSUES,
    }),
  ];
}

function generationsFor(branchThree: string): LoopRunGeneration[] {
  return [
    {
      generation: 2,
      outputs: [
        { node_id: "fetch_issues", status: "reused", generation: 2 },
        { node_id: "fix_batches", status: "reused", generation: 2, item_index: 1 },
        { node_id: "fix_batches", status: "reused", generation: 2, item_index: 2 },
        {
          node_id: "fix_batches",
          status: branchThree,
          generation: 2,
          item_index: 3,
          task_run_id: "tr_204",
        },
        { node_id: "check_all", status: "pending", generation: 2 },
        { node_id: "resolve_threads", status: "pending", generation: 2 },
        { node_id: "push_changes", status: "pending", generation: 2 },
      ],
    },
  ];
}

export interface LoopRunStoryScenario {
  run: LoopRunRecord;
  frames: LoopRunEventFrame[];
  generations: LoopRunGeneration[];
  watchEvents?: LoopWatchEventsState;
  goalTurns?: GoalTurnTimelineItem[];
}

export function runningScenario(): LoopRunStoryScenario {
  frameSeq = 0;
  const frames = [
    ...roundOneFrames(),
    frame("generation_started", 4, { generation: 2, reattempt_strategy: "failed_only" }),
    frame(
      "node_running",
      4,
      nodePayload("fix_batches", 2, {
        item_index: 3,
        task_id: "task_fix",
        task_run_id: "tr_204",
      })
    ),
    frame("token_tick", 3, { tokens_used: 268_000 }),
  ];
  return { run: reviewsWatchRun(), frames, generations: generationsFor("running") };
}

export function needsApprovalScenario(): LoopRunStoryScenario {
  frameSeq = 0;
  const frames = [
    ...roundOneFrames(20),
    frame("gate_verdict", 7, {
      node_id: "check_all",
      generation: 2,
      verdict: "revise",
      confidence: 0.88,
      criteria: [
        {
          id: "all_issues_handled",
          type: "agent-judge",
          status: "revise",
          note: "still two open points",
        },
      ],
      blocking_issues: REVISE_ISSUES,
    }),
    frame("generation_started", 6, { generation: 3, reattempt_strategy: "failed_only" }),
    frame("needs_approval", 2, {
      gate_id: "budget",
      title: "Time limit reached — continue this run?",
      generation: 3,
      facts: [
        { label: "Time used", value: "45m of 45m" },
        { label: "Tokens", value: "412K / 1.5M" },
        { label: "Cost", value: "~$1.71 est." },
        { label: "Round", value: "3" },
      ],
    }),
    frame("status_changed", 2, {
      from: "running",
      to: "needs-approval",
      status: "needs-approval",
      cause: "budget",
    }),
  ];
  return {
    run: reviewsWatchRun({
      status: "needs-approval",
      generation: 3,
      tokens_used: 412_000,
      created_at: minutesAgo(46),
      started_at: minutesAgo(46),
      last_progress_at: minutesAgo(1),
      active_gate_id: "budget",
    }),
    frames,
    generations: generationsFor("pending"),
  };
}

export function watchingScenario(): LoopRunStoryScenario {
  frameSeq = 0;
  const frames = [
    ...roundOneFrames(16),
    frame("generation_started", 20, { generation: 2, reattempt_strategy: "failed_only" }),
    frame(
      "node_succeeded",
      19,
      nodePayload("fix_batches", 2, { item_index: 3, task_id: "task_fix", task_run_id: "tr_204" })
    ),
    frame("gate_verdict", 18, {
      node_id: "check_all",
      generation: 2,
      verdict: "pass",
      confidence: 0.94,
      criteria: [
        {
          id: "all_issues_handled",
          type: "agent-judge",
          status: "pass",
          note: "every comment handled",
        },
      ],
      blocking_issues: [],
    }),
    frame(
      "node_succeeded",
      17,
      nodePayload("resolve_threads", 2, { task_id: "task_resolve", task_run_id: "tr_205" })
    ),
    frame(
      "node_succeeded",
      17,
      nodePayload("push_changes", 2, { task_id: "task_push", task_run_id: "tr_206" })
    ),
    frame("status_changed", 16, {
      from: "running",
      to: "watching",
      status: "watching",
      cause: "contract",
    }),
  ];
  const generations: LoopRunGeneration[] = [
    {
      generation: 2,
      outputs: [
        { node_id: "fetch_issues", status: "reused", generation: 2 },
        { node_id: "fix_batches", status: "reused", generation: 2, item_index: 1 },
        { node_id: "fix_batches", status: "reused", generation: 2, item_index: 2 },
        { node_id: "fix_batches", status: "succeeded", generation: 2, item_index: 3 },
        { node_id: "check_all", status: "succeeded", generation: 2 },
        { node_id: "resolve_threads", status: "succeeded", generation: 2 },
        { node_id: "push_changes", status: "succeeded", generation: 2 },
      ],
    },
  ];
  return {
    run: reviewsWatchRun({
      status: "watching",
      tokens_used: 341_000,
      created_at: minutesAgo(38),
      started_at: minutesAgo(38),
      last_progress_at: minutesAgo(16),
    }),
    frames,
    generations,
    watchEvents: {
      subscriptions: [{ kind: "event.post_record", filter: "payload.pr == input.pr" }],
      cursors: { workspace_events: 4_182 },
      last_wake_at: minutesAgo(16),
    },
  };
}

export function pausedScenario(): LoopRunStoryScenario {
  frameSeq = 0;
  const frames = [
    ...roundOneFrames(5),
    frame("generation_started", 8, { generation: 2, reattempt_strategy: "failed_only" }),
    frame("status_changed", 3, {
      from: "running",
      to: "paused",
      status: "paused",
      cause: "pause_boundary",
    }),
  ];
  return {
    run: reviewsWatchRun({
      status: "paused",
      tokens_used: 296_000,
      created_at: minutesAgo(29),
      started_at: minutesAgo(29),
      last_progress_at: minutesAgo(3),
    }),
    frames,
    generations: generationsFor("pending"),
  };
}

export function failedScenario(): LoopRunStoryScenario {
  frameSeq = 0;
  const frames = [
    ...roundOneFrames(61),
    frame("generation_started", 62, { generation: 2, reattempt_strategy: "failed_only" }),
    frame(
      "node_running",
      61,
      nodePayload("fix_batches", 2, { item_index: 3, task_id: "task_fix", task_run_id: "tr_204" })
    ),
    frame("status_changed", 60, {
      from: "running",
      to: "failed",
      status: "failed",
      cause: "coordinator_failure",
      failure: {
        kind: "coordinator_failure",
        code: "provider_auth",
        cause: "GitHub authentication expired while fetching review threads.",
        recovery: "Fix the GitHub credential in Vault, then start a new run.",
      },
    }),
  ];
  return {
    run: reviewsWatchRun({
      status: "failed",
      tokens_used: 310_000,
      created_at: minutesAgo(87),
      started_at: minutesAgo(87),
      last_progress_at: minutesAgo(60),
    }),
    frames,
    generations: generationsFor("running"),
  };
}

export function exhaustedScenario(): LoopRunStoryScenario {
  frameSeq = 0;
  const frames = [
    ...roundOneFrames(10),
    frame("status_changed", 30, {
      from: "running",
      to: "exhausted",
      status: "exhausted",
      cause: "budget",
    }),
  ];
  return {
    run: reviewsWatchRun({
      status: "exhausted",
      tokens_used: 1_500_000,
      budget_on_exceeded: "halt",
      created_at: minutesAgo(80),
      started_at: minutesAgo(80),
      last_progress_at: minutesAgo(30),
    }),
    frames,
    generations: generationsFor("pending"),
  };
}

export function noOpScenario(): LoopRunStoryScenario {
  frameSeq = 0;
  const frames = [
    frame("status_changed", 50, {
      from: "watching",
      to: "running",
      status: "running",
      cause: "watch_poll",
    }),
    frame("node_succeeded", 50, nodePayload("fetch_issues", 1, {})),
    frame("status_changed", 49, {
      from: "running",
      to: "no-op",
      status: "no-op",
      cause: "contract",
    }),
  ];
  return {
    run: reviewsWatchRun({
      status: "no-op",
      generation: 1,
      tokens_used: 12_000,
      created_at: minutesAgo(51),
      started_at: minutesAgo(51),
      last_progress_at: minutesAgo(49),
    }),
    frames,
    generations: [],
  };
}

/** Reduces the scenario's frames through the production reducer, like the page. */
function reduceLiveState(frames: readonly LoopRunEventFrame[]): LoopRunLiveState {
  return frames.reduce(applyLoopEventFrame, emptyLoopRunLiveState());
}

type ScenarioBodyProps = Omit<LoopRunPageBodyProps, "inspect">;

/** Derives the full page-body prop set from a scenario via the production libs. */
export function buildScenarioProps(scenario: LoopRunStoryScenario): ScenarioBodyProps {
  const { run, generations, watchEvents } = scenario;
  const live = reduceLiveState(scenario.frames);
  const graph = readLoopGraph(reviewsWatchDefinition);
  const story = buildRunStory(live.frames, {
    status: run.status,
    reattemptStrategy: run.reattempt_strategy,
    graph,
    generations,
  });
  const verdict = latestGateVerdict(live.gateVerdicts);
  const openPoints = verdict ? verdict.blockingIssues.length : null;
  const elapsedSeconds = runElapsedSeconds(run, NOW);
  const isLive = !isTerminalLoopStatus(run.status);
  const watchNode = findWatchNode(graph);
  const showNextNote =
    run.status === "running" || run.status === "watching" || run.status === "queued";
  const stepStartedMs = story.now ? Date.parse(story.now.startedAt) : Number.NaN;
  return {
    run,
    definition: reviewsWatchDefinition,
    graph,
    isLive,
    subject: watchedSubject(run, reviewsWatchDefinition),
    hasWatchSource: watchNode !== null,
    elapsedLabel: formatClockDuration(elapsedSeconds),
    stepElapsedLabel:
      run.status === "running" && !Number.isNaN(stepStartedMs)
        ? formatClockDuration((NOW - stepStartedMs) / 1000)
        : null,
    progress: buildRunProgress(run, generations, openPoints),
    story,
    goalIds: goalNodeIds(graph),
    goalTurns: scenario.goalTurns ?? [],
    usageRows: buildRunUsage(run, elapsedSeconds),
    usageNote: usageNote(run),
    approvalRequest: live.needsApproval,
    approvalFallbackFacts: usageSnapshotFacts(run, elapsedSeconds),
    failure: live.failure,
    latestVerdict: verdict,
    watchEvents,
    watchCadence:
      typeof watchNode?.watch?.poll === "string" ? `checks every ${watchNode.watch.poll}` : null,
    frames: live.frames,
    inputRows: buildInputRows(run, reviewsWatchDefinition),
    startedBy: humanizeStartOrigin(run),
    workspaceLabel: "Home",
    versionLabel: `v${run.definition_version} · pinned`,
    nextNote: showNextNote ? buildNextNote(graph, generations) : null,
    showNowCard: run.status === "running" || run.status === "watching" || run.status === "paused",
    terminalFromStatus: isTerminalLoopStatus(run.status) ? "running" : undefined,
    onDecision: () => undefined,
    onStartNewRun: () => undefined,
  };
}
