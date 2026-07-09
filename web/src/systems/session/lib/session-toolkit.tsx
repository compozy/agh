import type { DataMessagePartProps, ToolDefinition, Toolkit } from "@assistant-ui/react";
import { makeAssistantDataUI } from "@assistant-ui/react";

import { PermissionDataPart } from "../components/permission-prompt";
import { RuntimeActivityNotice } from "../components/runtime-activity-notice";
import type { AgentEventPayload, AghPermissionData } from "../types";

// Every session tool executes server-side: the daemon streams each call and its
// result as already-resolved transcript parts, so the client registers these
// tools only to mark them backend-owned. Tool rendering is centralized in the
// session timeline's `ToolCallRow` (the single fallback renderer), so no toolkit
// entry carries a per-tool `render` — one shared backend definition is reused for
// every registered tool.
const backendTool: ToolDefinition = { type: "backend" };

export const sessionToolkit: Toolkit = {
  Bash: backendTool,
  Read: backendTool,
  Write: backendTool,
  Edit: backendTool,
  Grep: backendTool,
  Glob: backendTool,
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
