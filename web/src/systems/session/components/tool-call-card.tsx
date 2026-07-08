import { memo, useMemo } from "react";

import {
  CodeBlock,
  CopyIconButton,
  ToolCallRow as PrimitiveToolCallRow,
  type ToolCallStatus,
} from "@agh/ui";

import { deriveToolRowStatus, hasToolInput, toolResultIsEmpty } from "../lib/message-parts";
import {
  getToolCompactSummary,
  getToolIcon,
  getToolLabel,
  resolveRegisteredToolName,
} from "../lib/tool-labels";
import type { UIMessage } from "../types";
import { ExpandedToolContent } from "./tool-renderers/expanded-tool-content";

export interface ToolCallRowProps {
  message: UIMessage;
  defaultExpanded?: boolean;
  /**
   * True once the owning turn has settled. Neutral (empty-output) tools show
   * `empty` (Minus) while the turn streams and promote to `success` (Check) only
   * after it settles. Defaults to `false` (assume mid-stream unless told).
   */
  turnSettled?: boolean;
}

function formatJsonSource(input: Record<string, unknown> | undefined): string {
  if (!input || Object.keys(input).length === 0) return "";
  try {
    return JSON.stringify(input, null, 2);
  } catch {
    return String(input);
  }
}

function formatToolPayload(message: UIMessage): string {
  try {
    return JSON.stringify(
      {
        tool: message.toolName,
        input: message.toolInput ?? {},
        output: message.toolResult ?? null,
        error: message.toolError === true,
      },
      null,
      2
    );
  } catch {
    return String(message.toolName ?? "tool");
  }
}

function progressLabelFor(
  message: UIMessage,
  status: ToolCallStatus,
  runtimeError: boolean
): string {
  const toolName = resolveRegisteredToolName(message.toolName ?? "tool");
  if (status === "pending" || status === "running") {
    return getToolLabel(toolName, "active");
  }
  // Only a true runtime error takes the "Failed to …" verb (danger heading);
  // error-shaped output keeps the neutral past-tense verb + the X glyph.
  if (status === "failed" && runtimeError) {
    return `Failed to ${getToolLabel(toolName, "failure")}`;
  }
  return getToolLabel(toolName, "past");
}

/**
 * Chat-thread tool surface composing `<ToolCallRow>` from `@agh/ui`. Maps the
 * legacy `UIMessage.toolResult / toolError / toolName` shape onto the compound
 * `<ToolCallRow.Input>` + `<ToolCallRow.Output>` slots and drives one row-state
 * language via `deriveToolRowStatus`: pending / running / failed / success /
 * empty, with neutral→success promotion gated on `turnSettled`.
 */
export const ToolCallRow = memo(
  function ToolCallRow({
    message,
    defaultExpanded = false,
    turnSettled = false,
  }: ToolCallRowProps) {
    const { status, runtimeError } = deriveToolRowStatus({
      toolError: message.toolError,
      toolResult: message.toolResult,
      hasInput: hasToolInput(message.toolInput),
      turnSettled,
    });
    const registryTool = resolveRegisteredToolName(message.toolName ?? "tool");
    const compactSummary = getToolCompactSummary(registryTool, message.toolInput);
    const progressLabel = progressLabelFor(message, status, runtimeError);
    const toolIcon = getToolIcon(registryTool, message.toolInput);
    const inputJson = useMemo(() => formatJsonSource(message.toolInput), [message.toolInput]);
    const copyPayload = useMemo(() => formatToolPayload(message), [message]);
    const hasOutput = !toolResultIsEmpty(message.toolResult);
    const errorText =
      typeof message.toolResult?.error === "string" ? message.toolResult.error : undefined;
    const errorMessage = status === "failed" ? errorText : undefined;
    return (
      <div data-testid="tool-call-row">
        <PrimitiveToolCallRow
          toolName={progressLabel}
          icon={toolIcon}
          preview={compactSummary}
          status={status}
          runtimeError={runtimeError}
          errorMessage={errorMessage}
          defaultExpanded={defaultExpanded || status === "failed"}
        >
          <div className="flex min-w-0 items-center justify-end">
            <CopyIconButton
              value={copyPayload}
              copyLabel="Copy tool payload"
              copiedLabel="Tool payload copied"
              copyFailedLabel="Tool payload copy failed"
              className="text-subtle hover:text-fg"
            />
          </div>
          {inputJson ? (
            <PrimitiveToolCallRow.Input>
              <CodeBlock
                code={inputJson}
                language="json"
                density="compact"
                showPrompt={false}
                copyable={false}
              />
            </PrimitiveToolCallRow.Input>
          ) : null}
          {hasOutput ? (
            <PrimitiveToolCallRow.Output>
              <ExpandedToolContent message={message} />
            </PrimitiveToolCallRow.Output>
          ) : null}
        </PrimitiveToolCallRow>
      </div>
    );
  },
  (previous, next) =>
    previous.message.toolInput === next.message.toolInput &&
    previous.message.toolResult === next.message.toolResult &&
    previous.message.toolError === next.message.toolError &&
    previous.message.toolName === next.message.toolName &&
    previous.defaultExpanded === next.defaultExpanded &&
    previous.turnSettled === next.turnSettled
);
