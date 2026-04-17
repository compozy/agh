import { useCallback, useEffect, useMemo, useState } from "react";

import { useSettingsPage } from "@/hooks/routes/use-settings-page";
import {
  SettingsApiError,
  useSettingsGeneral,
  useUpdateSettingsGeneral,
  type SettingsGeneralSection,
  type SettingsUpdateGeneralRequest,
} from "@/systems/settings";

type GeneralConfig = SettingsGeneralSection["config"];

export function useSettingsGeneralPage() {
  const query = useSettingsGeneral();
  const mutation = useUpdateSettingsGeneral();
  const page = useSettingsPage({ currentSlug: "general" });

  const envelope = query.data ?? null;

  const [draft, setDraft] = useState<GeneralConfig | null>(null);
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
