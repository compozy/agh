import { useCallback, useEffect, useMemo, useState } from "react";

import { useSettingsPage } from "@/hooks/routes/use-settings-page";
import {
  SettingsApiError,
  useSettingsSkills,
  useUpdateSettingsSkills,
  type SettingsSkillsSection,
  type SettingsUpdateSkillsRequest,
} from "@/systems/settings";

type SkillsConfig = SettingsSkillsSection["config"];

function cloneDisabled(config: SkillsConfig): string[] {
  return [...(config.disabled_skills ?? [])];
}

function sameDisabled(a: string[] | undefined, b: string[] | undefined): boolean {
  const left = a ?? [];
  const right = b ?? [];
  if (left.length !== right.length) return false;
  for (let i = 0; i < left.length; i += 1) {
    if (left[i] !== right[i]) return false;
  }
  return true;
}

function samePolicy(a: SkillsConfig, b: SkillsConfig): boolean {
  if (a.enabled !== b.enabled) return false;
  if (a.poll_interval !== b.poll_interval) return false;
  if (a.marketplace.registry !== b.marketplace.registry) return false;
  if ((a.marketplace.base_url ?? "") !== (b.marketplace.base_url ?? "")) return false;
  if (
    JSON.stringify(a.allowed_marketplace_hooks ?? []) !==
    JSON.stringify(b.allowed_marketplace_hooks ?? [])
  ) {
    return false;
  }
  if (
    JSON.stringify(a.allowed_marketplace_mcp ?? []) !==
    JSON.stringify(b.allowed_marketplace_mcp ?? [])
  ) {
    return false;
  }
  return true;
}

function errorMessage(error: unknown): string | null {
  if (error instanceof SettingsApiError) return error.message;
  if (error instanceof Error) return error.message;
  return null;
}

export function useSettingsSkillsPage() {
  const query = useSettingsSkills();
  const disabledMutation = useUpdateSettingsSkills();
  const policyMutation = useUpdateSettingsSkills();
  const page = useSettingsPage({ currentSlug: "skills" });

  const envelope = query.data ?? null;

  const [draft, setDraft] = useState<SkillsConfig | null>(null);
  const [lastDisabledLabel, setLastDisabledLabel] = useState<string | null>(null);
  const [lastPolicyLabel, setLastPolicyLabel] = useState<string | null>(null);

  useEffect(() => {
    if (envelope && draft === null) {
      setDraft(envelope.config);
    }
  }, [envelope, draft]);

  const isDisabledDirty = useMemo(() => {
    if (!envelope || !draft) return false;
    return !sameDisabled(envelope.config.disabled_skills, draft.disabled_skills);
  }, [envelope, draft]);

  const isPolicyDirty = useMemo(() => {
    if (!envelope || !draft) return false;
    return !samePolicy(envelope.config, draft);
  }, [envelope, draft]);

  const handleResetDisabled = useCallback(() => {
    if (!envelope || !draft) return;
    setDraft({ ...draft, disabled_skills: cloneDisabled(envelope.config) });
  }, [envelope, draft]);

  const handleResetPolicy = useCallback(() => {
    if (!envelope || !draft) return;
    setDraft({
      ...envelope.config,
      disabled_skills: draft.disabled_skills,
    });
  }, [envelope, draft]);

  const handleSaveDisabled = useCallback(() => {
    if (!envelope || !draft) return;
    const body: SettingsUpdateSkillsRequest = {
      config: {
        ...envelope.config,
        disabled_skills: draft.disabled_skills ?? [],
      },
    };
    disabledMutation.mutate(body, {
      onSuccess: () => {
        setLastDisabledLabel("Saved · applied immediately");
      },
    });
  }, [envelope, draft, disabledMutation]);

  const handleSavePolicy = useCallback(() => {
    if (!envelope || !draft) return;
    const body: SettingsUpdateSkillsRequest = {
      config: {
        ...draft,
        disabled_skills: envelope.config.disabled_skills ?? [],
      },
    };
    policyMutation.mutate(body, {
      onSuccess: () => {
        setLastPolicyLabel("Saved · restart required to apply");
      },
    });
  }, [envelope, draft, policyMutation]);

  const toggleDisabled = useCallback(
    (name: string) => {
      if (!draft) return;
      const current = draft.disabled_skills ?? [];
      const next = current.includes(name)
        ? current.filter(entry => entry !== name)
        : [...current, name].sort();
      setDraft({ ...draft, disabled_skills: next });
    },
    [draft]
  );

  return {
    isLoading: query.isLoading,
    error: query.error,
    envelope,
    draft,
    setDraft,
    toggleDisabled,
    isDisabledDirty,
    isPolicyDirty,
    handleResetDisabled,
    handleResetPolicy,
    handleSaveDisabled,
    handleSavePolicy,
    isSavingDisabled: disabledMutation.isPending,
    isSavingPolicy: policyMutation.isPending,
    saveDisabledError: errorMessage(disabledMutation.error),
    savePolicyError: errorMessage(policyMutation.error),
    disabledWarnings: disabledMutation.data?.warnings,
    policyWarnings: policyMutation.data?.warnings,
    lastDisabledLabel,
    lastPolicyLabel,
    restart: page.restart,
  };
}
