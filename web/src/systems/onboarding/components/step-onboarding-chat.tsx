import { useCallback, useEffect, useRef } from "react";
import { useAui, useAuiState } from "@assistant-ui/react";

import { SessionThread } from "@/components/assistant-ui/session-thread";
import { cancelSessionPrompt, SessionChatRuntimeProvider } from "@/systems/session";
import { Button, Spinner } from "@agh/ui";

import { ONBOARDING_AGENT_NAME, type OnboardingChatApi } from "../hooks/use-onboarding-chat";

const KICKOFF_MESSAGE =
  "Help me set up the channels and agents for my workspace. Suggest a few sensible defaults first.";

interface StepOnboardingChatProps {
  chat: OnboardingChatApi;
}

export function StepOnboardingChat({ chat }: StepOnboardingChatProps) {
  const {
    session,
    kickoffSessionId,
    isCreating,
    error,
    ensureSession,
    retry,
    markKickoffSent,
    reportError,
  } = chat;

  useEffect(() => {
    void ensureSession();
  }, [ensureSession]);

  if (error && !session) {
    return (
      <div className="flex min-h-0 flex-1 flex-col items-center justify-center gap-3 px-8 py-10">
        <p className="max-w-md text-center text-sm text-danger" role="alert">
          {error}
        </p>
        <Button
          variant="outline"
          size="sm"
          onClick={() => void retry()}
          disabled={isCreating}
          data-testid="onboarding-chat-retry"
        >
          {isCreating ? <Spinner /> : null}
          Try again
        </Button>
      </div>
    );
  }

  if (!session || isCreating) {
    return (
      <div className="flex min-h-0 flex-1 items-center justify-center gap-2 px-8 py-10 text-sm text-muted">
        <Spinner /> Starting your onboarding agent…
      </div>
    );
  }

  return (
    <SessionChatRuntimeProvider sessionId={session.sessionId} workspaceId={session.workspaceId}>
      <OnboardingChatPanel
        sessionId={session.sessionId}
        workspaceId={session.workspaceId}
        kickoffSessionId={kickoffSessionId}
        canPrompt={session.canPrompt}
        recoveryMessage={error ?? session.recoveryMessage}
        canRestart={session.canRestart}
        isRestarting={isCreating}
        retry={retry}
        markKickoffSent={markKickoffSent}
        reportError={reportError}
      />
    </SessionChatRuntimeProvider>
  );
}

function OnboardingChatPanel({
  sessionId,
  workspaceId,
  kickoffSessionId,
  canPrompt,
  recoveryMessage,
  canRestart,
  isRestarting,
  retry,
  markKickoffSent,
  reportError,
}: {
  sessionId: string;
  workspaceId: string;
  kickoffSessionId: string;
  canPrompt: boolean;
  recoveryMessage: string | null;
  canRestart: boolean;
  isRestarting: boolean;
  retry: () => Promise<void>;
  markKickoffSent: (sessionId: string) => void;
  reportError: (message: string) => void;
}) {
  const aui = useAui();
  const threadListItem = useAuiState(state => state.threadListItem);
  const pendingKickoffRef = useRef<string | null>(null);
  const sessionThreadReady =
    threadListItem?.remoteId === sessionId &&
    (threadListItem.status === "regular" || threadListItem.status === "archived");

  useEffect(() => {
    if (!canPrompt || !sessionThreadReady) {
      return;
    }
    if (kickoffSessionId === sessionId || pendingKickoffRef.current === sessionId) {
      return;
    }
    pendingKickoffRef.current = sessionId;
    // Seed the conversation so the onboarding agent starts without the operator typing first.
    let cancelled = false;
    const sendKickoff = async () => {
      try {
        await aui.thread().append({
          role: "user",
          content: [{ type: "text", text: KICKOFF_MESSAGE }],
        });
        if (!cancelled) {
          markKickoffSent(sessionId);
        }
      } catch (error) {
        if (pendingKickoffRef.current === sessionId) {
          pendingKickoffRef.current = null;
        }
        if (!cancelled) {
          reportError(
            error instanceof Error
              ? `Failed to send the onboarding kickoff message. ${error.message}`
              : "Failed to send the onboarding kickoff message."
          );
        }
      }
    };
    void sendKickoff();
    return () => {
      cancelled = true;
    };
  }, [
    aui,
    canPrompt,
    kickoffSessionId,
    markKickoffSent,
    reportError,
    sessionId,
    sessionThreadReady,
  ]);

  const handleCancelPrompt = useCallback(() => {
    void cancelSessionPrompt(workspaceId, sessionId).catch(error => {
      reportError(
        error instanceof Error
          ? `Failed to cancel the onboarding prompt. ${error.message}`
          : "Failed to cancel the onboarding prompt."
      );
    });
  }, [reportError, sessionId, workspaceId]);

  return (
    <div className="flex min-h-0 flex-1 flex-col" data-testid="onboarding-step-chat">
      {recoveryMessage ? (
        <div className="border-b border-line bg-canvas-soft px-8 py-3">
          <div className="flex w-full min-w-0 flex-wrap items-center justify-between gap-3">
            <p
              className="min-w-0 flex-1 text-sm leading-6 text-muted"
              data-testid="onboarding-chat-status"
              role="status"
            >
              {recoveryMessage}
            </p>
            {canRestart ? (
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => void retry()}
                disabled={isRestarting}
                data-testid="onboarding-chat-restart"
              >
                {isRestarting ? <Spinner className="size-3" /> : null}
                Start new chat
              </Button>
            ) : null}
          </div>
        </div>
      ) : null}
      <SessionThread
        sessionId={sessionId}
        agentName={ONBOARDING_AGENT_NAME}
        canPrompt={canPrompt}
        contentInset="px-8"
        onCancelPrompt={handleCancelPrompt}
      />
    </div>
  );
}
