import type { MonoBadgeTone } from "@agh/ui";

import type { KnowledgeMemoryItem, KnowledgeScope, MemoryType } from "@/systems/knowledge/types";

const SCOPE_ORDER: Record<KnowledgeScope, number> = {
  global: 0,
  workspace: 1,
};

export function deriveScopeFromFilename(filename: string): KnowledgeScope {
  if (filename.startsWith("workspace/") || filename.startsWith("ws/")) {
    return "workspace";
  }
  return "global";
}

export function resolveKnowledgeScope(memory: Pick<KnowledgeMemoryItem, "filename" | "scope">) {
  return memory.scope ?? deriveScopeFromFilename(memory.filename);
}

export function knowledgeMemoryKey(
  memory: Pick<KnowledgeMemoryItem, "filename" | "scope" | "key">
) {
  return memory.key ?? `${resolveKnowledgeScope(memory)}:${memory.filename}`;
}

export function compareKnowledgeScope(left: KnowledgeScope, right: KnowledgeScope): number {
  return (SCOPE_ORDER[left] ?? 99) - (SCOPE_ORDER[right] ?? 99);
}

export function knowledgeScopeLabel(scope: KnowledgeScope): string {
  return scope === "workspace" ? "WORKSPACE" : "GLOBAL";
}

export function knowledgeScopeShortLabel(scope: KnowledgeScope): string {
  return scope === "workspace" ? "WS" : "GLOBAL";
}

const TYPE_TONE: Record<string, MonoBadgeTone> = {
  user: "accent",
  feedback: "accent",
  project: "success",
  reference: "info",
};

export function memoryTypeTone(type: MemoryType): MonoBadgeTone {
  return TYPE_TONE[type] ?? "accent";
}

export function memoryScopeTone(scope: KnowledgeScope): MonoBadgeTone {
  return scope === "workspace" ? "info" : "neutral";
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
