import type { QueryClient } from "@tanstack/react-query";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { AlertCircle, Eraser, MessageCircle, Trash2 } from "lucide-react";
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
import { useSessionWorkspaceGuard } from "@/hooks/routes/use-session-workspace-guard";
import {
  SessionChatRuntimeProvider,
  SessionInspector,
  SessionResumeFailure,
  sessionByIdOptions,
  sessionDetailOptions,
  sessionKeys,
  SessionNotFoundError,
  sessionTranscriptOptions,
  useSessionById,
  type SessionPayload,
} from "@/systems/session";
import type { TopbarRouteContext } from "@/types/topbar";
import { adoptRouteWorkspaceWhenSelectionInvalid } from "./-route-preload";

interface AgentSessionRouteLoaderData {
  workspaceId: string | null;
}

export const Route = createFileRoute("/_app/agents/$name/sessions/$id")({
  beforeLoad: ({ params }): { topbar: TopbarRouteContext } => ({
    topbar: { title: `${params.name} · Session`, icon: MessageCircle },
  }),
  loader: ({ context, params }) => {
    return prefetchAgentSessionRoute({
      queryClient: context.queryClient,
      sessionId: params.id,
    });
  },
  pendingComponent: SessionRouteLoading,
  component: SessionPage,
});

export async function prefetchAgentSessionRoute({
  queryClient,
  sessionId,
}: {
  queryClient: QueryClient;
  sessionId: string;
}): Promise<AgentSessionRouteLoaderData> {
  const workspaceId = await resolveSessionRouteWorkspace(queryClient, sessionId);
  if (!workspaceId) {
    return { workspaceId: null };
  }

  await adoptRouteWorkspaceWhenSelectionInvalid(queryClient, workspaceId);
  await Promise.allSettled([
    queryClient.ensureQueryData(sessionDetailOptions(workspaceId, sessionId)),
    queryClient.ensureInfiniteQueryData(sessionTranscriptOptions(workspaceId, sessionId)),
  ]);

  return { workspaceId };
}

async function resolveSessionRouteWorkspace(
  queryClient: QueryClient,
  sessionId: string
): Promise<string | null> {
  const cachedWorkspaceId = normalizeWorkspaceId(
    queryClient.getQueryData<SessionPayload>(sessionKeys.byId(sessionId))?.workspace_id
  );
  if (cachedWorkspaceId) {
    return cachedWorkspaceId;
  }
  try {
    const session = await queryClient.ensureQueryData(sessionByIdOptions(sessionId));
    return normalizeWorkspaceId(session.workspace_id);
  } catch (error) {
    if (error instanceof SessionNotFoundError) {
      return null;
    }
    throw error;
  }
}

function normalizeWorkspaceId(value: string | null | undefined): string | null {
  const trimmed = value?.trim();
  return trimmed ? trimmed : null;
}

function SessionPageContent({
  agentName,
  sessionId,
  session,
  onDeleteSuccess,
}: SessionPageContentProps) {
  const page = useSessionDetailPage({ sessionId, session, onDeleteSuccess });
  const { controls, inspectorMemory, inspectorUsage, sessionVault, deleteDialog, clearDialog } =
    page;

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
          workspaceId={session.workspace_id}
          agentName={agentName}
          canPrompt={controls.canPrompt}
          onCancelPrompt={controls.handleStop}
          onQueuePrompt={controls.handleQueuePrompt}
          onInterruptPrompt={controls.handleInterruptPrompt}
          onSteerPrompt={controls.handleSteerPrompt}
          isBusyInputPending={controls.isBusyInputPending}
          isSessionRunning={controls.isSessionRunning}
          allowBusyInput={controls.allowBusyInput}
          queuedPrompts={controls.queuedPrompts}
          onRemoveQueuedPrompt={controls.handleRemoveQueuedPrompt}
          onSteerQueuedPrompt={controls.handleSteerQueuedPrompt}
        />
      </div>
      <SessionInspector
        messages={controls.messages}
        sessionId={sessionId}
        usage={inspectorUsage}
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
      <SessionClearDialog
        open={clearDialog.open}
        onOpenChange={clearDialog.setOpen}
        isClearing={controls.isClearing}
        onConfirm={clearDialog.confirmClear}
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

interface SessionClearDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  isClearing: boolean;
  onConfirm: () => void;
}

function SessionClearDialog({
  open,
  onOpenChange,
  isClearing,
  onConfirm,
}: SessionClearDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        showCloseButton={!isClearing}
        className="max-w-md"
        data-testid="composer-clear-dialog"
      >
        <DialogHeader>
          <DialogTitle>Clear conversation</DialogTitle>
          <DialogDescription>
            This removes the visible transcript for this session and starts a fresh runtime
            conversation on the same session id.
          </DialogDescription>
        </DialogHeader>
        <DialogFooter className="gap-2">
          <Button
            type="button"
            variant="ghost"
            onClick={() => onOpenChange(false)}
            disabled={isClearing}
            data-testid="composer-clear-cancel"
          >
            Cancel
          </Button>
          <Button
            type="button"
            variant="destructive"
            onClick={onConfirm}
            disabled={isClearing}
            data-testid="composer-clear-confirm"
          >
            {isClearing ? (
              <>
                <Spinner className="size-3" />
                Clearing
              </>
            ) : (
              <>
                <Eraser className="size-3" />
                Clear conversation
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

function SessionPage() {
  const { name, id } = Route.useParams();
  const { workspaceId } = Route.useLoaderData();

  return <SessionPageResolved name={name} id={id} workspaceId={workspaceId} />;
}

function SessionRouteLoading() {
  return (
    <div className="flex flex-1 items-center justify-center" data-testid="session-route-loading">
      <Spinner className="size-5 text-subtle" />
    </div>
  );
}

interface SessionPageResolvedProps {
  name: string;
  id: string;
  workspaceId: string | null;
}

function SessionPageResolved({ name, id, workspaceId }: SessionPageResolvedProps) {
  const navigate = useNavigate();

  if (!workspaceId) {
    return (
      <div className="flex flex-1 items-center justify-center">
        <div className="flex flex-col items-center gap-2 text-center">
          <AlertCircle className="size-6 text-danger" />
          <p className="text-sm text-subtle">Session not found</p>
        </div>
      </div>
    );
  }

  return (
    <SessionPageWithWorkspace name={name} id={id} workspaceId={workspaceId} navigate={navigate} />
  );
}

interface SessionPageWithWorkspaceProps {
  name: string;
  id: string;
  workspaceId: string;
  navigate: ReturnType<typeof useNavigate>;
}

function SessionPageWithWorkspace({
  name,
  id,
  workspaceId,
  navigate,
}: SessionPageWithWorkspaceProps) {
  const { data: session, isLoading, error } = useSessionById(id, workspaceId);
  const sessionWorkspaceId = session?.workspace_id?.trim();

  useSessionWorkspaceGuard({
    sessionWorkspaceId,
    agentName: name,
    sessionId: id,
    sessionName: session?.name,
  });

  useEffect(() => {
    if (error?.message?.includes("not found")) {
      toast.error("Session not found");
      void navigate({ to: "/agents/$name", params: { name }, replace: true });
    }
  }, [error, navigate, name]);

  if (isLoading) {
    return <SessionRouteLoading />;
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

  if (!sessionWorkspaceId) {
    return (
      <div className="flex flex-1 items-center justify-center">
        <div className="flex flex-col items-center gap-2 text-center">
          <AlertCircle className="size-6 text-danger" />
          <p className="text-sm text-subtle">Session workspace unavailable</p>
        </div>
      </div>
    );
  }

  const resolvedAgentName = session.agent_name ?? name;

  return (
    <SessionChatRuntimeProvider key={id} sessionId={id} workspaceId={sessionWorkspaceId}>
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
