import { ArrowRight, SlidersHorizontal } from "lucide-react";

import { PillGroup, type PillGroupItem } from "@agh/ui";

import type { TaskScope } from "../../types";

export type TaskFormMode = "simple" | "advanced";

interface ModeToolbarProps {
  mode: TaskFormMode;
  onModeChange: (mode: TaskFormMode) => void;
  scope: TaskScope;
  /** Active workspace name — surfaces inside the Workspace scope pill label. */
  workspaceName?: string | null;
  onScopeChange: (scope: TaskScope) => void;
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

function buildScopeItems(workspaceName?: string | null): PillGroupItem<TaskScope>[] {
  return [
    {
      value: "workspace",
      label: workspaceName ? `Workspace · ${workspaceName}` : "Workspace",
      testId: "task-scope-workspace",
    },
    { value: "global", label: "Global", testId: "task-scope-global" },
  ];
}

/**
 * Pinned mode toolbar — Simple/Advanced segmented control on the left and the
 * task Scope control on the right. The task binds to the active workspace, so
 * the workspace name lives inside the Workspace pill label rather than a
 * separate select (truthful UI: there is no workspace re-pick here).
 */
export function ModeToolbar({
  mode,
  onModeChange,
  scope,
  workspaceName,
  onScopeChange,
}: ModeToolbarProps) {
  return (
    <div className="flex items-center gap-3 border-y border-line-soft bg-canvas-tint px-6 py-3">
      <PillGroup
        aria-label="Editor mode"
        items={MODE_ITEMS}
        onChange={onModeChange}
        size="sm"
        value={mode}
      />
      <div className="flex-1" />
      <span className="text-form-hint text-subtle">Scope</span>
      <PillGroup
        aria-label="Task scope"
        items={buildScopeItems(workspaceName)}
        onChange={onScopeChange}
        size="sm"
        value={scope}
      />
    </div>
  );
}
