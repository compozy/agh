import { Info } from "lucide-react";

import { ScopeSelector } from "@/systems/workspace";

import type { WorkspaceOption } from "../../lib/trigger-preview";
import type { AutomationScope } from "../../types";

interface ScopeStepProps {
  scope: AutomationScope;
  workspaceId: string | undefined;
  workspaces: ReadonlyArray<WorkspaceOption>;
  onScopeChange: (scope: AutomationScope) => void;
  onWorkspaceChange: (workspaceId: string) => void;
}

/** "For" step: global vs workspace scope. Global jobs aren't bound to a workspace. */
export function ScopeStep({
  scope,
  workspaceId,
  workspaces,
  onScopeChange,
  onWorkspaceChange,
}: ScopeStepProps) {
  return (
    <div>
      <ScopeSelector
        ariaLabel="Job scope"
        onScopeChange={onScopeChange}
        onWorkspaceChange={onWorkspaceChange}
        scope={scope}
        testIdPrefix="job"
        workspaceId={workspaceId}
        workspaces={workspaces}
      />
      {scope === "global" ? (
        <div className="mt-3 flex items-center gap-2 text-form-hint text-subtle">
          <Info aria-hidden="true" className="size-3 shrink-0" />
          Global jobs aren&apos;t bound to a workspace; leave the workspace field empty.
        </div>
      ) : null}
    </div>
  );
}
