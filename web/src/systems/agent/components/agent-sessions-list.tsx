import { Link } from "@tanstack/react-router";
import { MessageSquare } from "lucide-react";

import {
  Empty,
  MonoBadge,
  Skeleton,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
  cn,
} from "@agh/ui";

import { getAgentSessionStatus } from "../lib/session-status";
import type { SessionPayload } from "@/systems/session";

export interface AgentSessionsListProps {
  agentName: string;
  sessions: SessionPayload[];
  isLoading: boolean;
  isError: boolean;
}

export function AgentSessionsList({
  agentName,
  sessions,
  isLoading,
  isError,
}: AgentSessionsListProps) {
  if (isLoading) {
    return <AgentSessionsSkeleton />;
  }

  if (isError) {
    return (
      <Empty
        icon={MessageSquare}
        title="Couldn't load sessions"
        description="The session list failed to load. Try refreshing the page."
        data-testid="agent-sessions-error"
        fill={false}
      />
    );
  }

  if (sessions.length === 0) {
    return (
      <Empty
        icon={MessageSquare}
        title="No sessions yet"
        description={`Start a new session for ${agentName} from the toolbar above.`}
        data-testid="agent-sessions-empty"
        fill={false}
      />
    );
  }

  return (
    <div className="overflow-x-auto" data-testid="agent-sessions-table-wrapper">
      <Table data-testid="agent-sessions-table">
        <TableHeader>
          <TableRow>
            <TableHead className="w-[42%]">Session</TableHead>
            <TableHead>Status</TableHead>
            <TableHead className="text-right">Duration</TableHead>
            <TableHead className="text-right">Iterations</TableHead>
            <TableHead className="text-right">Last activity</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {sessions.map(session => (
            <AgentSessionRow key={session.id} agentName={agentName} session={session} />
          ))}
        </TableBody>
      </Table>
    </div>
  );
}

interface AgentSessionRowProps {
  agentName: string;
  session: SessionPayload;
}

function AgentSessionRow({ agentName, session }: AgentSessionRowProps) {
  const status = getAgentSessionStatus(session);
  const title = session.name?.trim() || session.id.slice(0, 12);
  return (
    <TableRow data-testid={`agent-session-row-${session.id}`} data-state={status.kind}>
      <TableCell>
        <Link
          to="/agents/$name/sessions/$id"
          params={{ name: agentName, id: session.id }}
          className={cn(
            "flex flex-col gap-0.5 text-[13px] text-[color:var(--color-text-primary)]",
            "transition-colors hover:text-[color:var(--color-accent)]"
          )}
          data-testid={`agent-session-link-${session.id}`}
        >
          <span className="truncate font-medium">{title}</span>
          <span className="font-mono text-[10px] uppercase tracking-[0.06em] text-[color:var(--color-text-tertiary)]">
            {session.provider}
          </span>
        </Link>
      </TableCell>
      <TableCell>
        <MonoBadge tone={status.tone} data-testid={`agent-session-status-${session.id}`}>
          {status.label}
        </MonoBadge>
      </TableCell>
      <TableCell className="text-right font-mono text-[12px] text-[color:var(--color-text-secondary)]">
        {formatDuration(session.activity?.elapsed_seconds)}
      </TableCell>
      <TableCell className="text-right font-mono text-[12px] text-[color:var(--color-text-secondary)]">
        {formatIterations(session.activity?.iteration_current, session.activity?.iteration_max)}
      </TableCell>
      <TableCell className="text-right font-mono text-[12px] text-[color:var(--color-text-secondary)]">
        {formatRelativeTime(session.activity?.last_activity_at ?? session.updated_at)}
      </TableCell>
    </TableRow>
  );
}

function AgentSessionsSkeleton() {
  return (
    <div
      className="flex flex-col gap-2 px-1 py-2"
      data-testid="agent-sessions-loading"
      role="status"
      aria-live="polite"
    >
      {Array.from({ length: 4 }, (_, index) => (
        <Skeleton key={index} className="h-9 w-full rounded-[var(--radius-md)]" />
      ))}
    </div>
  );
}

function formatDuration(seconds: number | undefined | null): string {
  if (typeof seconds !== "number" || !Number.isFinite(seconds) || seconds <= 0) return "—";
  const total = Math.round(seconds);
  if (total < 60) return `${total}s`;
  const minutes = Math.floor(total / 60);
  const remainder = total % 60;
  if (minutes < 60) return remainder === 0 ? `${minutes}m` : `${minutes}m ${remainder}s`;
  const hours = Math.floor(minutes / 60);
  const remainderMinutes = minutes % 60;
  return remainderMinutes === 0 ? `${hours}h` : `${hours}h ${remainderMinutes}m`;
}

function formatIterations(current: number | undefined, max: number | undefined): string {
  if (typeof current !== "number" || !Number.isFinite(current)) return "—";
  if (typeof max === "number" && Number.isFinite(max) && max > 0) {
    return `${current}/${max}`;
  }
  return `${current}`;
}

function formatRelativeTime(value: string | null | undefined): string {
  if (!value) return "—";
  const ts = new Date(value).getTime();
  if (!Number.isFinite(ts)) return "—";
  const diffMs = Date.now() - ts;
  if (diffMs < 0) return "just now";
  const seconds = Math.floor(diffMs / 1000);
  if (seconds < 45) return "just now";
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.floor(hours / 24);
  if (days < 7) return `${days}d ago`;
  return new Date(ts).toLocaleDateString();
}
