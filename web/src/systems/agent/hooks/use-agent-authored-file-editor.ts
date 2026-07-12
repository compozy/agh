import { useEffect, useEffectEvent, useRef, useState, type ReactNode } from "react";
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

export type AuthoredFilePayload = AgentSoulPayload | AgentHeartbeatPayload;

export interface UseAgentAuthoredFileEditorArgs {
  resourceKey: string;
  kind: AuthoredFileKind;
  payload: AuthoredFilePayload | undefined;
  history: AgentSoulHistoryResponse | AgentHeartbeatHistoryResponse | undefined;
  onValidate: (body: string) => Promise<{
    diagnostics?: Array<{ message: string; line?: number; source_path?: string }>;
    validation_status?: string;
  }>;
  onSave: (body: string, expectedDigest: string) => Promise<AuthoredFilePayload>;
  onRestore: (revisionId: string, expectedDigest: string) => Promise<AuthoredFilePayload>;
}

const CREATE_BODIES: Record<AuthoredFileKind, string> = {
  soul: "# Soul\n\nDefine persona and constraints.\n",
  heartbeat: "---\nenabled: true\n---\n\nCheck for pending work.\n",
};

const CONFLICT_MESSAGE = "This file changed elsewhere. Reload and retry.";

interface LocalBaseline {
  digest: string;
  body: string;
}

/** Missing only for an explicit successful domain payload — never unknown/undefined. */
export function isAuthoredFileMissing(payload: AuthoredFilePayload | undefined): boolean {
  if (!payload) return false;
  return payload.validation_status === "missing" || payload.present === false;
}

export function buildAuthoredFileResourceKey(
  workspaceId: string | null,
  agentName: string,
  kind: AuthoredFileKind
): string {
  return JSON.stringify([workspaceId, agentName, kind]);
}

function readBody(kind: AuthoredFileKind, payload: AuthoredFilePayload): string {
  if (kind === "soul") {
    return serializeAgentSoulSource(payload as AgentSoulPayload);
  }
  return serializeAgentHeartbeatSource(payload as AgentHeartbeatPayload);
}

function readDigest(payload: AuthoredFilePayload): string {
  return payload.digest ?? "";
}

function seedFromPayload(
  kind: AuthoredFileKind,
  payload: AuthoredFilePayload | undefined
): {
  draft: string;
  baseline: LocalBaseline;
  diagnostics: Array<{ message: string; line?: number; source_path?: string }>;
} {
  if (!payload) {
    return { draft: "", baseline: { digest: "", body: "" }, diagnostics: [] };
  }
  if (isAuthoredFileMissing(payload)) {
    return {
      draft: "",
      baseline: { digest: readDigest(payload), body: "" },
      diagnostics: [],
    };
  }
  const body = readBody(kind, payload);
  return {
    draft: body,
    baseline: { digest: readDigest(payload), body },
    diagnostics: payload.diagnostics ?? [],
  };
}

export function useAgentAuthoredFileEditor({
  resourceKey,
  kind,
  payload,
  history,
  onValidate,
  onSave,
  onRestore,
}: UseAgentAuthoredFileEditorArgs) {
  const fileLabel = kind === "soul" ? "SOUL.md" : "HEARTBEAT.md";
  const [trackedResourceKey, setTrackedResourceKey] = useState(resourceKey);
  const initial = seedFromPayload(kind, payload);
  const [draft, setDraft] = useState(initial.draft);
  const [baseline, setBaseline] = useState<LocalBaseline>(initial.baseline);
  const [diagnostics, setDiagnostics] = useState(initial.diagnostics);
  const [effectivePayload, setEffectivePayload] = useState(payload);
  const [validating, setValidating] = useState(false);
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);
  const [conflict, setConflict] = useState(false);
  const [showHistory, setShowHistory] = useState(false);
  const [validationStatus, setValidationStatus] = useState<string | undefined>(
    payload?.validation_status
  );

  const draftRef = useRef(draft);
  const baselineRef = useRef(baseline);
  draftRef.current = draft;
  baselineRef.current = baseline;

  const applyAuthoritativePayload = (authoritative: AuthoredFilePayload | undefined) => {
    const next = seedFromPayload(kind, authoritative);
    setDraft(next.draft);
    setBaseline(next.baseline);
    setDiagnostics(next.diagnostics);
    setEffectivePayload(authoritative);
    setValidationStatus(authoritative?.validation_status);
    setConflict(false);
    setSaveError(null);
  };
  const applyPayloadFromEffect = useEffectEvent(applyAuthoritativePayload);

  useEffect(() => {
    const resourceChanged = trackedResourceKey !== resourceKey;
    const dirty = draftRef.current !== baselineRef.current.body;

    if (!resourceChanged && dirty) {
      // Same resource + dirty: freeze draft and CAS baseline against any background payload.
      return;
    }

    if (resourceChanged) {
      setTrackedResourceKey(resourceKey);
      setShowHistory(false);
    }

    applyPayloadFromEffect(payload);
  }, [payload, resourceKey, trackedResourceKey]);

  const dirty = draft !== baseline.body;

  const guard = useUnsavedGuard({ dirty, entityName: fileLabel });

  const applyWriteError = (err: unknown, fallback: string) => {
    if (isAgentDigestConflict(err)) {
      setConflict(true);
      setSaveError(CONFLICT_MESSAGE);
      return;
    }
    setConflict(false);
    setSaveError(err instanceof Error ? err.message : fallback);
  };

  const handleReload = () => {
    applyAuthoritativePayload(payload);
  };

  const handleValidate = async () => {
    setValidating(true);
    setSaveError(null);
    setConflict(false);
    try {
      const result = await onValidate(draft);
      setDiagnostics(result.diagnostics ?? []);
      if (result.validation_status) {
        setValidationStatus(result.validation_status);
      }
    } catch (err) {
      setSaveError(err instanceof Error ? err.message : `Couldn't validate ${fileLabel}`);
    } finally {
      setValidating(false);
    }
  };

  const handleSave = async () => {
    setSaving(true);
    setSaveError(null);
    setConflict(false);
    try {
      const authoritative = await onSave(draft, baseline.digest);
      applyAuthoritativePayload(authoritative);
      toast.success(`${fileLabel} saved`);
    } catch (err) {
      applyWriteError(err, `Couldn't save ${fileLabel}`);
    } finally {
      setSaving(false);
    }
  };

  const handleCreate = async () => {
    const body = CREATE_BODIES[kind];
    setSaving(true);
    setSaveError(null);
    setConflict(false);
    try {
      const authoritative = await onSave(body, baseline.digest);
      applyAuthoritativePayload(authoritative);
      toast.success(`${fileLabel} created`);
    } catch (err) {
      applyWriteError(err, `Couldn't create ${fileLabel}`);
    } finally {
      setSaving(false);
    }
  };

  const handleRestore = async (revisionId: string) => {
    setSaving(true);
    setSaveError(null);
    setConflict(false);
    try {
      const authoritative = await onRestore(revisionId, baseline.digest);
      applyAuthoritativePayload(authoritative);
      toast.success(`${fileLabel} restored`);
    } catch (err) {
      applyWriteError(err, `Couldn't restore ${fileLabel}`);
    } finally {
      setSaving(false);
    }
  };

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
    handleReload,
    payload: effectivePayload,
    isMissing: isAuthoredFileMissing(effectivePayload) && draft === "" && baseline.body === "",
    status: validationStatus ?? effectivePayload?.validation_status ?? "valid",
    revisions,
  };
}
