import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { AlertCircle, MessageCircle, Trash2 } from "lucide-react";
import { useEffect } from "react";
import { toast } from "sonner";

import {
  Button,
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  Spinner,
} from "@agh/ui";

import { SessionThread } from "@/components/assistant-ui/session-thread";
import { useSessionDetailPage } from "@/hooks/routes/use-session-detail-page";
import {
  SessionChatRuntimeProvider,
  SessionInspector,
  SessionResumeFailure,
  useSession,
  type SessionPayload,
} from "@/systems/session";
import type { TopbarRouteContext } from "@/types/topbar";

export const Route = createFileRoute("/_app/agents/$name/sessions/$id")({
  beforeLoad: ({ params }): { topbar: TopbarRouteContext } => ({
    topbar: { title: `${params.name} · Session`, icon: MessageCircle },
  }),
  component: SessionPage,
});

function SessionPageContent({
  agentName,
  sessionId,
  session,
  onDeleteSuccess,
}: SessionPageContentProps) {
  const page = useSessionDetailPage({ sessionId, session, onDeleteSuccess });
  const { controls, inspectorMemory, sessionVault, deleteDialog } = page;

  return (
    <div className="flex min-h-0 min-w-0 flex-1 overflow-hidden">
      <div className="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden">
        {controls.resumeFailure ? (
          <SessionResumeFailure
            agentName={controls.resumeFailure.providerUnavailable?.agentName ?? agentName}
            isRetrying={controls.isResuming}
            message={controls.resumeFailure.message}
            missingProvider={controls.resumeFailure.providerUnavailable?.missingProvider ?? null}
            onDismiss={controls.handleDismissResumeFailure}
            onRetry={controls.handleResume}
            sessionId={sessionId}
          />
        ) : null}
        <SessionThread
          sessionId={sessionId}
          agentName={agentName}
          canPrompt={controls.canPrompt}
          onCancelPrompt={controls.handleCancelPrompt}
          onClearConversation={controls.handleClear}
          canClearConversation={controls.canClear}
          isClearingConversation={controls.isClearing}
        />
      </div>
      <SessionInspector
        messages={controls.messages}
        sessionId={sessionId}
        memory={inspectorMemory}
        vaultSecrets={sessionVault.data ?? []}
        vaultIsLoading={sessionVault.isLoading}
        vaultError={sessionVault.error}
      />
      <SessionDeleteDialog
        open={deleteDialog.open}
        onOpenChange={deleteDialog.setOpen}
        session={session}
        isDeleting={controls.isDeleting}
        onConfirm={deleteDialog.confirmDelete}
      />
    </div>
  );
}

interface SessionDeleteDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  session: SessionPayload;
  isDeleting: boolean;
  onConfirm: () => void;
}

function SessionDeleteDialog({
  open,
  onOpenChange,
  session,
  isDeleting,
  onConfirm,
}: SessionDeleteDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent showCloseButton={!isDeleting} className="max-w-md" data-testid="delete-dialog">
        <DialogHeader>
          <DialogTitle>Delete session</DialogTitle>
          <DialogDescription>
            This permanently removes <strong>{session.name?.trim() || session.id}</strong>,
            including its transcript and history, and removes it from the session list.
          </DialogDescription>
        </DialogHeader>
        <DialogFooter className="gap-2">
          <Button
            type="button"
            variant="ghost"
            onClick={() => onOpenChange(false)}
            disabled={isDeleting}
            data-testid="delete-dialog-cancel"
          >
            Cancel
          </Button>
          <Button
            type="button"
            variant="destructive"
            onClick={onConfirm}
            disabled={isDeleting}
            data-testid="delete-dialog-confirm"
          >
            {isDeleting ? (
              <>
                <Spinner className="size-3" />
                Deleting
              </>
            ) : (
              <>
                <Trash2 className="size-3" />
                Delete session
              </>
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

interface SessionPageContentProps {
  agentName: string;
  sessionId: string;
  session: SessionPayload;
  onDeleteSuccess: () => void;
}

export function SessionPage() {
  const { name, id } = Route.useParams();
  const navigate = useNavigate();
  const { data: session, isLoading, error } = useSession(id);

  useEffect(() => {
    if (error?.message?.includes("not found")) {
      toast.error("Session not found");
      void navigate({ to: "/agents/$name", params: { name }, replace: true });
    }
  }, [error, navigate, name]);

  if (isLoading) {
    return (
      <div className="flex flex-1 items-center justify-center">
        <Spinner className="size-5 text-subtle" />
      </div>
    );
  }

  if (!session) {
    return (
      <div className="flex flex-1 items-center justify-center">
        <div className="flex flex-col items-center gap-2 text-center">
          <AlertCircle className="size-6 text-danger" />
          <p className="text-sm text-subtle">{error?.message ?? "Session not found"}</p>
        </div>
      </div>
    );
  }

  const resolvedAgentName = session.agent_name ?? name;

  return (
    <SessionChatRuntimeProvider key={id} sessionId={id} workspaceId={session.workspace_id}>
      <SessionPageContent
        agentName={resolvedAgentName}
        sessionId={id}
        session={session}
        onDeleteSuccess={() => {
          void navigate({ to: "/agents/$name", params: { name: resolvedAgentName } });
        }}
      />
    </SessionChatRuntimeProvider>
  );
}
