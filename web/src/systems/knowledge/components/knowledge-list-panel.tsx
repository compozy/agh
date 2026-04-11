import { Search } from "lucide-react";
import { useMemo } from "react";

import { cn } from "@/lib/utils";

import type { MemoryHeader, MemoryScope } from "../types";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface MemoryGroup {
  scope: MemoryScope;
  label: string;
  memories: MemoryHeader[];
}

interface KnowledgeListPanelProps {
  memories: MemoryHeader[];
  selectedFilename: string | null;
  onSelectMemory: (filename: string) => void;
  searchQuery: string;
  onSearchChange: (query: string) => void;
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

const SCOPE_ORDER: Record<string, number> = {
  global: 0,
  workspace: 1,
};

function groupMemoriesByScope(memories: MemoryHeader[]): MemoryGroup[] {
  const groups: Record<string, MemoryHeader[]> = {};
  for (const mem of memories) {
    // Derive scope from filename prefix: global memories start with "global/" or similar
    const scope = deriveScope(mem);
    if (!groups[scope]) groups[scope] = [];
    groups[scope].push(mem);
  }

  return Object.entries(groups)
    .sort(([a], [b]) => (SCOPE_ORDER[a] ?? 99) - (SCOPE_ORDER[b] ?? 99))
    .map(([scope, items]) => ({
      scope: scope as MemoryScope,
      label: scope.toUpperCase(),
      memories: items.sort((a, b) => a.name.localeCompare(b.name)),
    }));
}

function deriveScope(mem: MemoryHeader): string {
  // If filename contains path separator indicating scope, use it;
  // otherwise fall back to "global"
  if (mem.filename.startsWith("workspace/") || mem.filename.startsWith("ws/")) {
    return "workspace";
  }
  return "global";
}

function formatDate(dateStr: string): string {
  try {
    const date = new Date(dateStr);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffH = Math.floor(diffMs / (1000 * 60 * 60));
    if (diffH < 1) return "just now";
    if (diffH < 24) return `${diffH}h ago`;
    const diffD = Math.floor(diffH / 24);
    if (diffD < 7) return `${diffD}d ago`;
    return date.toLocaleDateString("en-US", { month: "short", day: "numeric" });
  } catch {
    return dateStr;
  }
}

// ---------------------------------------------------------------------------
// Type Badge
// ---------------------------------------------------------------------------

const TYPE_COLORS: Record<string, { bg: string; text: string }> = {
  user: { bg: "bg-[#e8572a26]", text: "text-[#e8572a]" },
  feedback: { bg: "bg-[#e8572a26]", text: "text-[#e8572a]" },
  project: { bg: "bg-[#30d15826]", text: "text-[#30d158]" },
  reference: { bg: "bg-[#bf5af226]", text: "text-[#bf5af2]" },
};

const SCOPE_COLORS: Record<string, { bg: string; text: string }> = {
  global: { bg: "bg-[#63636626]", text: "text-[#636366]" },
  workspace: { bg: "bg-[#63636626]", text: "text-[#636366]" },
};

function TypeBadge({ type }: { type: string }) {
  const colors = TYPE_COLORS[type] ?? TYPE_COLORS.user;
  return (
    <span
      className={cn(
        "inline-flex h-[18px] items-center rounded px-1.5 font-mono text-[9px] font-semibold uppercase tracking-[0.08em]",
        colors.bg,
        colors.text
      )}
      data-testid={`type-badge-${type}`}
    >
      {type}
    </span>
  );
}

function ScopeBadge({ scope }: { scope: string }) {
  const colors = SCOPE_COLORS[scope] ?? SCOPE_COLORS.global;
  return (
    <span
      className={cn(
        "inline-flex h-[18px] items-center rounded px-1.5 font-mono text-[9px] font-semibold uppercase tracking-[0.08em]",
        colors.bg,
        colors.text
      )}
      data-testid={`scope-badge-${scope}`}
    >
      {scope === "workspace" ? "WS" : scope}
    </span>
  );
}

// ---------------------------------------------------------------------------
// Knowledge List Item
// ---------------------------------------------------------------------------

function KnowledgeListItem({
  memory,
  isSelected,
  onSelect,
}: {
  memory: MemoryHeader;
  isSelected: boolean;
  onSelect: () => void;
}) {
  const scope = deriveScope(memory);

  return (
    <button
      onClick={onSelect}
      className={cn(
        "relative flex w-full flex-col gap-1 border-b border-[color:rgba(58,58,60,0.45)] px-3 py-2.5 text-left transition-colors",
        "hover:bg-[color:var(--color-surface)]",
        isSelected && "bg-[color:var(--color-surface)]"
      )}
      data-testid={`memory-item-${memory.filename}`}
    >
      {isSelected && (
        <span
          className="absolute left-0 top-1 bottom-1 w-[3px] rounded-r bg-[#E8572A]"
          data-testid="memory-active-indicator"
        />
      )}
      {/* Title row */}
      <div className="flex items-center gap-2">
        <span className="flex-1 truncate text-sm font-medium text-[color:var(--color-text-primary)]">
          {memory.name}
        </span>
        <span className="shrink-0 text-[10px] text-[color:var(--color-text-tertiary)]">
          {formatDate(memory.mod_time)}
        </span>
      </div>
      {/* Description */}
      {memory.description && (
        <span className="truncate text-xs text-[color:var(--color-text-secondary)]">
          {memory.description}
        </span>
      )}
      {/* Badges */}
      <div className="flex items-center gap-1.5">
        <TypeBadge type={memory.type} />
        <ScopeBadge scope={scope} />
      </div>
    </button>
  );
}

// ---------------------------------------------------------------------------
// Knowledge List Panel
// ---------------------------------------------------------------------------

function KnowledgeListPanel({
  memories,
  selectedFilename,
  onSelectMemory,
  searchQuery,
  onSearchChange,
}: KnowledgeListPanelProps) {
  const filtered = useMemo(() => {
    if (!searchQuery) return memories;
    const q = searchQuery.toLowerCase();
    return memories.filter(
      m =>
        m.name.toLowerCase().includes(q) ||
        (m.description ?? "").toLowerCase().includes(q) ||
        m.type.toLowerCase().includes(q)
    );
  }, [memories, searchQuery]);

  const groups = useMemo(() => groupMemoriesByScope(filtered), [filtered]);

  return (
    <div
      className="flex w-[280px] shrink-0 flex-col border-r border-[color:var(--color-divider)] bg-[color:var(--color-surface-panel)]"
      data-testid="knowledge-list-panel"
    >
      {/* Search input */}
      <div className="border-b border-[color:var(--color-divider)] p-3">
        <div className="flex items-center gap-2 rounded-lg bg-[color:var(--color-surface)] px-3 py-2">
          <Search className="size-3.5 shrink-0 text-[color:var(--color-text-tertiary)]" />
          <input
            type="text"
            placeholder="Search knowledge..."
            value={searchQuery}
            onChange={e => onSearchChange(e.target.value)}
            className="w-full bg-transparent text-sm text-[color:var(--color-text-primary)] placeholder:text-[color:var(--color-text-tertiary)] outline-none"
            data-testid="knowledge-search-input"
          />
        </div>
      </div>

      {/* Grouped memory list */}
      <div className="flex-1 overflow-y-auto">
        {groups.length === 0 && (
          <div
            className="px-3 py-6 text-center text-sm text-[color:var(--color-text-tertiary)]"
            data-testid="knowledge-list-empty"
          >
            No knowledge items found
          </div>
        )}
        {groups.map(group => (
          <div key={group.scope} data-testid={`knowledge-group-${group.scope}`}>
            <div className="flex items-center justify-between px-3 pb-1 pt-3">
              <span className="font-mono text-[10px] font-semibold uppercase tracking-[0.08em] text-[color:var(--color-text-tertiary)]">
                {group.label}
              </span>
              <span className="font-mono text-[10px] text-[color:var(--color-text-tertiary)]">
                {group.memories.length}
              </span>
            </div>
            {group.memories.map(memory => (
              <KnowledgeListItem
                key={memory.filename}
                memory={memory}
                isSelected={memory.filename === selectedFilename}
                onSelect={() => onSelectMemory(memory.filename)}
              />
            ))}
          </div>
        ))}
      </div>
    </div>
  );
}

export { KnowledgeListPanel };
export type { KnowledgeListPanelProps };
