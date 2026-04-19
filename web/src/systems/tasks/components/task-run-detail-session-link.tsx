import { Link } from "@tanstack/react-router";
import { ArrowUpRight, MessageSquare } from "lucide-react";

import { MonoBadge, Section } from "@agh/ui";

import type { TaskRunDetailView } from "../types";

export interface TaskRunDetailSessionLinkProps {
  run: TaskRunDetailView;
}

export function TaskRunDetailSessionLink({ run }: TaskRunDetailSessionLinkProps) {
  const session = run.session;

  if (!session?.session_id) {
    return (
      <Section aria-label="Linked session" label="Linked session">
        <div
          className="flex items-center gap-3 rounded-[var(--radius-diagram)] border border-dashed border-[color:var(--color-divider)] bg-[color:var(--color-surface)] px-4 py-4"
          data-testid="task-run-detail-session-none"
        >
          <MessageSquare className="size-5 text-[color:var(--color-text-tertiary)]" />
          <p className="text-[13px] text-[color:var(--color-text-secondary)]">
            This run is not attached to a session yet. Session drill-down will appear once a session
            is claimed.
          </p>
        </div>
      </Section>
    );
  }

  return (
    <Section aria-label="Linked session" label="Linked session">
      <div
        className="flex items-start justify-between gap-3 rounded-[var(--radius-diagram)] border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] px-4 py-4"
        data-testid="task-run-detail-session-link-panel"
      >
        <div className="min-w-0 flex-1">
          <MonoBadge data-testid="task-run-detail-session-id">{session.session_id}</MonoBadge>
          <div className="mt-2 flex flex-wrap items-center gap-2 text-[13px] text-[color:var(--color-text-secondary)]">
            {session.agent_name ? (
              <span data-testid="task-run-detail-session-agent">Agent {session.agent_name}</span>
            ) : null}
            {session.workspace_id ? (
              <span className="font-mono">· workspace {session.workspace_id}</span>
            ) : null}
            {session.state ? <span>· state {session.state}</span> : null}
          </div>
        </div>
        <Link
          className="inline-flex shrink-0 items-center gap-1 rounded-md border border-[color:var(--color-divider)] px-2.5 py-1 font-mono text-[11px] uppercase tracking-[0.14em] text-[color:var(--color-accent)] hover:border-[color:var(--color-accent)]"
          data-testid="task-run-detail-session-drilldown"
          params={{ id: session.session_id }}
          to="/session/$id"
        >
          Open session
          <ArrowUpRight className="size-3" />
        </Link>
      </div>
    </Section>
  );
}
