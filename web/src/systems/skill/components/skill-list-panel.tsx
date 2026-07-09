import { AlertCircle, Wrench } from "lucide-react";
import { Link } from "@tanstack/react-router";
import { useMemo } from "react";

import {
  Button,
  CatalogCard,
  Empty,
  ListingRow,
  Pill,
  Spinner,
  Switch,
  type ListingViewMode,
} from "@agh/ui";

import {
  filterInstalledSkills,
  type SkillEnabledFilter,
  type SkillFilterState,
  type SkillSourceFilter,
} from "../lib/skill-list-filters";
import { skillSourceLabel, skillSourceTone } from "../lib/skill-formatters";
import type { SkillPayload } from "../types";

export interface SkillListPanelProps {
  skills: SkillPayload[];
  searchQuery: string;
  view: ListingViewMode;
  sourceFilter: SkillSourceFilter | null;
  enabledFilter: SkillEnabledFilter | null;
  onClearFilters: () => void;
  onDisable: (name: string) => void;
  onEnable: (name: string) => void;
  isActionPending?: boolean;
  isLoading?: boolean;
  errorMessage?: string | null;
}

interface SkillRowProps {
  skill: SkillPayload;
  onDisable: (name: string) => void;
  onEnable: (name: string) => void;
  isActionPending: boolean;
}

function SkillEnabledSwitch({ skill, onDisable, onEnable, isActionPending }: SkillRowProps) {
  return (
    <Switch
      aria-label={skill.enabled ? `Disable ${skill.name}` : `Enable ${skill.name}`}
      checked={skill.enabled}
      data-testid={`skill-enabled-switch-${skill.name}`}
      disabled={isActionPending}
      onCheckedChange={checked => {
        if (checked) {
          onEnable(skill.name);
          return;
        }
        onDisable(skill.name);
      }}
      size="sm"
    />
  );
}

function SkillListingRow({ skill, onDisable, onEnable, isActionPending }: SkillRowProps) {
  const source = skill.provenance?.precedence_tier ?? skill.source;
  return (
    <ListingRow data-skill={skill.name} data-testid={`skill-item-${skill.name}`}>
      <ListingRow.Link
        render={
          <Link
            aria-label={`Open ${skill.name}`}
            params={{ name: skill.name }}
            to="/skills/$name"
          />
        }
      >
        <ListingRow.Icon>
          <Wrench aria-hidden="true" className="size-4" />
        </ListingRow.Icon>
        <ListingRow.Main>
          <ListingRow.Name>
            <ListingRow.Title>{skill.name}</ListingRow.Title>
            <Pill
              data-testid={`skill-tier-badge-${skill.name}`}
              size="xs"
              tone={skillSourceTone(source)}
            >
              {skillSourceLabel(source)}
            </Pill>
            {skill.version ? <ListingRow.Slug>v{skill.version}</ListingRow.Slug> : null}
          </ListingRow.Name>
          {skill.description ? (
            <ListingRow.Description>{skill.description}</ListingRow.Description>
          ) : null}
          {skill.dir ? (
            <ListingRow.Meta>
              <span className="font-mono text-badge text-subtle">{skill.dir}</span>
            </ListingRow.Meta>
          ) : null}
        </ListingRow.Main>
      </ListingRow.Link>
      <ListingRow.Trail>
        <SkillEnabledSwitch
          isActionPending={isActionPending}
          onDisable={onDisable}
          onEnable={onEnable}
          skill={skill}
        />
      </ListingRow.Trail>
    </ListingRow>
  );
}

function SkillCatalogCard({ skill, onDisable, onEnable, isActionPending }: SkillRowProps) {
  const source = skill.provenance?.precedence_tier ?? skill.source;
  return (
    <CatalogCard actionable data-skill={skill.name} data-testid={`skill-card-${skill.name}`}>
      <Link
        aria-label={`Open ${skill.name}`}
        className="flex min-w-0 flex-col gap-3"
        params={{ name: skill.name }}
        to="/skills/$name"
      >
        <div className="flex items-start gap-3">
          <CatalogCard.Logo>
            <Wrench className="size-4" />
          </CatalogCard.Logo>
          <div className="flex min-w-0 flex-1 flex-col gap-1">
            <CatalogCard.Title>{skill.name}</CatalogCard.Title>
            <CatalogCard.Meta>
              {skill.version ? <span>{`v${skill.version}`}</span> : null}
              <span>{skillSourceLabel(source)}</span>
            </CatalogCard.Meta>
          </div>
        </div>
        {skill.description ? (
          <CatalogCard.Description>{skill.description}</CatalogCard.Description>
        ) : null}
      </Link>
      <CatalogCard.Actions className="justify-between">
        <Pill mono size="sm" tone={skillSourceTone(source)}>
          {skillSourceLabel(source)}
        </Pill>
        <SkillEnabledSwitch
          isActionPending={isActionPending}
          onDisable={onDisable}
          onEnable={onEnable}
          skill={skill}
        />
      </CatalogCard.Actions>
    </CatalogCard>
  );
}

function SkillListPanel({
  skills,
  searchQuery,
  view,
  sourceFilter,
  enabledFilter,
  onClearFilters,
  onDisable,
  onEnable,
  isActionPending = false,
  isLoading = false,
  errorMessage = null,
}: SkillListPanelProps) {
  const filterState: SkillFilterState = useMemo(
    () => ({ source: sourceFilter, enabled: enabledFilter }),
    [enabledFilter, sourceFilter]
  );
  const filtered = useMemo(
    () => filterInstalledSkills(skills, searchQuery, filterState),
    [filterState, searchQuery, skills]
  );
  const hasActiveFilters =
    searchQuery.trim() !== "" || sourceFilter !== null || enabledFilter !== null;
  const isEmpty = filtered.length === 0;

  if (isLoading && isEmpty) {
    return (
      <div
        className="flex min-h-60 items-center justify-center px-6 py-10"
        data-testid="skill-list-loading"
      >
        <Spinner aria-hidden="true" className="size-5 text-subtle" />
      </div>
    );
  }

  if (errorMessage && isEmpty) {
    return (
      <div className="flex min-h-60 items-center justify-center p-4" data-testid="skill-list-error">
        <Empty
          className="max-w-sm"
          description={errorMessage}
          icon={AlertCircle}
          title="Unable to load skills"
        />
      </div>
    );
  }

  if (isEmpty) {
    return (
      <div className="flex min-h-60 items-center justify-center p-4" data-testid="skill-list-empty">
        <Empty
          action={
            hasActiveFilters ? (
              <Button
                data-testid="skill-list-clear-filters"
                onClick={onClearFilters}
                size="sm"
                type="button"
                variant="ghost"
              >
                Clear filters
              </Button>
            ) : undefined
          }
          className="max-w-sm"
          description={
            hasActiveFilters
              ? "Try clearing search or filters."
              : "No skills are installed in this workspace yet."
          }
          icon={Wrench}
          title={hasActiveFilters ? "No skills match" : "No skills yet"}
        />
      </div>
    );
  }

  if (view === "cards") {
    return (
      <div
        className="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3"
        data-testid="skill-list-card-grid"
      >
        {filtered.map(skill => (
          <SkillCatalogCard
            isActionPending={isActionPending}
            key={skill.name}
            onDisable={onDisable}
            onEnable={onEnable}
            skill={skill}
          />
        ))}
      </div>
    );
  }

  return (
    <div
      className="overflow-hidden rounded-lg border border-line bg-canvas-soft"
      data-testid="skill-list-rows"
    >
      {filtered.map(skill => (
        <SkillListingRow
          isActionPending={isActionPending}
          key={skill.name}
          onDisable={onDisable}
          onEnable={onEnable}
          skill={skill}
        />
      ))}
    </div>
  );
}

export { SkillListPanel };
export type { SkillFilterState };
