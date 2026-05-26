import { beforeEach, describe, expect, it } from "vitest";

import { useOnboardingDraftStore } from "../use-onboarding-draft-store";

const STORAGE_KEY = "agh:onboarding:draft:v2";

function persistedState(): Record<string, unknown> {
  const raw = window.localStorage.getItem(STORAGE_KEY);
  if (!raw) {
    return {};
  }
  return (JSON.parse(raw).state ?? {}) as Record<string, unknown>;
}

describe("useOnboardingDraftStore", () => {
  beforeEach(() => {
    window.localStorage.clear();
    useOnboardingDraftStore.getState().reset();
  });

  it("grows maxStep but never shrinks it as the user moves between steps", () => {
    const store = useOnboardingDraftStore.getState();
    store.setStep(3);
    expect(useOnboardingDraftStore.getState().maxStep).toBe(3);
    store.setStep(2);
    expect(useOnboardingDraftStore.getState().step).toBe(2);
    expect(useOnboardingDraftStore.getState().maxStep).toBe(3);
  });

  it("adds workspaces without duplicating the same path", () => {
    const store = useOnboardingDraftStore.getState();
    store.addWorkspace({ path: "/a", name: "a" });
    store.addWorkspace({ path: "/a", name: "a-again" });
    store.addWorkspace({ path: "/b", name: "b" });
    expect(useOnboardingDraftStore.getState().workspaces).toEqual([
      { path: "/a", name: "a" },
      { path: "/b", name: "b" },
    ]);
  });

  it("removes a workspace by path", () => {
    const store = useOnboardingDraftStore.getState();
    store.addWorkspace({ path: "/a", name: "a" });
    store.addWorkspace({ path: "/b", name: "b" });
    store.removeWorkspace("/a");
    expect(useOnboardingDraftStore.getState().workspaces).toEqual([{ path: "/b", name: "b" }]);
  });

  it("never persists the API key to local storage", () => {
    useOnboardingDraftStore.getState().patch({ apiKey: "sk-super-secret", provider: "claude" });
    const persisted = persistedState();
    expect(persisted.provider).toBe("claude");
    expect(persisted.apiKey).toBe("");
    expect(useOnboardingDraftStore.getState().apiKey).toBe("sk-super-secret");
  });

  it("persists onboarding chat session identity without persisting secrets", () => {
    useOnboardingDraftStore.getState().patch({
      apiKey: "sk-super-secret",
      onboardingSessionId: "sess_onboarding",
      onboardingWorkspaceId: "ws_alpha",
      onboardingKickoffSessionId: "sess_onboarding",
    });

    const persisted = persistedState();
    expect(persisted.apiKey).toBe("");
    expect(persisted.onboardingSessionId).toBe("sess_onboarding");
    expect(persisted.onboardingWorkspaceId).toBe("ws_alpha");
    expect(persisted.onboardingKickoffSessionId).toBe("sess_onboarding");
  });

  it("resets to the initial draft", () => {
    const store = useOnboardingDraftStore.getState();
    store.patch({ provider: "claude", model: "opus", apiKey: "x" });
    store.addWorkspace({ path: "/a", name: "a" });
    store.reset();
    const state = useOnboardingDraftStore.getState();
    expect(state.provider).toBe("");
    expect(state.model).toBe("");
    expect(state.apiKey).toBe("");
    expect(state.workspaces).toEqual([]);
    expect(state.onboardingSessionId).toBe("");
    expect(state.onboardingWorkspaceId).toBe("");
    expect(state.onboardingKickoffSessionId).toBe("");
    expect(state.step).toBe(1);
  });
});
