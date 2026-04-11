import { useCallback, useEffect, useRef } from "react";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { Loader2, AlertCircle } from "lucide-react";
import { toast } from "sonner";

import { useSession } from "@/systems/session/hooks/use-sessions";
import { useSessionChat } from "@/systems/session/hooks/use-session-chat";
import { useSessionTranscript } from "@/systems/session/hooks/use-session-transcript";
import { useSessionStore } from "@/systems/session/stores/session-store";
import { useStopSession, useResumeSession } from "@/systems/session/hooks/use-session-actions";
import { ChatHeader } from "@/systems/session/components/chat-header";
import { ChatView } from "@/systems/session/components/chat-view";
import { MessageComposer } from "@/systems/session/components/message-composer";
import { PermissionPrompt } from "@/systems/session/components/permission-prompt";
import { useWorkspaces } from "@/systems/workspace";

export const Route = createFileRoute("/_app/session/$id")({
  component: SessionPage,
});

function SessionPage() {
  const { id } = Route.useParams();
  const navigate = useNavigate();
  const hydratedSessionIdRef = useRef<string | null>(null);

  const { data: session, isLoading, error } = useSession(id);
  const { data: workspaces } = useWorkspaces();
  const messages = useSessionStore(s => s.messages);
  const isStreaming = useSessionStore(s => s.isStreaming);
  const pendingPermission = useSessionStore(s => s.pendingPermission);
  const activeSessionId = useSessionStore(s => s.activeSessionId);

  const {
    transcriptMessages,
    isLoadingTranscript,
    error: transcriptError,
  } = useSessionTranscript(id);
  const canPrompt = session?.state === "active";
  const { sendMessage, status } = useSessionChat({ sessionId: id });
  const stopMutation = useStopSession();
  const resumeMutation = useResumeSession();

  // Session switch: reset the shell immediately for the target session.
  useEffect(() => {
    hydratedSessionIdRef.current = null;
    useSessionStore.setState({
      activeSessionId: id,
      messages: [],
      isStreaming: false,
      pendingPermission: null,
    });
  }, [id]);

  // Transcript hydration: apply the canonical transcript exactly once per session id.
  useEffect(() => {
    if (!transcriptMessages) return;
    if (activeSessionId !== id) return;
    if (hydratedSessionIdRef.current === id) return;

    useSessionStore.getState().setActiveSession(id, transcriptMessages);
    hydratedSessionIdRef.current = id;
  }, [activeSessionId, id, transcriptMessages]);

  // Handle 404 — navigate away with toast
  useEffect(() => {
    if (error?.message?.includes("not found")) {
      toast.error("Session not found");
      navigate({ to: "/" });
    }
  }, [error, navigate]);

  const handlePermissionResolved = useCallback(() => {
    useSessionStore.getState().setPendingPermission(null);
  }, []);

  const isDisabled =
    !canPrompt || isStreaming || status === "submitted" || pendingPermission !== null;
  const workspaceName = workspaces?.find(workspace => workspace.id === session?.workspace_id)?.name;

  if (isLoading || isLoadingTranscript) {
    return (
      <div className="flex flex-1 items-center justify-center">
        <Loader2 className="size-5 animate-spin text-[color:var(--color-text-tertiary)]" />
      </div>
    );
  }

  if (error || transcriptError || !session) {
    return (
      <div className="flex flex-1 items-center justify-center">
        <div className="flex flex-col items-center gap-2 text-center">
          <AlertCircle className="size-6 text-[color:var(--color-danger)]" />
          <p className="text-sm text-[color:var(--color-text-tertiary)]">
            {error?.message ?? transcriptError?.message ?? "Session not found"}
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="flex flex-1 flex-col overflow-hidden">
      <ChatHeader
        session={session}
        onStop={() => stopMutation.mutate(id)}
        onResume={() => resumeMutation.mutate(id)}
        workspaceName={workspaceName}
      />
      <ChatView messages={messages} isStreaming={isStreaming} agentName={session.agent_name} />
      {pendingPermission && (
        <PermissionPrompt
          permission={pendingPermission}
          sessionId={id}
          onResolved={handlePermissionResolved}
        />
      )}
      {canPrompt && <MessageComposer onSend={sendMessage} disabled={isDisabled} />}
    </div>
  );
}
