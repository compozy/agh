import type { HistoryState } from "@tanstack/react-router";

export interface SessionReturnNavigationState {
  sessionId: string;
  workspaceId: string;
}

type SessionReturnHistoryState = HistoryState & {
  sessionReturn: SessionReturnNavigationState;
};

export function createSessionReturnHistoryState(
  sessionId: string,
  workspaceId: string
): SessionReturnHistoryState {
  return {
    sessionReturn: {
      sessionId,
      workspaceId,
    },
  };
}

export function sessionReturnWorkspaceIdFromState(
  state: HistoryState,
  sessionId: string
): string | undefined {
  if (!("sessionReturn" in state) || !isSessionReturnNavigationState(state.sessionReturn)) {
    return undefined;
  }
  const intent = state.sessionReturn;
  if (intent.sessionId !== sessionId) return undefined;
  const workspaceId = intent.workspaceId.trim();
  return workspaceId || undefined;
}

function isSessionReturnNavigationState(value: unknown): value is SessionReturnNavigationState {
  if (!value || typeof value !== "object") return false;
  return (
    "sessionId" in value &&
    typeof value.sessionId === "string" &&
    "workspaceId" in value &&
    typeof value.workspaceId === "string"
  );
}
