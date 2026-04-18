import { describe, expect, it } from "vitest";

import {
  findSettingsSection,
  SETTINGS_ROOT_PATH,
  SETTINGS_SECTIONS,
  settingsSectionPath,
} from "./sections";

describe("settings sections metadata", () => {
  it("mirrors the Paper-mapped screen order", () => {
    expect(SETTINGS_SECTIONS.map(section => section.slug)).toEqual([
      "general",
      "providers",
      "mcp-servers",
      "environments",
      "memory",
      "skills",
      "automation",
      "network",
      "observability",
      "hooks-extensions",
    ]);
  });

  it("provides nested paths rooted under the settings shell", () => {
    expect(SETTINGS_ROOT_PATH).toBe("/settings");
    expect(settingsSectionPath("providers")).toBe("/settings/providers");
    expect(settingsSectionPath("hooks-extensions")).toBe("/settings/hooks-extensions");
  });

  it("looks sections up by slug", () => {
    expect(findSettingsSection("memory")?.label).toBe("Memory");
    expect(findSettingsSection("nope")).toBeUndefined();
    expect(findSettingsSection(null)).toBeUndefined();
  });
});
