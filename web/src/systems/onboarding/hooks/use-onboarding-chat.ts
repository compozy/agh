import { useCallback, useRef, useState } from "react";

import { useCreateSession } from "@/systems/session";

import { useOnboardingDraftStore } from "../stores/use-onboarding-draft-store";

export const ONBOARDING_AGENT_NAME = "onboarding";

export interface OnboardingChatSession {
  sessionId: string;
  workspaceId: string;
}

export interface OnboardingChatApi {
  session: OnboardingChatSession | null;
  isCreating: boolean;
  error: string | null;
  ensureSession: () => Promise<void>;
  retry: () => Promise<void>;
}

export function useOnboardingChat(): OnboardingChatApi {
  const createSession = useCreateSession();
  const [session, setSession] = useState<OnboardingChatSession | null>(null);
  const [error, setError] = useState<string | null>(null);
  const creatingRef = useRef(false);

  const startSession = useCallback(async () => {
    if (creatingRef.current) {
      return;
    }
    const draft = useOnboardingDraftStore.getState();
    const firstWorkspace = draft.workspaces[0];
    if (!firstWorkspace) {
      setError("Add a workspace before starting the onboarding chat.");
      return;
    }
    creatingRef.current = true;
    setError(null);
    try {
      const created = await createSession.mutateAsync({
        agent_name: ONBOARDING_AGENT_NAME,
        workspace_path: firstWorkspace.path,
        ...(draft.provider.length > 0 ? { provider: draft.provider } : {}),
        ...(draft.model.length > 0 ? { model: draft.model } : {}),
        ...(draft.reasoning.length > 0 ? { reasoning_effort: draft.reasoning } : {}),
      });
      const workspaceId = created.workspace_id ?? "";
      if (workspaceId.length === 0) {
        setError("The onboarding session was created without a workspace.");
        return;
      }
      setSession({ sessionId: created.id, workspaceId });
      useOnboardingDraftStore.getState().patch({ chatStarted: true });
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to start the onboarding agent session."
      );
    } finally {
      creatingRef.current = false;
    }
  }, [createSession]);

  const ensureSession = useCallback(async () => {
    if (session !== null) {
      return;
    }
    await startSession();
  }, [session, startSession]);

  const retry = useCallback(async () => {
    setSession(null);
    await startSession();
  }, [startSession]);

  return {
    session,
    isCreating: createSession.isPending,
    error,
    ensureSession,
    retry,
  };
}
