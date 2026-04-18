import { ChevronRight, Plus } from "lucide-react";
import type { ReactNode } from "react";

import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@agh/ui";

import type { AgentPayload } from "../types";
import { AgentIcon } from "./agent-icon";

interface AgentSidebarGroupProps {
  agent: AgentPayload;
  onNewSession?: (agentName: string) => void;
  newSessionDisabled?: boolean;
  children?: ReactNode;
}

function AgentSidebarGroup({
  agent,
  onNewSession,
  newSessionDisabled = false,
  children,
}: AgentSidebarGroupProps) {
  const hasChildren = children != null;

  return (
    <Collapsible defaultOpen className="group/collapsible relative flex w-full min-w-0 flex-col">
      <div className="relative flex items-center">
        <CollapsibleTrigger className="flex min-h-7 flex-1 items-center gap-1.5 rounded-md px-2 text-left font-mono text-[0.64rem] uppercase tracking-[0.22em] text-[color:var(--color-text-label)] transition-colors hover:bg-[color:var(--color-hover)] hover:text-foreground focus-visible:ring-2 focus-visible:ring-[color:var(--color-accent)] focus-visible:outline-none">
          <AgentIcon provider={agent.provider} className="mr-1 size-3.5" />
          <span className="truncate">{agent.name}</span>
          <ChevronRight className="ml-auto size-3 transition-transform group-data-[panel-open]/collapsible:rotate-90" />
        </CollapsibleTrigger>
        <button
          type="button"
          title="New Session"
          aria-label="New Session"
          disabled={newSessionDisabled}
          onClick={() => onNewSession?.(agent.name)}
          className="absolute right-1 inline-flex size-6 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-[color:var(--color-hover)] hover:text-foreground focus-visible:ring-2 focus-visible:ring-[color:var(--color-accent)] focus-visible:outline-none disabled:pointer-events-none disabled:opacity-50"
        >
          <Plus className="size-3.5" />
          <span className="sr-only">New Session</span>
        </button>
      </div>
      <CollapsibleContent>
        <ul className="mx-3 mt-1 flex min-w-0 flex-col gap-0.5 border-l border-border pl-2">
          {hasChildren ? (
            children
          ) : (
            <li className="px-2 py-1 text-xs text-[color:var(--color-text-tertiary)]">
              No sessions
            </li>
          )}
        </ul>
      </CollapsibleContent>
    </Collapsible>
  );
}

export { AgentSidebarGroup };
