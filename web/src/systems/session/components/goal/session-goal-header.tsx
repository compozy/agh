import { useSessionGoalHeader } from "../../hooks/use-session-goal-header";
import { GoalStatusChip } from "./goal-status-chip";

interface SessionGoalHeaderProps {
  sessionId: string;
  workspaceId: string;
}

export function SessionGoalHeader({ sessionId, workspaceId }: SessionGoalHeaderProps) {
  const goal = useSessionGoalHeader(workspaceId, sessionId);

  if (goal.error) {
    return (
      <div
        className="border-b border-line bg-danger-tint px-4 py-2 text-small-body text-danger"
        role="alert"
      >
        Goal status unavailable. {goal.error.message}
      </div>
    );
  }
  if (!goal.snapshot) return null;

  return (
    <div className="border-b border-line bg-canvas px-4 py-3" data-testid="session-goal-header">
      <GoalStatusChip
        snapshot={goal.snapshot}
        composerAffordance={goal.composerAffordance}
        pendingAction={goal.pendingAction}
        onPause={goal.onPause}
        onResume={goal.onResume}
        onApprove={goal.onApprove}
        onClear={goal.onClear}
        onPrefillComposer={goal.onPrefillComposer}
      />
    </div>
  );
}
