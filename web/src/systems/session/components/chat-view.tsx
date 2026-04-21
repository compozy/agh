import { memo } from "react";
import { ArrowDown, MessageSquare } from "lucide-react";

import { Button, Empty } from "@agh/ui";

import { cn } from "@/lib/utils";
import { useChatViewContent } from "../hooks/use-chat-view-content";
import { mergeToolPairs, type RowDescriptor } from "../hooks/use-chat-view-rows";
import type { UIMessage } from "../types";
import { MessageBubble } from "./message-bubble";
import { ProcessingIndicator } from "./processing-indicator";
import { ToolGroupSection } from "./tool-group-section";

export { buildRows, mergeToolPairs, type RowDescriptor } from "../hooks/use-chat-view-rows";

interface ChatMessageRowProps {
  row: RowDescriptor;
  agentName?: string;
}

const ChatMessageRow = memo(
  function ChatMessageRow({ row, agentName }: ChatMessageRowProps) {
    if (row.kind === "processing") {
      return <ProcessingIndicator />;
    }

    if (row.kind === "tool_group") {
      const cards = mergeToolPairs(row.tools);
      return <ToolGroupSection tools={cards} />;
    }

    return <MessageBubble message={row.msg} agentName={agentName} />;
  },
  (previous, next) => previous.row === next.row && previous.agentName === next.agentName
);

export interface ChatViewProps {
  messages: UIMessage[];
  isStreaming: boolean;
  agentName?: string;
}

export const ChatView = memo(function ChatView({
  messages,
  isStreaming,
  agentName,
}: ChatViewProps) {
  if (messages.length === 0 && !isStreaming) {
    return (
      <div
        className="flex flex-1 items-center justify-center px-6 py-10"
        data-testid="chat-empty-state"
      >
        <Empty
          icon={
            <MessageSquare
              aria-hidden="true"
              data-testid="chat-empty-icon"
              className="size-5 text-[color:var(--color-text-tertiary)]/60"
            />
          }
          title="Start the conversation"
          description="Send a message to begin this session."
          className="max-w-md bg-transparent"
        />
      </div>
    );
  }

  return <ChatViewContent messages={messages} isStreaming={isStreaming} agentName={agentName} />;
});

function ChatViewContent({ messages, isStreaming, agentName }: ChatViewProps) {
  const view = useChatViewContent({ messages, isStreaming });

  return (
    <div className="relative flex flex-1 flex-col overflow-hidden" data-testid="chat-view">
      <div
        ref={view.scrollRef}
        className="min-h-0 flex-1 overflow-y-auto"
        style={{ overscrollBehaviorY: "contain" }}
        onPointerDown={view.markUserIntent}
        onScroll={view.handleScroll}
        data-testid="chat-view-scroll"
      >
        <div
          className="w-full"
          style={{
            height: `${view.virtualizer.getTotalSize()}px`,
            position: "relative",
          }}
        >
          {view.virtualizer.getVirtualItems().map(virtualRow => {
            const row = view.rows[virtualRow.index];

            return (
              <div
                key={virtualRow.key}
                ref={view.virtualizer.measureElement}
                data-index={virtualRow.index}
                style={{
                  left: 0,
                  position: "absolute",
                  top: 0,
                  transform: `translateY(${virtualRow.start}px)`,
                  width: "100%",
                }}
              >
                <ChatMessageRow row={row} agentName={agentName} />
              </div>
            );
          })}
        </div>
      </div>

      {view.showScrollButton ? (
        <div className="absolute bottom-3 left-1/2 -translate-x-1/2">
          <Button
            type="button"
            variant="secondary"
            size="sm"
            onClick={view.scrollToBottom}
            className={cn(
              "gap-1.5 rounded-full border border-[color:var(--color-divider)] bg-[color:var(--color-surface-panel)]/95 px-3 shadow-none backdrop-blur",
              "text-[color:var(--color-text-secondary)] hover:text-[color:var(--color-text-primary)]"
            )}
            data-testid="scroll-to-bottom"
          >
            <ArrowDown className="size-3.5" />
            <span className="text-[12px]">Scroll to bottom</span>
          </Button>
        </div>
      ) : null}
    </div>
  );
}
