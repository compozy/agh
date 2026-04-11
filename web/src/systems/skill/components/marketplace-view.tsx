import { Download, Search } from "lucide-react";
import { useMemo, useState } from "react";

import { PillButton } from "@/components/design-system";
import type { SkillPayload } from "../types";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface MarketplaceViewProps {
  skills: SkillPayload[];
  installedSkillNames: Set<string>;
  onInstall?: (name: string) => void;
  isInstalling: boolean;
  installUnavailableReason?: string;
}

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const CATEGORIES = ["ALL", "TESTING", "DATABASE", "DEPLOY", "AI", "DEVOPS", "SECURITY"] as const;

type Category = (typeof CATEGORIES)[number];

// ---------------------------------------------------------------------------
// Marketplace Row
// ---------------------------------------------------------------------------

function MarketplaceRow({
  skill,
  isInstalled,
  onInstall,
  isInstalling,
  installUnavailableReason,
}: {
  skill: SkillPayload;
  isInstalled: boolean;
  onInstall?: () => void;
  isInstalling: boolean;
  installUnavailableReason?: string;
}) {
  const tags = skill.metadata?.tags;
  const tagList: string[] = Array.isArray(tags) ? (tags as string[]) : [];
  const downloads = skill.metadata?.downloads;

  const installDisabled = isInstalling || !onInstall;

  return (
    <div
      className="flex items-center gap-4 rounded-lg bg-[color:var(--color-surface)] px-4 py-3"
      data-testid={`marketplace-row-${skill.name}`}
    >
      {/* Info */}
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <span className="text-[15px] font-medium text-[color:var(--color-text-primary)]">
            {skill.name}
          </span>
          {skill.provenance && (
            <span className="text-xs text-[color:var(--color-text-tertiary)]">
              @{skill.provenance.slug}
            </span>
          )}
          {skill.version && (
            <span className="text-xs text-[color:var(--color-text-tertiary)]">
              v{skill.version}
            </span>
          )}
        </div>
        <p className="mt-0.5 truncate text-xs text-[color:var(--color-text-secondary)]">
          {skill.description}
        </p>
        {tagList.length > 0 && (
          <div className="mt-1.5 flex flex-wrap gap-1.5">
            {tagList.map(tag => (
              <span
                key={tag}
                className="inline-flex h-[22px] items-center rounded-md border border-[color:var(--color-divider)] px-2 text-[10px] text-[color:var(--color-text-tertiary)]"
              >
                {tag}
              </span>
            ))}
          </div>
        )}
      </div>

      {/* Downloads */}
      {downloads != null && (
        <div className="flex shrink-0 items-center gap-1 text-xs text-[color:var(--color-text-tertiary)]">
          <Download className="size-3" />
          <span>{String(downloads)}</span>
        </div>
      )}

      {/* Action */}
      {isInstalled ? (
        <span
          className="inline-flex h-8 shrink-0 items-center rounded-full border border-[color:var(--color-divider)] px-3.5 text-xs text-[color:var(--color-text-tertiary)]"
          data-testid={`installed-pill-${skill.name}`}
        >
          INSTALLED
        </span>
      ) : (
        <button
          onClick={() => onInstall?.()}
          disabled={installDisabled}
          aria-disabled={installDisabled}
          title={!onInstall ? installUnavailableReason : undefined}
          className="inline-flex h-8 shrink-0 items-center rounded-full bg-[color:var(--color-accent)] px-3.5 text-xs font-medium text-white transition-colors hover:bg-[color:var(--color-accent-hover)] disabled:opacity-50"
          data-testid={`install-btn-${skill.name}`}
          type="button"
        >
          INSTALL
        </button>
      )}
    </div>
  );
}

// ---------------------------------------------------------------------------
// Marketplace View
// ---------------------------------------------------------------------------

function MarketplaceView({
  skills,
  installedSkillNames,
  onInstall,
  isInstalling,
  installUnavailableReason,
}: MarketplaceViewProps) {
  const [search, setSearch] = useState("");
  const [activeCategory, setActiveCategory] = useState<Category>("ALL");

  const filtered = useMemo(() => {
    let result = skills;

    if (search) {
      const q = search.toLowerCase();
      result = result.filter(
        s => s.name.toLowerCase().includes(q) || (s.description ?? "").toLowerCase().includes(q)
      );
    }

    if (activeCategory !== "ALL") {
      const cat = activeCategory.toLowerCase();
      result = result.filter(s => {
        const tags = s.metadata?.tags;
        if (Array.isArray(tags)) {
          return (tags as string[]).some(t => t.toLowerCase() === cat);
        }
        return false;
      });
    }

    return result;
  }, [skills, search, activeCategory]);

  return (
    <div className="flex flex-1 flex-col overflow-hidden" data-testid="marketplace-view">
      {/* Search */}
      <div className="border-b border-[color:var(--color-divider)] p-4">
        <div className="flex items-center gap-2 rounded-lg bg-[color:var(--color-surface-elevated)] px-3 py-2">
          <Search className="size-4 shrink-0 text-[color:var(--color-text-tertiary)]" />
          <input
            type="text"
            placeholder="Search skills on marketplace..."
            value={search}
            onChange={e => setSearch(e.target.value)}
            className="w-full bg-transparent text-sm text-[color:var(--color-text-primary)] placeholder:text-[color:var(--color-text-tertiary)] outline-none"
            data-testid="marketplace-search-input"
          />
        </div>
      </div>

      {/* Category filter chips */}
      <div className="flex flex-wrap items-center gap-1.5 border-b border-[color:var(--color-divider)] px-4 py-3">
        {CATEGORIES.map(cat => (
          <PillButton
            key={cat}
            active={activeCategory === cat}
            size="dense"
            onClick={() => setActiveCategory(cat)}
            data-testid={`category-chip-${cat}`}
          >
            {cat}
          </PillButton>
        ))}
      </div>

      {/* Skill rows */}
      <div className="flex-1 overflow-y-auto p-4">
        <div className="flex flex-col gap-2">
          {filtered.length === 0 && (
            <div
              className="py-12 text-center text-sm text-[color:var(--color-text-tertiary)]"
              data-testid="marketplace-empty"
            >
              No skills found
            </div>
          )}
          {filtered.map(skill => (
            <MarketplaceRow
              key={skill.name}
              skill={skill}
              isInstalled={installedSkillNames.has(skill.name)}
              onInstall={onInstall ? () => onInstall(skill.name) : undefined}
              isInstalling={isInstalling}
              installUnavailableReason={installUnavailableReason}
            />
          ))}
        </div>
      </div>
    </div>
  );
}

export { MarketplaceView };
export type { MarketplaceViewProps };
