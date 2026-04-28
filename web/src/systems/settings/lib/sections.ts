import {
  Activity,
  Brain,
  Cpu,
  Network,
  Puzzle,
  Server,
  SlidersHorizontal,
  Workflow,
  Wrench,
} from "lucide-react";

import type { SettingsSectionDescriptor, SettingsSectionSlug } from "../types";

export const SETTINGS_ROOT_PATH = "/settings" as const;

export const SETTINGS_SECTIONS: readonly SettingsSectionDescriptor[] = [
  { slug: "general", label: "General", icon: SlidersHorizontal },
  { slug: "providers", label: "Providers", icon: Cpu },
  { slug: "mcp-servers", label: "MCP Servers", icon: Server },
  { slug: "memory", label: "Memory", icon: Brain },
  { slug: "skills", label: "Skills", icon: Wrench },
  { slug: "automation", label: "Automation", icon: Workflow },
  { slug: "network", label: "Network", icon: Network },
  { slug: "observability", label: "Observability", icon: Activity },
  { slug: "hooks-extensions", label: "Hooks & Extensions", icon: Puzzle },
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
