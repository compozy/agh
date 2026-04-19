import { Download, Wrench } from "lucide-react";
import { useMemo, useState } from "react";

import {
  Button,
  Card,
  CardContent,
  CardFooter,
  CardHeader,
  Empty,
  MonoBadge,
  Pills,
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
}

interface MarketplaceCardProps {
  skill: SkillPayload;
  isInstalled: boolean;
  onInstall?: () => void;
  isInstalling: boolean;
  installUnavailableReason?: string;
}

function MarketplaceCard({
  skill,
  isInstalled,
  onInstall,
  isInstalling,
  installUnavailableReason,
}: MarketplaceCardProps) {
  const author = deriveSkillAuthor(skill);
  const tags = deriveSkillTags(skill);
  const downloads = skill.metadata?.downloads;
  const installDisabled = isInstalling || !onInstall;

  return (
    <Card className="flex flex-col gap-3" data-testid={`marketplace-row-${skill.name}`} size="sm">
      <CardHeader>
        <div className="flex items-start gap-3">
          <span
            aria-hidden="true"
            className="inline-flex size-9 shrink-0 items-center justify-center rounded-lg bg-[color:var(--color-surface-elevated)] text-[color:var(--color-accent)]"
          >
            <Wrench className="size-4" />
          </span>
          <div className="flex min-w-0 flex-1 flex-col gap-1">
            <span className="truncate text-[13px] font-medium text-[color:var(--color-text-primary)]">
              {skill.name}
            </span>
            <div className="flex flex-wrap items-center gap-2 font-mono text-[10px] uppercase tracking-[0.08em] text-[color:var(--color-text-tertiary)]">
              {author ? <span>{`@${author}`}</span> : null}
              {skill.version ? <span>{`v${skill.version}`}</span> : null}
              {downloads !== undefined && downloads !== null ? (
                <span className="inline-flex items-center gap-1">
                  <Download aria-hidden="true" className="size-3" />
                  {String(downloads)}
                </span>
              ) : null}
            </div>
          </div>
        </div>
      </CardHeader>
      <CardContent className="flex flex-col gap-2">
        <p className="text-[12.5px] leading-[1.55] text-[color:var(--color-text-secondary)]">
          {skill.description}
        </p>
        {tags.length > 0 ? (
          <div className="flex flex-wrap items-center gap-1.5">
            {tags.map(tag => (
              <MonoBadge
                data-testid={`marketplace-tag-${skill.name}-${tag}`}
                key={tag}
                tone="neutral"
                uppercase={false}
              >
                {tag}
              </MonoBadge>
            ))}
          </div>
        ) : null}
      </CardContent>
      <CardFooter className="bg-transparent">
        {isInstalled ? (
          <MonoBadge data-testid={`installed-pill-${skill.name}`} tone="success">
            INSTALLED
          </MonoBadge>
        ) : (
          <Button
            aria-disabled={installDisabled}
            data-testid={`install-btn-${skill.name}`}
            disabled={installDisabled}
            onClick={() => onInstall?.()}
            size="sm"
            title={!onInstall ? installUnavailableReason : undefined}
            type="button"
            variant="outline"
          >
            Install
          </Button>
        )}
      </CardFooter>
    </Card>
  );
}

function MarketplaceView({
  skills,
  installedSkillNames,
  onInstall,
  isInstalling,
  installUnavailableReason,
}: MarketplaceViewProps) {
  const [search, setSearch] = useState("");
  const [activeCategory, setActiveCategory] = useState<MarketplaceCategory>("ALL");

  const filtered = useMemo(() => {
    const byQuery = filterSkillsByQuery(skills, search);
    return byQuery.filter(skill => matchesMarketplaceCategory(skill, activeCategory));
  }, [skills, search, activeCategory]);

  return (
    <div className="flex min-h-0 flex-1 flex-col overflow-hidden" data-testid="marketplace-view">
      <div className="flex flex-col gap-3 border-b border-[color:var(--color-divider)] px-4 py-3">
        <SearchInput
          data-testid="marketplace-search-input"
          onChange={setSearch}
          placeholder="Search skills on marketplace…"
          value={search}
        />
        <Pills
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
                search.trim() !== "" || activeCategory !== "ALL"
                  ? "No skills match the current filters."
                  : "No skills found on the marketplace."
              }
              icon={Wrench}
              title="No skills found"
            />
          </div>
        ) : (
          <div
            className="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3"
            data-testid="marketplace-grid"
          >
            {filtered.map(skill => (
              <MarketplaceCard
                installUnavailableReason={installUnavailableReason}
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
