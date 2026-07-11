import type { ReactNode } from "react";
import { FileText } from "lucide-react";

import { Button, Empty, Pill, Skeleton, Spinner, Textarea } from "@agh/ui";

import {
  useAgentAuthoredFileEditor,
  type AuthoredFileKind,
} from "../hooks/use-agent-authored-file-editor";
import type {
  AgentHeartbeatHistoryResponse,
  AgentHeartbeatPayload,
  AgentSoulHistoryResponse,
  AgentSoulPayload,
} from "../types";

export type { AuthoredFileKind };

export interface AgentAuthoredFileEditorProps {
  kind: AuthoredFileKind;
  payload: AgentSoulPayload | AgentHeartbeatPayload | undefined;
  isLoading: boolean;
  isError: boolean;
  history: AgentSoulHistoryResponse | AgentHeartbeatHistoryResponse | undefined;
  onValidate: (body: string) => Promise<{
    diagnostics?: Array<{ message: string; line?: number; source_path?: string }>;
    validation_status?: string;
  }>;
  onSave: (body: string, expectedDigest: string) => Promise<void>;
  onRestore: (revisionId: string, expectedDigest: string) => Promise<void>;
  onRetry: () => void;
  /** Extra content rendered above the editor (heartbeat status/wake). */
  headerSlot?: ReactNode;
}

export function AgentAuthoredFileEditor({
  kind,
  payload,
  isLoading,
  isError,
  history,
  onValidate,
  onSave,
  onRestore,
  onRetry,
  headerSlot,
}: AgentAuthoredFileEditorProps) {
  const editor = useAgentAuthoredFileEditor({
    kind,
    payload,
    history,
    onValidate,
    onSave,
    onRestore,
  });

  if (isLoading) {
    return (
      <div className="flex flex-col gap-3" data-testid={`agent-${kind}-loading`}>
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-64 rounded-md" />
      </div>
    );
  }

  if (isError) {
    return (
      <Empty
        icon={FileText}
        title={`Couldn't load ${editor.fileLabel}`}
        description="Retry to fetch the authored file."
        action={
          <Button type="button" size="sm" variant="ghost" onClick={onRetry}>
            Retry
          </Button>
        }
        data-testid={`agent-${kind}-error`}
        fill={false}
      />
    );
  }

  if (editor.isMissing) {
    return (
      <div data-testid={`agent-${kind}-missing`}>
        {headerSlot}
        <Empty
          icon={FileText}
          title={`No ${editor.fileLabel}`}
          description={
            kind === "soul"
              ? "This agent has no soul file. Create one to define persona and constraints."
              : "This agent has no heartbeat policy. Create one to schedule wake checks."
          }
          action={
            <Button
              type="button"
              size="sm"
              onClick={() => void editor.handleCreate()}
              disabled={editor.saving}
              data-testid={`agent-${kind}-create`}
            >
              {editor.saving ? <Spinner className="size-3" /> : null}
              Create {editor.fileLabel}
            </Button>
          }
          fill={false}
        />
        {editor.saveError ? (
          <p className="mt-3 text-small-body text-danger" role="alert">
            {editor.saveError}
          </p>
        ) : null}
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-4" data-testid={`agent-${kind}-editor`}>
      {editor.guardDialog}
      {headerSlot}
      <div className="flex flex-wrap items-center gap-2">
        <Pill
          mono
          tone={
            editor.status === "valid"
              ? "success"
              : editor.status === "invalid"
                ? "danger"
                : "neutral"
          }
        >
          {editor.status}
        </Pill>
        {payload && "enabled" in payload ? (
          <Pill mono tone={payload.enabled ? "success" : "neutral"}>
            {payload.enabled ? "enabled" : "disabled"}
          </Pill>
        ) : null}
        {payload?.source_path ? (
          <code className="font-mono text-badge tracking-mono text-muted">
            {payload.source_path}
          </code>
        ) : null}
      </div>

      <Textarea
        variant="mono"
        value={editor.draft}
        onChange={event => editor.setDraft(event.target.value)}
        className="min-h-64"
        data-testid={`agent-${kind}-textarea`}
        aria-label={`${editor.fileLabel} body`}
      />

      {editor.diagnostics.length > 0 ? (
        <ul className="flex flex-col gap-1" data-testid={`agent-${kind}-diagnostics`}>
          {editor.diagnostics.map((item, index) => (
            <li key={`${item.message}-${index}`} className="text-small-body text-danger">
              {item.message}
              {item.line != null ? (
                <span className="ml-2 font-mono text-muted">line {item.line}</span>
              ) : null}
              {item.source_path ? (
                <code className="ml-2 font-mono text-muted">{item.source_path}</code>
              ) : null}
            </li>
          ))}
        </ul>
      ) : null}

      {editor.conflict || editor.saveError ? (
        <div className="flex flex-wrap items-center gap-2" role="alert">
          <p className="text-small-body text-danger">{editor.saveError}</p>
          {editor.conflict ? (
            <Button
              type="button"
              size="sm"
              variant="ghost"
              onClick={onRetry}
              data-testid={`agent-${kind}-reload`}
            >
              Reload
            </Button>
          ) : null}
        </div>
      ) : null}

      <div className="flex flex-wrap items-center gap-2">
        <Button
          type="button"
          size="sm"
          variant="ghost"
          onClick={() => void editor.handleValidate()}
          disabled={editor.validating || editor.saving}
          data-testid={`agent-${kind}-validate`}
        >
          {editor.validating ? <Spinner className="size-3" /> : null}
          {editor.validating ? "Validating…" : "Validate"}
        </Button>
        <Button
          type="button"
          size="sm"
          onClick={() => void editor.handleSave()}
          disabled={!editor.dirty || editor.saving}
          data-testid={`agent-${kind}-save`}
        >
          {editor.saving ? <Spinner className="size-3" /> : null}
          {editor.saving ? "Saving…" : "Save"}
        </Button>
        <Button
          type="button"
          size="sm"
          variant="ghost"
          onClick={() => editor.setShowHistory(current => !current)}
          data-testid={`agent-${kind}-history-toggle`}
        >
          History
        </Button>
      </div>

      {editor.showHistory ? (
        <div className="rounded-md border border-line p-3" data-testid={`agent-${kind}-history`}>
          {editor.revisions.length === 0 ? (
            <p className="text-small-body text-muted">No revisions yet.</p>
          ) : (
            <ul className="flex flex-col gap-2">
              {editor.revisions.map(revision => {
                const id = "id" in revision ? String(revision.id) : "";
                const created =
                  "created_at" in revision && typeof revision.created_at === "string"
                    ? revision.created_at
                    : "";
                return (
                  <li
                    key={id || created}
                    className="flex items-center justify-between gap-3"
                    data-testid={`agent-${kind}-revision-${id}`}
                  >
                    <span className="font-mono text-badge tracking-mono text-muted">
                      {id || "revision"} {created ? `· ${created}` : ""}
                    </span>
                    <Button
                      type="button"
                      size="sm"
                      variant="ghost"
                      onClick={() => void editor.handleRestore(id)}
                      disabled={!id || editor.saving}
                      data-testid={`agent-${kind}-restore-${id}`}
                    >
                      Restore
                    </Button>
                  </li>
                );
              })}
            </ul>
          )}
        </div>
      ) : null}
      <span className="sr-only" aria-live="polite">
        {editor.saving
          ? `Saving ${editor.fileLabel}`
          : editor.validating
            ? `Validating ${editor.fileLabel}`
            : ""}
      </span>
    </div>
  );
}
