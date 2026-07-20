import { useTopbarSlot } from "@agh/ui";

import { SETTINGS_SECTIONS } from "../lib/sections";
import type { SettingsSectionSlug } from "../types";

export interface SettingsTopbarPublisherProps {
  slug: SettingsSectionSlug;
  /** Runtime status summary published into the window-head trail. */
  statusLine?: React.ReactNode;
}

/** Publishes one settings section's identity and status into its window head. */
export function SettingsTopbarPublisher({ slug, statusLine }: SettingsTopbarPublisherProps) {
  const section = SETTINGS_SECTIONS.find(entry => entry.slug === slug);
  const Icon = section?.icon;
  useTopbarSlot({
    glyph: Icon ? <Icon /> : undefined,
    crumb: section?.label ?? slug,
    status: statusLine,
  });
  return null;
}
