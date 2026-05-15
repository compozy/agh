import type { PillTone } from "@agh/ui";

/**
 * Canonical option shape for the shared provider command select. Snake_case keys
 * are intentional: workspace `SessionProviderOption` (OpenAPI-derived) and the
 * settings-mapped agent provider options both satisfy this with zero call-site
 * mapping.
 */
export interface ProviderSelectOption {
  name: string;
  display_name?: string;
  harness?: string;
  runtime_provider?: string;
}

/**
 * Canonical option shape for the shared model command select. Availability is
 * precomputed by the caller so `systems/runtime` stays decoupled from
 * `systems/model-catalog`.
 */
export interface ModelSelectOption {
  id: string;
  label: string;
  availability?: {
    label: string;
    tone: PillTone;
    state?: string;
  };
}

/** Canonical option shape for the shared reasoning-effort command select. */
export interface ReasoningSelectOption {
  value: string;
  label: string;
  source?: string;
}
