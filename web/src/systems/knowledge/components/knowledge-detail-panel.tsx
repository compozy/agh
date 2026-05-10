import { AlertCircle, BookOpen, Pencil, Trash2 } from "lucide-react";
import { useEffect, useState } from "react";

import { Button, CodeBlock, Empty, Pill, Section, Spinner } from "@agh/ui";

import {
  formatKnowledgeDateTime,
  formatKnowledgeRelativeTime,
  knowledgeAgentTierLabel,
  knowledgeScopeLabel,
  memoryScopeTone,
  memoryTypeTone,
} from "@/systems/knowledge/lib/knowledge-formatters";
import type {
  KnowledgeMemoryItem,
  KnowledgeScope,
  MemoryDecision,
} from "@/systems/knowledge/types";

import { KnowledgeDecisionsSection } from "./knowledge-decisions-section";
import { KnowledgeDeleteDialog } from "./knowledge-delete-dialog";
import { KnowledgeEditDialog } from "./knowledge-edit-dialog";
import { pillToneFromKnowledgeTone } from "./knowledge-pill-tone";

interface KnowledgeDetailPanelProps {
  memory: KnowledgeMemoryItem | undefined;
  content: string | undefined;
  scope?: KnowledgeScope;
  status: {
    isLoading: boolean;
    isDeletePending: boolean;
    isEditPending?: boolean;
    isDecisionsLoading?: boolean;
  };
  error: Error | null;
  onDelete: (memory: KnowledgeMemoryItem) => Promise<void>;
  deleteError?: string | null;
  onEdit?: (
    memory: KnowledgeMemoryItem,
    input: { content: string; description?: string }
  ) => Promise<void>;
  editError?: string | null;
  decisions?: MemoryDecision[];
  decisionsError?: Error | null;
  onRevertDecision?: (decision: MemoryDecision) => Promise<void>;
  revertingDecisionId?: string | null;
  revertError?: string | null;
}

interface MetadataRow {
  key: string;
  value: string;
  tone?: "mono" | "plain";
}

function buildMetadataRows(memory: KnowledgeMemoryItem): MetadataRow[] {
  const rows: MetadataRow[] = [
    { key: "Type", value: memory.type, tone: "mono" },
    { key: "Scope", value: memory.scope, tone: "mono" },
  ];
  if (memory.scope === "agent" && memory.agent_tier) {
    rows.push({
      key: "Agent tier",
      value: knowledgeAgentTierLabel(memory.agent_tier),
      tone: "mono",
    });
  }
  if (memory.agent_name) {
    rows.push({ key: "Agent", value: memory.agent_name, tone: "mono" });
  }
  if (memory.workspace_id) {
    rows.push({ key: "Workspace", value: memory.workspace_id, tone: "mono" });
  }
  rows.push({
    key: "Modified",
    value: formatKnowledgeDateTime(memory.mod_time),
    tone: "plain",
  });
  rows.push({
    key: "Recalls",
    value: String(memory.recall_count),
    tone: "mono",
  });
  if (memory.last_recalled_at) {
    rows.push({
      key: "Last recalled",
      value: formatKnowledgeDateTime(memory.last_recalled_at),
      tone: "plain",
    });
  }
  if (memory.staleness_banner) {
    rows.push({ key: "Staleness", value: memory.staleness_banner, tone: "plain" });
  }
  if (memory.superseded_by) {
    rows.push({ key: "Superseded by", value: memory.superseded_by, tone: "mono" });
  }
  rows.push({
    key: "Injection",
    value: memory.injection ? "true" : "false",
    tone: "mono",
  });
  if (memory.system_managed) {
    rows.push({ key: "System managed", value: "true", tone: "mono" });
  }
  return rows;
}

function KnowledgeDetailPanel({
  memory,
  content,
  scope,
  status,
  error,
  onDelete,
  deleteError,
  onEdit,
  editError,
  decisions,
  decisionsError = null,
  onRevertDecision,
  revertingDecisionId = null,
  revertError = null,
}: KnowledgeDetailPanelProps) {
  const { isLoading, isDeletePending, isEditPending = false, isDecisionsLoading = false } = status;
  const [confirmDeleteOpen, setConfirmDeleteOpen] = useState(false);
  const [editOpen, setEditOpen] = useState(false);

  useEffect(() => {
    setConfirmDeleteOpen(false);
    setEditOpen(false);
  }, [memory?.filename, memory?.scope]);

  if (isLoading) {
    return (
      <div
        className="flex min-h-0 flex-1 items-center justify-center"
        data-testid="knowledge-detail-loading"
      >
        <Spinner className="size-5 text-(--subtle)" />
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
        <Empty className="max-w-md" icon={BookOpen} title="Select a memory to view details" />
      </div>
    );
  }

  const resolvedScope: KnowledgeScope = scope ?? memory.scope;
  const scopeTone = pillToneFromKnowledgeTone(memoryScopeTone(resolvedScope));
  const typeTone = pillToneFromKnowledgeTone(memoryTypeTone(memory.type));

  const metadataRows = buildMetadataRows(memory);

  const handleConfirmDelete = async () => {
    try {
      await onDelete(memory);
      setConfirmDeleteOpen(false);
    } catch {
      // Error state is surfaced through `deleteError` and the dialog stays open.
    }
  };

  const handleConfirmEdit = async (input: { content: string; description?: string }) => {
    if (!onEdit) return;
    try {
      await onEdit(memory, input);
      setEditOpen(false);
    } catch {
      // Error state is surfaced through `editError` and the dialog stays open.
    }
  };

  return (
    <div
      className="flex min-h-0 flex-1 flex-col overflow-y-auto"
      data-testid="knowledge-detail-panel"
    >
      <header className="flex flex-col gap-3 border-b border-(--line) px-6 py-5">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div className="flex min-w-0 items-center gap-3">
            <span
              aria-hidden="true"
              className="inline-flex size-8 shrink-0 items-center justify-center rounded-lg bg-(--elevated) text-(--accent)"
            >
              <BookOpen className="size-4" />
            </span>
            <div className="flex min-w-0 flex-col">
              <h2 className="truncate text-item-title font-medium tracking-tight text-(--fg)">
                {memory.name}
              </h2>
              <span className="truncate font-mono text-eyebrow text-(--subtle)">
                {memory.filename}
              </span>
            </div>
          </div>
          <div className="flex shrink-0 items-center gap-1.5">
            <Pill mono data-testid="detail-type-badge" tone={typeTone}>
              {memory.type}
            </Pill>
            <Pill mono data-testid="detail-scope-badge" tone={scopeTone}>
              {knowledgeScopeLabel(resolvedScope)}
            </Pill>
            {memory.scope === "agent" && memory.agent_tier ? (
              <Pill mono data-testid="detail-agent-tier-badge" tone="warning">
                {knowledgeAgentTierLabel(memory.agent_tier)}
              </Pill>
            ) : null}
          </div>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <Pill.Dot tone={memory.staleness_banner ? "warning" : "success"} />
          <span className="text-small-body text-(--muted)">
            {memory.staleness_banner ?? "Active"}
          </span>
          <span className="font-mono text-eyebrow text-(--subtle)">
            Updated {formatKnowledgeRelativeTime(memory.mod_time)}
          </span>
          {memory.superseded_by ? (
            <Pill mono data-testid="detail-superseded-badge" tone="warning">
              Superseded
            </Pill>
          ) : null}
        </div>
      </header>

      <div className="flex flex-col gap-6 px-6 py-5">
        {memory.description ? (
          <Section label="Description">
            <p className="text-small-body leading-relaxed text-(--muted)">{memory.description}</p>
          </Section>
        ) : null}

        {content ? (
          <Section label="Content">
            <CodeBlock code={content} copyable data-testid="content-preview" showPrompt={false} />
          </Section>
        ) : null}

        <Section label="Metadata">
          <dl
            className="flex flex-col divide-y divide-(--line) rounded-lg border border-(--line) bg-(--canvas-soft)"
            data-testid="metadata-table"
          >
            {metadataRows.map(row => (
              <div
                className="flex items-center justify-between gap-3 px-4 py-2.5"
                data-testid={`metadata-row-${row.key}`}
                key={row.key}
              >
                <dt className="font-mono text-eyebrow uppercase tracking-badge text-(--muted)">
                  {row.key}
                </dt>
                <dd className="min-w-0 text-right">
                  {row.tone === "mono" ? (
                    <Pill mono>{row.value}</Pill>
                  ) : (
                    <span className="text-small-body text-(--fg)">{row.value}</span>
                  )}
                </dd>
              </div>
            ))}
          </dl>
        </Section>

        <KnowledgeDecisionsSection
          decisions={decisions}
          error={decisionsError}
          isLoading={isDecisionsLoading}
          onRevertDecision={onRevertDecision}
          revertError={revertError}
          revertingDecisionId={revertingDecisionId}
        />
      </div>

      <footer className="mt-auto flex flex-wrap items-center gap-2 border-t border-(--line) px-6 py-4">
        {onEdit ? (
          <Button
            data-testid="edit-memory-btn"
            disabled={isEditPending || content === undefined}
            onClick={() => setEditOpen(true)}
            size="sm"
            type="button"
            variant="outline"
          >
            <Pencil className="size-3.5" />
            Edit
          </Button>
        ) : null}
        <Button
          data-testid="delete-memory-btn"
          disabled={isDeletePending}
          onClick={() => setConfirmDeleteOpen(true)}
          size="sm"
          type="button"
          variant="outline"
        >
          <Trash2 className="size-3.5" />
          Delete
        </Button>
        {deleteError ? (
          <span className="text-xs text-(--danger)" data-testid="knowledge-delete-error">
            {deleteError}
          </span>
        ) : null}
        {editError ? (
          <span className="text-xs text-(--danger)" data-testid="knowledge-edit-error">
            {editError}
          </span>
        ) : null}
      </footer>

      <KnowledgeDeleteDialog
        error={deleteError}
        filename={memory.filename}
        isPending={isDeletePending}
        onConfirm={handleConfirmDelete}
        onOpenChange={setConfirmDeleteOpen}
        open={confirmDeleteOpen}
        scope={resolvedScope}
      />

      {onEdit ? (
        <KnowledgeEditDialog
          error={editError}
          filename={memory.filename}
          initialContent={content ?? ""}
          initialDescription={memory.description ?? ""}
          isPending={isEditPending}
          onConfirm={handleConfirmEdit}
          onOpenChange={setEditOpen}
          open={editOpen}
          scope={resolvedScope}
        />
      ) : null}
    </div>
  );
}

export { KnowledgeDetailPanel };
export type { KnowledgeDetailPanelProps };
