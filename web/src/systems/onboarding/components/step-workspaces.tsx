import { Folder, X } from "lucide-react";

import { Button, Eyebrow } from "@agh/ui";

import type { OnboardingWorkspacesApi } from "../hooks/use-onboarding-workspaces";
import { DirectoryBrowser } from "./directory-browser";

interface StepWorkspacesProps {
  workspaces: OnboardingWorkspacesApi;
}

export function StepWorkspaces({ workspaces }: StepWorkspacesProps) {
  const selected = workspaces.workspaces;

  return (
    <div className="flex flex-col gap-6" data-testid="onboarding-step-workspaces">
      <section className="flex flex-col gap-3">
        <Eyebrow className="text-subtle">Browse for a folder</Eyebrow>
        <DirectoryBrowser workspaces={workspaces} />
        {workspaces.resolveError ? (
          <p className="text-sm text-danger" role="alert" data-testid="onboarding-resolve-error">
            {workspaces.resolveError}
          </p>
        ) : null}
      </section>

      <section className="flex flex-col gap-3">
        <div className="flex items-center justify-between">
          <Eyebrow className="text-subtle">Selected workspaces</Eyebrow>
          <span className="text-xs text-faint tabular-nums">
            {selected.length} folder{selected.length === 1 ? "" : "s"}
          </span>
        </div>
        {selected.length === 0 ? (
          <p className="rounded-md border border-dashed border-line px-4 py-5 text-center text-sm text-faint">
            No folders yet — browse above and add at least one workspace to continue.
          </p>
        ) : (
          <ul className="flex flex-col gap-2">
            {selected.map(workspace => (
              <li
                key={workspace.path}
                className="flex items-center gap-3 rounded-md bg-canvas-soft px-3 py-2.5 ring-1 ring-inset ring-line"
                data-testid="onboarding-selected-workspace"
              >
                <span className="grid size-8 flex-none place-items-center rounded bg-elevated text-warning">
                  <Folder className="size-4" />
                </span>
                <span className="min-w-0 flex-1">
                  <span className="block truncate text-sm font-medium text-fg-strong">
                    {workspace.name}
                  </span>
                  <span className="block truncate font-mono text-xs text-subtle">
                    {workspace.path}
                  </span>
                </span>
                <Button
                  variant="ghost"
                  size="icon-sm"
                  onClick={() => workspaces.removeWorkspace(workspace.path)}
                  aria-label={`Remove ${workspace.name}`}
                >
                  <X className="size-3.5" />
                </Button>
              </li>
            ))}
          </ul>
        )}
      </section>
    </div>
  );
}
