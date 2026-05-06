import { useEffect, useMemo } from "react";
import { AlertCircle, Loader2 } from "lucide-react";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { toast } from "sonner";

import { SessionThread } from "@/components/assistant-ui/session-thread";
import { useSessionPageControls } from "@/hooks/routes/use-session-page-controls";
import {
  ChatHeader,
  SessionChatRuntimeProvider,
  SessionInspector,
  SessionResumeFailure,
  useSession,
  useSessionLedger,
  type InspectorMemoryState,
  type SessionPayload,
} from "@/systems/session";
import { useSessionVaultSecrets } from "@/systems/vault";
import { useWorkspaces } from "@/systems/workspace";

export const Route = createFileRoute("/_app/agents/$name/sessions/$id")({
  component: SessionPage,
});

function SessionPageContent({
  agentName,
  sessionId,
  session,
  workspaceName,
  onDeleteSuccess,
}: SessionPageContentProps) {
  const {
    canClear,
    canPrompt,
    handleCancelPrompt,
    handleClear,
    handleDelete,
    handleDismissResumeFailure,
    handleResume,
    handleStop,
    isClearing,
    isDeleting,
    isResuming,
    isStopping,
    messages,
    resumeFailure,
  } = useSessionPageControls(sessionId, session.state, { onDeleteSuccess });
  const sessionVault = useSessionVaultSecrets(sessionId);
  const ledgerEnabled = session.state === "stopped";
  const sessionLedger = useSessionLedger(sessionId, { enabled: ledgerEnabled });
  const inspectorMemory = useMemo<InspectorMemoryState>(
    () => ({
      ledger: sessionLedger.data ?? null,
      isLoading: sessionLedger.isLoading,
      error: sessionLedger.error,
    }),
    [sessionLedger.data, sessionLedger.isLoading, sessionLedger.error]
  );

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
        {resumeFailure ? (
          <SessionResumeFailure
            agentName={resumeFailure.providerUnavailable?.agentName ?? agentName}
            isRetrying={isResuming}
            message={resumeFailure.message}
            missingProvider={resumeFailure.providerUnavailable?.missingProvider ?? null}
            onDismiss={handleDismissResumeFailure}
            onRetry={handleResume}
            sessionId={sessionId}
          />
        ) : null}
        <SessionThread
          sessionId={sessionId}
          agentName={agentName}
          canPrompt={canPrompt}
          onCancelPrompt={handleCancelPrompt}
          onClearConversation={handleClear}
          canClearConversation={canClear}
          isClearingConversation={isClearing}
        />
      </div>
      <SessionInspector
        messages={messages}
        sessionId={sessionId}
        memory={inspectorMemory}
        vaultSecrets={sessionVault.data ?? []}
        vaultIsLoading={sessionVault.isLoading}
        vaultError={sessionVault.error}
      />
    </div>
  );
}

interface SessionPageContentProps {
  agentName: string;
  sessionId: string;
  session: SessionPayload;
  workspaceName?: string;
  onDeleteSuccess: () => void;
}

export function SessionPage() {
  const { name, id } = Route.useParams();
  const navigate = useNavigate();
  const { data: session, isLoading, error } = useSession(id);
  const { data: workspaces } = useWorkspaces();

  useEffect(() => {
    if (error?.message?.includes("not found")) {
      toast.error("Session not found");
      void navigate({ to: "/agents/$name", params: { name }, replace: true });
    }
  }, [error, navigate, name]);

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
  const resolvedAgentName = session.agent_name ?? name;

  return (
    <SessionChatRuntimeProvider key={id} sessionId={id} workspaceId={session.workspace_id}>
      <SessionPageContent
        agentName={resolvedAgentName}
        sessionId={id}
        session={session}
        workspaceName={workspaceName}
        onDeleteSuccess={() => {
          void navigate({ to: "/agents/$name", params: { name: resolvedAgentName } });
        }}
      />
    </SessionChatRuntimeProvider>
  );
}
