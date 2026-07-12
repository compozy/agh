import type { DataMessagePartProps } from "@assistant-ui/react";

import { PermissionDataPart } from "../components/permission-prompt";
import { RuntimeActivityNotice } from "../components/runtime-activity-notice";
import { useSessionRuntimeRenderContext } from "../hooks/use-session-runtime-render-context";
import type { AgentEventPayload, AghPermissionData } from "../types";

export function AghPermissionDataRenderer({ data }: DataMessagePartProps<AghPermissionData>) {
  const renderContext = useSessionRuntimeRenderContext();
  if (!renderContext) {
    throw new Error("AghPermissionDataUI requires SessionRuntimeRenderProvider");
  }
  return (
    <PermissionDataPart
      data={data}
      sessionId={renderContext.sessionId}
      workspaceId={renderContext.workspaceId}
    />
  );
}

export function AghEventDataRenderer({ data }: DataMessagePartProps<AgentEventPayload>) {
  return <RuntimeActivityNotice event={data} />;
}
