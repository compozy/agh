import { AlertCircle, Loader2, Wrench } from "lucide-react";
import { useMemo } from "react";

import { Empty, MonoBadge, SearchInput, StatusDot } from "@agh/ui";
import { cn } from "@/lib/utils";

import {
  compareSkillSource,
  filterSkillsByQuery,
  skillSourceLabel,
  skillStatusTone,
} from "../lib/skill-formatters";
import type { SkillPayload } from "../types";

interface SkillGroup {
  source: string;
  label: string;
  skills: SkillPayload[];
}

interface SkillListPanelProps {
  skills: SkillPayload[];
  selectedSkillName: string | null;
  onSelectSkill: (name: string) => void;
  searchQuery: string;
  onSearchChange: (query: string) => void;
  isLoading?: boolean;
  errorMessage?: string | null;
}

function groupSkillsBySource(skills: SkillPayload[]): SkillGroup[] {
  const buckets = new Map<string, SkillPayload[]>();
  for (const skill of skills) {
    const list = buckets.get(skill.source);
    if (list) {
      list.push(skill);
    } else {
      buckets.set(skill.source, [skill]);
    }
  }
  return Array.from(buckets.entries())
    .sort(([left], [right]) => compareSkillSource(left, right))
    .map(([source, items]) => ({
      source,
      label: skillSourceLabel(source),
      skills: items.slice().sort((a, b) => a.name.localeCompare(b.name)),
    }));
}

interface SkillListItemProps {
  skill: SkillPayload;
  isSelected: boolean;
  onSelect: () => void;
}

function SkillListItem({ skill, isSelected, onSelect }: SkillListItemProps) {
  return (
    <button
      aria-pressed={isSelected}
      className={cn(
        "relative flex w-full flex-col gap-1.5 border-b border-[color:var(--color-divider)] px-4 py-3 text-left transition-colors",
        "hover:bg-[color:var(--color-hover)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[color:var(--color-accent)]",
        isSelected && "bg-[color:var(--color-surface)]"
      )}
      data-state={isSelected ? "selected" : undefined}
      data-testid={`skill-item-${skill.name}`}
      onClick={onSelect}
      type="button"
    >
      {isSelected ? (
        <span
          aria-hidden="true"
          className="absolute left-0 top-2 bottom-2 w-[3px] rounded-r bg-[color:var(--color-accent)]"
          data-testid="skill-active-indicator"
        />
      ) : null}
      <div className="flex items-center gap-2">
        <StatusDot
          data-testid={`skill-status-dot-${skill.name}`}
          tone={skillStatusTone(skill.enabled)}
        />
        <span className="min-w-0 flex-1 truncate text-[13px] font-medium text-[color:var(--color-text-primary)]">
          {skill.name}
        </span>
        {skill.version ? (
          <span className="shrink-0 font-mono text-[10px] uppercase tracking-[0.08em] text-[color:var(--color-text-tertiary)]">
            v{skill.version}
          </span>
        ) : null}
      </div>
      {skill.description ? (
        <span className="truncate text-[12px] text-[color:var(--color-text-secondary)]">
          {skill.description}
        </span>
      ) : null}
    </button>
  );
}

function SkillListPanel({
  skills,
  selectedSkillName,
  onSelectSkill,
  searchQuery,
  onSearchChange,
  isLoading = false,
  errorMessage = null,
}: SkillListPanelProps) {
  const filtered = useMemo(() => filterSkillsByQuery(skills, searchQuery), [skills, searchQuery]);
  const groups = useMemo(() => groupSkillsBySource(filtered), [filtered]);
  const isEmpty = filtered.length === 0;

  return (
    <aside className="flex min-h-0 flex-1 flex-col" data-testid="skill-list-panel">
      <div className="border-b border-[color:var(--color-divider)] p-3">
        <SearchInput
          aria-label="Search installed skills"
          data-testid="skill-search-input"
          onChange={onSearchChange}
          placeholder="Filter skills…"
          value={searchQuery}
        />
      </div>

      <div className="min-h-0 flex-1 overflow-y-auto">
        {isLoading && isEmpty ? (
          <div
            className="flex min-h-full items-center justify-center px-6 py-10"
            data-testid="skill-list-loading"
          >
            <Loader2
              aria-hidden="true"
              className="size-5 animate-spin text-[color:var(--color-text-tertiary)]"
            />
          </div>
        ) : errorMessage && isEmpty ? (
          <div
            className="flex min-h-full items-center justify-center p-4"
            data-testid="skill-list-error"
          >
            <Empty
              className="max-w-sm"
              description={errorMessage}
              icon={AlertCircle}
              title="Unable to load skills"
            />
          </div>
        ) : isEmpty ? (
          <div
            className="flex min-h-full items-center justify-center p-4"
            data-testid="skill-list-empty"
          >
            <Empty
              className="max-w-sm"
              description={
                searchQuery.trim() !== "" ? "Try a different search term." : "No skills found"
              }
              icon={Wrench}
              title="No skills found"
            />
          </div>
        ) : (
          <div data-testid="skill-list-groups">
            {groups.map(group => (
              <div data-testid={`skill-group-${group.source}`} key={group.source}>
                <div
                  className="flex items-center justify-between gap-2 border-b border-[color:var(--color-divider)] bg-[color:var(--color-surface-panel)] px-4 py-2"
                  data-testid={`skill-group-header-${group.source}`}
                >
                  <span className="font-mono text-[10px] uppercase tracking-[0.14em] text-[color:var(--color-text-label)]">
                    {group.label}
                  </span>
                  <MonoBadge>{group.skills.length}</MonoBadge>
                </div>
                {group.skills.map(skill => (
                  <SkillListItem
                    isSelected={skill.name === selectedSkillName}
                    key={skill.name}
                    onSelect={() => onSelectSkill(skill.name)}
                    skill={skill}
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

export { SkillListPanel };
export type { SkillListPanelProps };
