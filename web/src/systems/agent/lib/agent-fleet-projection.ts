import type { SessionPayload } from "@/systems/session";

import type { AgentPayload } from "../types";
import { AGENT_CATEGORY_LABEL_SEPARATOR, formatCategoryLabel } from "./agent-category";
import type { AgentsFleetSearch } from "./agent-fleet-search";
import { deriveAgentFleetSignals, type AgentFleetSignals } from "./fleet-signals";

const META_SEPARATOR = " · ";
const CATEGORY_ELLIPSIS_LIMIT = 40;

export interface AgentFleetRowModel {
  agent: AgentPayload;
  signals: AgentFleetSignals | null;
  meta: string;
  cardMeta: string;
  ariaLabel: string;
  hasDiagnostics: boolean;
  sessionsAvailable: boolean;
}

export function compareAgentNamesStable(a: string, b: string): number {
  const aLower = a.toLowerCase();
  const bLower = b.toLowerCase();
  if (aLower === bLower) return a.localeCompare(b);
  return aLower < bLower ? -1 : 1;
}

export function sortAgentsByNameStable(agents: readonly AgentPayload[]): AgentPayload[] {
  return [...agents].sort((left, right) => compareAgentNamesStable(left.name, right.name));
}

export function indexSessionsByAgent(
  sessions: readonly SessionPayload[]
): Map<string, SessionPayload[]> {
  const byAgent = new Map<string, SessionPayload[]>();
  for (const session of sessions) {
    const existing = byAgent.get(session.agent_name);
    if (existing) {
      existing.push(session);
    } else {
      byAgent.set(session.agent_name, [session]);
    }
  }
  return byAgent;
}

export function formatAgentOriginLabel(origin: AgentPayload["origin"]): string {
  return origin === "workspace" ? "Workspace" : "Global";
}

/** Middle-truncate long category joins while preserving first and last segments. */
export function formatCategoryMetaSegment(path: string[] | null | undefined): string {
  const label = formatCategoryLabel(path);
  if (label.length <= CATEGORY_ELLIPSIS_LIMIT) return label;
  if (!Array.isArray(path) || path.length <= 2) {
    if (label.length <= CATEGORY_ELLIPSIS_LIMIT) return label;
    const head = Math.max(8, Math.floor((CATEGORY_ELLIPSIS_LIMIT - 1) / 2));
    const tail = CATEGORY_ELLIPSIS_LIMIT - 1 - head;
    return `${label.slice(0, head)}…${label.slice(-tail)}`;
  }
  const first = path[0] ?? "";
  const last = path[path.length - 1] ?? "";
  return `${first}${AGENT_CATEGORY_LABEL_SEPARATOR}…${AGENT_CATEGORY_LABEL_SEPARATOR}${last}`;
}

export function formatAgentFleetMeta(agent: AgentPayload): string {
  const segments: string[] = [];
  const category = formatCategoryMetaSegment(agent.category_path);
  if (category) segments.push(category);
  if (agent.provider) segments.push(agent.provider);
  if (agent.model) segments.push(agent.model);
  segments.push(formatAgentOriginLabel(agent.origin));
  return segments.join(META_SEPARATOR);
}

/** Card eyebrow: category (or provider) · model · origin — keeps definition truth without decorative chrome. */
export function formatAgentFleetCardMeta(agent: AgentPayload): string {
  const segments: string[] = [];
  const category = formatCategoryMetaSegment(agent.category_path);
  if (category) {
    segments.push(category);
  } else if (agent.provider) {
    segments.push(agent.provider);
  }
  if (agent.model) segments.push(agent.model);
  segments.push(formatAgentOriginLabel(agent.origin));
  return segments.join(META_SEPARATOR);
}

export function formatAgentFleetAriaLabel(
  agent: AgentPayload,
  signals: AgentFleetSignals | null,
  sessionsAvailable: boolean
): string {
  if (!sessionsAvailable || signals === null) {
    return `${agent.name}, session status unavailable`;
  }
  const statusLabel = signals.status === "active" ? "Active" : "Idle";
  return `${agent.name}, ${statusLabel}, ${signals.active} of ${signals.total} sessions active`;
}

export function collectAgentCategoryOptions(agents: readonly AgentPayload[]): string[] {
  const options = new Set<string>();
  for (const agent of agents) {
    const label = formatCategoryLabel(agent.category_path);
    if (label) options.add(label);
  }
  return [...options].sort(compareAgentNamesStable);
}

function matchesQuery(agent: AgentPayload, query: string): boolean {
  const needle = query.trim().toLowerCase();
  if (needle === "") return true;
  if (agent.name.toLowerCase().includes(needle)) return true;
  const category = formatCategoryLabel(agent.category_path).toLowerCase();
  if (category.includes(needle)) return true;
  if (Array.isArray(agent.category_path)) {
    for (const segment of agent.category_path) {
      if (segment.toLowerCase().includes(needle)) return true;
    }
  }
  return false;
}

function matchesCategory(agent: AgentPayload, category: string | undefined): boolean {
  if (!category) return true;
  return formatCategoryLabel(agent.category_path) === category;
}

function matchesStatus(
  signals: AgentFleetSignals | null,
  status: AgentsFleetSearch["status"],
  sessionsAvailable: boolean
): boolean {
  if (!status) return true;
  if (!sessionsAvailable || signals === null) return true;
  return signals.status === status;
}

export function projectAgentFleetRows(input: {
  agents: readonly AgentPayload[];
  sessions: readonly SessionPayload[] | null;
  search: AgentsFleetSearch;
}): AgentFleetRowModel[] {
  const sessionsAvailable = input.sessions !== null;
  const byAgent = sessionsAvailable
    ? indexSessionsByAgent(input.sessions ?? [])
    : new Map<string, SessionPayload[]>();
  const sorted = sortAgentsByNameStable(input.agents);
  const rows: AgentFleetRowModel[] = [];

  for (const agent of sorted) {
    if (!matchesQuery(agent, input.search.q ?? "")) continue;
    if (!matchesCategory(agent, input.search.category)) continue;

    const agentSessions = byAgent.get(agent.name) ?? [];
    const signals = sessionsAvailable ? deriveAgentFleetSignals(agentSessions) : null;
    if (!matchesStatus(signals, input.search.status, sessionsAvailable)) continue;

    rows.push({
      agent,
      signals,
      meta: formatAgentFleetMeta(agent),
      cardMeta: formatAgentFleetCardMeta(agent),
      ariaLabel: formatAgentFleetAriaLabel(agent, signals, sessionsAvailable),
      hasDiagnostics: Array.isArray(agent.diagnostics) && agent.diagnostics.length > 0,
      sessionsAvailable,
    });
  }

  return rows;
}
