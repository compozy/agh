import { Square, Play } from "lucide-react";

import { cn } from "@/lib/utils";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import type { SessionPayload } from "../types";

export interface ChatHeaderProps {
  session: SessionPayload;
  onStop: () => void;
  onResume: () => void;
  workspaceName?: string;
}

const STATE_BADGE_VARIANT: Record<string, "default" | "secondary" | "destructive" | "outline"> = {
  active: "default",
  starting: "secondary",
  stopping: "secondary",
  stopped: "outline",
};

export function ChatHeader({ session, onStop, onResume, workspaceName }: ChatHeaderProps) {
  const isActive = session.state === "active" || session.state === "starting";
  const isStopped = session.state === "stopped";

  return (
    <div
      className={cn(
        "flex items-center justify-between border-b px-4 py-2",
        "border-[color:var(--color-divider)] bg-[color:var(--color-surface)]"
      )}
      data-testid="chat-header"
    >
      <div className="flex items-center gap-3 overflow-hidden">
        <div className="min-w-0">
          <div className="flex items-center gap-2">
            <h2 className="truncate text-sm font-medium text-[color:var(--color-text-primary)]">
              {session.name ?? session.id}
            </h2>
            <Badge
              variant={STATE_BADGE_VARIANT[session.state] ?? "secondary"}
              className={cn("text-[0.625rem]", session.state === "starting" && "animate-pulse")}
              data-testid="session-state-badge"
            >
              {session.state}
            </Badge>
          </div>
          <div className="mt-1 flex items-center gap-2 overflow-hidden text-xs text-[color:var(--color-text-tertiary)]">
            <span>{session.agent_name}</span>
            {workspaceName && (
              <Badge
                variant="outline"
                className="h-4 shrink-0 px-1 text-[0.55rem] leading-none"
                data-testid="session-workspace-badge"
              >
                {workspaceName}
              </Badge>
            )}
            <span
              className="truncate font-mono text-[0.68rem]"
              data-testid="session-workspace-id"
              title={session.workspace_id}
            >
              {session.workspace_id}
            </span>
          </div>
        </div>
      </div>

      <div className="flex items-center gap-1">
        {isActive && (
          <Button variant="ghost" size="icon-sm" onClick={onStop} data-testid="stop-button">
            <Square className="size-3.5" />
          </Button>
        )}
        {isStopped && (
          <Button variant="ghost" size="icon-sm" onClick={onResume} data-testid="resume-button">
            <Play className="size-3.5" />
          </Button>
        )}
      </div>
    </div>
  );
}
