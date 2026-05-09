import { AlertCircle, Wrench } from "lucide-react";
import { useMemo } from "react";

import {
  Empty,
  Item,
  ItemContent,
  ItemDescription,
  ItemHeader,
  ItemMedia,
  ItemSelectionIndicator,
  ItemTitle,
  ListGroup,
  Pill,
  SearchInput,
  Spinner,
} from "@agh/ui";

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
    <Item
      as="button"
      className="rounded-none border-x-0 border-t-0 border-b border-(--color-divider) px-4 py-3"
      data-state={isSelected ? "selected" : undefined}
      data-testid={`skill-item-${skill.name}`}
      onClick={onSelect}
      selectable
      selected={isSelected}
    >
      {isSelected ? <ItemSelectionIndicator data-testid="skill-active-indicator" /> : null}
      <ItemHeader>
        <ItemMedia>
          <Pill.Dot
            data-testid={`skill-status-dot-${skill.name}`}
            tone={skillStatusTone(skill.enabled)}
          />
        </ItemMedia>
        <ItemContent>
          <ItemTitle className="w-full">
            <span className="min-w-0 flex-1 truncate">{skill.name}</span>
            {skill.version ? (
              <span className="shrink-0 font-mono text-badge uppercase tracking-badge text-(--color-text-tertiary)">
                v{skill.version}
              </span>
            ) : null}
          </ItemTitle>
        </ItemContent>
      </ItemHeader>
      {skill.description ? (
        <ItemDescription className="basis-full truncate text-xs">
          {skill.description}
        </ItemDescription>
      ) : null}
    </Item>
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
      <div className="border-b border-(--color-divider) p-3">
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
            <Spinner aria-hidden="true" className="size-5 text-(--color-text-tertiary)" />
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
              <ListGroup
                count={group.skills.length}
                data-testid={`skill-group-${group.source}`}
                headerProps={{ "data-testid": `skill-group-header-${group.source}` }}
                key={group.source}
                label={group.label}
              >
                {group.skills.map(skill => (
                  <SkillListItem
                    isSelected={skill.name === selectedSkillName}
                    key={skill.name}
                    onSelect={() => onSelectSkill(skill.name)}
                    skill={skill}
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

export { SkillListPanel };
export type { SkillListPanelProps };
