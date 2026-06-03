import { ArrowRight, SlidersHorizontal } from "lucide-react";

import { ScopeSelector, type WorkspaceCommandSelectOption } from "@/systems/workspace";
import { PillGroup, type PillGroupItem } from "@agh/ui";

import type { TaskScope } from "../../types";

export type TaskFormMode = "simple" | "advanced";

interface ModeToolbarProps {
  mode: TaskFormMode;
  onModeChange: (mode: TaskFormMode) => void;
  scope: TaskScope;
  workspaceId: string | null;
  workspaces: ReadonlyArray<WorkspaceCommandSelectOption> | undefined;
  onScopeChange: (scope: TaskScope) => void;
  onWorkspaceChange: (workspaceId: string) => void;
}

const MODE_ITEMS: PillGroupItem<TaskFormMode>[] = [
  {
    value: "simple",
    label: (
      <span className="flex items-center gap-1.5">
        <ArrowRight aria-hidden="true" className="size-3" />
        Simple
      </span>
    ),
    testId: "task-mode-simple",
  },
  {
    value: "advanced",
    label: (
      <span className="flex items-center gap-1.5">
        <SlidersHorizontal aria-hidden="true" className="size-3" />
        Advanced
      </span>
    ),
    testId: "task-mode-advanced",
  },
];

/**
 * Pinned mode toolbar — Simple/Advanced segmented control on the left and the
 * shared task scope control on the right.
 */
export function ModeToolbar({
  mode,
  onModeChange,
  scope,
  workspaceId,
  workspaces,
  onScopeChange,
  onWorkspaceChange,
}: ModeToolbarProps) {
  return (
    <div className="flex flex-wrap items-center gap-3 border-y border-line-soft bg-canvas-tint px-6 py-3">
      <PillGroup
        aria-label="Editor mode"
        items={MODE_ITEMS}
        onChange={onModeChange}
        size="sm"
        value={mode}
      />
      <div className="flex-1" />
      <div className="flex min-w-0 flex-wrap items-center gap-2">
        <span className="text-form-hint text-subtle">Scope</span>
        <ScopeSelector
          ariaLabel="Task scope"
          onScopeChange={onScopeChange}
          onWorkspaceChange={onWorkspaceChange}
          scope={scope}
          testIdPrefix="task"
          workspaceId={workspaceId}
          workspaces={workspaces}
        />
      </div>
    </div>
  );
}
