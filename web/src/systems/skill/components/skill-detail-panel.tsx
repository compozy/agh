import { ExternalLink, Loader2, Power } from "lucide-react";

import type { SkillPayload } from "../types";
import { cn } from "@/lib/utils";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface SkillDetailPanelProps {
  skill: SkillPayload | undefined;
  isLoading: boolean;
  error: Error | null;
  onDisable: (name: string) => void;
  onEnable: (name: string) => void;
  isActionPending: boolean;
}

// ---------------------------------------------------------------------------
// Source Badge
// ---------------------------------------------------------------------------

const SOURCE_BADGE_COLORS: Record<string, { bg: string; text: string }> = {
  bundled: {
    bg: "bg-[color:var(--color-success-tint)]",
    text: "text-[color:var(--color-success)]",
  },
  workspace: {
    bg: "bg-[color:var(--color-info-tint)]",
    text: "text-[color:var(--color-info)]",
  },
  marketplace: {
    bg: "bg-[color:var(--color-accent-tint)]",
    text: "text-[color:var(--color-accent)]",
  },
  user: {
    bg: "bg-[color:var(--color-warning-tint)]",
    text: "text-[color:var(--color-warning)]",
  },
  additional: {
    bg: "bg-[color:var(--color-neutral-tint)]",
    text: "text-[color:var(--color-text-tertiary)]",
  },
};

function SourceBadge({ source }: { source: string }) {
  const colors = SOURCE_BADGE_COLORS[source] ?? SOURCE_BADGE_COLORS.additional;

  return (
    <span
      className={cn(
        "inline-flex h-[22px] items-center rounded-md px-2 font-mono text-[10px] font-semibold uppercase tracking-[0.08em]",
        colors.bg,
        colors.text
      )}
      data-testid="source-badge"
    >
      {source}
    </span>
  );
}

// ---------------------------------------------------------------------------
// Content Preview Card
// ---------------------------------------------------------------------------

function ContentPreviewCard({ content }: { content: string }) {
  const preview = content.length > 300 ? `${content.slice(0, 300)}...` : content;

  return (
    <div className="rounded-xl bg-[color:var(--color-surface)] p-4" data-testid="content-preview">
      <h4 className="mb-2 font-mono text-[10px] font-semibold uppercase tracking-[0.08em] text-[color:var(--color-text-tertiary)]">
        Content
      </h4>
      <pre className="whitespace-pre-wrap font-mono text-xs leading-relaxed text-[color:var(--color-text-secondary)]">
        {preview}
      </pre>
      {content.length > 300 && (
        <button
          type="button"
          disabled
          aria-disabled="true"
          title="Full content view is not implemented yet"
          className="mt-2 text-sm text-[color:var(--color-accent)] hover:text-[color:var(--color-accent-hover)]"
        >
          View full content
        </button>
      )}
    </div>
  );
}

// ---------------------------------------------------------------------------
// Skill Detail Panel
// ---------------------------------------------------------------------------

function SkillDetailPanel({
  skill,
  isLoading,
  error,
  onDisable,
  onEnable,
  isActionPending,
}: SkillDetailPanelProps) {
  if (isLoading) {
    return (
      <div className="flex flex-1 items-center justify-center" data-testid="skill-detail-loading">
        <Loader2 className="size-5 animate-spin text-[color:var(--color-text-tertiary)]" />
      </div>
    );
  }

  if (error) {
    return (
      <div
        className="flex flex-1 items-center justify-center text-sm text-[color:var(--color-danger)]"
        data-testid="skill-detail-error"
      >
        Failed to load skill details
      </div>
    );
  }

  if (!skill) {
    return (
      <div
        className="flex flex-1 items-center justify-center text-sm text-[color:var(--color-text-tertiary)]"
        data-testid="skill-detail-empty"
      >
        Select a skill to view details
      </div>
    );
  }

  return (
    <div className="flex flex-1 flex-col overflow-y-auto p-6" data-testid="skill-detail-panel">
      {/* Header */}
      <div className="mb-6">
        <div className="flex items-center gap-3">
          <h2 className="text-base font-semibold text-[color:var(--color-text-primary)]">
            {skill.name}
          </h2>
          <SourceBadge source={skill.source} />
        </div>

        {skill.version && (
          <span className="mt-1 block text-xs text-[color:var(--color-text-tertiary)]">
            v{skill.version}
          </span>
        )}

        {/* Status line */}
        <div className="mt-2 flex items-center gap-2">
          <span
            className={cn(
              "size-2 rounded-full",
              skill.enabled
                ? "bg-[color:var(--color-success)]"
                : "bg-[color:var(--color-text-tertiary)]"
            )}
          />
          <span className="text-xs text-[color:var(--color-text-secondary)]">
            {skill.enabled ? "Enabled" : "Disabled"}
          </span>
          <span className="text-xs text-[color:var(--color-text-tertiary)]">{skill.dir}</span>
        </div>
      </div>

      {/* Description */}
      <div className="mb-6">
        <h3 className="mb-2 font-mono text-[10px] font-semibold uppercase tracking-[0.08em] text-[color:var(--color-text-tertiary)]">
          Description
        </h3>
        <p className="text-sm leading-relaxed text-[color:var(--color-text-secondary)]">
          {skill.description}
        </p>
      </div>

      {/* Content preview */}
      {skill.content && <ContentPreviewCard content={skill.content} />}

      {/* Metadata */}
      {skill.metadata && Object.keys(skill.metadata).length > 0 && (
        <div className="mt-6">
          <h3 className="mb-2 font-mono text-[10px] font-semibold uppercase tracking-[0.08em] text-[color:var(--color-text-tertiary)]">
            Metadata
          </h3>
          <div className="overflow-hidden rounded-lg">
            {Object.entries(skill.metadata).map(([key, value], idx) => (
              <div
                key={key}
                className={cn(
                  "flex items-center justify-between px-3 py-2",
                  idx % 2 === 0 ? "bg-transparent" : "bg-[color:var(--color-surface)]"
                )}
              >
                <span className="text-xs text-[color:var(--color-text-tertiary)]">{key}</span>
                <span className="text-sm font-medium text-[color:var(--color-text-primary)]">
                  {String(value)}
                </span>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Actions */}
      <div className="mt-6 flex items-center gap-3">
        {skill.enabled ? (
          <button
            onClick={() => onDisable(skill.name)}
            disabled={isActionPending}
            className="inline-flex h-9 items-center gap-2 rounded-lg border border-[color:var(--color-divider)] bg-transparent px-5 text-sm font-medium text-[color:var(--color-text-primary)] transition-colors hover:bg-[color:var(--color-hover)] disabled:opacity-50"
            data-testid="disable-skill-btn"
            type="button"
          >
            <Power className="size-3.5" />
            Disable
          </button>
        ) : (
          <button
            onClick={() => onEnable(skill.name)}
            disabled={isActionPending}
            className="inline-flex h-9 items-center gap-2 rounded-lg bg-[color:var(--color-accent)] px-5 text-sm font-medium text-white transition-colors hover:bg-[color:var(--color-accent-hover)] disabled:opacity-50"
            data-testid="enable-skill-btn"
            type="button"
          >
            <Power className="size-3.5" />
            Enable
          </button>
        )}
        <button
          type="button"
          disabled
          aria-disabled="true"
          title="CLI deep links are not implemented yet"
          className="inline-flex h-9 items-center gap-2 rounded-lg border border-[color:var(--color-divider)] bg-transparent px-5 text-sm font-medium text-[color:var(--color-text-primary)] transition-colors hover:bg-[color:var(--color-hover)]"
          data-testid="view-in-cli-btn"
        >
          <ExternalLink className="size-3.5" />
          View in CLI
        </button>
      </div>
    </div>
  );
}

export { SkillDetailPanel };
export type { SkillDetailPanelProps };
