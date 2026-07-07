import type { LoopRunEventFrame } from "../types";

/**
 * The SSE contract the run page consumes (techspec §observability). Each payload
 * type below is the shape the daemon writes into `frame.payload`; the reducer reads
 * it through runtime guards so an unknown or malformed frame degrades to a bare
 * event line instead of throwing.
 */

/** One collapsed line for the "Live events" rail (every frame yields one). */
export interface LoopLiveEvent {
  seq: number;
  at: string;
  kind: string;
  /** Success/danger/warning accent for the event kind (rail-only, cosmetic). */
  tone: "neutral" | "ok" | "warn" | "err";
  message: string;
}

export interface LoopChannelMessage {
  id: string;
  author: string;
  role?: string;
  text: string;
  /** The harvested `result` coordination message that completes converse-and-decide. */
  isResult: boolean;
}

export interface LoopGateVerdict {
  nodeId: string;
  generation: number;
  verdict: "pass" | "revise";
  confidence?: number;
  reason?: string;
  route?: string;
  criteria: { id: string; type: string; status: "pass" | "revise"; note?: string }[];
  blockingIssues: { id: string; note?: string }[];
}

export interface LoopApprovalFact {
  label: string;
  value: string;
}

export interface LoopApprovalRequest {
  gateId: string;
  title: string;
  prompt?: string;
  facts: LoopApprovalFact[];
}

export interface LoopRunLiveState {
  events: LoopLiveEvent[];
  channelMessages: LoopChannelMessage[];
  /** Latest gate verdict keyed by `nodeId` (a re-run overwrites the prior verdict). */
  gateVerdicts: Record<string, LoopGateVerdict>;
  needsApproval: LoopApprovalRequest | null;
  /** Latest token count from `token_tick`, overlaid on the polled run when fresher. */
  tokensUsed: number | null;
}

export function emptyLoopRunLiveState(): LoopRunLiveState {
  return {
    events: [],
    channelMessages: [],
    gateVerdicts: {},
    needsApproval: null,
    tokensUsed: null,
  };
}

const MAX_RAIL_EVENTS = 40;

const KIND_TONE: Record<string, LoopLiveEvent["tone"]> = {
  node_running: "neutral",
  node_succeeded: "ok",
  node_failed: "err",
  gate_verdict: "warn",
  generation_started: "warn",
  channel_msg: "ok",
  token_tick: "ok",
  needs_approval: "warn",
  status_changed: "neutral",
};

function asRecord(value: unknown): Record<string, unknown> | null {
  return typeof value === "object" && value !== null ? (value as Record<string, unknown>) : null;
}

function str(value: unknown, fallback = ""): string {
  return typeof value === "string" ? value : fallback;
}

function num(value: unknown): number | undefined {
  return typeof value === "number" && Number.isFinite(value) ? value : undefined;
}

function numOrString(value: unknown, fallback = ""): string {
  if (typeof value === "number" && Number.isFinite(value)) return String(value);
  return str(value, fallback);
}

function reattemptStrategyLabel(value: unknown): string {
  if (value === "failed_only") return "failed-only";
  if (value === "full_body") return "full-body";
  return str(value);
}

/** Human summary for the rail line, derived per kind from the frame payload. */
function eventMessage(kind: string, payload: Record<string, unknown> | null): string {
  if (!payload) return kind;
  switch (kind) {
    case "node_running":
    case "node_succeeded":
    case "node_failed":
      return str(payload.node_id, kind);
    case "gate_verdict":
      return `${str(payload.node_id, "gate")} → ${str(payload.verdict, "verdict")}`;
    case "generation_started": {
      const strategy = reattemptStrategyLabel(payload.reattempt_strategy);
      return `gen ${numOrString(payload.generation, "?")}${strategy ? ` · ${strategy}` : ""}`;
    }
    case "channel_msg":
      return `${str(payload.author, "peer")}: ${str(payload.text)}`.slice(0, 80);
    case "token_tick": {
      const total = num(payload.tokens_used);
      return total === undefined ? "token tick" : `${total.toLocaleString()} tokens`;
    }
    case "needs_approval":
      return str(payload.title, "human approval requested");
    case "status_changed":
      return str(payload.status, "status changed");
    default:
      return kind;
  }
}

function applyGateVerdict(
  verdicts: Record<string, LoopGateVerdict>,
  payload: Record<string, unknown>
): Record<string, LoopGateVerdict> {
  const nodeId = str(payload.node_id);
  if (nodeId === "") return verdicts;
  const rawCriteria = Array.isArray(payload.criteria) ? payload.criteria : [];
  const rawIssues = Array.isArray(payload.blocking_issues) ? payload.blocking_issues : [];
  return {
    ...verdicts,
    [nodeId]: {
      nodeId,
      generation: num(payload.generation) ?? 0,
      verdict: str(payload.verdict) === "pass" ? "pass" : "revise",
      confidence: num(payload.confidence),
      reason: str(payload.reason) || undefined,
      route: str(payload.route) || undefined,
      criteria: rawCriteria.map(item => {
        const record = asRecord(item);
        return {
          id: str(record?.id),
          type: str(record?.type),
          status: str(record?.status) === "pass" ? "pass" : "revise",
          note: str(record?.note) || undefined,
        };
      }),
      blockingIssues: rawIssues.map(item => {
        const record = asRecord(item);
        return { id: str(record?.id), note: str(record?.note) || undefined };
      }),
    },
  };
}

function appendChannelMessage(
  messages: LoopChannelMessage[],
  payload: Record<string, unknown>
): LoopChannelMessage[] {
  const text = str(payload.text);
  if (text === "") return messages;
  return [
    ...messages,
    {
      id: str(payload.id) || `msg-${messages.length}`,
      author: str(payload.author, "peer"),
      role: str(payload.role) || undefined,
      text,
      isResult: payload.is_result === true || str(payload.kind) === "result",
    },
  ];
}

function parseApproval(payload: Record<string, unknown>): LoopApprovalRequest {
  const rawFacts = Array.isArray(payload.facts) ? payload.facts : [];
  return {
    gateId: str(payload.gate_id, "approve"),
    title: str(payload.title, "Approve to resume"),
    prompt: str(payload.prompt) || undefined,
    facts: rawFacts.map(item => {
      const record = asRecord(item);
      return { label: str(record?.label), value: str(record?.value) };
    }),
  };
}

/**
 * Folds one SSE frame into the run-page live state: every frame appends a rail line
 * (bounded), and the structured kinds (`gate_verdict`, `channel_msg`, `needs_approval`,
 * `token_tick`) also update their dedicated surface. Malformed payloads degrade to the
 * rail line only.
 */
export function applyLoopEventFrame(
  state: LoopRunLiveState,
  frame: LoopRunEventFrame
): LoopRunLiveState {
  const kind = str(frame.kind);
  const payload = asRecord(frame.payload);
  const event: LoopLiveEvent = {
    seq: num(frame.seq) ?? 0,
    at: str(frame.at),
    kind,
    tone: KIND_TONE[kind] ?? "neutral",
    message: eventMessage(kind, payload),
  };
  const next: LoopRunLiveState = {
    ...state,
    events: [event, ...state.events].slice(0, MAX_RAIL_EVENTS),
  };
  if (!payload) return next;
  switch (kind) {
    case "gate_verdict":
      next.gateVerdicts = applyGateVerdict(state.gateVerdicts, payload);
      break;
    case "channel_msg":
      next.channelMessages = appendChannelMessage(state.channelMessages, payload);
      break;
    case "needs_approval":
      next.needsApproval = parseApproval(payload);
      break;
    case "token_tick": {
      const total = num(payload.tokens_used);
      if (total !== undefined) next.tokensUsed = total;
      break;
    }
    default:
      break;
  }
  return next;
}
