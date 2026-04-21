import { Inbox } from "lucide-react";

import { Avatar, AvatarFallback, cn, Empty, Pill, StatusDot, type StatusDotTone } from "@agh/ui";

import type { WorkspacePayload } from "../types";

interface WorkspaceSelectorProps {
  workspaces: WorkspacePayload[];
  activeWorkspaceId: string | null;
  onSelectWorkspace: (workspaceId: string) => void;
  globalWorkspaceId?: string | null;
  disabled?: boolean;
  className?: string;
}

function workspaceInitial(name: string): string {
  return name.trim().charAt(0).toUpperCase() || "·";
}

function WorkspaceSelector({
  workspaces,
  activeWorkspaceId,
  onSelectWorkspace,
  globalWorkspaceId = null,
  disabled = false,
  className,
}: WorkspaceSelectorProps) {
  if (workspaces.length === 0) {
    return (
      <Empty
        icon={Inbox}
        title="No workspaces"
        description="Register a workspace to activate AGH for this machine."
        data-testid="workspace-selector-empty"
        className={className}
        fill={false}
      />
    );
  }

  return (
    <ul
      data-testid="workspace-selector"
      aria-label="Workspaces"
      className={cn("flex flex-col gap-1", className)}
    >
      {workspaces.map(workspace => {
        const isActive = workspace.id === activeWorkspaceId;
        const isGlobal = globalWorkspaceId !== null && workspace.id === globalWorkspaceId;
        const dotTone: StatusDotTone = isActive ? "success" : "neutral";
        const initial = workspaceInitial(workspace.name);
        const rootDirId = `workspace-selector-root-dir-${workspace.id}`;

        return (
          <li key={workspace.id}>
            <button
              type="button"
              aria-current={isActive ? "true" : undefined}
              aria-describedby={rootDirId}
              disabled={disabled}
              data-testid={`workspace-selector-item-${workspace.id}`}
              data-active={isActive}
              onClick={() => onSelectWorkspace(workspace.id)}
              className={cn(
                "group flex w-full items-center gap-3 rounded-xl border border-transparent bg-[color:var(--color-surface)] px-2.5 py-2 text-left transition-colors",
                "hover:bg-[color:var(--color-hover)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[color:var(--color-accent)]",
                isActive &&
                  "border-[color:var(--color-accent)] bg-[color:var(--color-surface-elevated)]",
                "disabled:pointer-events-none disabled:opacity-50"
              )}
            >
              <Avatar size="sm" className="shrink-0">
                <AvatarFallback
                  className={cn(
                    "font-mono text-[11px] font-semibold tracking-[0.02em]",
                    isActive
                      ? "bg-[color:var(--color-accent-tint)] text-[color:var(--color-accent)]"
                      : "bg-[color:var(--color-surface-elevated)] text-[color:var(--color-text-secondary)]"
                  )}
                >
                  {initial}
                </AvatarFallback>
              </Avatar>
              <div className="flex min-w-0 flex-1 flex-col gap-0.5">
                <div className="flex items-center gap-2">
                  <span
                    className="truncate text-[13px] font-medium text-[color:var(--color-text-primary)]"
                    data-testid={`workspace-selector-name-${workspace.id}`}
                  >
                    {workspace.name}
                  </span>
                  {isGlobal ? (
                    <Pill variant="accent" data-testid={`workspace-selector-home-${workspace.id}`}>
                      HOME
                    </Pill>
                  ) : (
                    <Pill data-testid={`workspace-selector-path-${workspace.id}`}>PATH</Pill>
                  )}
                </div>
                <span
                  className="truncate font-mono text-[0.68rem] text-[color:var(--color-text-tertiary)]"
                  data-testid={`workspace-selector-root-dir-${workspace.id}`}
                  id={rootDirId}
                  title={workspace.root_dir}
                >
                  {workspace.root_dir}
                </span>
              </div>
              <StatusDot
                tone={dotTone}
                size="md"
                data-testid={`workspace-selector-dot-${workspace.id}`}
              />
            </button>
          </li>
        );
      })}
    </ul>
  );
}

export { WorkspaceSelector };
