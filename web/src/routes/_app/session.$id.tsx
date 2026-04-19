import { AlertCircle, Loader2 } from "lucide-react";
import { createFileRoute } from "@tanstack/react-router";

import { useSessionPage } from "@/hooks/routes/use-session-page";
import { ChatHeader } from "@/systems/session/components/chat-header";
import { ChatView } from "@/systems/session/components/chat-view";
import { MessageComposer } from "@/systems/session/components/message-composer";
import { PermissionPrompt } from "@/systems/session/components/permission-prompt";

export const Route = createFileRoute("/_app/session/$id")({
  component: SessionPage,
});

function SessionPage() {
  const { id } = Route.useParams();
  const page = useSessionPage(id);

  if (page.isLoading) {
    return (
      <div className="flex flex-1 items-center justify-center">
        <Loader2 className="size-5 animate-spin text-[color:var(--color-text-tertiary)]" />
      </div>
    );
  }

  if (!page.session) {
    return (
      <div className="flex flex-1 items-center justify-center">
        <div className="flex flex-col items-center gap-2 text-center">
          <AlertCircle className="size-6 text-[color:var(--color-danger)]" />
          <p className="text-sm text-[color:var(--color-text-tertiary)]">
            {page.fatalErrorMessage}
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="flex flex-1 flex-col overflow-hidden">
      <ChatHeader
        session={page.session}
        onStop={page.handleStop}
        onResume={page.handleResume}
        workspaceName={page.workspaceName}
      />
      <ChatView
        messages={page.messages}
        isStreaming={page.isStreaming}
        agentName={page.session.agent_name}
      />
      {page.pendingPermission && (
        <PermissionPrompt
          permission={page.pendingPermission}
          sessionId={id}
          onResolved={page.handlePermissionResolved}
        />
      )}
      {page.canPrompt && (
        <MessageComposer
          sessionId={id}
          onSend={page.handleSend}
          disabled={page.isDisabled}
          skills={page.skills}
          channels={page.channels}
        />
      )}
    </div>
  );
}
