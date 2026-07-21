import {
  Activity,
  Brain,
  Cpu,
  Network,
  Palette,
  Puzzle,
  SlidersHorizontal,
  Webhook,
  Wrench,
  Zap,
} from "lucide-react";

import type { SettingsSectionDescriptor, SettingsSectionSlug } from "../types";

export const SETTINGS_ROOT_PATH = "/settings" as const;

export const SETTINGS_SECTIONS: readonly SettingsSectionDescriptor[] = [
  { slug: "general", label: "General", icon: SlidersHorizontal },
  { slug: "appearance", label: "Appearance", icon: Palette },
  { slug: "providers", label: "Providers", icon: Cpu },
  { slug: "memory", label: "Memory", icon: Brain },
  { slug: "skills", label: "Skills", icon: Wrench },
  { slug: "automation", label: "Automation", icon: Zap },
  { slug: "network", label: "Network", icon: Network },
  { slug: "observability", label: "Observability", icon: Activity },
  { slug: "hooks", label: "Hooks", icon: Webhook },
  { slug: "extensions", label: "Extensions", icon: Puzzle },
] as const;

export const SETTINGS_SECTION_SLUGS: readonly SettingsSectionSlug[] = SETTINGS_SECTIONS.map(
  section => section.slug
);

export function settingsSectionPath(slug: SettingsSectionSlug): string {
  return `${SETTINGS_ROOT_PATH}/${slug}`;
}

export function findSettingsSection(
  slug: string | undefined | null
): SettingsSectionDescriptor | undefined {
  if (!slug) {
    return undefined;
  }

  return SETTINGS_SECTIONS.find(section => section.slug === slug);
}
