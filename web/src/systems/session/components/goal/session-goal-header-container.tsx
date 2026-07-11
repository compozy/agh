import { useSessionGoalHeader } from "../../hooks/use-session-goal-header";
import { SessionGoalHeader } from "./session-goal-header";

interface SessionGoalHeaderContainerProps {
  sessionId: string;
  workspaceId: string;
}

export function SessionGoalHeaderContainer({
  sessionId,
  workspaceId,
}: SessionGoalHeaderContainerProps) {
  const goal = useSessionGoalHeader(workspaceId, sessionId);
  return <SessionGoalHeader {...goal} />;
}
