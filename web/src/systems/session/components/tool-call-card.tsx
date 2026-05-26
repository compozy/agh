import { memo, useMemo } from "react";

import { CodeBlock, ToolCallCard as PrimitiveToolCallCard, type ToolCallStatus } from "@agh/ui";

import { getToolCompactSummary, getToolLabel, resolveRegisteredToolName } from "../lib/tool-labels";
import type { UIMessage } from "../types";
import { ExpandedToolContent } from "./tool-renderers/expanded-tool-content";

export interface ToolCallCardProps {
  message: UIMessage;
}

function statusFromMessage(message: UIMessage): ToolCallStatus {
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

function labelTestIdFor(status: ToolCallStatus): string {
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

function progressLabelFor(message: UIMessage, status: ToolCallStatus): string {
  const toolName = resolveRegisteredToolName(message.toolName ?? "tool");
  if (status === "in_progress") {
    return getToolLabel(toolName, "active");
  }
  if (status === "failed") {
    return `Failed to ${getToolLabel(toolName, "failure")}`;
  }
  return getToolLabel(toolName, "past");
}

/**
 * Chat-thread tool surface composing `<ToolCallCard>` from `@agh/ui`. Maps the
 * legacy `UIMessage.toolResult / toolError / toolName` shape onto the
 * compound `<ToolCallCard.Input>` + `<ToolCallCard.Output>` slots and
 * delegates per-tool output rendering to the existing `ExpandedToolContent`
 * dispatcher.
 */
export const ToolCallCard = memo(
  function ToolCallCard({ message }: ToolCallCardProps) {
    const status = statusFromMessage(message);
    const registryTool = resolveRegisteredToolName(message.toolName ?? "tool");
    const compactSummary = getToolCompactSummary(registryTool, message.toolInput);
    const progressLabel = progressLabelFor(message, status);
    const labelTestId = labelTestIdFor(status);
    const inputJson = useMemo(() => formatJsonSource(message.toolInput), [message.toolInput]);
    const hasOutput = message.toolResult !== undefined;
    const errorMessage = status === "failed" ? progressLabel : undefined;
    return (
      <div data-testid="tool-call-card">
        <PrimitiveToolCallCard
          toolName={registryTool}
          filePath={compactSummary}
          status={status}
          errorMessage={errorMessage}
        >
          {inputJson ? (
            <PrimitiveToolCallCard.Input>
              <CodeBlock language="json" code={inputJson} />
            </PrimitiveToolCallCard.Input>
          ) : null}
          {hasOutput ? (
            <PrimitiveToolCallCard.Output>
              <ExpandedToolContent message={message} />
            </PrimitiveToolCallCard.Output>
          ) : null}
        </PrimitiveToolCallCard>
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
