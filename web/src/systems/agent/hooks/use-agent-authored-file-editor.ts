import { useCallback, useEffect, useMemo, useState, type ReactNode } from "react";
import { toast } from "sonner";

import { isAgentDigestConflict } from "../adapters/agent-api";
import {
  serializeAgentHeartbeatSource,
  serializeAgentSoulSource,
} from "../lib/agent-authored-file-source";
import type {
  AgentHeartbeatHistoryResponse,
  AgentHeartbeatPayload,
  AgentSoulHistoryResponse,
  AgentSoulPayload,
} from "../types";
import { useUnsavedGuard } from "./use-unsaved-guard";

export type AuthoredFileKind = "soul" | "heartbeat";

export interface UseAgentAuthoredFileEditorArgs {
  kind: AuthoredFileKind;
  payload: AgentSoulPayload | AgentHeartbeatPayload | undefined;
  history: AgentSoulHistoryResponse | AgentHeartbeatHistoryResponse | undefined;
  onValidate: (body: string) => Promise<{
    diagnostics?: Array<{ message: string; line?: number; source_path?: string }>;
    validation_status?: string;
  }>;
  onSave: (body: string, expectedDigest: string) => Promise<void>;
  onRestore: (revisionId: string, expectedDigest: string) => Promise<void>;
}

const CREATE_BODIES: Record<AuthoredFileKind, string> = {
  soul: "# Soul\n\nDefine persona and constraints.\n",
  heartbeat: "---\nenabled: true\n---\n\nCheck for pending work.\n",
};

export function isAuthoredFileMissing(
  payload: AgentSoulPayload | AgentHeartbeatPayload | undefined
): boolean {
  if (!payload) return true;
  return payload.validation_status === "missing" || payload.present === false;
}

function readBody(
  kind: AuthoredFileKind,
  payload: AgentSoulPayload | AgentHeartbeatPayload
): string {
  if (kind === "soul") {
    return serializeAgentSoulSource(payload as AgentSoulPayload);
  }
  return serializeAgentHeartbeatSource(payload as AgentHeartbeatPayload);
}

function readDigest(payload: AgentSoulPayload | AgentHeartbeatPayload): string {
  return payload.digest ?? "";
}

export function useAgentAuthoredFileEditor({
  kind,
  payload,
  history,
  onValidate,
  onSave,
  onRestore,
}: UseAgentAuthoredFileEditorArgs) {
  const fileLabel = kind === "soul" ? "SOUL.md" : "HEARTBEAT.md";
  const [draft, setDraft] = useState("");
  const [seededDigest, setSeededDigest] = useState("");
  const [diagnostics, setDiagnostics] = useState<
    Array<{ message: string; line?: number; source_path?: string }>
  >([]);
  const [validating, setValidating] = useState(false);
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);
  const [conflict, setConflict] = useState(false);
  const [showHistory, setShowHistory] = useState(false);

  useEffect(() => {
    if (!payload || isAuthoredFileMissing(payload)) {
      setDraft("");
      setSeededDigest(payload ? readDigest(payload) : "");
      return;
    }
    setDraft(readBody(kind, payload));
    setSeededDigest(readDigest(payload));
    setDiagnostics(payload.diagnostics ?? []);
    setConflict(false);
    setSaveError(null);
  }, [kind, payload]);

  const dirty = useMemo(() => {
    if (!payload || isAuthoredFileMissing(payload)) return false;
    return draft !== readBody(kind, payload);
  }, [draft, kind, payload]);

  const guard = useUnsavedGuard({ dirty, entityName: fileLabel });

  const handleValidate = useCallback(async () => {
    setValidating(true);
    setSaveError(null);
    try {
      const result = await onValidate(draft);
      setDiagnostics(result.diagnostics ?? []);
    } catch (err) {
      setSaveError(err instanceof Error ? err.message : `Couldn't validate ${fileLabel}`);
    } finally {
      setValidating(false);
    }
  }, [draft, fileLabel, onValidate]);

  const handleSave = useCallback(async () => {
    setSaving(true);
    setSaveError(null);
    setConflict(false);
    try {
      await onSave(draft, seededDigest);
      toast.success(`${fileLabel} saved`);
    } catch (err) {
      if (isAgentDigestConflict(err)) {
        setConflict(true);
        setSaveError("This file changed elsewhere. Reload and retry.");
      } else {
        setSaveError(err instanceof Error ? err.message : `Couldn't save ${fileLabel}`);
      }
    } finally {
      setSaving(false);
    }
  }, [draft, fileLabel, onSave, seededDigest]);

  const handleCreate = useCallback(async () => {
    const body = CREATE_BODIES[kind];
    const digest = payload ? readDigest(payload) : "";
    setSaving(true);
    setSaveError(null);
    try {
      await onSave(body, digest);
      toast.success(`${fileLabel} created`);
    } catch (err) {
      setSaveError(err instanceof Error ? err.message : `Couldn't create ${fileLabel}`);
    } finally {
      setSaving(false);
    }
  }, [fileLabel, kind, onSave, payload]);

  const handleRestore = useCallback(
    async (revisionId: string) => {
      setSaving(true);
      setSaveError(null);
      try {
        await onRestore(revisionId, seededDigest);
        toast.success(`${fileLabel} restored`);
      } catch (err) {
        setSaveError(err instanceof Error ? err.message : `Couldn't restore ${fileLabel}`);
      } finally {
        setSaving(false);
      }
    },
    [fileLabel, onRestore, seededDigest]
  );

  const revisions = history && "revisions" in history ? (history.revisions ?? []) : [];

  return {
    fileLabel,
    draft,
    setDraft,
    diagnostics,
    validating,
    saving,
    saveError,
    conflict,
    showHistory,
    setShowHistory,
    dirty,
    guardDialog: guard.confirmDialog as ReactNode,
    handleValidate,
    handleSave,
    handleCreate,
    handleRestore,
    isMissing: isAuthoredFileMissing(payload),
    status: payload?.validation_status ?? "valid",
    revisions,
  };
}
