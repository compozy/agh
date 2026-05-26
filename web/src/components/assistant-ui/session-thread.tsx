import {
  ComposerPrimitive,
  type DataMessagePartProps,
  type EmptyMessagePartProps,
  MessagePrimitive,
  type ReasoningMessagePartProps,
  type TextMessagePartProps,
  ThreadPrimitive,
  type ToolCallMessagePartProps,
  useAuiState,
} from "@assistant-ui/react";
import {
  Activity,
  CornerDownRight,
  ListPlus,
  Scissors,
  SendHorizontal,
  Square,
  Trash2,
} from "lucide-react";
import { type ComponentPropsWithoutRef, type ReactNode, useCallback, useState } from "react";

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

type SessionBusyInputHandler = (message: string) => void | Promise<void>;

export type SessionThreadContentInset = "px-4" | "px-8";

const SESSION_THREAD_CONTENT_INSET_DEFAULT: SessionThreadContentInset = "px-4";

interface SessionThreadProps {
  sessionId: string;
  agentName: string;
  canPrompt: boolean;
  onCancelPrompt: () => void;
  onQueuePrompt?: SessionBusyInputHandler;
  onInterruptPrompt?: SessionBusyInputHandler;
  onSteerPrompt?: SessionBusyInputHandler;
  isBusyInputPending?: boolean;
  onClearConversation?: () => void;
  canClearConversation?: boolean;
  isClearingConversation?: boolean;
  contentInset?: SessionThreadContentInset;
}

function ThreadContentRail({
  inset,
  className,
  children,
  ...props
}: {
  inset: SessionThreadContentInset;
  className?: string;
  children: ReactNode;
} & Omit<ComponentPropsWithoutRef<"div">, "className" | "children">) {
  return (
    <div
      className={cn("w-full min-w-0", inset, className)}
      data-testid="thread-content-rail"
      {...props}
    >
      {children}
    </div>
  );
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
        "my-2 flex w-full min-w-0 items-start gap-2 rounded-lg border px-3 py-2",
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

function textField(record: Record<string, unknown>, key: string): string | null {
  const value = record[key];
  if (typeof value !== "string") {
    return null;
  }
  const message = value.trim();
  return message.length > 0 ? message : null;
}

function recordField(record: Record<string, unknown>, key: string): Record<string, unknown> | null {
  const value = record[key];
  if (typeof value !== "object" || value === null || Array.isArray(value)) {
    return null;
  }
  return value as Record<string, unknown>;
}

function messageFromErrorRecord(record: Record<string, unknown>): string | null {
  const data = recordField(record, "data");
  if (data) {
    const dataMessage = textField(data, "error") ?? textField(data, "message");
    if (dataMessage) {
      return dataMessage;
    }
  }

  const failure = recordField(record, "failure");
  if (failure) {
    const failureMessage = textField(failure, "summary") ?? textField(failure, "message");
    if (failureMessage) {
      return failureMessage;
    }
  }

  return (
    textField(record, "error") ??
    textField(record, "summary") ??
    textField(record, "detail") ??
    textField(record, "message")
  );
}

export function formatMessageError(error: unknown): string | null {
  if (error instanceof Error) {
    return formatMessageError(error.message);
  }

  if (typeof error === "object" && error !== null && !Array.isArray(error)) {
    return messageFromErrorRecord(error as Record<string, unknown>);
  }

  if (typeof error === "string") {
    const message = error.trim();
    if (message.length === 0) {
      return null;
    }

    try {
      const parsed = JSON.parse(message) as unknown;
      return formatMessageError(parsed);
    } catch {
      // Non-JSON provider errors are already human-readable enough to display.
    }

    return message;
  }

  return null;
}

function SessionMessageErrorNotice() {
  const error = useAuiState(state => {
    const status = state.message.status;
    if (status?.type !== "incomplete" || status.reason !== "error") {
      return null;
    }
    return formatMessageError(status.error);
  });

  if (error === null) {
    return null;
  }

  return (
    <div
      role="alert"
      data-testid="session-message-error"
      className={cn(
        "rounded-md border px-3 py-2 text-sm",
        "border-danger/30 bg-danger/8",
        "text-danger"
      )}
    >
      {error}
    </div>
  );
}

function UserMessage() {
  return (
    <MessagePrimitive.Root className="flex w-full min-w-0 justify-end py-3">
      <div
        className={cn(
          "max-w-[min(80%,42rem)] rounded-xl border px-4 py-3",
          "border-line bg-canvas-soft"
        )}
      >
        <MessagePrimitive.Parts
          components={{
            Text: ({ text, status }: TextMessagePartProps) => (
              <SessionTextPart text={text} state={status} />
            ),
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
    <MessagePrimitive.Root className="flex w-full min-w-0 py-3">
      <div className="flex min-w-0 flex-1 flex-col gap-3">
        <MessagePrimitive.Parts
          components={{
            Text: ({ text, status }: TextMessagePartProps) => (
              <SessionTextPart text={text} state={status} />
            ),
            Reasoning: ({ text, status }: ReasoningMessagePartProps) => (
              <SessionReasoningPart text={text} state={status} />
            ),
            Empty: ({ status }: EmptyMessagePartProps) => <SessionMessageEmpty status={status} />,
            tools: {
              Fallback: SessionToolPart,
            },
            data: {
              Fallback: SessionDataPart,
            },
          }}
        />
        <SessionMessageErrorNotice />
      </div>
    </MessagePrimitive.Root>
  );
}

function SessionComposer({
  sessionId,
  contentInset,
  canPrompt,
  onCancelPrompt,
  onQueuePrompt,
  onInterruptPrompt,
  onSteerPrompt,
  isBusyInputPending = false,
  onClearConversation,
  canClearConversation = false,
  isClearingConversation = false,
}: Pick<
  SessionThreadProps,
  | "sessionId"
  | "contentInset"
  | "canPrompt"
  | "onCancelPrompt"
  | "onQueuePrompt"
  | "onInterruptPrompt"
  | "onSteerPrompt"
  | "isBusyInputPending"
  | "onClearConversation"
  | "canClearConversation"
  | "isClearingConversation"
>) {
  const { clearComposer, composerText, isRunning } = useSessionComposerState(sessionId);
  const [clearDialogOpen, setClearDialogOpen] = useState(false);
  const trimmedComposerText = composerText.trim();
  const canSubmitBusyInput =
    isRunning && canPrompt && trimmedComposerText.length > 0 && !isBusyInputPending;
  const showBusyInputControls = isRunning || isBusyInputPending;

  const handleConfirmClear = useCallback(() => {
    setClearDialogOpen(false);
    onClearConversation?.();
  }, [onClearConversation]);

  const handleBusyInputAction = useCallback(
    (handler?: SessionBusyInputHandler) => {
      if (!handler || !canSubmitBusyInput) {
        return;
      }

      void Promise.resolve(handler(trimmedComposerText))
        .then(clearComposer)
        .catch(() => undefined);
    },
    [canSubmitBusyInput, clearComposer, trimmedComposerText]
  );

  return (
    <>
      <div className={cn("border-t border-line bg-canvas-soft")} data-testid="composer-shell">
        <ThreadContentRail
          inset={contentInset ?? SESSION_THREAD_CONTENT_INSET_DEFAULT}
          className="py-3"
        >
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
            <div className="flex flex-wrap items-center justify-between gap-3">
              <div className="flex min-w-0 flex-wrap items-center gap-2">
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
                ) : null}
              </div>

              {showBusyInputControls ? (
                <div className="flex flex-wrap items-center justify-end gap-2">
                  {onQueuePrompt ? (
                    <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      onClick={() => handleBusyInputAction(onQueuePrompt)}
                      disabled={!canSubmitBusyInput}
                      data-testid="composer-queue-button"
                    >
                      <ListPlus className="size-3" />
                      Queue
                    </Button>
                  ) : null}
                  {onSteerPrompt ? (
                    <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      onClick={() => handleBusyInputAction(onSteerPrompt)}
                      disabled={!canSubmitBusyInput}
                      data-testid="composer-steer-button"
                    >
                      <CornerDownRight className="size-3" />
                      Steer
                    </Button>
                  ) : null}
                  {onInterruptPrompt ? (
                    <Button
                      type="button"
                      variant="destructive"
                      size="sm"
                      onClick={() => handleBusyInputAction(onInterruptPrompt)}
                      disabled={!canSubmitBusyInput}
                      data-testid="composer-interrupt-button"
                    >
                      <Scissors className="size-3" />
                      Interrupt
                    </Button>
                  ) : null}
                  {isRunning ? (
                    <Button
                      type="button"
                      variant="destructive"
                      size="sm"
                      onClick={onCancelPrompt}
                      data-testid="composer-stop-button"
                    >
                      <Square className="size-3 fill-current" />
                      Stop
                    </Button>
                  ) : null}
                </div>
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
        </ThreadContentRail>
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
    <div className="flex size-full w-full min-w-0 items-center justify-center py-12">
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

function ThreadViewport({
  contentInset,
  children,
  className,
  ...props
}: ThreadViewportProps & {
  contentInset: SessionThreadContentInset;
  children: ReactNode;
}) {
  return (
    <ThreadPrimitive.Viewport
      {...props}
      className={cn("min-h-0 flex-1 overflow-y-auto", className)}
      data-testid="chat-view"
    >
      <ThreadContentRail inset={contentInset} className="min-h-full">
        {children}
      </ThreadContentRail>
    </ThreadPrimitive.Viewport>
  );
}

export function SessionThread({
  sessionId,
  agentName,
  canPrompt,
  onCancelPrompt,
  onQueuePrompt,
  onInterruptPrompt,
  onSteerPrompt,
  isBusyInputPending = false,
  onClearConversation,
  canClearConversation = false,
  isClearingConversation = false,
  contentInset = SESSION_THREAD_CONTENT_INSET_DEFAULT,
}: SessionThreadProps) {
  return (
    <ThreadPrimitive.Root className="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden">
      <ThreadViewport contentInset={contentInset}>
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
        contentInset={contentInset}
        canPrompt={canPrompt}
        onCancelPrompt={onCancelPrompt}
        onQueuePrompt={onQueuePrompt}
        onInterruptPrompt={onInterruptPrompt}
        onSteerPrompt={onSteerPrompt}
        isBusyInputPending={isBusyInputPending}
        onClearConversation={onClearConversation}
        canClearConversation={canClearConversation}
        isClearingConversation={isClearingConversation}
      />
    </ThreadPrimitive.Root>
  );
}
