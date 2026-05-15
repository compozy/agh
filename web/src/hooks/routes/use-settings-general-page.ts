import { useCallback, useEffect, useMemo, useRef, useState } from "react";

import { useSettingsPage } from "@/hooks/routes/use-settings-page";
import {
  SettingsApiError,
  useSettingsGeneral,
  useSettingsUpdate,
  useUpdateSettingsGeneral,
  type SettingsGeneralSection,
  type SettingsUpdateGeneralRequest,
} from "@/systems/settings";
import { useActiveWorkspace } from "@/systems/workspace";

type GeneralConfig = SettingsGeneralSection["config"];

export function useSettingsGeneralPage() {
  const query = useSettingsGeneral();
  const update = useSettingsUpdate();
  const mutation = useUpdateSettingsGeneral();
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
        setLastAppliedLabel(
          result.restart_required
            ? "Saved · restart required to apply"
            : "Saved · applied immediately"
        );
      },
    });
  }, [draft, mutation]);

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
  };
}
