import { useCallback, useEffect, useMemo, useState } from "react";

import { useSettingsPage } from "@/hooks/routes/use-settings-page";
import {
  SettingsApiError,
  useSettingsObservability,
  useUpdateSettingsObservability,
  type SettingsObservabilitySection,
  type SettingsUpdateObservabilityRequest,
} from "@/systems/settings";

type ObservabilityConfig = SettingsObservabilitySection["config"];

export function useSettingsObservabilityPage() {
  const query = useSettingsObservability();
  const mutation = useUpdateSettingsObservability();
  const page = useSettingsPage({ currentSlug: "observability" });

  const envelope = query.data ?? null;

  const [draft, setDraft] = useState<ObservabilityConfig | null>(null);
  const [lastAppliedLabel, setLastAppliedLabel] = useState<string | null>(null);

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
    const body: SettingsUpdateObservabilityRequest = { config: draft };
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
    restart: page.restart,
  };
}
