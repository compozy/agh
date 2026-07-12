import type { AgentCatalogItem, AgentPayload } from "../types";
import { AGENT_CATEGORY_LABEL_SEPARATOR, formatCategoryLabel } from "./agent-category";
import type { AgentFleetStatus } from "./fleet-signals";

const META_SEPARATOR = " · ";
const CATEGORY_ELLIPSIS_LIMIT = 40;

export interface AgentFleetRowModel {
  agent: AgentPayload;
  signals: AgentFleetSessionSignals | null;
  meta: string;
  cardMeta: string;
  ariaLabel: string;
  hasDiagnostics: boolean;
  sessionsAvailable: boolean;
}

export interface AgentFleetSessionSignals {
  status: AgentFleetStatus;
  active: number;
  total: number;
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
  signals: AgentFleetSessionSignals | null,
  sessionsAvailable: boolean
): string {
  if (!sessionsAvailable || signals === null) {
    return `${agent.name}, session status unavailable`;
  }
  const statusLabel = signals.status === "active" ? "Active" : "Idle";
  return `${agent.name}, ${statusLabel}, ${signals.active} of ${signals.total} sessions active`;
}

export function projectAgentFleetRows(input: {
  items: readonly AgentCatalogItem[];
  sessionsAvailable: boolean;
}): AgentFleetRowModel[] {
  return input.items.map(item => {
    const agent = item.agent;
    const signals =
      input.sessionsAvailable && item.sessions
        ? {
            status: item.sessions.active > 0 ? ("active" as const) : ("idle" as const),
            active: item.sessions.active,
            total: item.sessions.total,
          }
        : null;
    return {
      agent,
      signals,
      meta: formatAgentFleetMeta(agent),
      cardMeta: formatAgentFleetCardMeta(agent),
      ariaLabel: formatAgentFleetAriaLabel(agent, signals, input.sessionsAvailable),
      hasDiagnostics: Array.isArray(agent.diagnostics) && agent.diagnostics.length > 0,
      sessionsAvailable: input.sessionsAvailable,
    };
  });
}
