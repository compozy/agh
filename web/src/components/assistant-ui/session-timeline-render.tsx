import { ChevronRight, CircleStop } from "lucide-react";
import { useRef } from "react";
import type { ReactNode } from "react";

import { cn } from "@/lib/utils";
import { MessageMarkdown } from "@/systems/session/components/message-markdown";
import { PermissionDataPart } from "@/systems/session/components/permission-prompt";
import { RuntimeActivityNotice } from "@/systems/session/components/runtime-activity-notice";
import { ThinkingBlock } from "@/systems/session/components/thinking-block";
import { ToolCallRow } from "@/systems/session/components/tool-call-card";
import { useSessionRuntimeRenderContext } from "@/systems/session/hooks/use-session-runtime-render-context";
import { isAgentEventPayload, resolveToolResult } from "@/systems/session/lib/message-parts";
import type { AghPermissionData } from "@/systems/session/types";
import type { UIMessage } from "@/systems/session/types";
import { Button, Eyebrow } from "@agh/ui";
import { useAssistantMessageTimeline } from "./hooks/use-assistant-message-timeline";
import { SessionChangedFilesRowView } from "./session-changed-files-row";
import { SessionWorkingRowView } from "./session-working-row";
import {
  type SessionDataRow,
  type SessionReasoningRow,
  type SessionRow,
  type SessionTextRow,
  type SessionTimelineToolPart,
  type SessionWorkToggleRow,
  visibleWorkEntries,
} from "./session-timeline.logic";

function formatDataPreview(data: unknown): string | null {
  if (data === undefined || data === null) return null;
  if (typeof data === "string") return data;
  try {
    return JSON.stringify(data);
  } catch {
    return String(data);
  }
}

function isAghPermissionData(value: unknown): value is AghPermissionData {
  return isAgentEventPayload(value) && typeof value.request_id === "string";
}

function SessionTextRowView({ row }: { row: SessionTextRow }) {
  return (
    <div className="text-sm leading-7 text-fg">
      <MessageMarkdown content={row.part.text} streaming={row.part.state === "running"} />
    </div>
  );
}

function SessionReasoningRowView({ row }: { row: SessionReasoningRow }) {
  return (
    <ThinkingBlock
      thinking={row.text}
      thinkingComplete={!row.streaming}
      updateCount={row.updateCount}
    />
  );
}

function SessionDataRowView({ row }: { row: SessionDataRow }) {
  const renderContext = useSessionRuntimeRenderContext();
  if (row.part.name === "data-agh-event" && isAgentEventPayload(row.part.data)) {
    return <RuntimeActivityNotice event={row.part.data} />;
  }
  if (
    row.part.name === "data-agh-permission" &&
    renderContext &&
    isAghPermissionData(row.part.data)
  ) {
    return (
      <PermissionDataPart
        data={row.part.data}
        sessionId={renderContext.sessionId}
        workspaceId={renderContext.workspaceId}
      />
    );
  }

  const preview = formatDataPreview(row.part.data);
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
      <div className="min-w-0">
        <div className="text-card-title text-fg">Data event</div>
        <div className="truncate text-form-label text-subtle">{row.part.name}</div>
        {clippedPreview ? (
          <pre className="mt-1 max-h-24 overflow-auto whitespace-pre-wrap break-words font-mono text-small-body text-muted">
            {clippedPreview}
          </pre>
        ) : null}
      </div>
    </div>
  );
}

function toolMessageFromPart(part: SessionTimelineToolPart): UIMessage {
  return {
    id: part.toolCallId,
    role: part.result !== undefined || part.isError ? "tool_result" : "tool_call",
    content: "",
    toolName: part.toolName,
    toolInput: part.args,
    toolResult: resolveToolResult(part.result),
    toolError: part.isError,
    isStreaming: part.status === "running",
    timestamp: part.timestamp ? Date.parse(part.timestamp) : Date.now(),
  };
}

function workToggleLabel(row: SessionWorkToggleRow): string {
  if (row.expanded) {
    return "Show fewer tool calls";
  }
  return `+${row.hiddenCount} previous tool call${row.hiddenCount === 1 ? "" : "s"}`;
}

function WorkToggleButton({
  row,
  onToggle,
}: {
  row: SessionWorkToggleRow;
  onToggle: (button: HTMLElement | null) => void;
}) {
  const ref = useRef<HTMLButtonElement | null>(null);
  return (
    <Button
      ref={ref}
      type="button"
      variant="ghost"
      size="xs"
      data-testid="work-toggle-row"
      aria-expanded={row.expanded}
      onClick={() => onToggle(ref.current)}
      className="ml-7 w-fit gap-1 px-1 text-subtle hover:text-fg"
    >
      <ChevronRight
        className={cn(
          "size-3 transition-transform duration-base ease-out motion-reduce:transition-none",
          row.expanded ? "rotate-90" : null
        )}
        aria-hidden="true"
      />
      {workToggleLabel(row)}
    </Button>
  );
}

function TurnFoldButton({
  row,
  expanded,
  onToggle,
}: {
  row: Extract<SessionRow, { kind: "turn-fold" }>;
  expanded: boolean;
  onToggle: (button: HTMLElement | null) => void;
}) {
  const ref = useRef<HTMLButtonElement | null>(null);
  return (
    <Button
      ref={ref}
      type="button"
      variant="ghost"
      size="xs"
      data-testid="turn-fold-row"
      aria-expanded={expanded}
      onClick={() => onToggle(ref.current)}
      className="w-fit gap-1 px-1 text-subtle hover:text-fg"
    >
      <ChevronRight
        className={cn("size-3 transition-transform", expanded ? "rotate-90" : null)}
        aria-hidden="true"
      />
      {row.label}
    </Button>
  );
}

export function AssistantMessageTimeline() {
  const { expandedTurns, rows, toggleChangedFiles, toggleTurn, toggleWorkGroup } =
    useAssistantMessageTimeline();

  function renderRows(targetRows: readonly SessionRow[]): ReactNode {
    return targetRows.map(row => {
      switch (row.kind) {
        case "text":
          return <SessionTextRowView key={row.id} row={row} />;
        case "reasoning":
          return <SessionReasoningRowView key={row.id} row={row} />;
        case "data":
          return <SessionDataRowView key={row.id} row={row} />;
        case "working":
          return <SessionWorkingRowView key={row.id} row={row} />;
        case "work": {
          const tools = visibleWorkEntries(row);
          return (
            <div key={row.id} data-testid="work-row" className="flex min-w-0 flex-col gap-0.5">
              {row.grouped ? (
                <Eyebrow className="mb-0.5 px-1 text-subtle">
                  {row.entries.length} tool calls
                </Eyebrow>
              ) : null}
              {tools.map(tool => (
                <ToolCallRow
                  key={tool.id}
                  message={toolMessageFromPart(tool)}
                  turnSettled={!row.active}
                />
              ))}
            </div>
          );
        }
        case "work-toggle":
          return (
            <WorkToggleButton
              key={row.id}
              row={row}
              onToggle={button => toggleWorkGroup(row.groupId, button)}
            />
          );
        case "changed-files":
          return (
            <SessionChangedFilesRowView
              key={row.id}
              row={row}
              onToggle={button => toggleChangedFiles(row.id, button)}
            />
          );
        case "turn-fold": {
          // An interrupted turn keeps its work expanded so the operator keeps
          // their place; the fold row becomes a danger-toned interruption label
          // above the always-visible work (never a collapsing disclosure).
          if (row.interrupted) {
            return (
              <div
                key={row.id}
                data-testid="turn-fold-interrupted"
                className="flex min-w-0 flex-col gap-1"
              >
                <div
                  data-testid="turn-fold-interrupt-label"
                  className="flex w-fit items-center gap-1.5 px-1 text-small-body text-danger"
                >
                  <CircleStop className="size-3" aria-hidden="true" />
                  <span>{row.label}</span>
                </div>
                <div className="ml-4 flex min-w-0 flex-col gap-1">{renderRows(row.rows)}</div>
              </div>
            );
          }
          const expanded = expandedTurns.has(row.turnId ?? row.id);
          return (
            <div key={row.id} className="flex min-w-0 flex-col gap-1">
              <TurnFoldButton
                row={row}
                expanded={expanded}
                onToggle={button => toggleTurn(row.turnId ?? row.id, button)}
              />
              {expanded ? (
                <div className="ml-4 flex min-w-0 flex-col gap-1">{renderRows(row.rows)}</div>
              ) : null}
            </div>
          );
        }
      }
    });
  }

  return <>{renderRows(rows)}</>;
}
