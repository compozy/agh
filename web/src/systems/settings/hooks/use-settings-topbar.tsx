import { useTopbarSlot } from "@agh/ui";

import { SETTINGS_SECTIONS } from "../lib/sections";
import type { SettingsSectionSlug } from "../types";

export interface UseSettingsTopbarOptions {
  status?: React.ReactNode;
  actions?: React.ReactNode;
}

/** Publishes one settings section's identity and actions into the window topbar. */
export function useSettingsTopbar(
  slug: SettingsSectionSlug,
  { status, actions }: UseSettingsTopbarOptions = {}
) {
  const section = SETTINGS_SECTIONS.find(entry => entry.slug === slug);
  const Icon = section?.icon;

  useTopbarSlot({
    glyph: Icon ? <Icon /> : undefined,
    crumb: section?.label ?? slug,
    status,
    actions,
  });
}
