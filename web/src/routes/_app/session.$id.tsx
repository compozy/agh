import { useEffect } from "react";
import { AlertCircle, Loader2 } from "lucide-react";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { toast } from "sonner";

import { SessionThread } from "@/components/assistant-ui/session-thread";
import { useSessionPageControls } from "@/hooks/routes/use-session-page-controls";
import { ChatHeader } from "@/systems/session/components/chat-header";
import { SessionChatRuntimeProvider } from "@/systems/session/components/session-chat-runtime-provider";
import { SessionInspector } from "@/systems/session/components/session-inspector";
import { useSession } from "@/systems/session/hooks/use-sessions";
import type { SessionPayload } from "@/systems/session/types";
import { useWorkspaces } from "@/systems/workspace";

export const Route = createFileRoute("/_app/session/$id")({
  component: SessionPage,
});

function SessionPageContent({
  sessionId,
  session,
  workspaceName,
  onDeleteSuccess,
}: {
  sessionId: string;
  session: SessionPayload;
  workspaceName?: string;
  onDeleteSuccess: () => void;
}) {
  const {
    canClear,
    canPrompt,
    handleCancelPrompt,
    handleClear,
    handleDelete,
    handleResume,
    handleStop,
    isClearing,
    isDeleting,
    isResuming,
    isStopping,
    messages,
  } = useSessionPageControls(sessionId, session.state, { onDeleteSuccess });

  return (
    <div className="flex min-h-0 min-w-0 flex-1 overflow-hidden">
      <div className="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden">
        <ChatHeader
          session={session}
          onDelete={handleDelete}
          onStop={handleStop}
          onResume={handleResume}
          isDeleting={isDeleting}
          isStopping={isStopping}
          isResuming={isResuming}
          workspaceName={workspaceName}
        />
        <SessionThread
          sessionId={sessionId}
          agentName={session.agent_name}
          canPrompt={canPrompt}
          onCancelPrompt={handleCancelPrompt}
          onClearConversation={handleClear}
          canClearConversation={canClear}
          isClearingConversation={isClearing}
        />
      </div>
      <SessionInspector messages={messages} />
    </div>
  );
}

function SessionPage() {
  const { id } = Route.useParams();
  const navigate = useNavigate();
  const { data: session, isLoading, error } = useSession(id);
  const { data: workspaces } = useWorkspaces();

  useEffect(() => {
    if (error?.message?.includes("not found")) {
      toast.error("Session not found");
      navigate({ to: "/" });
    }
  }, [error, navigate]);

  if (isLoading) {
    return (
      <div className="flex flex-1 items-center justify-center">
        <Loader2 className="size-5 animate-spin text-[color:var(--color-text-tertiary)]" />
      </div>
    );
  }

  if (!session) {
    return (
      <div className="flex flex-1 items-center justify-center">
        <div className="flex flex-col items-center gap-2 text-center">
          <AlertCircle className="size-6 text-[color:var(--color-danger)]" />
          <p className="text-sm text-[color:var(--color-text-tertiary)]">
            {error?.message ?? "Session not found"}
          </p>
        </div>
      </div>
    );
  }

  const workspaceName = workspaces?.find(workspace => workspace.id === session.workspace_id)?.name;

  return (
    <SessionChatRuntimeProvider key={id} sessionId={id} workspaceId={session.workspace_id}>
      <SessionPageContent
        sessionId={id}
        session={session}
        workspaceName={workspaceName}
        onDeleteSuccess={() => {
          void navigate({ to: "/" });
        }}
      />
    </SessionChatRuntimeProvider>
  );
}
