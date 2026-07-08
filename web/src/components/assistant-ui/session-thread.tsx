import {
  type DataMessagePartProps,
  MessagePrimitive,
  type TextMessagePartProps,
  ThreadPrimitive,
  ReadonlyThreadProvider,
  useAuiState,
} from "@assistant-ui/react";
import { Activity, ArrowDown } from "lucide-react";
import { type ComponentPropsWithoutRef, useEffect, useRef, useState } from "react";

import { cn } from "@/lib/utils";
import { MessageMarkdown } from "@/systems/session/components/message-markdown";
import { useSessionTranscriptThreadState } from "@/systems/session";
import { SessionComposer, type SessionComposerProps } from "./session-composer";
import {
  SESSION_THREAD_CONTENT_INSET_DEFAULT,
  ThreadContentRail,
  type SessionThreadContentInset,
} from "./session-thread-content-rail";
import { useVirtualizedThreadMessages } from "./hooks/use-virtualized-thread-messages";
import { MessageActions } from "./message-actions";
import { formatMessageError } from "./session-thread-error";
import { AssistantMessageTimeline } from "./session-timeline-render";
import { ThreadStatePane } from "./session-thread-states";
import { VIRTUAL_MESSAGE_ESTIMATE } from "./timeline-row-estimates";

export { formatMessageError };

interface SessionThreadProps extends SessionComposerProps {
  agentName: string;
}

function SessionTextPart({ text, state }: { text: string; state?: { type: string } }) {
  return (
    <div className="text-sm leading-7 text-fg">
      <MessageMarkdown content={text} streaming={state?.type === "running"} />
    </div>
  );
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
    <MessagePrimitive.Root className="group/message flex w-full min-w-0 justify-end py-3">
      <div className="flex min-w-0 max-w-[min(80%,42rem)] flex-col items-end gap-1">
        <div className={cn("w-full rounded-xl border px-4 py-3", "border-line bg-canvas-soft")}>
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
        <MessageActions align="end" copyLabel="Copy message" testId="user-message-actions" />
      </div>
    </MessagePrimitive.Root>
  );
}

function AssistantMessage() {
  return (
    <MessagePrimitive.Root className="group/message flex w-full min-w-0 py-3">
      <div className="flex min-w-0 flex-1 flex-col gap-3">
        <AssistantMessageTimeline />
        <SessionMessageErrorNotice />
        <MessageActions align="start" copyLabel="Copy message" testId="assistant-message-actions" />
      </div>
    </MessagePrimitive.Root>
  );
}

type ThreadViewportProps = ComponentPropsWithoutRef<typeof ThreadPrimitive.Viewport>;

export function ScrollToBottomPill({
  visible,
  onClick,
}: {
  visible: boolean;
  onClick: () => void;
}) {
  // Synara's floating scroll-to-bottom affordance remapped to AGH tokens: a
  // neutral `size-8` disc (no glass/backdrop-blur) that fades + drifts in with the
  // shared disclosure motion and stays mounted so its exit animates too.
  return (
    <div
      className={cn(
        "pointer-events-none absolute inset-x-0 bottom-3 z-20 flex justify-center",
        "transition-all duration-base ease-out motion-reduce:transition-none",
        visible ? "translate-y-0 opacity-100" : "translate-y-1 opacity-0"
      )}
      aria-hidden={!visible}
    >
      <button
        type="button"
        data-testid="scroll-to-bottom-pill"
        data-visible={visible}
        aria-label="Scroll to latest"
        onClick={onClick}
        tabIndex={visible ? 0 : -1}
        className={cn(
          "flex size-8 items-center justify-center rounded-full",
          "border border-line bg-canvas-soft text-muted shadow-[var(--shadow-overlay)]",
          "transition-colors hover:bg-hover hover:text-fg",
          "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-line-strong",
          visible ? "pointer-events-auto" : "pointer-events-none"
        )}
      >
        <ArrowDown className="size-3.5" aria-hidden="true" />
      </button>
    </div>
  );
}

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
  const messageCount = transcriptMessages.length;
  const { virtualizer, showScrollToBottom, scrollToEnd } = useVirtualizedThreadMessages(
    viewportRef,
    transcriptMessages
  );

  return (
    <div className="relative flex min-h-0 min-w-0 flex-1 flex-col">
      <ThreadPrimitive.Viewport
        {...props}
        ref={viewportRef}
        className={cn("min-h-0 flex-1 overflow-y-auto", className)}
        data-testid="chat-view"
      >
        <ThreadContentRail inset={contentInset} className="min-h-full">
          <VirtualizedThreadMessages
            agentName={agentName}
            virtualizer={virtualizer}
            messageCount={messageCount}
            transcriptStatus={transcriptStatus}
            transcriptError={transcriptError}
            retryTranscript={retryTranscript}
            transcriptMessages={transcriptMessages}
          />
        </ThreadContentRail>
      </ThreadPrimitive.Viewport>
      <ScrollToBottomPill visible={showScrollToBottom} onClick={scrollToEnd} />
    </div>
  );
}

const SESSION_MESSAGE_COMPONENTS = {
  UserMessage,
  AssistantMessage,
};

export function VirtualizedThreadMessages({
  agentName,
  virtualizer,
  messageCount,
  transcriptStatus,
  transcriptError,
  retryTranscript,
  transcriptMessages,
}: {
  agentName: string;
  virtualizer: ReturnType<typeof useVirtualizedThreadMessages>["virtualizer"];
  messageCount: number;
  transcriptStatus: ReturnType<typeof useSessionTranscriptThreadState>["status"];
  transcriptError: ReturnType<typeof useSessionTranscriptThreadState>["error"];
  retryTranscript: () => void;
  transcriptMessages: ReturnType<typeof useSessionTranscriptThreadState>["messages"];
}) {
  // `ReadonlyThreadProvider` applies `messages` in a passive effect, so a freshly
  // grown transcript is not yet readable by index during the growth render. Gate
  // the rendered rows on the committed count so a streaming append never asks the
  // core for an index it does not have yet — this replaces the old id-join `key`
  // remount that used to force synchronous re-init at the cost of resetting scroll.
  const [committedCount, setCommittedCount] = useState(messageCount);
  useEffect(() => {
    setCommittedCount(messageCount);
  }, [messageCount]);
  const renderableCount = Math.min(messageCount, committedCount);

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

  const virtualItems = virtualizer.getVirtualItems().filter(item => item.index < renderableCount);
  const visibleItems =
    virtualItems.length > 0
      ? virtualItems
      : Array.from({ length: Math.min(renderableCount, 12) }, (_, index) => ({
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

  // No `key` remount here: `ReadonlyThreadProvider` reactively re-applies
  // `messages` via `setMessages`, so re-keying on the joined id set only reset the
  // virtualizer's scroll position when the first live message arrived. Reusing the
  // provider keeps the viewport stable across streaming appends and reconnects.
  return <ReadonlyThreadProvider messages={transcriptMessages}>{rows}</ReadonlyThreadProvider>;
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
  queuedPrompts = [],
  onRemoveQueuedPrompt,
  onSteerQueuedPrompt,
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
        queuedPrompts={queuedPrompts}
        onRemoveQueuedPrompt={onRemoveQueuedPrompt}
        onSteerQueuedPrompt={onSteerQueuedPrompt}
      />
    </ThreadPrimitive.Root>
  );
}
