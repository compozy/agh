import { useCallback, useEffect, useMemo, useRef, useState } from "react";

import { useSettingsPage } from "@/hooks/routes/use-settings-page";
import {
  SettingsApiError,
  useReloadSettings,
  useSettingsApplyRecords,
  useSettingsGeneral,
  useSettingsUpdate,
  useUpdateSettingsGeneral,
  type SettingsGeneralSection,
  type SettingsUpdateGeneralRequest,
} from "@/systems/settings";
import { useActiveWorkspace } from "@/systems/workspace";

type GeneralConfig = SettingsGeneralSection["config"];

function applyResultLabel(result: {
  active_generation: number;
  next_action: string;
  restart_required?: boolean;
  skipped?: boolean;
  skipped_reason?: string;
}) {
  if (result.skipped) {
    return result.skipped_reason ?? "No config changes detected";
  }
  if (result.restart_required || result.next_action === "restart-daemon") {
    return "Saved · restart required to apply";
  }
  if (result.next_action === "new-session") {
    return "Saved · new sessions use this config";
  }
  if (result.next_action === "retry") {
    return "Saved · reload required";
  }
  return `Saved · active generation ${result.active_generation}`;
}

export function useSettingsGeneralPage() {
  const query = useSettingsGeneral();
  const update = useSettingsUpdate();
  const applyRecords = useSettingsApplyRecords({ limit: 8 });
  const mutation = useUpdateSettingsGeneral();
  const reload = useReloadSettings();
  const page = useSettingsPage({ currentSlug: "general" });
  const { activeWorkspaceId } = useActiveWorkspace();

  const envelope = query.data ?? null;
  const workspaceContextKey = activeWorkspaceId ?? "__none__";

  const [draft, setDraft] = useState<GeneralConfig | null>(null);
  const [lastAppliedLabel, setLastAppliedLabel] = useState<string | null>(null);
  const draftWorkspaceContext = useRef<string | null>(null);

  useEffect(() => {
    if (envelope && draft === null) {
      setDraft(envelope.config);
      draftWorkspaceContext.current = workspaceContextKey;
      return;
    }

    if (!envelope || draft === null) {
      return;
    }

    if (draftWorkspaceContext.current === null) {
      draftWorkspaceContext.current = workspaceContextKey;
      return;
    }

    if (draftWorkspaceContext.current !== workspaceContextKey) {
      setDraft(envelope.config);
      setLastAppliedLabel(null);
      draftWorkspaceContext.current = workspaceContextKey;
    }
  }, [envelope, draft, workspaceContextKey]);

  const isDirty = useMemo(() => {
    if (!envelope || !draft) return false;
    return JSON.stringify(envelope.config) !== JSON.stringify(draft);
  }, [envelope, draft]);

  const handleReset = useCallback(() => {
    if (envelope) {
      setDraft(envelope.config);
    }
  }, [envelope]);

  const handleSave = useCallback(() => {
    if (!draft) return;
    const body: SettingsUpdateGeneralRequest = { config: draft };
    mutation.mutate(body, {
      onSuccess: result => {
        setLastAppliedLabel(applyResultLabel(result));
      },
    });
  }, [draft, mutation]);

  const handleReload = useCallback(() => {
    reload.mutate(undefined, {
      onSuccess: result => {
        setLastAppliedLabel(applyResultLabel(result));
      },
    });
  }, [reload]);

  const saveError =
    mutation.error instanceof SettingsApiError
      ? mutation.error.message
      : mutation.error instanceof Error
        ? mutation.error.message
        : null;

  const handleRetry = useCallback(() => {
    void query.refetch();
  }, [query]);

  return {
    isLoading: query.isLoading,
    error: query.error,
    envelope,
    draft,
    setDraft,
    isDirty,
    handleReset,
    handleSave,
    isSaving: mutation.isPending,
    saveError,
    warnings: mutation.data?.warnings,
    lastAppliedLabel,
    handleRetry,
    restart: page.restart,
    update,
    applyRecords,
    handleReload,
    isReloading: reload.isPending,
    reloadError:
      reload.error instanceof SettingsApiError
        ? reload.error.message
        : reload.error instanceof Error
          ? reload.error.message
          : null,
    reloadResult: reload.data ?? null,
  };
}
