import { Link, useMatchRoute } from "@tanstack/react-router";
import { Loader2 } from "lucide-react";

import { Badge } from "@agh/ui";

import { cn } from "@/lib/utils";
import type { SessionPayload, SessionState } from "../types";

interface SessionSidebarItemProps {
  session: SessionPayload;
  hasPendingPermission?: boolean;
  workspaceName?: string;
}

function StateBadge({ state }: { state: SessionState }) {
  switch (state) {
    case "active":
      return (
        <Badge variant="default" className="h-4 px-1 text-[0.55rem] leading-none">
          active
        </Badge>
      );
    case "starting":
      return (
        <Badge
          variant="outline"
          className="h-4 animate-pulse px-1 text-[0.55rem] leading-none text-amber-500 border-amber-500/30"
        >
          starting
        </Badge>
      );
    case "stopping":
      return (
        <Badge variant="outline" className="h-4 px-1 text-[0.55rem] leading-none">
          <Loader2 className="mr-0.5 size-2.5 animate-spin" />
          stopping
        </Badge>
      );
    case "stopped":
      return (
        <Badge variant="secondary" className="h-4 px-1 text-[0.55rem] leading-none">
          stopped
        </Badge>
      );
  }
}

function SessionSidebarItem({
  session,
  hasPendingPermission,
  workspaceName,
}: SessionSidebarItemProps) {
  const matchRoute = useMatchRoute();
  const isActive = !!matchRoute({ to: "/session/$id", params: { id: session.id } });
  const displayTitle = session.name || session.id.slice(0, 8);

  return (
    <li data-slot="session-sidebar-item" className="relative">
      <Link
        to="/session/$id"
        params={{ id: session.id }}
        data-active={isActive}
        data-testid="sidebar-sub-button"
        className={cn(
          "flex min-w-0 items-start gap-2 rounded-md px-2 py-2 text-xs text-muted-foreground transition-colors hover:bg-[color:var(--color-hover)] hover:text-foreground data-[active=true]:bg-[color:var(--color-surface-panel)] data-[active=true]:text-foreground focus-visible:ring-2 focus-visible:ring-[color:var(--color-accent)] focus-visible:outline-none",
          isActive && "font-medium"
        )}
      >
        <div className="min-w-0 flex-1">
          <span className="truncate text-xs">{displayTitle}</span>
          <div className="mt-1 flex items-center gap-1.5 overflow-hidden">
            {workspaceName && (
              <Badge
                variant="outline"
                className="h-4 shrink-0 px-1 text-[0.55rem] leading-none"
                data-testid="workspace-name-badge"
              >
                {workspaceName}
              </Badge>
            )}
            <span
              className="truncate font-mono text-[0.6rem] text-[color:var(--color-text-tertiary)]"
              data-testid="workspace-id-text"
              title={session.workspace_id}
            >
              {session.workspace_id}
            </span>
          </div>
        </div>
        <span className="ml-auto flex shrink-0 items-center gap-1.5">
          {hasPendingPermission && (
            <span
              className="size-2 animate-pulse rounded-full bg-amber-500"
              title="Permission pending"
              data-testid="permission-indicator"
            />
          )}
          <StateBadge state={session.state} />
        </span>
      </Link>
    </li>
  );
}

export { SessionSidebarItem };
