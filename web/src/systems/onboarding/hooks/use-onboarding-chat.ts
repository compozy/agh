import { useCallback, useMemo, useRef, useState } from "react";

import {
  fetchSession,
  SessionApiError,
  SessionNotFoundError,
  useCreateSession,
  type SessionPayload,
} from "@/systems/session";

import { useOnboardingDraftStore } from "../stores/use-onboarding-draft-store";

export const ONBOARDING_AGENT_NAME = "onboarding";

export interface OnboardingChatSession {
  sessionId: string;
  workspaceId: string;
  canPrompt: boolean;
  recoveryMessage: string | null;
  canRestart: boolean;
}

export interface OnboardingChatApi {
  session: OnboardingChatSession | null;
  kickoffSessionId: string;
  isCreating: boolean;
  error: string | null;
  ensureSession: () => Promise<void>;
  retry: () => Promise<void>;
  markKickoffSent: (sessionId: string) => void;
  reportError: (message: string) => void;
}

function clearPersistedOnboardingSession() {
  useOnboardingDraftStore.getState().patch({
    onboardingSessionId: "",
    onboardingWorkspaceId: "",
    onboardingKickoffSessionId: "",
  });
}

function canPromptOnboardingSession(session: SessionPayload): boolean {
  return session.state === "active" || session.state === "starting";
}

function isMissingPersistedSession(error: unknown): boolean {
  return (
    error instanceof SessionNotFoundError ||
    (error instanceof SessionApiError && error.status === 404)
  );
}

function onboardingRecoveryMessage(session: SessionPayload): string | null {
  if (session.state === "stopping") {
    return "This onboarding session is stopping. The history below is preserved.";
  }
  if (canPromptOnboardingSession(session)) {
    return null;
  }
  return "This onboarding session stopped before setup finished. The history below is preserved.";
}

function onboardingSessionFromPayload(
  session: SessionPayload,
  fallback: OnboardingChatSession
): OnboardingChatSession {
  const canPrompt = canPromptOnboardingSession(session);
  return {
    sessionId: session.id || fallback.sessionId,
    workspaceId: session.workspace_id || fallback.workspaceId,
    canPrompt,
    recoveryMessage: onboardingRecoveryMessage(session),
    canRestart: !canPrompt,
  };
}

export function useOnboardingChat(): OnboardingChatApi {
  const createSession = useCreateSession();
  const persistedSessionId = useOnboardingDraftStore(state => state.onboardingSessionId);
  const persistedWorkspaceId = useOnboardingDraftStore(state => state.onboardingWorkspaceId);
  const kickoffSessionId = useOnboardingDraftStore(state => state.onboardingKickoffSessionId);
  const [localSession, setLocalSession] = useState<OnboardingChatSession | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [isValidating, setIsValidating] = useState(false);
  const creatingRef = useRef(false);
  const validatingRef = useRef(false);
  const persistedSession = useMemo(
    () =>
      persistedSessionId.length > 0 && persistedWorkspaceId.length > 0
        ? { sessionId: persistedSessionId, workspaceId: persistedWorkspaceId }
        : null,
    [persistedSessionId, persistedWorkspaceId]
  );
  const session = localSession;

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
        clearPersistedOnboardingSession();
        return;
      }
      setLocalSession({
        sessionId: created.id,
        workspaceId,
        canPrompt: true,
        recoveryMessage: null,
        canRestart: false,
      });
      useOnboardingDraftStore.getState().patch({
        onboardingSessionId: created.id,
        onboardingWorkspaceId: workspaceId,
        onboardingKickoffSessionId: "",
      });
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to start the onboarding agent session."
      );
    } finally {
      creatingRef.current = false;
    }
  }, [createSession]);

  const ensureSession = useCallback(async () => {
    if (error !== null || localSession !== null || validatingRef.current) {
      return;
    }
    if (persistedSession !== null) {
      validatingRef.current = true;
      setIsValidating(true);
      setError(null);
      try {
        const existing = await fetchSession(
          persistedSession.workspaceId,
          persistedSession.sessionId
        );
        setLocalSession(
          onboardingSessionFromPayload(existing, {
            ...persistedSession,
            canPrompt: false,
            recoveryMessage: null,
            canRestart: true,
          })
        );
        return;
      } catch (err) {
        if (!isMissingPersistedSession(err)) {
          setError(
            err instanceof Error ? err.message : "Failed to verify the onboarding agent session."
          );
          return;
        }
        setLocalSession(null);
        clearPersistedOnboardingSession();
      } finally {
        validatingRef.current = false;
        setIsValidating(false);
      }
    }
    await startSession();
  }, [error, localSession, persistedSession, startSession]);

  const retry = useCallback(async () => {
    setLocalSession(null);
    clearPersistedOnboardingSession();
    await startSession();
  }, [startSession]);

  const markKickoffSent = useCallback((sessionId: string) => {
    useOnboardingDraftStore.getState().patch({ onboardingKickoffSessionId: sessionId });
  }, []);

  const reportError = useCallback((message: string) => {
    setError(message);
  }, []);

  return {
    session,
    kickoffSessionId,
    isCreating: createSession.isPending || isValidating,
    error,
    ensureSession,
    retry,
    markKickoffSent,
    reportError,
  };
}
