import {
  AuiIf,
  ComposerPrimitive,
  type DataMessagePartProps,
  MessagePrimitive,
  ThreadPrimitive,
  type ToolCallMessagePartProps,
} from "@assistant-ui/react";
import { Activity, SendHorizontal, Square, Trash2 } from "lucide-react";
import { type ComponentPropsWithoutRef, useCallback, useState } from "react";

import { cn } from "@/lib/utils";
import { MessageMarkdown } from "@/systems/session/components/message-markdown";
import { ThinkingBlock } from "@/systems/session/components/thinking-block";
import { BackendToolPart } from "@/systems/session/lib/session-toolkit";
import {
  Button,
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  Eyebrow,
  Spinner,
} from "@agh/ui";
import { useSessionComposerState } from "./hooks/use-session-composer-state";

interface SessionThreadProps {
  sessionId: string;
  agentName: string;
  canPrompt: boolean;
  onCancelPrompt: () => void;
  onClearConversation?: () => void;
  canClearConversation?: boolean;
  isClearingConversation?: boolean;
}

function SessionTextPart({ text, state }: { text: string; state?: { type: string } }) {
  return (
    <div className="text-sm leading-7 text-fg">
      <MessageMarkdown content={text} streaming={state?.type === "running"} />
    </div>
  );
}

function SessionReasoningPart({ text, state }: { text: string; state?: { type: string } }) {
  return <ThinkingBlock thinking={text} thinkingComplete={state?.type !== "running"} />;
}

function formatDataPreview(data: unknown): string | null {
  if (data === undefined || data === null) {
    return null;
  }

  if (typeof data === "string") {
    return data;
  }

  try {
    return JSON.stringify(data);
  } catch {
    return String(data);
  }
}

function SessionDataPart(part: DataMessagePartProps<unknown>) {
  const preview = formatDataPreview(part.data);
  const clippedPreview =
    preview && preview.length > 180 ? `${preview.slice(0, 180).trimEnd()}...` : preview;

  return (
    <div
      data-testid="session-data-part"
      className={cn(
        "my-2 flex max-w-3xl items-start gap-2 rounded-lg border px-3 py-2",
        "border-line bg-canvas-soft text-form-input text-muted"
      )}
    >
      <Activity aria-hidden="true" className="mt-0.5 size-3 shrink-0 text-info" />
      <div className="min-w-0">
        <div className="text-card-title text-fg">Data event</div>
        <div className="truncate text-form-label text-subtle">{part.name}</div>
        {clippedPreview ? (
          <pre className="mt-1 max-h-24 overflow-auto whitespace-pre-wrap break-words font-mono text-small-body text-muted">
            {clippedPreview}
          </pre>
        ) : null}
      </div>
    </div>
  );
}

function SessionToolPart(part: ToolCallMessagePartProps<Record<string, unknown>, unknown>) {
  return <BackendToolPart {...part} />;
}

function SessionMessageEmpty({ status }: { status: { type: string } }) {
  if (status.type !== "running") {
    return null;
  }

  return (
    <div className="flex items-center gap-2 text-sm text-subtle">
      <Spinner />
      <span>Thinking…</span>
    </div>
  );
}

function UserMessage() {
  return (
    <MessagePrimitive.Root className="mx-auto flex w-full max-w-3xl justify-end py-3">
      <div
        className={cn(
          "max-w-[min(80%,42rem)] rounded-xl border px-4 py-3",
          "border-line bg-canvas-soft"
        )}
      >
        <MessagePrimitive.Parts
          components={{
            Text: ({ text, status }) => <SessionTextPart text={text} state={status} />,
            data: {
              Fallback: SessionDataPart,
            },
          }}
        />
      </div>
    </MessagePrimitive.Root>
  );
}

function AssistantMessage() {
  return (
    <MessagePrimitive.Root className="mx-auto flex w-full max-w-3xl py-3">
      <div className="flex min-w-0 flex-1 flex-col gap-3">
        <MessagePrimitive.Parts
          components={{
            Text: ({ text, status }) => <SessionTextPart text={text} state={status} />,
            Reasoning: ({ text, status }) => <SessionReasoningPart text={text} state={status} />,
            Empty: ({ status }) => <SessionMessageEmpty status={status} />,
            tools: {
              Fallback: SessionToolPart,
            },
            data: {
              Fallback: SessionDataPart,
            },
          }}
        />
        <AuiIf
          condition={state =>
            state.message.status?.type === "incomplete" && state.message.status.reason === "error"
          }
        >
          <div
            className={cn(
              "rounded-md border px-3 py-2 text-sm",
              "border-danger/30 bg-danger/8",
              "text-danger"
            )}
          >
            <MessagePrimitive.Error />
          </div>
        </AuiIf>
      </div>
    </MessagePrimitive.Root>
  );
}

function SessionComposer({
  sessionId,
  canPrompt,
  onCancelPrompt,
  onClearConversation,
  canClearConversation = false,
  isClearingConversation = false,
}: Pick<
  SessionThreadProps,
  | "sessionId"
  | "canPrompt"
  | "onCancelPrompt"
  | "onClearConversation"
  | "canClearConversation"
  | "isClearingConversation"
>) {
  const { isRunning } = useSessionComposerState(sessionId);
  const [clearDialogOpen, setClearDialogOpen] = useState(false);

  const handleConfirmClear = useCallback(() => {
    setClearDialogOpen(false);
    onClearConversation?.();
  }, [onClearConversation]);

  return (
    <>
      <div className={cn("border-t px-4 py-3", "border-line bg-canvas-soft")}>
        <ComposerPrimitive.Root
          className={cn(
            "flex flex-col gap-2 rounded-xl border px-3 pt-2.5 pb-2",
            "border-line bg-canvas-soft",
            "focus-within:border-accent transition-colors"
          )}
        >
          <ComposerPrimitive.Input
            aria-label="Session prompt"
            data-testid="composer-textarea"
            disabled={!canPrompt}
            placeholder={canPrompt ? "Send a message…" : "Session is not active"}
            rows={1}
            maxRows={12}
            submitMode="enter"
            className={cn(
              "min-h-6 w-full resize-none border-none bg-transparent p-0 text-sm leading-relaxed",
              "text-fg placeholder:text-subtle",
              "outline-none focus-visible:border-transparent focus-visible:ring-0",
              "dark:bg-transparent"
            )}
          />
          <div className="flex items-center justify-between gap-3">
            {onClearConversation ? (
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => setClearDialogOpen(true)}
                disabled={!canClearConversation || isRunning || isClearingConversation}
                data-testid="composer-clear-button"
              >
                {isClearingConversation ? (
                  <Spinner className="size-3" />
                ) : (
                  <Trash2 className="size-3" />
                )}
                Clear conversation
              </Button>
            ) : (
              <span />
            )}

            {isRunning ? (
              <button
                type="button"
                onClick={onCancelPrompt}
                className={cn(
                  "inline-flex h-9 items-center gap-2 rounded-full px-3",
                  "bg-danger/12 text-danger",
                  "transition-colors hover:bg-danger/18"
                )}
              >
                <Square className="size-3 fill-current" />
                <span className="text-sm font-medium">Stop</span>
              </button>
            ) : (
              <ComposerPrimitive.Send
                aria-label="Send message"
                className={cn(
                  "inline-flex size-9 items-center justify-center rounded-full",
                  "bg-accent text-accent-ink transition-colors",
                  "hover:bg-accent-hover disabled:cursor-not-allowed disabled:opacity-50"
                )}
                data-testid="composer-send-button"
              >
                <SendHorizontal className="size-4" />
              </ComposerPrimitive.Send>
            )}
          </div>
        </ComposerPrimitive.Root>
      </div>

      <Dialog open={clearDialogOpen} onOpenChange={setClearDialogOpen}>
        <DialogContent
          showCloseButton={!isClearingConversation}
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
              onClick={() => setClearDialogOpen(false)}
              disabled={isClearingConversation}
              data-testid="composer-clear-cancel"
            >
              Cancel
            </Button>
            <Button
              type="button"
              variant="destructive"
              onClick={handleConfirmClear}
              disabled={isClearingConversation}
              data-testid="composer-clear-confirm"
            >
              {isClearingConversation ? (
                <>
                  <Spinner className="size-3" />
                  Clearing
                </>
              ) : (
                <>
                  <Trash2 className="size-3" />
                  Clear conversation
                </>
              )}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}

function ThreadEmpty({ agentName }: Pick<SessionThreadProps, "agentName">) {
  return (
    <div className="mx-auto flex size-full max-w-3xl items-center justify-center px-4 py-12">
      <div className="max-w-md text-center">
        <Eyebrow className="text-subtle">{agentName}</Eyebrow>
        <p className="mt-2 text-sm text-muted">
          Start a conversation. The assistant thread replays persisted history and continues live
          over the daemon stream.
        </p>
      </div>
    </div>
  );
}

type ThreadViewportProps = ComponentPropsWithoutRef<typeof ThreadPrimitive.Viewport>;

function ThreadViewport(props: ThreadViewportProps) {
  return (
    <ThreadPrimitive.Viewport
      {...props}
      className={cn("min-h-0 flex-1 overflow-y-auto px-4", props.className)}
      data-testid="chat-view"
    />
  );
}

export function SessionThread({
  sessionId,
  agentName,
  canPrompt,
  onCancelPrompt,
  onClearConversation,
  canClearConversation = false,
  isClearingConversation = false,
}: SessionThreadProps) {
  return (
    <ThreadPrimitive.Root className="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden">
      <ThreadViewport>
        <ThreadPrimitive.Empty>
          <ThreadEmpty agentName={agentName} />
        </ThreadPrimitive.Empty>
        <ThreadPrimitive.Messages
          components={{
            UserMessage,
            AssistantMessage,
          }}
        />
      </ThreadViewport>
      <SessionComposer
        sessionId={sessionId}
        canPrompt={canPrompt}
        onCancelPrompt={onCancelPrompt}
        onClearConversation={onClearConversation}
        canClearConversation={canClearConversation}
        isClearingConversation={isClearingConversation}
      />
    </ThreadPrimitive.Root>
  );
}
