import { describe, expect, it } from "vitest";

import type { RuntimeProviderOption } from "@/systems/runtime";

import {
  buildCreateAgentParams,
  createDefaultAgentCreateDraft,
  validateAgentCreateDraft,
  type AgentCreateDialogDraft,
  type AgentCreateValidationContext,
} from "../agent-create-draft";

const providerOptions: readonly RuntimeProviderOption[] = [{ id: "codex", name: "Codex" }];

const context: AgentCreateValidationContext = {
  hasActiveWorkspace: false,
  providerOptions,
  providersError: null,
  providersLoading: false,
};

/** A globally-scoped draft that is valid apart from the reasoning field under test. */
function baseDraft(reasoningEffort: string): AgentCreateDialogDraft {
  return {
    ...createDefaultAgentCreateDraft(false),
    scope: "global",
    name: "release-captain",
    provider: "codex",
    prompt: "Own release readiness.",
    reasoningEffort: reasoningEffort as AgentCreateDialogDraft["reasoningEffort"],
  };
}

describe("agent-create-draft reasoning effort validation", () => {
  it("Should flag an off-contract reasoning effort as a runtime-step field error", () => {
    const validation = validateAgentCreateDraft(baseDraft("ultra"), context);

    expect(validation.fields.reasoningEffort).toBe("Choose a valid reasoning effort.");
    expect(validation.stepValidity.runtime).toBe(false);
    expect(validation.canSubmit).toBe(false);
  });

  it("Should return null from buildCreateAgentParams for an off-contract reasoning effort", () => {
    expect(buildCreateAgentParams(baseDraft("ultra"), null, context)).toBeNull();
  });

  it("Should accept the canonical empty effort as provider default and omit it from the request", () => {
    const validation = validateAgentCreateDraft(baseDraft(""), context);
    expect(validation.fields.reasoningEffort).toBeUndefined();
    expect(validation.stepValidity.runtime).toBe(true);

    const params = buildCreateAgentParams(baseDraft(""), null, context);
    expect(params).not.toBeNull();
    expect(params?.agent).not.toHaveProperty("reasoning_effort");
  });

  it("Should carry a canonical reasoning effort through to the built request", () => {
    const params = buildCreateAgentParams(baseDraft("high"), null, context);
    expect(params?.agent.reasoning_effort).toBe("high");
  });
});
