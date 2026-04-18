import { useCallback, useEffect, useMemo, useState } from "react";

import { useSettingsPage } from "@/hooks/routes/use-settings-page";
import {
  SettingsApiError,
  useDisableSettingsExtension,
  useEnableSettingsExtension,
  usePutSettingsHook,
  useSettingsExtensions,
  useSettingsHooksExtensions,
  useUpdateSettingsHooksExtensions,
  type SettingsExtensionEntry,
  type SettingsHookEntry,
  type SettingsHookRequest,
  type SettingsHooksExtensionsSection,
  type SettingsHooksExtensionsTransportParity,
  type SettingsMutationResult,
  type SettingsUpdateHooksExtensionsRequest,
} from "@/systems/settings";

type PolicyConfig = SettingsHooksExtensionsSection["config"];

export type HooksPolicyLastAction =
  | { kind: "saved"; result: SettingsMutationResult }
  | { kind: "hook-toggled"; name: string; enabled: boolean; result: SettingsMutationResult };

export type ExtensionLastAction = {
  kind: "extension-toggled";
  name: string;
  enabled: boolean;
};

type LastAction = HooksPolicyLastAction | ExtensionLastAction | null;

function clonePolicy(config: PolicyConfig): PolicyConfig {
  return {
    marketplace: { ...config.marketplace },
    resources: {
      ...config.resources,
      allowed_kinds: config.resources.allowed_kinds ? [...config.resources.allowed_kinds] : [],
      operator_write_rate_limit: { ...config.resources.operator_write_rate_limit },
      snapshot_rate_limit: { ...config.resources.snapshot_rate_limit },
    },
  };
}

function samePolicy(a: PolicyConfig, b: PolicyConfig): boolean {
  if ((a.marketplace.registry ?? "") !== (b.marketplace.registry ?? "")) return false;
  if ((a.marketplace.base_url ?? "") !== (b.marketplace.base_url ?? "")) return false;

  const aKinds = [...(a.resources.allowed_kinds ?? [])].sort();
  const bKinds = [...(b.resources.allowed_kinds ?? [])].sort();
  if (aKinds.length !== bKinds.length) return false;
  for (let i = 0; i < aKinds.length; i += 1) {
    if (aKinds[i] !== bKinds[i]) return false;
  }

  if ((a.resources.max_scope ?? "") !== (b.resources.max_scope ?? "")) return false;

  const aOp = a.resources.operator_write_rate_limit;
  const bOp = b.resources.operator_write_rate_limit;
  if (aOp.queue !== bOp.queue || aOp.requests !== bOp.requests || aOp.window !== bOp.window) {
    return false;
  }

  const aSnap = a.resources.snapshot_rate_limit;
  const bSnap = b.resources.snapshot_rate_limit;
  if (
    aSnap.queue !== bSnap.queue ||
    aSnap.requests !== bSnap.requests ||
    aSnap.window !== bSnap.window
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

export function useSettingsHooksExtensionsPage() {
  const query = useSettingsHooksExtensions();
  const extensionsQuery = useSettingsExtensions();
  const policyMutation = useUpdateSettingsHooksExtensions();
  const hookMutation = usePutSettingsHook();
  const enableMutation = useEnableSettingsExtension();
  const disableMutation = useDisableSettingsExtension();
  const page = useSettingsPage({ currentSlug: "hooks-extensions" });

  const envelope = query.data ?? null;

  const [draft, setDraft] = useState<PolicyConfig | null>(null);
  const [lastAction, setLastAction] = useState<LastAction>(null);
  const [pendingHookName, setPendingHookName] = useState<string | null>(null);
  const [pendingExtensionName, setPendingExtensionName] = useState<string | null>(null);

  useEffect(() => {
    if (envelope && draft === null) {
      setDraft(clonePolicy(envelope.config));
    }
  }, [envelope, draft]);

  const hooks: SettingsHookEntry[] = useMemo(() => envelope?.hooks ?? [], [envelope]);
  const installedFromEnvelope = useMemo(() => envelope?.installed ?? [], [envelope]);
  const extensions = useMemo<SettingsExtensionEntry[]>(() => {
    const live = extensionsQuery.data;
    if (live && live.length > 0) return live;
    return installedFromEnvelope.map(entry => ({
      name: entry.name,
      enabled: entry.enabled,
      version: entry.version ?? "",
      state: entry.state ?? "",
      health: entry.health,
      health_message: entry.health_message,
      last_error: entry.last_error,
      source: "settings",
      type: "unknown",
      daemon_running: true,
    }));
  }, [extensionsQuery.data, installedFromEnvelope]);

  const transportParity: SettingsHooksExtensionsTransportParity | null =
    envelope?.transport_parity ?? null;

  const isPolicyDirty = useMemo(() => {
    if (!envelope || !draft) return false;
    return !samePolicy(envelope.config, draft);
  }, [envelope, draft]);

  const handleResetPolicy = useCallback(() => {
    if (envelope) setDraft(clonePolicy(envelope.config));
    policyMutation.reset();
  }, [envelope, policyMutation]);

  const handleSavePolicy = useCallback(() => {
    if (!draft) return;
    const body: SettingsUpdateHooksExtensionsRequest = { config: draft };
    policyMutation.mutate(body, {
      onSuccess: result => {
        setLastAction({ kind: "saved", result });
      },
    });
  }, [draft, policyMutation]);

  const updateDraft = useCallback(
    (updater: (current: PolicyConfig) => PolicyConfig) => {
      if (!draft) return;
      setDraft(updater(draft));
    },
    [draft]
  );

  const toggleAllowedKind = useCallback(
    (kind: string) => {
      updateDraft(current => {
        const existing = current.resources.allowed_kinds ?? [];
        const next = existing.includes(kind)
          ? existing.filter(value => value !== kind)
          : [...existing, kind].sort();
        return {
          ...current,
          resources: { ...current.resources, allowed_kinds: next },
        };
      });
    },
    [updateDraft]
  );

  const toggleHookEnabled = useCallback(
    (entry: SettingsHookEntry, nextEnabled: boolean) => {
      hookMutation.reset();
      setPendingHookName(entry.name);
      const declaration: SettingsHookRequest["declaration"] = {
        ...entry.declaration,
        required: nextEnabled,
      };
      hookMutation.mutate(
        { name: entry.name, body: { declaration } },
        {
          onSuccess: result => {
            setLastAction({
              kind: "hook-toggled",
              name: entry.name,
              enabled: nextEnabled,
              result,
            });
          },
          onSettled: () => {
            setPendingHookName(null);
          },
        }
      );
    },
    [hookMutation]
  );

  const toggleExtensionEnabled = useCallback(
    (entry: SettingsExtensionEntry, nextEnabled: boolean) => {
      enableMutation.reset();
      disableMutation.reset();
      setPendingExtensionName(entry.name);
      const mutation = nextEnabled ? enableMutation : disableMutation;
      mutation.mutate(entry.name, {
        onSuccess: () => {
          setLastAction({
            kind: "extension-toggled",
            name: entry.name,
            enabled: nextEnabled,
          });
        },
        onSettled: () => {
          setPendingExtensionName(null);
        },
      });
    },
    [disableMutation, enableMutation]
  );

  const dismissLastAction = useCallback(() => setLastAction(null), []);

  const hooksCounts = useMemo(() => {
    const total = hooks.length;
    const enabled = hooks.filter(entry => entry.declaration.required !== false).length;
    return { total, enabled };
  }, [hooks]);

  const extensionsCounts = useMemo(() => {
    const total = extensions.length;
    const enabled = extensions.filter(entry => entry.enabled).length;
    return { total, enabled };
  }, [extensions]);

  const canMutateExtensions = transportParity?.extensions_http !== false;

  return {
    isLoading: query.isLoading,
    error: query.error,
    envelope,
    draft,

    hooks,
    hooksCounts,
    pendingHookName,
    toggleHookEnabled,
    hookError: errorMessage(hookMutation.error),

    extensions,
    extensionsCounts,
    extensionsLoading: extensionsQuery.isLoading,
    extensionsError: errorMessage(extensionsQuery.error),
    pendingExtensionName,
    toggleExtensionEnabled,
    extensionActionError: errorMessage(enableMutation.error) ?? errorMessage(disableMutation.error),
    canMutateExtensions,

    transportParity,

    isPolicyDirty,
    isSavingPolicy: policyMutation.isPending,
    savePolicyError: errorMessage(policyMutation.error),
    policyWarnings: policyMutation.data?.warnings,
    handleSavePolicy,
    handleResetPolicy,
    updatePolicyDraft: updateDraft,
    toggleAllowedKind,

    lastAction,
    dismissLastAction,

    restart: page.restart,
  };
}
