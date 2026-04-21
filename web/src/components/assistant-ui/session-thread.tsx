import { type ComponentPropsWithoutRef } from "react";
import { Loader2, SendHorizontal, Square } from "lucide-react";
import { AuiIf, ComposerPrimitive, MessagePrimitive, ThreadPrimitive } from "@assistant-ui/react";

import { cn } from "@/lib/utils";
import { MessageMarkdown } from "@/systems/session/components/message-markdown";
import { ThinkingBlock } from "@/systems/session/components/thinking-block";
import { useSessionComposerState } from "./hooks/use-session-composer-state";

interface SessionThreadProps {
  sessionId: string;
  agentName: string;
  canPrompt: boolean;
  onCancelPrompt: () => void;
}

function SessionTextPart({ text }: { text: string }) {
  return (
    <div className="text-sm leading-7 text-[color:var(--color-text-primary)]">
      <MessageMarkdown content={text} />
    </div>
  );
}

function SessionReasoningPart({ text, state }: { text: string; state?: { type: string } }) {
  return <ThinkingBlock thinking={text} thinkingComplete={state?.type !== "running"} />;
}

function SessionMessageEmpty({ status }: { status: { type: string } }) {
  if (status.type !== "running") {
    return null;
  }

  return (
    <div className="flex items-center gap-2 text-sm text-[color:var(--color-text-tertiary)]">
      <Loader2 className="size-4 animate-spin" />
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
          "border-[color:var(--color-divider)] bg-[color:var(--color-surface-panel)]"
        )}
      >
        <MessagePrimitive.Parts
          components={{
            Text: ({ text }) => <SessionTextPart text={text} />,
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
            Text: ({ text }) => <SessionTextPart text={text} />,
            Reasoning: ({ text, status }) => <SessionReasoningPart text={text} state={status} />,
            Empty: ({ status }) => <SessionMessageEmpty status={status} />,
          }}
        />
        <AuiIf
          condition={state =>
            state.message.status?.type === "incomplete" && state.message.status.reason === "error"
          }
        >
          <div
            className={cn(
              "rounded-[var(--radius-md)] border px-3 py-2 text-sm",
              "border-[color:var(--color-danger)]/30 bg-[color:var(--color-danger)]/8",
              "text-[color:var(--color-danger)]"
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
}: Pick<SessionThreadProps, "sessionId" | "canPrompt" | "onCancelPrompt">) {
  const { isRunning } = useSessionComposerState(sessionId);

  return (
    <div
      className={cn(
        "border-t px-4 py-3",
        "border-[color:var(--color-divider)] bg-[color:var(--color-surface-panel)]"
      )}
    >
      <ComposerPrimitive.Root
        className={cn(
          "flex flex-col gap-2 rounded-xl border px-3 pt-2.5 pb-2",
          "border-[color:var(--color-divider)] bg-[color:var(--color-surface)]",
          "focus-within:border-[color:var(--color-accent)] transition-colors"
        )}
      >
        <ComposerPrimitive.Input
          disabled={!canPrompt}
          placeholder={canPrompt ? "Send a message..." : "Session is not active"}
          rows={1}
          maxRows={12}
          submitMode="enter"
          className={cn(
            "min-h-6 w-full resize-none border-none bg-transparent p-0 text-sm leading-relaxed",
            "text-[color:var(--color-text-primary)] placeholder:text-[color:var(--color-text-tertiary)]",
            "shadow-none outline-none focus-visible:border-transparent focus-visible:ring-0",
            "dark:bg-transparent"
          )}
        />
        <div className="flex items-center justify-end">
          {isRunning ? (
            <button
              type="button"
              onClick={onCancelPrompt}
              className={cn(
                "inline-flex h-9 items-center gap-2 rounded-full px-3",
                "bg-[color:var(--color-danger)]/12 text-[color:var(--color-danger)]",
                "transition-colors hover:bg-[color:var(--color-danger)]/18"
              )}
            >
              <Square className="size-3.5 fill-current" />
              <span className="text-sm font-medium">Stop</span>
            </button>
          ) : (
            <ComposerPrimitive.Send
              className={cn(
                "inline-flex size-9 items-center justify-center rounded-full",
                "bg-[color:var(--color-accent)] text-white transition-colors",
                "hover:bg-[color:var(--color-accent-hover)] disabled:cursor-not-allowed disabled:opacity-50"
              )}
            >
              <SendHorizontal className="size-4" />
            </ComposerPrimitive.Send>
          )}
        </div>
      </ComposerPrimitive.Root>
    </div>
  );
}

function ThreadEmpty({ agentName }: Pick<SessionThreadProps, "agentName">) {
  return (
    <div className="mx-auto flex h-full w-full max-w-3xl items-center justify-center px-4 py-12">
      <div className="max-w-md text-center">
        <p className="font-mono text-[11px] tracking-[0.08em] text-[color:var(--color-text-tertiary)] uppercase">
          {agentName}
        </p>
        <p className="mt-2 text-sm text-[color:var(--color-text-secondary)]">
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
    />
  );
}

export function SessionThread({
  sessionId,
  agentName,
  canPrompt,
  onCancelPrompt,
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
      />
    </ThreadPrimitive.Root>
  );
}
