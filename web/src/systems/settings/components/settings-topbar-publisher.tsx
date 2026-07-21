import { useTopbarSlot } from "@agh/ui";

import { SETTINGS_SECTIONS } from "../lib/sections";
import type { SettingsSectionSlug } from "../types";

export interface SettingsTopbarPublisherProps {
  slug: SettingsSectionSlug;
  /** Runtime status summary published into the window-head trail. */
  statusLine?: React.ReactNode;
  /** Head actions (single accent target, per the topbar contract). */
  actions?: React.ReactNode;
}

/** Publishes one settings section's identity, status, and actions into its window head. */
export function SettingsTopbarPublisher({
  slug,
  statusLine,
  actions,
}: SettingsTopbarPublisherProps) {
  const section = SETTINGS_SECTIONS.find(entry => entry.slug === slug);
  const Icon = section?.icon;
  useTopbarSlot({
    glyph: Icon ? <Icon /> : undefined,
    crumb: section?.label ?? slug,
    status: statusLine,
    actions,
  });
  return null;
}
