import { ChevronRight, Plus } from "lucide-react";
import type { ReactNode } from "react";

import { cn, Collapsible, CollapsibleContent, CollapsibleTrigger, MonoBadge } from "@agh/ui";

import type { AgentPayload } from "../types";
import { AgentIcon } from "./agent-icon";

interface AgentSidebarGroupProps {
  agent: AgentPayload;
  onNewSession?: (agentName: string) => void;
  newSessionDisabled?: boolean;
  defaultOpen?: boolean;
  sessionCount?: number;
  children?: ReactNode;
}

function AgentSidebarGroup({
  agent,
  onNewSession,
  newSessionDisabled = false,
  defaultOpen = true,
  sessionCount,
  children,
}: AgentSidebarGroupProps) {
  const hasChildren = children != null;
  const showCount = typeof sessionCount === "number" && sessionCount > 0;

  return (
    <Collapsible
      defaultOpen={defaultOpen}
      data-testid={`agent-sidebar-group-${agent.name}`}
      className="group/agent-group relative flex w-full min-w-0 flex-col"
    >
      <div className="relative flex items-center">
        <CollapsibleTrigger
          data-testid={`agent-sidebar-group-trigger-${agent.name}`}
          className={cn(
            "flex min-h-7 flex-1 items-center gap-1.5 rounded-md px-2 text-left transition-colors",
            "font-mono text-[0.64rem] uppercase tracking-[0.22em] text-[color:var(--color-text-label)]",
            "hover:bg-[color:var(--color-hover)] hover:text-[color:var(--color-text-primary)]",
            "focus-visible:ring-2 focus-visible:ring-[color:var(--color-accent)] focus-visible:outline-none"
          )}
        >
          <ChevronRight
            aria-hidden="true"
            className="size-3 shrink-0 text-[color:var(--color-text-tertiary)] transition-transform group-data-[panel-open]/agent-group:rotate-90"
          />
          <AgentIcon provider={agent.provider} tone="muted" className="size-3.5" />
          <span className="truncate">{agent.name}</span>
          {showCount ? (
            <MonoBadge
              tone="neutral"
              className="ml-auto px-1 py-0 normal-case tracking-normal"
              data-testid={`agent-sidebar-group-count-${agent.name}`}
            >
              {sessionCount}
            </MonoBadge>
          ) : null}
        </CollapsibleTrigger>
        <button
          type="button"
          title="New Session"
          aria-label="New Session"
          disabled={newSessionDisabled}
          onClick={() => onNewSession?.(agent.name)}
          data-testid={`agent-sidebar-group-new-session-${agent.name}`}
          className={cn(
            "absolute right-1 inline-flex size-6 items-center justify-center rounded-md transition-colors",
            "text-[color:var(--color-text-tertiary)] hover:bg-[color:var(--color-hover)] hover:text-[color:var(--color-text-primary)]",
            "focus-visible:ring-2 focus-visible:ring-[color:var(--color-accent)] focus-visible:outline-none",
            "disabled:pointer-events-none disabled:opacity-50"
          )}
        >
          <Plus className="size-3.5" />
          <span className="sr-only">New Session</span>
        </button>
      </div>
      <CollapsibleContent>
        <ul
          data-testid={`agent-sidebar-group-items-${agent.name}`}
          className="mx-3 mt-1 flex min-w-0 flex-col gap-0.5 border-l border-[color:var(--color-divider)] pl-2"
        >
          {hasChildren ? (
            children
          ) : (
            <li
              className="px-2 py-1 text-xs text-[color:var(--color-text-tertiary)]"
              data-testid={`agent-sidebar-group-empty-${agent.name}`}
            >
              No sessions
            </li>
          )}
        </ul>
      </CollapsibleContent>
    </Collapsible>
  );
}

export { AgentSidebarGroup };
