import { AlertCircle, BookOpen, ExternalLink, Loader2, Trash2 } from "lucide-react";
import { useState } from "react";

import { Button, CodeBlock, Empty, MonoBadge, Section, StatusDot } from "@agh/ui";

import {
  deriveScopeFromFilename,
  formatKnowledgeDateTime,
  formatKnowledgeRelativeTime,
  knowledgeScopeLabel,
  memoryScopeTone,
  memoryTypeTone,
} from "@/systems/knowledge/lib/knowledge-formatters";
import type { MemoryHeader } from "@/systems/knowledge/types";

import { KnowledgeDeleteDialog } from "./knowledge-delete-dialog";

interface KnowledgeDetailPanelProps {
  memory: MemoryHeader | undefined;
  content: string | undefined;
  scope?: string;
  isLoading: boolean;
  error: Error | null;
  onDelete: (filename: string) => void;
  isDeletePending: boolean;
}

interface MetadataRow {
  key: string;
  value: string;
  tone?: "mono" | "plain";
}

function KnowledgeDetailPanel({
  memory,
  content,
  scope,
  isLoading,
  error,
  onDelete,
  isDeletePending,
}: KnowledgeDetailPanelProps) {
  const [confirmOpen, setConfirmOpen] = useState(false);

  if (isLoading) {
    return (
      <div
        className="flex min-h-0 flex-1 items-center justify-center"
        data-testid="knowledge-detail-loading"
      >
        <Loader2
          aria-hidden="true"
          className="size-5 animate-spin text-[color:var(--color-text-tertiary)]"
        />
      </div>
    );
  }

  if (error) {
    return (
      <div
        className="flex min-h-0 flex-1 items-center justify-center px-6 py-10"
        data-testid="knowledge-detail-error"
      >
        <Empty
          className="max-w-md"
          description={error.message ?? "Failed to load memory details"}
          icon={AlertCircle}
          title="Failed to load memory details"
        />
      </div>
    );
  }

  if (!memory) {
    return (
      <div
        className="flex min-h-0 flex-1 items-center justify-center px-6 py-10"
        data-testid="knowledge-detail-empty"
      >
        <Empty
          className="max-w-md"
          description="Select a memory to view details"
          icon={BookOpen}
          title="Select a memory to view details"
        />
      </div>
    );
  }

  const resolvedScope = scope ?? deriveScopeFromFilename(memory.filename);
  const scopeForTone = resolvedScope === "workspace" ? "workspace" : "global";
  const scopeTone = memoryScopeTone(scopeForTone);
  const typeTone = memoryTypeTone(memory.type);

  const metadataRows: MetadataRow[] = [
    { key: "Type", value: memory.type, tone: "mono" },
    { key: "Scope", value: resolvedScope, tone: "mono" },
    ...(memory.agent_name
      ? [{ key: "Agent", value: memory.agent_name, tone: "mono" as const }]
      : []),
    { key: "Modified", value: formatKnowledgeDateTime(memory.mod_time), tone: "plain" as const },
  ];

  const handleConfirmDelete = () => {
    onDelete(memory.filename);
    setConfirmOpen(false);
  };

  return (
    <div
      className="flex min-h-0 flex-1 flex-col overflow-y-auto"
      data-testid="knowledge-detail-panel"
    >
      <header className="flex flex-col gap-3 border-b border-[color:var(--color-divider)] px-6 py-5">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div className="flex min-w-0 items-center gap-3">
            <span
              aria-hidden="true"
              className="inline-flex size-8 shrink-0 items-center justify-center rounded-lg bg-[color:var(--color-surface-elevated)] text-[color:var(--color-accent)]"
            >
              <BookOpen className="size-4" />
            </span>
            <div className="flex min-w-0 flex-col">
              <h2 className="truncate text-[15px] font-semibold tracking-[-0.01em] text-[color:var(--color-text-primary)]">
                {memory.name}
              </h2>
              <span className="truncate font-mono text-[11px] text-[color:var(--color-text-tertiary)]">
                {memory.filename}
              </span>
            </div>
          </div>
          <div className="flex shrink-0 items-center gap-1.5">
            <MonoBadge data-testid="detail-type-badge" tone={typeTone}>
              {memory.type}
            </MonoBadge>
            <MonoBadge data-testid="detail-scope-badge" tone={scopeTone}>
              {knowledgeScopeLabel(scopeForTone)}
            </MonoBadge>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <StatusDot tone="success" />
          <span className="text-[13px] text-[color:var(--color-text-secondary)]">Active</span>
          <span className="font-mono text-[11px] text-[color:var(--color-text-tertiary)]">
            Updated {formatKnowledgeRelativeTime(memory.mod_time)}
          </span>
        </div>
      </header>

      <div className="flex flex-col gap-6 px-6 py-5">
        {memory.description ? (
          <Section label="Description">
            <p className="text-[13px] leading-relaxed text-[color:var(--color-text-secondary)]">
              {memory.description}
            </p>
          </Section>
        ) : null}

        {content ? (
          <Section label="Content">
            <CodeBlock code={content} copyable data-testid="content-preview" showPrompt={false} />
          </Section>
        ) : null}

        <Section label="Metadata">
          <dl
            className="flex flex-col divide-y divide-[color:var(--color-divider)] rounded-[var(--radius-diagram)] border border-[color:var(--color-divider)] bg-[color:var(--color-surface)]"
            data-testid="metadata-table"
          >
            {metadataRows.map(row => (
              <div
                className="flex items-center justify-between gap-3 px-4 py-2.5"
                data-testid={`metadata-row-${row.key}`}
                key={row.key}
              >
                <dt className="font-mono text-[11px] uppercase tracking-[0.06em] text-[color:var(--color-text-label)]">
                  {row.key}
                </dt>
                <dd className="min-w-0 text-right">
                  {row.tone === "mono" ? (
                    <MonoBadge>{row.value}</MonoBadge>
                  ) : (
                    <span className="text-[13px] text-[color:var(--color-text-primary)]">
                      {row.value}
                    </span>
                  )}
                </dd>
              </div>
            ))}
          </dl>
        </Section>
      </div>

      <footer className="mt-auto flex flex-wrap items-center gap-2 border-t border-[color:var(--color-divider)] px-6 py-4">
        <Button
          data-testid="delete-memory-btn"
          disabled={isDeletePending}
          onClick={() => setConfirmOpen(true)}
          size="sm"
          type="button"
          variant="outline"
        >
          <Trash2 className="size-3.5" />
          Delete
        </Button>
        <Button
          aria-disabled="true"
          data-testid="view-in-cli-btn"
          disabled
          size="sm"
          title="CLI deep links are not implemented yet"
          type="button"
          variant="ghost"
        >
          <ExternalLink className="size-3.5" />
          View in CLI
        </Button>
      </footer>

      <KnowledgeDeleteDialog
        filename={memory.filename}
        isPending={isDeletePending}
        onConfirm={handleConfirmDelete}
        onOpenChange={setConfirmOpen}
        open={confirmOpen}
        scope={resolvedScope}
      />
    </div>
  );
}

export { KnowledgeDetailPanel };
export type { KnowledgeDetailPanelProps };
