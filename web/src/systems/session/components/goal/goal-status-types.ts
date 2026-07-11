import type { ComponentProps } from "react";

import type { OperationResponse } from "@/lib/api-contract";

export type SessionGoalSnapshot = NonNullable<OperationResponse<"getSessionGoal", 200>["goal"]>;

export type GoalComposerAffordance =
  | {
      kind: "replace";
      expectedRunId: string;
      objective: string;
    }
  | {
      kind: "draft";
      expandedObjective: string;
    };

export type GoalControlAction = "pause" | "resume" | "approve" | "clear";

export interface GoalStatusChipProps extends Omit<ComponentProps<"section">, "children"> {
  snapshot: SessionGoalSnapshot;
  composerAffordance?: GoalComposerAffordance;
  pendingAction?: GoalControlAction | "prefill";
  onPause?: () => void;
  onResume?: () => void;
  onApprove?: () => void;
  onClear?: () => void;
  onPrefillComposer?: (text: string) => void;
}
