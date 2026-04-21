import { AlertCircle, Hash, Info, Loader2 } from "lucide-react";
import { Link } from "@tanstack/react-router";

import {
  CodeBlock,
  Empty,
  KindChip,
  MonoBadge,
  Pill,
  Section,
  StatusDot,
  type StatusDotTone,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@agh/ui";

import {
  formatChannelMemberCount,
  formatNetworkClockTime,
  formatNetworkNumber,
  getChannelDetailDescription,
  getPeerDisplayName,
} from "../lib/network-formatters";
import type { NetworkChannel, NetworkChannelMessage } from "../types";

interface NetworkChannelDetailPanelProps {
  channel: NetworkChannel | undefined;
  error: Error | null;
  isLoading: boolean;
  isMessagesLoading: boolean;
  messages: NetworkChannelMessage[];
}

type NetworkChannelMember = NonNullable<NetworkChannel["peers"]>[number];

const VALID_KINDS = new Set(["greet", "whois", "say", "direct", "capability", "receipt", "trace"]);

function getMessageKind(message: NetworkChannelMessage): string {
  const raw = message.intent;
  if (typeof raw === "string" && VALID_KINDS.has(raw)) {
    return raw;
  }
  return "say";
}

function memberStatusTone(member: NetworkChannelMember): StatusDotTone {
  if (member.local) {
    return "accent";
  }

  if (!member.last_seen) {
    return "neutral";
  }

  const parsed = new Date(member.last_seen);
  if (Number.isNaN(parsed.getTime())) {
    return "neutral";
  }

  return Date.now() - parsed.getTime() <= 60_000 ? "success" : "neutral";
}

function WireTraceTable({ channel }: { channel: NetworkChannel }) {
  const rows: Array<{ label: string; value: string }> = [
    { label: "channel", value: channel.channel },
    { label: "last message", value: formatNetworkClockTime(channel.last_message_at) },
    { label: "messages", value: formatNetworkNumber(channel.message_count ?? 0) },
    { label: "peers", value: formatNetworkNumber(channel.peer_count ?? 0) },
    { label: "local peers", value: formatNetworkNumber(channel.local_peer_count ?? 0) },
    { label: "remote peers", value: formatNetworkNumber(channel.remote_peer_count ?? 0) },
    { label: "sessions", value: formatNetworkNumber(channel.session_count ?? 0) },
  ];

  return (
    <div
      className="overflow-hidden rounded-[var(--radius-md)] border border-[color:var(--color-divider)] bg-[color:var(--color-surface)]"
      data-testid="network-channel-wire-trace"
    >
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead className="font-mono text-[10px] uppercase tracking-[0.14em] text-[color:var(--color-text-label)]">
              Key
            </TableHead>
            <TableHead className="font-mono text-[10px] uppercase tracking-[0.14em] text-[color:var(--color-text-label)]">
              Value
            </TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {rows.map(row => (
            <TableRow
              data-testid={`network-channel-wire-trace-row-${row.label.replaceAll(" ", "-")}`}
              key={row.label}
            >
              <TableCell className="w-40 font-mono text-[11px] uppercase tracking-[0.12em] text-[color:var(--color-text-label)]">
                {row.label}
              </TableCell>
              <TableCell className="text-[13px] text-[color:var(--color-text-primary)]">
                {row.value}
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}

function MembersList({ members }: { members: NetworkChannelMember[] }) {
  if (members.length === 0) {
    return (
      <Empty
        icon={Hash}
        title="No members yet"
        description="Peers appear here as soon as they join the channel."
      />
    );
  }

  return (
    <ul className="flex flex-col gap-1" data-testid="network-channel-members-list">
      {members.map(member => (
        <li
          className="flex items-center gap-3 rounded-[var(--radius-md)] border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] px-3 py-2"
          data-testid={`network-channel-member-${member.peer_id}`}
          key={member.peer_id}
        >
          <StatusDot size="md" tone={memberStatusTone(member)} />
          <div className="min-w-0 flex-1">
            <p className="truncate text-[13px] font-medium text-[color:var(--color-text-primary)]">
              {getPeerDisplayName(member)}
            </p>
          </div>
          <MonoBadge tone={member.local ? "accent" : "default"}>
            {member.local ? "local" : "remote"}
          </MonoBadge>
          <MonoBadge className="hidden sm:inline-flex" tone="default">
            {member.peer_id}
          </MonoBadge>
        </li>
      ))}
    </ul>
  );
}

function MessageItem({ message }: { message: NetworkChannelMessage }) {
  const author = message.display_name ?? message.peer_id;
  const kind = getMessageKind(message);

  return (
    <article
      className="space-y-2 rounded-[var(--radius-lg)] border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] px-4 py-3"
      data-testid={`network-channel-message-${message.message_id}`}
    >
      <header className="flex flex-wrap items-center gap-2">
        <KindChip data-testid={`network-channel-message-kind-${message.message_id}`} kind={kind} />
        <span className="text-[13px] font-medium text-[color:var(--color-text-primary)]">
          {author}
        </span>
        {message.local ? <Pill variant="accent">local</Pill> : null}
        <span className="font-mono text-[10.5px] text-[color:var(--color-text-tertiary)]">
          {formatNetworkClockTime(message.timestamp)}
        </span>
        {message.session_id ? (
          <Link
            className="ml-auto text-xs font-medium text-[color:var(--color-accent)] transition-colors hover:text-[color:var(--color-accent-hover)]"
            params={{ id: message.session_id }}
            to="/session/$id"
          >
            View Session
          </Link>
        ) : null}
      </header>
      <CodeBlock
        className="border border-[color:var(--color-divider)]"
        code={message.text}
        copyable={false}
        data-testid={`network-channel-message-payload-${message.message_id}`}
        showPrompt={false}
      />
    </article>
  );
}

function DetailStateFallback({ children, testId }: { children: React.ReactNode; testId: string }) {
  return (
    <div className="flex min-h-0 flex-1 items-center justify-center p-6" data-testid={testId}>
      {children}
    </div>
  );
}

export function NetworkChannelDetailPanel({
  channel,
  error,
  isLoading,
  isMessagesLoading,
  messages,
}: NetworkChannelDetailPanelProps) {
  if (isLoading) {
    return (
      <DetailStateFallback testId="network-channel-loading">
        <Loader2
          aria-hidden="true"
          className="size-5 animate-spin text-[color:var(--color-text-tertiary)]"
        />
      </DetailStateFallback>
    );
  }

  if (error) {
    return (
      <DetailStateFallback testId="network-channel-error">
        <Empty
          className="max-w-md"
          icon={AlertCircle}
          title="Unable to load channel"
          description={error.message ?? "Failed to load network channel"}
        />
      </DetailStateFallback>
    );
  }

  if (!channel) {
    return (
      <DetailStateFallback testId="network-channel-empty">
        <Empty
          className="max-w-md"
          icon={Hash}
          title="Select a channel"
          description="Inspect the wire trace, members, and message log of any materialized channel."
        />
      </DetailStateFallback>
    );
  }

  const members = channel.peers ?? [];

  return (
    <section
      className="flex min-h-0 flex-1 flex-col overflow-hidden"
      data-testid="network-channel-detail-panel"
    >
      <header className="border-b border-[color:var(--color-divider)] px-6 py-4">
        <div className="flex flex-wrap items-center gap-3">
          <span
            aria-hidden="true"
            className="inline-flex size-8 items-center justify-center rounded-[var(--radius-md)] border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] text-[color:var(--color-text-secondary)]"
          >
            <Hash className="size-4" />
          </span>
          <h2 className="font-mono text-[15px] font-semibold tracking-[-0.01em] text-[color:var(--color-text-primary)]">
            {channel.channel}
          </h2>
          <Pill variant="success">active</Pill>
          <Pill>{formatChannelMemberCount(channel)}</Pill>
        </div>
        <p className="mt-2 text-[13px] text-[color:var(--color-text-secondary)]">
          {getChannelDetailDescription(channel)}
        </p>
      </header>

      <div className="flex items-center gap-2 border-b border-[color:var(--color-divider)] bg-[color:var(--color-surface-panel)] px-6 py-3 text-[13px] text-[color:var(--color-text-secondary)]">
        <Info aria-hidden="true" className="size-4 text-[color:var(--color-text-tertiary)]" />
        <span>This channel is read-only. Use the CLI to send messages.</span>
      </div>

      <div className="min-h-0 flex-1 space-y-6 overflow-y-auto px-6 py-5">
        <Section label="Wire trace">
          <WireTraceTable channel={channel} />
        </Section>

        <Section label="Members" right={<MonoBadge>{members.length}</MonoBadge>}>
          <MembersList members={members} />
        </Section>

        <Section label="Messages" right={<MonoBadge>{messages.length}</MonoBadge>}>
          {isMessagesLoading && messages.length === 0 ? (
            <div
              className="flex min-h-40 items-center justify-center"
              data-testid="network-channel-messages-loading"
            >
              <Loader2
                aria-hidden="true"
                className="size-5 animate-spin text-[color:var(--color-text-tertiary)]"
              />
            </div>
          ) : messages.length === 0 ? (
            <div data-testid="network-channel-no-messages">
              <Empty
                icon={Hash}
                title="No messages yet"
                description="This channel exists and can accept members, but no read-only timeline messages have been recorded yet."
              />
            </div>
          ) : (
            <div className="flex flex-col gap-3">
              {messages.map(message => (
                <MessageItem key={message.message_id} message={message} />
              ))}
            </div>
          )}
        </Section>
      </div>
    </section>
  );
}
