import { Webhook } from "lucide-react";

import { ScopeSelector } from "@/systems/workspace";

import type { WorkspaceOption } from "../../lib/trigger-preview";
import type { AutomationScope } from "../../types";

interface ScopeStepProps {
  scope: AutomationScope;
  workspaceId: string | undefined;
  workspaces: ReadonlyArray<WorkspaceOption>;
  isWebhook: boolean;
  onScopeChange: (scope: AutomationScope) => void;
  onWorkspaceChange: (workspaceId: string) => void;
}

/** "For" step: global vs workspace scope, with the webhook-is-always-global rule. */
export function ScopeStep({
  scope,
  workspaceId,
  workspaces,
  isWebhook,
  onScopeChange,
  onWorkspaceChange,
}: ScopeStepProps) {
  return (
    <div>
      <ScopeSelector
        ariaLabel="Trigger scope"
        onScopeChange={onScopeChange}
        onWorkspaceChange={onWorkspaceChange}
        scope={scope}
        testIdPrefix="trigger"
        workspaceDisabled={isWebhook}
        workspaceId={workspaceId}
        workspaces={workspaces}
      />
      {isWebhook ? (
        <div className="mt-3 flex items-center gap-2 text-form-hint text-subtle">
          <Webhook aria-hidden="true" className="size-3 shrink-0 text-warning" />
          Webhook triggers are always global; they aren&apos;t tied to a workspace.
        </div>
      ) : null}
    </div>
  );
}
