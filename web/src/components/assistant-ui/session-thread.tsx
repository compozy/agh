import {
  ComposerPrimitive,
  type DataMessagePartProps,
  type EmptyMessagePartProps,
  MessagePrimitive,
  type ReasoningMessagePartProps,
  type TextMessagePartProps,
  ThreadPrimitive,
  type ToolCallMessagePartProps,
  ReadonlyThreadProvider,
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
import {
  type ComponentPropsWithoutRef,
  type RefObject,
  type ReactNode,
  useCallback,
  useMemo,
  useRef,
  useState,
} from "react";

import { cn } from "@/lib/utils";
import { MessageMarkdown } from "@/systems/session/components/message-markdown";
import { ThinkingBlock } from "@/systems/session/components/thinking-block";
import { BackendToolPart } from "@/systems/session/lib/session-toolkit";
import { useSessionTranscriptThreadState } from "@/systems/session";
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
import { useSessionComposerState } from "./hooks/use-session-composer-state";
import { useVirtualizedThreadMessages } from "./hooks/use-virtualized-thread-messages";
import { formatMessageError } from "./session-thread-error";
import { ThreadStatePane } from "./session-thread-states";

export { formatMessageError };

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
  isSessionRunning?: boolean;
  allowBusyInput?: boolean;
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
  isSessionRunning = false,
  allowBusyInput = true,
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
  | "isSessionRunning"
  | "allowBusyInput"
  | "onClearConversation"
  | "canClearConversation"
  | "isClearingConversation"
>) {
  const { clearComposer, composerText, isRunning } = useSessionComposerState(sessionId);
  const [clearDialogOpen, setClearDialogOpen] = useState(false);
  const trimmedComposerText = composerText.trim();
  const runtimeRunning = isRunning || isSessionRunning;
  const canSubmitBusyInput =
    runtimeRunning &&
    canPrompt &&
    allowBusyInput &&
    trimmedComposerText.length > 0 &&
    !isBusyInputPending;
  const showBusyInputControls = runtimeRunning || isBusyInputPending;

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
                    disabled={!canClearConversation || runtimeRunning || isClearingConversation}
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
                  {allowBusyInput && onQueuePrompt ? (
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
                  {allowBusyInput && onSteerPrompt ? (
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
                  {allowBusyInput && onInterruptPrompt ? (
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
                  {runtimeRunning ? (
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

type ThreadViewportProps = ComponentPropsWithoutRef<typeof ThreadPrimitive.Viewport>;

function ThreadViewport({
  agentName,
  contentInset,
  className,
  ...props
}: ThreadViewportProps & {
  agentName: string;
  contentInset: SessionThreadContentInset;
}) {
  const viewportRef = useRef<HTMLDivElement | null>(null);
  const {
    messages: transcriptMessages,
    status: transcriptStatus,
    error: transcriptError,
    retry: retryTranscript,
  } = useSessionTranscriptThreadState();
  const readonlyThreadKey = useMemo(
    () => transcriptMessages.map(message => message.id).join("\n"),
    [transcriptMessages]
  );
  const messageCount = transcriptMessages.length;

  return (
    <ThreadPrimitive.Viewport
      {...props}
      ref={viewportRef}
      className={cn("min-h-0 flex-1 overflow-y-auto", className)}
      data-testid="chat-view"
    >
      <ThreadContentRail inset={contentInset} className="min-h-full">
        <VirtualizedThreadMessages
          agentName={agentName}
          viewportRef={viewportRef}
          messageCount={messageCount}
          transcriptStatus={transcriptStatus}
          transcriptError={transcriptError}
          retryTranscript={retryTranscript}
          transcriptMessages={transcriptMessages}
          readonlyThreadKey={readonlyThreadKey}
        />
      </ThreadContentRail>
    </ThreadPrimitive.Viewport>
  );
}

const SESSION_MESSAGE_COMPONENTS = {
  UserMessage,
  AssistantMessage,
};
const VIRTUAL_MESSAGE_ESTIMATE = 144;

export function VirtualizedThreadMessages({
  agentName,
  viewportRef,
  messageCount,
  transcriptStatus,
  transcriptError,
  retryTranscript,
  transcriptMessages,
  readonlyThreadKey,
}: {
  agentName: string;
  viewportRef: RefObject<HTMLDivElement | null>;
  messageCount: number;
  transcriptStatus: ReturnType<typeof useSessionTranscriptThreadState>["status"];
  transcriptError: ReturnType<typeof useSessionTranscriptThreadState>["error"];
  retryTranscript: () => void;
  transcriptMessages: ReturnType<typeof useSessionTranscriptThreadState>["messages"];
  readonlyThreadKey: string;
}) {
  const { virtualizer } = useVirtualizedThreadMessages(viewportRef, messageCount);

  if (messageCount === 0) {
    return (
      <ThreadStatePane
        status={transcriptStatus}
        agentName={agentName}
        error={transcriptError}
        onRetry={retryTranscript}
      />
    );
  }

  const virtualItems = virtualizer.getVirtualItems();
  const visibleItems =
    virtualItems.length > 0
      ? virtualItems
      : Array.from({ length: Math.min(messageCount, 12) }, (_, index) => ({
          index,
          key: index,
          start: index * VIRTUAL_MESSAGE_ESTIMATE,
        }));
  const totalSize = Math.max(virtualizer.getTotalSize(), messageCount * VIRTUAL_MESSAGE_ESTIMATE);

  const rows = (
    <div
      className="relative w-full"
      data-testid="virtualized-thread-messages"
      style={{ height: totalSize }}
    >
      {visibleItems.map(item => (
        <div
          key={item.key}
          ref={virtualizer.measureElement}
          data-index={item.index}
          data-testid="virtualized-thread-row"
          className="absolute top-0 left-0 w-full"
          style={{ transform: `translateY(${item.start}px)` }}
        >
          <ThreadPrimitive.MessageByIndex
            index={item.index}
            components={SESSION_MESSAGE_COMPONENTS}
          />
        </div>
      ))}
    </div>
  );

  return (
    <ReadonlyThreadProvider key={readonlyThreadKey} messages={transcriptMessages}>
      {rows}
    </ReadonlyThreadProvider>
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
  isSessionRunning = false,
  allowBusyInput = true,
  onClearConversation,
  canClearConversation = false,
  isClearingConversation = false,
  contentInset = SESSION_THREAD_CONTENT_INSET_DEFAULT,
}: SessionThreadProps) {
  return (
    <ThreadPrimitive.Root className="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden">
      <ThreadViewport agentName={agentName} contentInset={contentInset} />
      <SessionComposer
        sessionId={sessionId}
        contentInset={contentInset}
        canPrompt={canPrompt}
        onCancelPrompt={onCancelPrompt}
        onQueuePrompt={onQueuePrompt}
        onInterruptPrompt={onInterruptPrompt}
        onSteerPrompt={onSteerPrompt}
        isBusyInputPending={isBusyInputPending}
        isSessionRunning={isSessionRunning}
        allowBusyInput={allowBusyInput}
        onClearConversation={onClearConversation}
        canClearConversation={canClearConversation}
        isClearingConversation={isClearingConversation}
      />
    </ThreadPrimitive.Root>
  );
}
