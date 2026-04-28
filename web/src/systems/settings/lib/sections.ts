import type { SettingsSectionDescriptor, SettingsSectionSlug } from "../types";

export const SETTINGS_ROOT_PATH = "/settings" as const;

export const SETTINGS_SECTIONS: readonly SettingsSectionDescriptor[] = [
  { slug: "general", label: "General" },
  { slug: "providers", label: "Providers" },
  { slug: "mcp-servers", label: "MCP Servers" },
  { slug: "memory", label: "Memory" },
  { slug: "skills", label: "Skills" },
  { slug: "automation", label: "Automation" },
  { slug: "network", label: "Network" },
  { slug: "observability", label: "Observability" },
  { slug: "hooks-extensions", label: "Hooks & Extensions" },
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
