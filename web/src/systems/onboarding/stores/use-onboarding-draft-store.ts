import { create } from "zustand";
import { createJSONStorage, persist } from "zustand/middleware";

export type OnboardingAuthMode = "native_cli" | "bound_secret";

export interface OnboardingWorkspaceDraft {
  path: string;
  name: string;
}

export interface OnboardingDraftState {
  step: number;
  maxStep: number;
  provider: string;
  model: string;
  reasoning: string;
  authMode: OnboardingAuthMode;
  envVar: string;
  apiKey: string;
  workspaces: OnboardingWorkspaceDraft[];
  onboardingSessionId: string;
  onboardingWorkspaceId: string;
  onboardingKickoffSessionId: string;
}

export interface OnboardingDraftStore extends OnboardingDraftState {
  setStep: (step: number) => void;
  patch: (patch: Partial<OnboardingDraftState>) => void;
  addWorkspace: (workspace: OnboardingWorkspaceDraft) => void;
  removeWorkspace: (path: string) => void;
  reset: () => void;
}

const initialState: OnboardingDraftState = {
  step: 1,
  maxStep: 1,
  provider: "",
  model: "",
  reasoning: "",
  authMode: "native_cli",
  envVar: "",
  apiKey: "",
  workspaces: [],
  onboardingSessionId: "",
  onboardingWorkspaceId: "",
  onboardingKickoffSessionId: "",
};

const storageKey = "agh:onboarding:draft:v2";

const draftStorage = createJSONStorage<OnboardingDraftState>(() => {
  if (typeof window === "undefined") {
    throw new Error("localStorage is unavailable");
  }
  return window.localStorage;
});

export const useOnboardingDraftStore = create<OnboardingDraftStore>()(
  persist(
    set => ({
      ...initialState,
      setStep: step => set(state => ({ step, maxStep: Math.max(state.maxStep, step) })),
      patch: patch => set(patch),
      addWorkspace: workspace =>
        set(state =>
          state.workspaces.some(item => item.path === workspace.path)
            ? state
            : { workspaces: [...state.workspaces, workspace] }
        ),
      removeWorkspace: path =>
        set(state => ({ workspaces: state.workspaces.filter(item => item.path !== path) })),
      reset: () => set({ ...initialState }),
    }),
    {
      name: storageKey,
      storage: draftStorage,
      // The API key is intentionally NOT persisted — it lives only in memory.
      partialize: state => ({
        step: state.step,
        maxStep: state.maxStep,
        provider: state.provider,
        model: state.model,
        reasoning: state.reasoning,
        authMode: state.authMode,
        envVar: state.envVar,
        apiKey: "",
        workspaces: state.workspaces,
        onboardingSessionId: state.onboardingSessionId,
        onboardingWorkspaceId: state.onboardingWorkspaceId,
        onboardingKickoffSessionId: state.onboardingKickoffSessionId,
      }),
    }
  )
);
