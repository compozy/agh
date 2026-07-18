import type * as React from "react";

import { PageHead } from "@agh/ui";

import { SETTINGS_SECTIONS } from "../lib/sections";
import type { SettingsSectionSlug } from "../types";

export interface SettingsPageHeadProps {
  slug: SettingsSectionSlug;
  /** Runtime status summary rendered beside the head (StatusLine). */
  statusLine?: React.ReactNode;
}

/**
 * Route identity row for a settings section: compact PageHead (section icon +
 * H1) with the runtime status line beside it. Status summaries are body
 * chrome — never topbar content (route chrome contract §04).
 */
export function SettingsPageHead({ slug, statusLine }: SettingsPageHeadProps) {
  const section = SETTINGS_SECTIONS.find(entry => entry.slug === slug);
  return (
    <div
      className="flex flex-wrap items-start justify-between gap-3"
      data-testid={`settings-page-${slug}-head-row`}
    >
      <PageHead
        data-testid={`settings-page-${slug}-head`}
        icon={section?.icon}
        title={section?.label ?? slug}
        variant="compact"
      />
      {statusLine}
    </div>
  );
}
