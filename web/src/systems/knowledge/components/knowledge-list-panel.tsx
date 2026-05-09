import { AlertCircle, BookOpen } from "lucide-react";

import {
  Empty,
  Item,
  ItemDescription,
  ItemFooter,
  ItemHeader,
  ItemTitle,
  ListGroup,
  Pill,
  SearchInput,
  Spinner,
} from "@agh/ui";

import {
  formatKnowledgeRelativeTime,
  knowledgeAgentTierShortLabel,
  knowledgeMemoryKey,
  knowledgeScopeShortLabel,
  memoryScopeTone,
  memoryTypeTone,
} from "../lib/knowledge-formatters";
import { groupKnowledgeMemoriesByScope } from "../lib/knowledge-list";
import type { KnowledgeMemoryItem } from "../types";
import { pillToneFromKnowledgeTone } from "./knowledge-pill-tone";

interface KnowledgeListPanelProps {
  memories: KnowledgeMemoryItem[];
  selectedMemoryKey: string | null;
  onSelectMemory: (memoryKey: string) => void;
  searchQuery: string;
  onSearchChange: (query: string) => void;
  isLoading?: boolean;
  errorMessage?: string | null;
  searchMode?: boolean;
  searchInfo?: string | null;
}

interface KnowledgeListItemProps {
  memory: KnowledgeMemoryItem;
  isSelected: boolean;
  onSelect: () => void;
}

function KnowledgeListItem({ memory, isSelected, onSelect }: KnowledgeListItemProps) {
  const memoryKey = knowledgeMemoryKey(memory);
  const scope = memory.scope;
  return (
    <Item
      as="button"
      className="rounded-none border-x-0 border-t-0 border-b border-[color:var(--color-divider)] px-4 py-3"
      data-state={isSelected ? "selected" : undefined}
      data-testid={`memory-item-${memoryKey}`}
      indicator={isSelected ? "rail" : "none"}
      onClick={onSelect}
      selectable
      selected={isSelected}
    >
      <ItemHeader>
        <ItemTitle className="min-w-0 flex-1 text-small-body text-(--color-text-primary)">
          {memory.name}
        </ItemTitle>
        <span className="shrink-0 font-mono text-badge uppercase tracking-badge text-(--color-text-tertiary)">
          {formatKnowledgeRelativeTime(memory.mod_time)}
        </span>
      </ItemHeader>
      {memory.description ? (
        <ItemDescription className="basis-full truncate text-xs text-(--color-text-secondary)">
          {memory.description}
        </ItemDescription>
      ) : null}
      <ItemFooter className="justify-start gap-1.5">
        <Pill
          mono
          data-testid={`type-badge-${memory.type}`}
          tone={pillToneFromKnowledgeTone(memoryTypeTone(memory.type))}
        >
          {memory.type}
        </Pill>
        <Pill
          mono
          data-testid={`scope-badge-${scope}`}
          tone={pillToneFromKnowledgeTone(memoryScopeTone(scope))}
        >
          {knowledgeScopeShortLabel(scope)}
        </Pill>
        {memory.scope === "agent" && memory.agent_tier ? (
          <Pill mono data-testid={`agent-tier-badge-${memory.agent_tier}`} tone="warning">
            {knowledgeAgentTierShortLabel(memory.agent_tier)}
          </Pill>
        ) : null}
        {memory.agent_name ? (
          <Pill mono data-testid="agent-name-badge" tone="neutral">
            {memory.agent_name}
          </Pill>
        ) : null}
        {memory.recall_count > 0 ? (
          <Pill mono data-testid="recall-count-badge" tone="info">
            ↻ {memory.recall_count}
          </Pill>
        ) : null}
        {memory.staleness_banner ? (
          <Pill mono data-testid="staleness-badge" tone="warning">
            STALE
          </Pill>
        ) : null}
        {memory.system_managed ? (
          <Pill mono data-testid="system-managed-badge" tone="neutral">
            SYSTEM
          </Pill>
        ) : null}
      </ItemFooter>
    </Item>
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
  searchMode = false,
  searchInfo = null,
}: KnowledgeListPanelProps) {
  const groups = groupKnowledgeMemoriesByScope(memories);
  const isEmpty = memories.length === 0;

  return (
    <aside className="flex min-h-0 flex-1 flex-col" data-testid="knowledge-list-panel">
      <div className="border-b border-(--color-divider) p-3">
        <SearchInput
          aria-label="Search knowledge"
          data-testid="knowledge-search-input"
          onChange={onSearchChange}
          placeholder={searchMode ? "Recall query..." : "Filter knowledge..."}
          value={searchQuery}
        />
        {searchInfo ? (
          <p
            className="mt-2 font-mono text-badge uppercase tracking-badge text-(--color-text-tertiary)"
            data-testid="knowledge-search-info"
          >
            {searchInfo}
          </p>
        ) : null}
      </div>

      <div className="min-h-0 flex-1 overflow-y-auto">
        {isLoading && isEmpty ? (
          <div
            className="flex min-h-full items-center justify-center px-6 py-10"
            data-testid="knowledge-list-loading"
          >
            <Spinner className="size-5 text-(--color-text-tertiary)" />
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
                searchMode
                  ? "No memories matched this recall query."
                  : searchQuery.trim() !== ""
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
              <ListGroup
                count={group.memories.length}
                data-testid={`knowledge-group-${group.scope}`}
                headerProps={{ "data-testid": `knowledge-group-header-${group.scope}` }}
                key={group.scope}
                label={group.label}
              >
                {group.memories.map(memory => (
                  <KnowledgeListItem
                    isSelected={knowledgeMemoryKey(memory) === selectedMemoryKey}
                    key={knowledgeMemoryKey(memory)}
                    memory={memory}
                    onSelect={() => onSelectMemory(knowledgeMemoryKey(memory))}
                  />
                ))}
              </ListGroup>
            ))}
          </div>
        )}
      </div>
    </aside>
  );
}

export { KnowledgeListPanel };
export type { KnowledgeListPanelProps };
