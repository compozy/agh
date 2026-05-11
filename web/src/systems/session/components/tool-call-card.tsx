import { memo, useMemo } from "react";

import { ChatToolCard as PrimitiveChatToolCard, type ChatToolStatus } from "@agh/ui";

import { getToolLabel } from "../lib/tool-labels";
import type { UIMessage } from "../types";
import { ExpandedToolContent } from "./tool-renderers/expanded-tool-content";

export interface ToolCallCardProps {
  message: UIMessage;
}

function statusFromMessage(message: UIMessage): ChatToolStatus {
  if (message.toolError) return "failed";
  if (message.toolResult !== undefined) return "completed";
  return "in_progress";
}

function formatJsonSource(input: Record<string, unknown> | undefined): string {
  if (!input || Object.keys(input).length === 0) return "";
  try {
    return JSON.stringify(input, null, 2);
  } catch {
    return String(input);
  }
}

function labelTestIdFor(status: ChatToolStatus): string {
  switch (status) {
    case "in_progress":
      return "tool-card-executing";
    case "failed":
      return "tool-card-error";
    case "completed":
      return "tool-card-success";
    case "pending":
      return "tool-card-pending";
  }
}

function progressLabelFor(message: UIMessage, status: ChatToolStatus): string {
  const toolName = message.toolName ?? "tool";
  if (status === "in_progress") {
    return getToolLabel(toolName, "active");
  }
  if (status === "failed") {
    return `Failed to ${getToolLabel(toolName, "failure")}`;
  }
  return getToolLabel(toolName, "past");
}

/**
 * Chat-thread tool surface consuming `<ChatToolCard>` per ADR-014 §6. Maps
 * the legacy `UIMessage.toolResult / toolError / toolName` shape onto the
 * primitive's `{ toolName, status, input, output, errorMessage }` API and
 * delegates per-tool output rendering to the existing `ExpandedToolContent`
 * dispatcher.
 */
export const ToolCallCard = memo(
  function ToolCallCard({ message }: ToolCallCardProps) {
    const status = statusFromMessage(message);
    const toolName = message.toolName ?? "tool";
    const progressLabel = progressLabelFor(message, status);
    const labelTestId = labelTestIdFor(status);

    const inputSource = useMemo(() => formatJsonSource(message.toolInput), [message.toolInput]);
    const input = inputSource ? { source: inputSource, format: "code" as const } : undefined;
    const output = message.toolResult
      ? { node: <ExpandedToolContent message={message} /> }
      : undefined;
    const errorMessage = status === "failed" ? progressLabel : undefined;

    return (
      <div data-testid="tool-call-card">
        <PrimitiveChatToolCard
          toolName={toolName}
          status={status}
          input={input}
          output={output}
          errorMessage={errorMessage}
        />
        <span data-testid={labelTestId} className="sr-only">
          {progressLabel}
        </span>
      </div>
    );
  },
  (previous, next) =>
    previous.message.toolInput === next.message.toolInput &&
    previous.message.toolResult === next.message.toolResult &&
    previous.message.toolError === next.message.toolError &&
    previous.message.toolName === next.message.toolName
);
