import { Box, Globe, Webhook } from "lucide-react";

import { Field, FieldTitle, NativeSelect, NativeSelectOption, PillGroup } from "@agh/ui";

import type { AutomationScope } from "../../types";
import type { WorkspaceOption } from "../../lib/trigger-preview";

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
      <div className="flex flex-wrap items-center gap-3">
        <PillGroup
          aria-label="Scope"
          items={[
            {
              value: "global",
              label: (
                <span className="flex items-center gap-1.5">
                  <Globe aria-hidden="true" className="size-3" />
                  Global
                </span>
              ),
              testId: "trigger-scope-global",
            },
            {
              value: "workspace",
              label: (
                <span className="flex items-center gap-1.5">
                  <Box aria-hidden="true" className="size-3" />
                  Workspace
                </span>
              ),
              testId: "trigger-scope-workspace",
              disabled: isWebhook,
            },
          ]}
          onChange={onScopeChange}
          value={scope}
        />
        {scope === "workspace" && !isWebhook ? (
          <div className="min-w-40 flex-1">
            <Field>
              <FieldTitle className="sr-only">Workspace</FieldTitle>
              <NativeSelect
                aria-label="Workspace"
                className="w-full"
                data-testid="trigger-workspace-select"
                onChange={event => onWorkspaceChange(event.target.value)}
                value={workspaceId ?? ""}
              >
                {workspaces.length === 0 ? (
                  <NativeSelectOption value="">No workspace available</NativeSelectOption>
                ) : null}
                {workspaces.map(workspace => (
                  <NativeSelectOption key={workspace.id} value={workspace.id}>
                    {workspace.name}
                  </NativeSelectOption>
                ))}
              </NativeSelect>
            </Field>
          </div>
        ) : null}
      </div>
      {isWebhook ? (
        <div className="mt-3 flex items-center gap-2 text-form-hint text-subtle">
          <Webhook aria-hidden="true" className="size-3 shrink-0 text-warning" />
          Webhook triggers are always global — they aren&apos;t tied to a workspace.
        </div>
      ) : null}
    </div>
  );
}
