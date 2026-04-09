import { Search } from "lucide-react";
import { useMemo } from "react";

import { cn } from "@/lib/utils";

import type { SkillPayload } from "../types";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

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
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

const SOURCE_ORDER: Record<string, number> = {
  bundled: 0,
  workspace: 1,
  marketplace: 2,
  user: 3,
  additional: 4,
};

function groupSkillsBySource(skills: SkillPayload[]): SkillGroup[] {
  const groups: Record<string, SkillPayload[]> = {};
  for (const skill of skills) {
    const src = skill.source;
    if (!groups[src]) groups[src] = [];
    groups[src].push(skill);
  }

  return Object.entries(groups)
    .sort(([a], [b]) => (SOURCE_ORDER[a] ?? 99) - (SOURCE_ORDER[b] ?? 99))
    .map(([source, items]) => ({
      source,
      label: source.toUpperCase(),
      skills: items.sort((a, b) => a.name.localeCompare(b.name)),
    }));
}

// ---------------------------------------------------------------------------
// Skill List Item
// ---------------------------------------------------------------------------

function SkillListItem({
  skill,
  isSelected,
  onSelect,
}: {
  skill: SkillPayload;
  isSelected: boolean;
  onSelect: () => void;
}) {
  return (
    <button
      onClick={onSelect}
      className={cn(
        "relative flex w-full items-center gap-2 border-b border-[color:var(--color-surface-elevated)] px-3 py-2 text-left transition-colors",
        "hover:bg-[color:var(--color-hover)]",
        isSelected && "bg-[color:var(--color-surface-elevated)]"
      )}
      data-testid={`skill-item-${skill.name}`}
    >
      {isSelected && (
        <span
          className="absolute left-0 top-1 bottom-1 w-[3px] rounded-r bg-[#E8572A]"
          data-testid="skill-active-indicator"
        />
      )}
      <span
        className={cn(
          "size-2 shrink-0 rounded-full",
          skill.enabled
            ? "bg-[color:var(--color-success)]"
            : "bg-[color:var(--color-text-tertiary)]"
        )}
        data-testid={`skill-status-dot-${skill.name}`}
      />
      <span className="flex-1 truncate text-sm font-medium text-[color:var(--color-text-primary)]">
        {skill.name}
      </span>
      {skill.version && (
        <span className="shrink-0 text-xs text-[color:var(--color-text-tertiary)]">
          {skill.version}
        </span>
      )}
    </button>
  );
}

// ---------------------------------------------------------------------------
// Skill List Panel
// ---------------------------------------------------------------------------

function SkillListPanel({
  skills,
  selectedSkillName,
  onSelectSkill,
  searchQuery,
  onSearchChange,
}: SkillListPanelProps) {
  const filtered = useMemo(() => {
    if (!searchQuery) return skills;
    const q = searchQuery.toLowerCase();
    return skills.filter(
      s => s.name.toLowerCase().includes(q) || s.description.toLowerCase().includes(q)
    );
  }, [skills, searchQuery]);

  const groups = useMemo(() => groupSkillsBySource(filtered), [filtered]);

  return (
    <div
      className="flex w-[280px] shrink-0 flex-col border-r border-[color:var(--color-divider)] bg-[color:var(--color-surface)]"
      data-testid="skill-list-panel"
    >
      {/* Search input */}
      <div className="border-b border-[color:var(--color-divider)] p-3">
        <div className="flex items-center gap-2 rounded-lg bg-[color:var(--color-surface-elevated)] px-3 py-2">
          <Search className="size-3.5 shrink-0 text-[color:var(--color-text-tertiary)]" />
          <input
            type="text"
            placeholder="Search skills..."
            value={searchQuery}
            onChange={e => onSearchChange(e.target.value)}
            className="w-full bg-transparent text-sm text-[color:var(--color-text-primary)] placeholder:text-[color:var(--color-text-tertiary)] outline-none"
            data-testid="skill-search-input"
          />
        </div>
      </div>

      {/* Grouped skill list */}
      <div className="flex-1 overflow-y-auto">
        {groups.length === 0 && (
          <div
            className="px-3 py-6 text-center text-sm text-[color:var(--color-text-tertiary)]"
            data-testid="skill-list-empty"
          >
            No skills found
          </div>
        )}
        {groups.map(group => (
          <div key={group.source} data-testid={`skill-group-${group.source}`}>
            <div className="flex items-center justify-between px-3 pb-1 pt-3">
              <span className="font-mono text-[10px] font-semibold uppercase tracking-[0.08em] text-[color:var(--color-text-tertiary)]">
                {group.label}
              </span>
              <span className="font-mono text-[10px] text-[color:var(--color-text-tertiary)]">
                {group.skills.length}
              </span>
            </div>
            {group.skills.map(skill => (
              <SkillListItem
                key={skill.name}
                skill={skill}
                isSelected={skill.name === selectedSkillName}
                onSelect={() => onSelectSkill(skill.name)}
              />
            ))}
          </div>
        ))}
      </div>
    </div>
  );
}

export { SkillListPanel };
export type { SkillListPanelProps };
