import { useCallback, useEffect, useRef } from "react";
import { useAui } from "@assistant-ui/react";

import { SessionThread } from "@/components/assistant-ui/session-thread";
import { cancelSessionPrompt, SessionChatRuntimeProvider } from "@/systems/session";
import { Button, Spinner } from "@agh/ui";

import { ONBOARDING_AGENT_NAME, type OnboardingChatApi } from "../hooks/use-onboarding-chat";

const KICKOFF_MESSAGE =
  "Hi! Help me set up the channels and agents for my workspace. Suggest a few sensible defaults first.";

interface StepOnboardingChatProps {
  chat: OnboardingChatApi;
}

export function StepOnboardingChat({ chat }: StepOnboardingChatProps) {
  const { session, isCreating, error, ensureSession, retry } = chat;

  useEffect(() => {
    void ensureSession();
  }, [ensureSession]);

  if (error) {
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
      <OnboardingChatPanel sessionId={session.sessionId} workspaceId={session.workspaceId} />
    </SessionChatRuntimeProvider>
  );
}

function OnboardingChatPanel({
  sessionId,
  workspaceId,
}: {
  sessionId: string;
  workspaceId: string;
}) {
  const aui = useAui();
  const seededRef = useRef(false);

  useEffect(() => {
    if (seededRef.current) {
      return;
    }
    seededRef.current = true;
    // Auto-open the conversation: append (and send) a hidden kickoff turn so the onboarding
    // agent greets and starts the interview without the operator typing first.
    void aui.thread().append({ role: "user", content: [{ type: "text", text: KICKOFF_MESSAGE }] });
  }, [aui]);

  const handleCancelPrompt = useCallback(() => {
    void cancelSessionPrompt(workspaceId, sessionId).catch(() => {
      // best-effort cancel; the wizard can still finish
    });
  }, [sessionId, workspaceId]);

  return (
    <div className="flex min-h-0 flex-1 flex-col" data-testid="onboarding-step-chat">
      <SessionThread
        sessionId={sessionId}
        agentName={ONBOARDING_AGENT_NAME}
        canPrompt
        onCancelPrompt={handleCancelPrompt}
      />
    </div>
  );
}
