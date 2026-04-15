import { memo } from "react";
import { ArrowDown, MessageSquare } from "lucide-react";

import { Button } from "@agh/ui";
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
      <div className="flex flex-1 items-center justify-center" data-testid="chat-empty-state">
        <div className="flex flex-col items-center gap-3">
          <MessageSquare className="size-8 text-[color:var(--color-text-tertiary)]/30" />
          <p className="text-sm italic text-[color:var(--color-text-tertiary)]">
            Send a message to start the conversation
          </p>
        </div>
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
      >
        <div
          style={{
            height: `${view.virtualizer.getTotalSize()}px`,
            position: "relative",
            width: "100%",
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

      {view.showScrollButton && (
        <div className="absolute bottom-2 left-1/2 -translate-x-1/2">
          <Button
            variant="secondary"
            size="sm"
            onClick={view.scrollToBottom}
            className={cn("border border-[color:var(--color-divider)]")}
            data-testid="scroll-to-bottom"
          >
            <ArrowDown className="size-3.5" />
            <span className="text-xs">Scroll to bottom</span>
          </Button>
        </div>
      )}
    </div>
  );
}
