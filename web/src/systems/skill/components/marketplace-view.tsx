import { AlertCircle, Download, Wrench } from "lucide-react";
import { useMemo, useState } from "react";

import {
  Alert,
  AlertDescription,
  AlertTitle,
  Button,
  CatalogCard,
  Empty,
  Pill,
  PillGroup,
  SearchInput,
} from "@agh/ui";

import {
  MARKETPLACE_CATEGORIES,
  type MarketplaceCategory,
  deriveSkillAuthor,
  deriveSkillTags,
  filterSkillsByQuery,
  matchesMarketplaceCategory,
} from "../lib/skill-formatters";
import type { SkillPayload } from "../types";

interface MarketplaceViewProps {
  skills: SkillPayload[];
  installedSkillNames: Set<string>;
  onInstall?: (name: string) => void;
  isInstalling: boolean;
  installUnavailableReason?: string;
  searchQuery?: string;
  onSearchChange?: (query: string) => void;
}

interface SkillCatalogItemProps {
  skill: SkillPayload;
  isInstalled: boolean;
  onInstall?: () => void;
  isInstalling: boolean;
}

function SkillCatalogItem({ skill, isInstalled, onInstall, isInstalling }: SkillCatalogItemProps) {
  const author = deriveSkillAuthor(skill);
  const tags = deriveSkillTags(skill);
  const downloads = skill.metadata?.downloads;

  return (
    <CatalogCard data-testid={`marketplace-row-${skill.name}`}>
      <div className="flex items-start gap-3">
        <CatalogCard.Logo>
          <Wrench className="size-4" />
        </CatalogCard.Logo>
        <div className="flex min-w-0 flex-1 flex-col gap-1">
          <CatalogCard.Title>{skill.name}</CatalogCard.Title>
          <CatalogCard.Meta>
            {author ? <span>{`@${author}`}</span> : null}
            {skill.version ? <span>{`v${skill.version}`}</span> : null}
            {downloads !== undefined && downloads !== null ? (
              <span className="inline-flex items-center gap-1">
                <Download aria-hidden="true" className="size-3" />
                {String(downloads)}
              </span>
            ) : null}
          </CatalogCard.Meta>
        </div>
      </div>
      <div className="flex flex-col gap-2">
        <CatalogCard.Description>{skill.description}</CatalogCard.Description>
        {tags.length > 0 ? (
          <div className="flex flex-wrap items-center gap-1.5">
            {tags.map(tag => (
              <Pill
                mono
                data-testid={`marketplace-tag-${skill.name}-${tag}`}
                key={tag}
                tone="neutral"
                uppercase={false}
              >
                {tag}
              </Pill>
            ))}
          </div>
        ) : null}
      </div>
      <CatalogCard.Actions>
        {isInstalled ? (
          <Pill mono data-testid={`installed-pill-${skill.name}`} tone="success">
            INSTALLED
          </Pill>
        ) : onInstall ? (
          <Button
            data-testid={`install-btn-${skill.name}`}
            disabled={isInstalling}
            onClick={() => onInstall()}
            size="sm"
            type="button"
            variant="outline"
          >
            Install
          </Button>
        ) : (
          <div
            className="flex items-center gap-2 text-eyebrow text-(--color-text-secondary)"
            data-testid={`catalog-state-${skill.name}`}
          >
            <Pill mono data-testid={`readonly-pill-${skill.name}`} tone="neutral">
              READ ONLY
            </Pill>
            <span>Metadata only</span>
          </div>
        )}
      </CatalogCard.Actions>
    </CatalogCard>
  );
}

function MarketplaceView({
  skills,
  installedSkillNames,
  onInstall,
  isInstalling,
  installUnavailableReason,
  searchQuery,
  onSearchChange,
}: MarketplaceViewProps) {
  const [localSearch, setLocalSearch] = useState("");
  const [activeCategory, setActiveCategory] = useState<MarketplaceCategory>("ALL");
  const search = searchQuery ?? localSearch;
  const handleSearchChange = onSearchChange ?? setLocalSearch;
  const isBrowseOnly = !onInstall;
  const hasFilters = search.trim() !== "" || activeCategory !== "ALL";

  const filtered = useMemo(() => {
    const byQuery = filterSkillsByQuery(skills, search);
    return byQuery.filter(skill => matchesMarketplaceCategory(skill, activeCategory));
  }, [skills, search, activeCategory]);

  return (
    <div className="flex min-h-0 flex-1 flex-col overflow-hidden" data-testid="marketplace-view">
      <div className="flex flex-col gap-3 border-b border-(--color-divider) px-4 py-3">
        {isBrowseOnly && installUnavailableReason ? (
          <Alert data-testid="marketplace-readonly-notice" variant="warning">
            <AlertCircle aria-hidden="true" className="size-4" />
            <AlertTitle>Installed marketplace metadata only</AlertTitle>
            <AlertDescription>{installUnavailableReason}</AlertDescription>
          </Alert>
        ) : null}
        <SearchInput
          aria-label={
            isBrowseOnly ? "Filter installed marketplace skills" : "Search marketplace skills"
          }
          data-testid="marketplace-search-input"
          onChange={handleSearchChange}
          placeholder={
            isBrowseOnly ? "Filter installed marketplace skills…" : "Search skills on marketplace…"
          }
          value={search}
        />
        <PillGroup
          aria-label="Marketplace category"
          data-testid="marketplace-category-pills"
          items={MARKETPLACE_CATEGORIES.map(cat => ({
            value: cat,
            label: cat,
            testId: `category-chip-${cat}`,
          }))}
          onChange={setActiveCategory}
          size="sm"
          value={activeCategory}
        />
      </div>
      <div className="min-h-0 flex-1 overflow-y-auto p-4">
        {filtered.length === 0 ? (
          <div
            className="flex min-h-[240px] items-center justify-center"
            data-testid="marketplace-empty"
          >
            <Empty
              className="max-w-sm"
              description={
                isBrowseOnly
                  ? hasFilters
                    ? "No installed marketplace skills match the current filters."
                    : "No marketplace-installed skills are available in this workspace yet."
                  : hasFilters
                    ? "No skills match the current filters."
                    : "No skills found on the marketplace."
              }
              icon={Wrench}
              title={isBrowseOnly ? "No marketplace-installed skills" : "No skills found"}
            />
          </div>
        ) : (
          <div
            className="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3"
            data-testid="marketplace-grid"
          >
            {filtered.map(skill => (
              <SkillCatalogItem
                isInstalled={installedSkillNames.has(skill.name)}
                isInstalling={isInstalling}
                key={skill.name}
                onInstall={onInstall ? () => onInstall(skill.name) : undefined}
                skill={skill}
              />
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

export { MarketplaceView };
export type { MarketplaceViewProps };
