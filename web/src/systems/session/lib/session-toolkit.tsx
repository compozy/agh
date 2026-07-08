import type { DataMessagePartProps, ToolCallMessagePartProps, Toolkit } from "@assistant-ui/react";
import { makeAssistantDataUI } from "@assistant-ui/react";

import { PermissionDataPart } from "../components/permission-prompt";
import { RuntimeActivityNotice } from "../components/runtime-activity-notice";
import { ToolCallRow } from "../components/tool-call-card";
import type { AgentEventPayload, AghPermissionData, UIMessage } from "../types";
import { resolveToolResult } from "./message-parts";

type SessionToolPartProps = ToolCallMessagePartProps<Record<string, unknown>, unknown>;

export function toLegacyToolMessage(part: SessionToolPartProps): UIMessage {
  return {
    id: part.toolCallId,
    role: part.result !== undefined || part.isError ? "tool_result" : "tool_call",
    content: "",
    toolName: part.toolName,
    toolInput: part.args,
    toolResult: resolveToolResult(part.result),
    toolError: part.isError,
    isStreaming: part.status.type === "running",
    timestamp: Date.now(),
  };
}

export function BackendToolPart(part: SessionToolPartProps) {
  // A live single-tool render has no broader turn context, so the tool's own
  // completion is the settle signal for neutral→success promotion.
  return (
    <ToolCallRow message={toLegacyToolMessage(part)} turnSettled={part.status.type !== "running"} />
  );
}

function createBackendTool() {
  return { type: "backend" as const };
}

export const sessionToolkit: Toolkit = {
  Bash: {
    ...createBackendTool(),
    render: (part: SessionToolPartProps) => <BackendToolPart {...part} />,
  },
  Read: {
    ...createBackendTool(),
    render: (part: SessionToolPartProps) => <BackendToolPart {...part} />,
  },
  Write: {
    ...createBackendTool(),
    render: (part: SessionToolPartProps) => <BackendToolPart {...part} />,
  },
  Edit: {
    ...createBackendTool(),
    render: (part: SessionToolPartProps) => <BackendToolPart {...part} />,
  },
  Grep: {
    ...createBackendTool(),
    render: (part: SessionToolPartProps) => <BackendToolPart {...part} />,
  },
  Glob: {
    ...createBackendTool(),
    render: (part: SessionToolPartProps) => <BackendToolPart {...part} />,
  },
};

export function createAghPermissionDataUI(workspaceId: string, sessionId: string) {
  return makeAssistantDataUI<AghPermissionData>({
    name: "agh-permission",
    render: ({ data }: DataMessagePartProps<AghPermissionData>) => (
      <PermissionDataPart data={data} sessionId={sessionId} workspaceId={workspaceId} />
    ),
  });
}

export function createAghEventDataUI() {
  return makeAssistantDataUI<AgentEventPayload>({
    name: "agh-event",
    render: ({ data }: DataMessagePartProps<AgentEventPayload>) => (
      <RuntimeActivityNotice event={data} />
    ),
  });
}
