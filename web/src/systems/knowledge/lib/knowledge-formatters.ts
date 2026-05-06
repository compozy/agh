import type {
  KnowledgeAgentTier,
  KnowledgeMemoryItem,
  KnowledgeScope,
  MemoryDecisionOp,
  MemoryDecisionSource,
  MemoryType,
} from "@/systems/knowledge/types";

const SCOPE_ORDER: Record<KnowledgeScope, number> = {
  global: 0,
  workspace: 1,
  agent: 2,
};

export function knowledgeMemoryKey(
  memory: Pick<KnowledgeMemoryItem, "filename" | "scope" | "key">
) {
  return memory.key ?? `${memory.scope}:${memory.filename}`;
}

export function compareKnowledgeScope(left: KnowledgeScope, right: KnowledgeScope): number {
  return (SCOPE_ORDER[left] ?? 99) - (SCOPE_ORDER[right] ?? 99);
}

export function knowledgeScopeLabel(scope: KnowledgeScope): string {
  if (scope === "workspace") return "WORKSPACE";
  if (scope === "agent") return "AGENT";
  return "GLOBAL";
}

export function knowledgeScopeShortLabel(scope: KnowledgeScope): string {
  if (scope === "workspace") return "WS";
  if (scope === "agent") return "AGENT";
  return "GLOBAL";
}

export function knowledgeAgentTierLabel(tier: KnowledgeAgentTier): string {
  return tier === "global" ? "AGENT-GLOBAL" : "AGENT-WORKSPACE";
}

export function knowledgeAgentTierShortLabel(tier: KnowledgeAgentTier): string {
  return tier === "global" ? "AG-GLOBAL" : "AG-WS";
}

export type KnowledgeTone = MemoryType | KnowledgeScope;

export function memoryTypeTone(type: MemoryType): KnowledgeTone {
  return type;
}

export function memoryScopeTone(scope: KnowledgeScope): KnowledgeTone {
  return scope;
}

const DECISION_OP_LABEL: Record<MemoryDecisionOp, string> = {
  noop: "NOOP",
  add: "ADD",
  update: "UPDATE",
  delete: "DELETE",
  reject: "REJECT",
};

export function decisionOpLabel(op: MemoryDecisionOp): string {
  return DECISION_OP_LABEL[op] ?? op.toUpperCase();
}

export function decisionSourceLabel(source: MemoryDecisionSource): string {
  return source === "rule" ? "RULE" : "LLM";
}

function safeDate(value: string): Date | null {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return null;
  return date;
}

export function formatKnowledgeRelativeTime(value: string): string {
  const date = safeDate(value);
  if (!date) return value;
  const diffMs = Date.now() - date.getTime();
  const diffH = Math.floor(diffMs / (1000 * 60 * 60));
  if (diffH < 1) return "just now";
  if (diffH < 24) return `${diffH}h ago`;
  const diffD = Math.floor(diffH / 24);
  if (diffD < 7) return `${diffD}d ago`;
  return date.toLocaleDateString("en-US", { month: "short", day: "numeric" });
}

export function formatKnowledgeDateTime(value: string): string {
  const date = safeDate(value);
  if (!date) return value;
  return date.toLocaleString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}
