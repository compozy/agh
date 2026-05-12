import type { ToolCallMessagePartProps, Toolkit } from "@assistant-ui/react";
import { makeAssistantDataUI } from "@assistant-ui/react";
import { AlertCircle } from "lucide-react";

import { Eyebrow, Spinner } from "@agh/ui";

import { cn } from "@/lib/utils";
import { PermissionDataPart } from "../components/permission-prompt";
import { RuntimeActivityNotice } from "../components/runtime-activity-notice";
import { ToolCallCard } from "../components/tool-call-card";
import type { AgentEventPayload, AghPermissionData, UIMessage } from "../types";
import { isAgentEventPayload, parseToolUseResult } from "./message-parts";

type SessionToolPartProps = ToolCallMessagePartProps<Record<string, unknown>, unknown>;

function toLegacyToolMessage(part: SessionToolPartProps): UIMessage {
  const result = isAgentEventPayload(part.result) ? parseToolUseResult(part.result) : null;

  return {
    id: part.toolCallId,
    role: part.result || part.isError ? "tool_result" : "tool_call",
    content: "",
    toolName: part.toolName,
    toolInput: part.args,
    toolResult: result ?? undefined,
    toolError: part.isError,
    isStreaming: part.status.type === "running",
    timestamp: Date.now(),
  };
}

function BackendToolPart({ part }: { part: SessionToolPartProps }) {
  if (part.status.type === "running" && Object.keys(part.args ?? {}).length === 0) {
    return (
      <div
        className={cn(
          "flex items-center gap-2 rounded-md border px-3 py-2",
          "border-line bg-canvas",
          "text-xs text-subtle"
        )}
      >
        <Spinner className="size-3" />
        <Eyebrow className="text-subtle">{part.toolName}</Eyebrow>
        <span>preparing input</span>
      </div>
    );
  }

  if (part.isError && !part.result) {
    return (
      <div
        className={cn(
          "flex items-center gap-2 rounded-md border px-3 py-2",
          "border-danger/30 bg-danger/8",
          "text-xs text-danger"
        )}
      >
        <AlertCircle className="size-3" />
        <span className="font-medium">{part.toolName}</span>
      </div>
    );
  }

  return <ToolCallCard message={toLegacyToolMessage(part)} />;
}

function createBackendTool() {
  return { type: "backend" as const };
}

export const sessionToolkit: Toolkit = {
  Bash: {
    ...createBackendTool(),
    render: part => <BackendToolPart part={part} />,
  },
  Read: {
    ...createBackendTool(),
    render: part => <BackendToolPart part={part} />,
  },
  Write: {
    ...createBackendTool(),
    render: part => <BackendToolPart part={part} />,
  },
  Edit: {
    ...createBackendTool(),
    render: part => <BackendToolPart part={part} />,
  },
  Grep: {
    ...createBackendTool(),
    render: part => <BackendToolPart part={part} />,
  },
  Glob: {
    ...createBackendTool(),
    render: part => <BackendToolPart part={part} />,
  },
};

export function createAghPermissionDataUI(sessionId: string) {
  return makeAssistantDataUI<AghPermissionData>({
    name: "agh-permission",
    render: ({ data }) => <PermissionDataPart data={data} sessionId={sessionId} />,
  });
}

export function createAghEventDataUI() {
  return makeAssistantDataUI<AgentEventPayload>({
    name: "agh-event",
    render: ({ data }) => <RuntimeActivityNotice event={data} />,
  });
}
