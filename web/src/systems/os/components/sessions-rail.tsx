import { ChevronLeft, ChevronRight, X } from "lucide-react";
import { useState } from "react";

import {
  Button,
  Eyebrow,
  Icon,
  PillDot,
  SearchInput,
  Sheet,
  SheetContent,
  SheetDescription,
  SheetTitle,
  StatusDot,
  Time,
  type PillTone,
} from "@agh/ui";

import { cn } from "@/lib/utils";
import { getSessionDisplayTitle, type SessionPayload } from "@/systems/session";

import { useDesktopSessionsRail } from "../hooks/use-desktop-sessions-rail";

type RailView = "recent" | "all";

interface SessionGroup {
  agentName: string;
  sessions: SessionPayload[];
}

function statusTone(badge: string): PillTone {
  if (badge === "running") return "accent";
  if (badge === "idle") return "success";
  if (badge === "waiting-for-auth" || badge === "hung") return "warning";
  if (badge === "failed" || badge === "unhealthy") return "danger";
  return "neutral";
}

function SessionStatusMark({ badge }: { badge: string }) {
  if (badge === "stopped") {
    return <StatusDot tone="faint" variant="ring" label="stopped" />;
  }
  return <PillDot tone={statusTone(badge)} pulse={badge === "running"} size="sm" />;
}

function SessionRow({ session, onSelect }: { session: SessionPayload; onSelect: () => void }) {
  return (
    <button
      type="button"
      className="grid w-full grid-cols-[8px_minmax(0,1fr)_auto] items-start gap-2 rounded-md px-2 py-1.5 text-left transition-colors hover:bg-row-hover focus-visible:shadow-focus-ring focus-visible:outline-none"
      data-status={session.badge}
      data-testid={`os-rail-session-${session.id}`}
      onClick={onSelect}
    >
      <span className="mt-1.5 grid place-items-center">
        <SessionStatusMark badge={session.badge} />
      </span>
      <span className="min-w-0">
        <span className="block truncate text-small-body text-fg-strong">
          {getSessionDisplayTitle(session)}
        </span>
        <span className="block truncate text-micro text-subtle">
          <span className="font-medium text-muted">{session.agent_name}</span>
          <span aria-hidden="true"> · </span>
          {session.badge}
        </span>
      </span>
      <Time iso={session.updated_at} className="mt-0.5 font-mono text-micro text-subtle" />
    </button>
  );
}

function SessionsRailBody({
  sessions,
  disconnected,
  collapsedAgentIds,
  onToggleGroup,
  onSelectSession,
  onClose,
}: {
  sessions: readonly SessionPayload[];
  disconnected: boolean;
  collapsedAgentIds: readonly string[];
  onToggleGroup: (agentName: string) => void;
  onSelectSession: (session: SessionPayload) => void;
  onClose: () => void;
}) {
  const [view, setView] = useState<RailView>("recent");
  const [filter, setFilter] = useState("");
  const normalizedFilter = filter.trim().toLocaleLowerCase();
  const filtered = sessions.filter(session => {
    if (normalizedFilter === "") return true;
    return (
      getSessionDisplayTitle(session).toLocaleLowerCase().includes(normalizedFilter) ||
      session.agent_name.toLocaleLowerCase().includes(normalizedFilter)
    );
  });
  const byAgent = new Map<string, SessionPayload[]>();
  for (const session of filtered) {
    const current = byAgent.get(session.agent_name) ?? [];
    current.push(session);
    byAgent.set(session.agent_name, current);
  }
  const groups: SessionGroup[] = [...byAgent.entries()].map(([agentName, groupedSessions]) => ({
    agentName,
    sessions: groupedSessions,
  }));
  const collapsedAgents = new Set(collapsedAgentIds);

  return (
    <div className="flex min-h-0 flex-1 flex-col" data-testid="os-sessions-rail-content">
      <div className="flex items-center gap-2 px-3 pt-3 pb-1.5">
        {view === "all" ? (
          <Button
            type="button"
            variant="ghost"
            size="icon-sm"
            aria-label="Back to recent sessions"
            onClick={() => setView("recent")}
          >
            <Icon as={ChevronLeft} size="sm" />
          </Button>
        ) : null}
        <Eyebrow className="min-w-0 flex-1 text-subtle">
          Sessions <span className="ml-1 text-faint">{filtered.length}</span>
        </Eyebrow>
        <Button
          type="button"
          variant="ghost"
          size="icon-sm"
          aria-label="Close sessions"
          onClick={onClose}
        >
          <Icon as={X} size="sm" />
        </Button>
      </div>
      <div className="px-3 py-1.5">
        <SearchInput
          value={filter}
          onChange={setFilter}
          placeholder="Filter sessions…"
          aria-label="Filter sessions"
          containerClassName="min-w-0"
        />
      </div>
      {disconnected ? (
        <p
          className="mx-3 my-1 rounded-md border border-warning/30 bg-warning-tint px-2.5 py-2 text-small-body text-warning"
          role="status"
        >
          Session updates are unavailable. Cached sessions remain visible.
        </p>
      ) : null}
      <div className="min-h-0 flex-1 overflow-hidden">
        <div
          className={cn(
            "flex h-full w-[200%] transition-transform duration-shell-slow ease-spring",
            view === "all" && "-translate-x-1/2"
          )}
          data-view={view}
        >
          <div className="flex h-full w-1/2 shrink-0 flex-col overflow-y-auto px-2.5 pb-3">
            <div className="flex flex-col gap-0.5">
              {filtered.slice(0, 6).map(session => (
                <SessionRow
                  key={session.id}
                  session={session}
                  onSelect={() => onSelectSession(session)}
                />
              ))}
            </div>
            {filtered.length === 0 ? (
              <p className="px-3 py-8 text-center text-small-body text-muted">No sessions match.</p>
            ) : null}
            <button
              type="button"
              className="mt-1 flex w-full items-center justify-center gap-1.5 rounded-md px-2 py-2 text-small-body font-medium text-subtle transition-colors hover:bg-row-hover hover:text-fg focus-visible:shadow-focus-ring focus-visible:outline-none"
              onClick={() => setView("all")}
            >
              Show all sessions
              <Icon as={ChevronRight} size="sm" />
            </button>
          </div>
          <div className="h-full w-1/2 shrink-0 overflow-y-auto px-2.5 pb-3">
            {groups.map(group => {
              const collapsed = collapsedAgents.has(group.agentName);
              return (
                <section
                  key={group.agentName}
                  className="border-b border-line-soft py-1 last:border-b-0"
                >
                  <button
                    type="button"
                    className="flex min-h-7 w-full items-center gap-2 rounded-sm px-2 py-1 text-left hover:bg-row-hover focus-visible:shadow-focus-ring focus-visible:outline-none"
                    aria-expanded={!collapsed}
                    onClick={() => onToggleGroup(group.agentName)}
                  >
                    <span className="grid size-5 place-items-center rounded-sm bg-elevated font-mono text-micro font-semibold text-muted">
                      {group.agentName.slice(0, 2).toUpperCase()}
                    </span>
                    <span className="min-w-0 flex-1 truncate text-small-body font-medium text-fg-strong">
                      {group.agentName}
                    </span>
                    <span className="font-mono text-micro text-faint">{group.sessions.length}</span>
                    <Icon
                      as={ChevronRight}
                      size="sm"
                      className={cn("text-faint transition-transform", !collapsed && "rotate-90")}
                    />
                  </button>
                  <div
                    className={cn(
                      "grid transition-[grid-template-rows] duration-base",
                      collapsed ? "grid-rows-[0fr]" : "grid-rows-[1fr]"
                    )}
                  >
                    <div className="min-h-0 overflow-hidden pl-2">
                      {group.sessions.map(session => (
                        <SessionRow
                          key={session.id}
                          session={session}
                          onSelect={() => onSelectSession(session)}
                        />
                      ))}
                    </div>
                  </div>
                </section>
              );
            })}
            {groups.length === 0 ? (
              <p className="px-3 py-8 text-center text-small-body text-muted">No sessions match.</p>
            ) : null}
          </div>
        </div>
      </div>
    </div>
  );
}

export function DesktopSessionsRail({
  sessions,
  disconnected,
}: {
  sessions: readonly SessionPayload[];
  disconnected: boolean;
}) {
  const model = useDesktopSessionsRail();
  const { coordinator, store, open, presentation, collapsedAgentIds } = model;

  const selectSession = (session: SessionPayload) => {
    coordinator.userOpen({
      app: "session",
      instanceKey: session.id,
      location: {
        pathname: `/agents/${encodeURIComponent(session.agent_name)}/sessions/${encodeURIComponent(session.id)}`,
        search: {},
      },
    });
    if (presentation === "compact") store.getState().closeRail();
  };
  const content = (
    <SessionsRailBody
      sessions={sessions}
      disconnected={disconnected}
      collapsedAgentIds={collapsedAgentIds}
      onToggleGroup={agentName => store.getState().toggleRailGroup(agentName)}
      onSelectSession={selectSession}
      onClose={() => store.getState().closeRail()}
    />
  );

  if (presentation === "compact") {
    return (
      <Sheet open={open} onOpenChange={next => !next && store.getState().closeRail()}>
        <SheetContent
          side="left"
          showCloseButton={false}
          className="w-[min(88vw,22rem)] gap-0 border-r border-line bg-shell-glass p-0 backdrop-blur-shell"
          data-testid="os-sessions-rail-sheet"
        >
          <SheetTitle className="sr-only">Sessions</SheetTitle>
          <SheetDescription className="sr-only">
            Filter sessions and open one in the current workspace.
          </SheetDescription>
          {content}
        </SheetContent>
      </Sheet>
    );
  }

  if (!open) return null;
  return (
    <aside
      aria-label="Sessions"
      className="absolute top-3 bottom-24 left-3 z-20 flex w-80 min-h-0 flex-col overflow-hidden rounded-window border border-line bg-shell-glass shadow-window backdrop-blur-shell"
      data-testid="os-sessions-rail"
    >
      {content}
    </aside>
  );
}
