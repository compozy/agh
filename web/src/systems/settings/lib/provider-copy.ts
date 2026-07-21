import type { ProviderStateView } from "./provider-state";
import type { SettingsProviderEntry } from "../types";

/** Plain-language sign-in summary for the provider listing (§ providers prototype). */
export function providerAuthSummary(provider: SettingsProviderEntry): string {
  switch (provider.settings.auth_mode) {
    case "bound_secret":
      return "Key stored in Vault";
    case "none":
      return "No sign-in needed";
    default:
      return "Uses your local login";
  }
}

/** Humanized one-line status for provider cards and rows. */
export function providerStatusCopy(
  state: ProviderStateView,
  modelCount: number | null
): { label: string; tone: "success" | "warning" } {
  switch (state.label) {
    case "installed":
      return {
        label: modelCount != null && modelCount > 0 ? `Ready · ${modelCount} models` : "Ready",
        tone: "success",
      };
    case "unconfigured":
      return { label: "Sign in to finish setup", tone: "warning" };
    case "binary-missing":
      return { label: "Binary not found", tone: "warning" };
  }
}
