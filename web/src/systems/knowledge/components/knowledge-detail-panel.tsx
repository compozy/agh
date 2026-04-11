import { ExternalLink, Loader2, Trash2 } from "lucide-react";

import { cn } from "@/lib/utils";
import type { MemoryHeader } from "@/systems/knowledge/types";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface KnowledgeDetailPanelProps {
  memory: MemoryHeader | undefined;
  content: string | undefined;
  scope?: string;
  isLoading: boolean;
  error: Error | null;
  onDelete: (filename: string) => void;
  isDeletePending: boolean;
}

// ---------------------------------------------------------------------------
// Type Badge (detail view)
// ---------------------------------------------------------------------------

const TYPE_COLORS: Record<string, { bg: string; text: string }> = {
  user: { bg: "bg-[#e8572a26]", text: "text-[#e8572a]" },
  feedback: { bg: "bg-[#e8572a26]", text: "text-[#e8572a]" },
  project: { bg: "bg-[#30d15826]", text: "text-[#30d158]" },
  reference: { bg: "bg-[#bf5af226]", text: "text-[#bf5af2]" },
};

function DetailTypeBadge({ type }: { type: string }) {
  const colors = TYPE_COLORS[type] ?? TYPE_COLORS.user;
  return (
    <span
      className={cn(
        "inline-flex h-[22px] items-center rounded-md px-2 font-mono text-[10px] font-semibold uppercase tracking-[0.08em]",
        colors.bg,
        colors.text
      )}
      data-testid="detail-type-badge"
    >
      {type}
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
          data-testid="view-full-content-link"
        >
          View full content &rarr;
        </button>
      )}
    </div>
  );
}

// ---------------------------------------------------------------------------
// Metadata Table
// ---------------------------------------------------------------------------

interface MetadataRow {
  key: string;
  value: string;
}

function MetadataTable({ rows }: { rows: MetadataRow[] }) {
  return (
    <div className="overflow-hidden rounded-lg" data-testid="metadata-table">
      {rows.map((row, idx) => (
        <div
          key={row.key}
          className={cn(
            "flex items-center justify-between px-3 py-2",
            idx % 2 === 0 ? "bg-transparent" : "bg-[color:var(--color-surface)]"
          )}
          data-testid={`metadata-row-${row.key}`}
        >
          <span className="text-xs text-[color:var(--color-text-tertiary)]">{row.key}</span>
          <span className="text-sm font-medium text-[color:var(--color-text-primary)]">
            {row.value}
          </span>
        </div>
      ))}
    </div>
  );
}

function formatDateTime(dateStr: string): string {
  const date = new Date(dateStr);
  if (Number.isNaN(date.getTime())) {
    return dateStr;
  }
  return date.toLocaleString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

// ---------------------------------------------------------------------------
// Knowledge Detail Panel
// ---------------------------------------------------------------------------

function KnowledgeDetailPanel({
  memory,
  content,
  scope,
  isLoading,
  error,
  onDelete,
  isDeletePending,
}: KnowledgeDetailPanelProps) {
  if (isLoading) {
    return (
      <div
        className="flex flex-1 items-center justify-center"
        data-testid="knowledge-detail-loading"
      >
        <Loader2 className="size-5 animate-spin text-[color:var(--color-text-tertiary)]" />
      </div>
    );
  }

  if (error) {
    return (
      <div
        className="flex flex-1 items-center justify-center text-sm text-[color:var(--color-danger)]"
        data-testid="knowledge-detail-error"
      >
        Failed to load memory details
      </div>
    );
  }

  if (!memory) {
    return (
      <div
        className="flex flex-1 items-center justify-center text-sm text-[color:var(--color-text-tertiary)]"
        data-testid="knowledge-detail-empty"
      >
        Select a memory to view details
      </div>
    );
  }

  const metadataRows: MetadataRow[] = [
    { key: "Type", value: memory.type },
    { key: "Scope", value: scope ?? "unknown" },
    ...(memory.agent_name ? [{ key: "Agent", value: memory.agent_name }] : []),
    { key: "Modified", value: formatDateTime(memory.mod_time) },
  ];

  return (
    <div className="flex flex-1 flex-col overflow-y-auto p-6" data-testid="knowledge-detail-panel">
      {/* Header */}
      <div className="mb-6">
        <div className="flex items-center gap-3">
          <h2 className="text-base font-semibold text-[color:var(--color-text-primary)]">
            {memory.name}
          </h2>
          <DetailTypeBadge type={memory.type} />
        </div>

        {/* Status line */}
        <div className="mt-2 flex items-center gap-2">
          <span className="size-2 rounded-full bg-[color:var(--color-success)]" />
          <span className="text-xs text-[color:var(--color-text-secondary)]">Active</span>
          <span className="text-xs text-[color:var(--color-text-tertiary)]">{memory.filename}</span>
        </div>
      </div>

      {/* Description */}
      {memory.description && (
        <div className="mb-6">
          <h3 className="mb-2 font-mono text-[10px] font-semibold uppercase tracking-[0.08em] text-[color:var(--color-text-tertiary)]">
            Description
          </h3>
          <p className="text-sm leading-relaxed text-[color:var(--color-text-secondary)]">
            {memory.description}
          </p>
        </div>
      )}

      {/* Content preview */}
      {content && <ContentPreviewCard content={content} />}

      {/* Actions */}
      <div className="mt-6 flex items-center gap-3">
        <button
          type="button"
          onClick={() => onDelete(memory.filename)}
          disabled={isDeletePending}
          className="inline-flex h-9 items-center gap-2 rounded-lg border border-[color:var(--color-divider)] bg-transparent px-5 text-sm font-medium text-[color:var(--color-danger)] transition-colors hover:bg-[color:var(--color-hover)] disabled:opacity-50"
          data-testid="delete-memory-btn"
        >
          <Trash2 className="size-3.5" />
          Delete
        </button>
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

      {/* Metadata */}
      <div className="mt-6">
        <h3 className="mb-2 font-mono text-[10px] font-semibold uppercase tracking-[0.08em] text-[color:var(--color-text-tertiary)]">
          Metadata
        </h3>
        <MetadataTable rows={metadataRows} />
      </div>
    </div>
  );
}

export { KnowledgeDetailPanel };
export type { KnowledgeDetailPanelProps };
