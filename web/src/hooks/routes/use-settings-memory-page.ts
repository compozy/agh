import { useCallback, useEffect, useMemo, useState } from "react";

import { useSettingsPage } from "@/hooks/routes/use-settings-page";
import { useTriggerMemoryDream } from "@/systems/knowledge";
import {
  SettingsApiError,
  useSettingsMemory,
  useUpdateSettingsMemory,
  type SettingsMemorySection,
  type SettingsUpdateMemoryRequest,
} from "@/systems/settings";

type MemoryConfig = SettingsMemorySection["config"];

export function useSettingsMemoryPage() {
  const query = useSettingsMemory();
  const mutation = useUpdateSettingsMemory();
  const triggerDream = useTriggerMemoryDream();
  const page = useSettingsPage({ currentSlug: "memory" });

  const envelope = query.data ?? null;

  const [draft, setDraft] = useState<MemoryConfig | null>(null);
  const [lastAppliedLabel, setLastAppliedLabel] = useState<string | null>(null);
  const [actionMessage, setActionMessage] = useState<string | null>(null);

  useEffect(() => {
    if (envelope && draft === null) {
      setDraft(envelope.config);
    }
  }, [envelope, draft]);

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
    const body: SettingsUpdateMemoryRequest = { config: draft };
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

  const handleTriggerDream = useCallback(() => {
    setActionMessage(null);
    triggerDream.mutate(
      {},
      {
        onSuccess: response => {
          setActionMessage(
            response.triggered ? "Dream triggered" : response.reason || "Dream not triggered"
          );
        },
        onError: error => {
          setActionMessage(
            error instanceof Error ? error.message : "Failed to trigger memory dream"
          );
        },
      }
    );
  }, [triggerDream]);

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
    handleTriggerDream,
    isTriggeringDream: triggerDream.isPending,
    actionMessage,
    handleRetry,
    restart: page.restart,
  };
}
