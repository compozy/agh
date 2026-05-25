import { describe, expect, it } from "vitest";

import { buildOnboardingProviderRequest } from "../provider-request";

const baseSettings = {
  command: "npx claude",
  display_name: "Claude Code",
  models: { default: "claude-sonnet-4-6", curated: [{ id: "claude-sonnet-4-6" }] },
};

describe("buildOnboardingProviderRequest", () => {
  it("persists the chosen default model and reasoning while preserving existing settings", () => {
    const body = buildOnboardingProviderRequest(baseSettings, {
      model: "claude-opus-4-7",
      reasoning: "xhigh",
      authMode: "native_cli",
      envVar: "",
      apiKey: "",
      provider: "claude",
    });
    expect(body.settings?.command).toBe("npx claude");
    expect(body.settings?.models?.default).toBe("claude-opus-4-7");
    const curated = body.settings?.models?.curated ?? [];
    const opus = curated.find(entry => entry.id === "claude-opus-4-7");
    expect(opus?.default_reasoning_effort).toBe("xhigh");
    // existing curated entry preserved
    expect(curated.some(entry => entry.id === "claude-sonnet-4-6")).toBe(true);
  });

  it("updates an existing curated entry's reasoning in place", () => {
    const body = buildOnboardingProviderRequest(baseSettings, {
      model: "claude-sonnet-4-6",
      reasoning: "high",
      authMode: "native_cli",
      envVar: "",
      apiKey: "",
      provider: "claude",
    });
    const curated = body.settings?.models?.curated ?? [];
    expect(curated).toHaveLength(1);
    expect(curated[0]?.default_reasoning_effort).toBe("high");
  });

  it("sets native_cli auth without credential slots or secrets", () => {
    const body = buildOnboardingProviderRequest(baseSettings, {
      model: "claude-opus-4-7",
      reasoning: "",
      authMode: "native_cli",
      envVar: "ANTHROPIC_API_KEY",
      apiKey: "sk-should-be-ignored",
      provider: "claude",
    });
    expect(body.settings?.auth_mode).toBe("native_cli");
    expect(body.settings?.credential_slots).toBeUndefined();
    expect(body.secrets).toBeUndefined();
  });

  it("binds an env-var reference without a secret value when no key is provided", () => {
    const body = buildOnboardingProviderRequest(baseSettings, {
      model: "claude-opus-4-7",
      reasoning: "",
      authMode: "bound_secret",
      envVar: "ANTHROPIC_API_KEY",
      apiKey: "",
      provider: "claude",
    });
    expect(body.settings?.auth_mode).toBe("bound_secret");
    expect(body.settings?.credential_slots?.[0]?.secret_ref).toBe("env:ANTHROPIC_API_KEY");
    expect(body.secrets).toBeUndefined();
  });

  it("stores a provided API key as a vault-backed secret", () => {
    const body = buildOnboardingProviderRequest(baseSettings, {
      model: "claude-opus-4-7",
      reasoning: "",
      authMode: "bound_secret",
      envVar: "ANTHROPIC_API_KEY",
      apiKey: "sk-real-key",
      provider: "claude",
    });
    expect(body.settings?.credential_slots?.[0]?.secret_ref).toBe("vault:providers/claude/api_key");
    expect(body.secrets).toEqual([
      {
        name: "api_key",
        secret_ref: "vault:providers/claude/api_key",
        kind: "api_key",
        value: "sk-real-key",
      },
    ]);
  });
});
