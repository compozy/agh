import type { PillTone } from "@agh/ui";

import type { SettingsProviderEntry } from "../types";

export type ProviderStateLabel = "installed" | "binary-missing" | "unconfigured";

export type ProviderStateIntent = "edit" | "configure";

export interface ProviderStateView {
  tone: PillTone;
  label: ProviderStateLabel;
  display: string;
  hint: string | null;
  cta: {
    label: string;
    intent: ProviderStateIntent;
  };
}

const DISPLAY: Record<ProviderStateLabel, string> = {
  installed: "Installed",
  "binary-missing": "Binary missing",
  unconfigured: "Unconfigured",
};

const STATE_TONE: Record<ProviderStateLabel, PillTone> = {
  installed: "success",
  "binary-missing": "warning",
  unconfigured: "warning",
};

export function providerCredentialsConfigured(provider: SettingsProviderEntry): boolean {
  const credentials = provider.credentials ?? [];
  if (credentials.length === 0) {
    return true;
  }
  return credentials.every(credential => !credential.required || credential.present);
}

export function deriveProviderStateLabel(provider: SettingsProviderEntry): ProviderStateLabel {
  if (!provider.command_available) {
    return "binary-missing";
  }
  if (!providerCredentialsConfigured(provider)) {
    return "unconfigured";
  }
  return "installed";
}

function firstMissingRequiredSlot(provider: SettingsProviderEntry): string | null {
  const missing = (provider.credentials ?? []).find(
    credential => credential.required && !credential.present
  );
  if (!missing) return null;
  return missing.target_env || missing.name || null;
}

export function getProviderStateView(provider: SettingsProviderEntry): ProviderStateView {
  const label = deriveProviderStateLabel(provider);
  switch (label) {
    case "installed":
      return {
        tone: STATE_TONE.installed,
        label,
        display: DISPLAY.installed,
        hint: null,
        cta: { label: "Edit settings", intent: "edit" },
      };
    case "unconfigured": {
      const slot = firstMissingRequiredSlot(provider);
      return {
        tone: STATE_TONE.unconfigured,
        label,
        display: DISPLAY.unconfigured,
        hint: slot ? `Bind ${slot} to continue.` : "Required credential is missing.",
        cta: { label: "Configure credentials", intent: "configure" },
      };
    }
    case "binary-missing": {
      const command = provider.settings.command?.trim() || provider.name;
      return {
        tone: STATE_TONE["binary-missing"],
        label,
        display: DISPLAY["binary-missing"],
        hint: `${command} not found on PATH.`,
        cta: { label: "Edit settings", intent: "edit" },
      };
    }
  }
}
