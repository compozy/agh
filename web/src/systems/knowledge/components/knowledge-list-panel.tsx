import { AlertCircle, BookOpen, Loader2 } from "lucide-react";
import { useMemo } from "react";

import { Empty, MonoBadge, SearchInput } from "@agh/ui";
import { cn } from "@/lib/utils";

import {
  formatKnowledgeRelativeTime,
  knowledgeMemoryKey,
  memoryScopeTone,
  resolveKnowledgeScope,
  memoryTypeTone,
} from "../lib/knowledge-formatters";
import { filterKnowledgeMemories, groupKnowledgeMemoriesByScope } from "../lib/knowledge-list";
import type { KnowledgeMemoryItem } from "../types";

interface KnowledgeListPanelProps {
  memories: KnowledgeMemoryItem[];
  selectedMemoryKey: string | null;
  onSelectMemory: (memoryKey: string) => void;
  searchQuery: string;
  onSearchChange: (query: string) => void;
  isLoading?: boolean;
  errorMessage?: string | null;
}

interface KnowledgeListItemProps {
  memory: KnowledgeMemoryItem;
  isSelected: boolean;
  onSelect: () => void;
}

function KnowledgeListItem({ memory, isSelected, onSelect }: KnowledgeListItemProps) {
  const scope = resolveKnowledgeScope(memory);
  return (
    <button
      aria-pressed={isSelected}
      className={cn(
        "relative flex w-full flex-col gap-1.5 border-b border-[color:var(--color-divider)] px-4 py-3 text-left transition-colors",
        "hover:bg-[color:var(--color-hover)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[color:var(--color-accent)]",
        isSelected && "bg-[color:var(--color-surface)]"
      )}
      data-state={isSelected ? "selected" : undefined}
      data-testid={`memory-item-${memory.key ?? memory.filename}`}
      onClick={onSelect}
      type="button"
    >
      {isSelected ? (
        <span
          aria-hidden="true"
          className="absolute left-0 top-2 bottom-2 w-[3px] rounded-r bg-[color:var(--color-accent)]"
          data-testid="memory-active-indicator"
        />
      ) : null}
      <div className="flex items-center gap-2">
        <span className="min-w-0 flex-1 truncate text-[13px] font-medium text-[color:var(--color-text-primary)]">
          {memory.name}
        </span>
        <span className="shrink-0 font-mono text-[10px] uppercase tracking-[0.08em] text-[color:var(--color-text-tertiary)]">
          {formatKnowledgeRelativeTime(memory.mod_time)}
        </span>
      </div>
      {memory.description ? (
        <span className="truncate text-[12px] text-[color:var(--color-text-secondary)]">
          {memory.description}
        </span>
      ) : null}
      <div className="flex flex-wrap items-center gap-1.5">
        <MonoBadge data-testid={`type-badge-${memory.type}`} tone={memoryTypeTone(memory.type)}>
          {memory.type}
        </MonoBadge>
        <MonoBadge data-testid={`scope-badge-${scope}`} tone={memoryScopeTone(scope)}>
          {scope === "workspace" ? "WS" : "GLOBAL"}
        </MonoBadge>
      </div>
    </button>
  );
}

function KnowledgeListPanel({
  memories,
  selectedMemoryKey,
  onSelectMemory,
  searchQuery,
  onSearchChange,
  isLoading = false,
  errorMessage = null,
}: KnowledgeListPanelProps) {
  const filtered = useMemo(
    () => filterKnowledgeMemories(memories, searchQuery),
    [memories, searchQuery]
  );
  const groups = useMemo(() => groupKnowledgeMemoriesByScope(filtered), [filtered]);
  const isEmpty = filtered.length === 0;

  return (
    <aside className="flex min-h-0 flex-1 flex-col" data-testid="knowledge-list-panel">
      <div className="border-b border-[color:var(--color-divider)] p-3">
        <SearchInput
          aria-label="Search knowledge"
          data-testid="knowledge-search-input"
          onChange={onSearchChange}
          placeholder="Filter knowledge…"
          value={searchQuery}
        />
      </div>

      <div className="min-h-0 flex-1 overflow-y-auto">
        {isLoading && isEmpty ? (
          <div
            className="flex min-h-full items-center justify-center px-6 py-10"
            data-testid="knowledge-list-loading"
          >
            <Loader2
              aria-hidden="true"
              className="size-5 animate-spin text-[color:var(--color-text-tertiary)]"
            />
          </div>
        ) : errorMessage && isEmpty ? (
          <div
            className="flex min-h-full items-center justify-center p-4"
            data-testid="knowledge-list-error"
          >
            <Empty
              className="max-w-sm"
              description={errorMessage}
              icon={AlertCircle}
              title="Unable to load knowledge"
            />
          </div>
        ) : isEmpty ? (
          <div
            className="flex min-h-full items-center justify-center p-4"
            data-testid="knowledge-list-empty"
          >
            <Empty
              className="max-w-sm"
              description={
                searchQuery.trim() !== ""
                  ? "Try a different search term or adjust the scope filter."
                  : "No knowledge items found"
              }
              icon={BookOpen}
              title="No knowledge items found"
            />
          </div>
        ) : (
          <div data-testid="knowledge-list-groups">
            {groups.map(group => (
              <div data-testid={`knowledge-group-${group.scope}`} key={group.scope}>
                <div
                  className="flex items-center justify-between gap-2 border-b border-[color:var(--color-divider)] bg-[color:var(--color-surface-panel)] px-4 py-2"
                  data-testid={`knowledge-group-header-${group.scope}`}
                >
                  <span className="font-mono text-[10px] uppercase tracking-[0.14em] text-[color:var(--color-text-label)]">
                    {group.label}
                  </span>
                  <MonoBadge>{group.memories.length}</MonoBadge>
                </div>
                {group.memories.map(memory => (
                  <KnowledgeListItem
                    isSelected={knowledgeMemoryKey(memory) === selectedMemoryKey}
                    key={knowledgeMemoryKey(memory)}
                    memory={memory}
                    onSelect={() => onSelectMemory(knowledgeMemoryKey(memory))}
                  />
                ))}
              </div>
            ))}
          </div>
        )}
      </div>
    </aside>
  );
}

export { KnowledgeListPanel };
export type { KnowledgeListPanelProps };
