import type { PillTone } from "@agh/ui";

import type { SessionPayload } from "@/systems/session";

export type AgentSessionStatusKind = "active" | "starting" | "stopping" | "failed" | "done";

export interface AgentSessionStatus {
  kind: AgentSessionStatusKind;
  label: string;
  tone: PillTone;
}

const ACTIVE_STATUS: AgentSessionStatus = { kind: "active", label: "ACTIVE", tone: "success" };
const STARTING_STATUS: AgentSessionStatus = {
  kind: "starting",
  label: "STARTING",
  tone: "warning",
};
const STOPPING_STATUS: AgentSessionStatus = {
  kind: "stopping",
  label: "STOPPING",
  tone: "warning",
};
const FAILED_STATUS: AgentSessionStatus = { kind: "failed", label: "FAILED", tone: "danger" };
const DONE_STATUS: AgentSessionStatus = { kind: "done", label: "DONE", tone: "neutral" };

export function isAgentSessionFailure(session: SessionPayload): boolean {
  return (
    session.state === "stopped" &&
    (Boolean(session.failure) ||
      session.stop_reason === "agent_crashed" ||
      session.stop_reason === "error")
  );
}

export function getAgentSessionStatus(session: SessionPayload): AgentSessionStatus {
  switch (session.state) {
    case "active":
      return ACTIVE_STATUS;
    case "starting":
      return STARTING_STATUS;
    case "stopping":
      return STOPPING_STATUS;
    case "stopped":
      if (isAgentSessionFailure(session)) {
        return FAILED_STATUS;
      }
      return DONE_STATUS;
  }
}
