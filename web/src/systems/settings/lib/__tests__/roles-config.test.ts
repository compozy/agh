import { describe, expect, it } from "vitest";

import {
  rolesStatusFixture,
  settingsRolesConfigFixture,
  settingsRolesConfigWithFallbackFixture,
} from "../../mocks/roles-fixtures";
import {
  addFallbackEntry,
  applyRoleFieldEdit,
  removeFallbackEntry,
  ROLE_ORDER,
  rolesConfigPath,
  updateFallbackEntry,
} from "../roles-config";
import { buildRolesViewModel } from "../roles-view-model";
import { collectRoleValidationErrors, fallbackFieldId } from "../roles-validation";

describe("rolesConfigPath", () => {
  it("Should map a role field to the exact roles.<role>.<field> config path", () => {
    expect(rolesConfigPath("auto_title", "model")).toBe("roles.auto_title.model");
    expect(rolesConfigPath("memory_controller", "timeout")).toBe("roles.memory_controller.timeout");
  });

  it("Should map the fallback chain to the array-level roles.<role>.fallback_chain path", () => {
    expect(rolesConfigPath("dream", "fallback_chain")).toBe("roles.dream.fallback_chain");
  });
});

describe("applyRoleFieldEdit", () => {
  it("Should set one scalar field immutably without touching other roles or the original", () => {
    const next = applyRoleFieldEdit(
      settingsRolesConfigFixture,
      "auto_title",
      "model",
      "claude-haiku-4-5"
    );

    expect(next.auto_title.model).toBe("claude-haiku-4-5");
    expect(settingsRolesConfigFixture.auto_title.model).toBe("");
    expect(next.dream).toEqual(settingsRolesConfigFixture.dream);
    expect(next).not.toBe(settingsRolesConfigFixture);
  });

  it("Should apply boolean and numeric edits by kind", () => {
    const toggled = applyRoleFieldEdit(settingsRolesConfigFixture, "coordinator", "enabled", true);
    expect(toggled.coordinator.enabled).toBe(true);
    const bumped = applyRoleFieldEdit(settingsRolesConfigFixture, "coordinator", "max_children", 3);
    expect(bumped.coordinator.max_children).toBe(3);
  });
});

describe("fallback chain operations", () => {
  it("Should append an empty entry immutably", () => {
    const next = addFallbackEntry(settingsRolesConfigFixture, "dream");
    expect(next.dream.fallback_chain).toHaveLength(1);
    expect(next.dream.fallback_chain[0]).toEqual({ provider: "", model: "", reasoning_effort: "" });
    expect(settingsRolesConfigFixture.dream.fallback_chain).toHaveLength(0);
  });

  it("Should remove the entry at the given index", () => {
    const next = removeFallbackEntry(settingsRolesConfigWithFallbackFixture, "dream", 0);
    expect(next.dream.fallback_chain).toHaveLength(1);
    expect(next.dream.fallback_chain[0].provider).toBe("openai");
  });

  it("Should update a single entry field immutably", () => {
    const next = updateFallbackEntry(
      settingsRolesConfigWithFallbackFixture,
      "dream",
      1,
      "provider",
      "google"
    );
    expect(next.dream.fallback_chain[1].provider).toBe("google");
    expect(settingsRolesConfigWithFallbackFixture.dream.fallback_chain[1].provider).toBe("openai");
  });
});

describe("buildRolesViewModel", () => {
  it("Should return the six roles in fixed product order regardless of API order", () => {
    const models = buildRolesViewModel(rolesStatusFixture.roles, settingsRolesConfigFixture);
    expect(models.map(model => model.role)).toEqual([...ROLE_ORDER]);
  });

  it("Should flatten draft values and preserve null effective values without fabrication", () => {
    const models = buildRolesViewModel(rolesStatusFixture.roles, settingsRolesConfigFixture);
    const controller = models.find(model => model.role === "memory_controller");
    const dream = models.find(model => model.role === "dream");

    expect(controller?.values.model).toBe("anthropic/claude-haiku-4");
    expect(controller?.effective.timeout).toBe("250ms");
    expect(dream?.effective.model).toBeNull();
  });
});

describe("collectRoleValidationErrors", () => {
  it("Should report required provider/model for an empty fallback entry, provider first", () => {
    const config = addFallbackEntry(settingsRolesConfigFixture, "dream");
    const errors = collectRoleValidationErrors(config);

    expect(errors[0].id).toBe(fallbackFieldId("dream", 0, "provider"));
    expect(errors[1].id).toBe(fallbackFieldId("dream", 0, "model"));
  });

  it("Should accept a fully specified fallback entry", () => {
    expect(collectRoleValidationErrors(settingsRolesConfigWithFallbackFixture)).toHaveLength(0);
  });
});
