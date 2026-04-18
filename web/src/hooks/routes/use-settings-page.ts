import { useMatchRoute } from "@tanstack/react-router";
import { useMemo } from "react";

import {
  SETTINGS_ROOT_PATH,
  SETTINGS_SECTIONS,
  settingsSectionPath,
  useSettingsRestart,
  type SettingsSectionDescriptor,
} from "@/systems/settings";

interface UseSettingsPageOptions {
  currentSlug?: string;
}

function useSettingsPage(options: UseSettingsPageOptions = {}) {
  const matchRoute = useMatchRoute();
  const restart = useSettingsRestart();

  const matchedSection = useMemo<SettingsSectionDescriptor | null>(() => {
    if (options.currentSlug) {
      const explicit = SETTINGS_SECTIONS.find(section => section.slug === options.currentSlug);
      if (explicit) {
        return explicit;
      }
    }

    for (const section of SETTINGS_SECTIONS) {
      if (matchRoute({ to: settingsSectionPath(section.slug), fuzzy: true })) {
        return section;
      }
    }

    return null;
  }, [matchRoute, options.currentSlug]);

  const isRestartBannerVisible =
    restart.isRestartRequired || restart.isPolling || restart.isSuccessful || restart.isFailed;

  const restartBanner = {
    isVisible: isRestartBannerVisible,
    isRestartRequired: restart.isRestartRequired,
    isPolling: restart.isPolling,
    isSuccessful: restart.isSuccessful,
    isFailed: restart.isFailed,
    operationId: restart.operationId,
    status: restart.status,
    failureReason: restart.failureReason,
    activeSessionCount: restart.activeSessionCount,
    lastMutation: restart.lastMutation,
    trigger: restart.trigger,
    isTriggerPending: restart.isTriggerPending,
    triggerError: restart.triggerError,
    dismiss: restart.dismiss,
  } as const;

  return {
    sections: SETTINGS_SECTIONS,
    rootPath: SETTINGS_ROOT_PATH,
    activeSection: matchedSection,
    activeSectionSlug: matchedSection?.slug ?? null,
    sectionPath: (slug: SettingsSectionDescriptor["slug"]) => settingsSectionPath(slug),
    restart: restartBanner,
  };
}

export { useSettingsPage };
