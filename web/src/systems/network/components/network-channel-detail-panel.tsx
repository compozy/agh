import { AlertCircle, Hash, Info, Loader2 } from "lucide-react";
import { Link } from "@tanstack/react-router";

import { Pill } from "@/components/design-system";
import { cn } from "@/lib/utils";

import {
  formatChannelMemberCount,
  formatNetworkClockTime,
  getChannelDetailDescription,
  getMessageAuthorInitial,
} from "../lib/network-formatters";
import type { NetworkChannel, NetworkChannelMessage } from "../types";

interface NetworkChannelDetailPanelProps {
  channel: NetworkChannel | undefined;
  error: Error | null;
  isLoading: boolean;
  isMessagesLoading: boolean;
  messages: NetworkChannelMessage[];
}

function MessageItem({ message }: { message: NetworkChannelMessage }) {
  const author = message.display_name ?? message.peer_id;

  return (
    <article
      className="flex items-start gap-3 rounded-xl border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] px-4 py-3"
      data-testid={`network-channel-message-${message.message_id}`}
    >
      <div
        className={cn(
          "flex size-8 shrink-0 items-center justify-center rounded-lg border text-sm font-semibold",
          message.local
            ? "border-[color:var(--color-accent)] bg-[color:var(--color-accent-tint)] text-[color:var(--color-accent)]"
            : "border-[color:var(--color-divider)] bg-[color:var(--color-surface-panel)] text-[color:var(--color-text-primary)]"
        )}
      >
        {getMessageAuthorInitial(message)}
      </div>

      <div className="min-w-0 flex-1">
        <div className="flex flex-wrap items-center gap-2">
          <p className="text-sm font-medium text-[color:var(--color-text-primary)]">{author}</p>
          {message.local ? (
            <Pill kind="state" tone="amber">
              local
            </Pill>
          ) : null}
          <span className="text-xs text-[color:var(--color-text-tertiary)]">
            {formatNetworkClockTime(message.timestamp)}
          </span>
          {message.session_id ? (
            <Link
              className="ml-auto text-xs font-medium text-[color:var(--color-accent)] transition-colors hover:text-[color:var(--color-accent)]/80"
              params={{ id: message.session_id }}
              to="/session/$id"
            >
              View Session
            </Link>
          ) : null}
        </div>
        <p className="mt-1 whitespace-pre-wrap text-sm leading-relaxed text-[color:var(--color-text-secondary)]">
          {message.text}
        </p>
      </div>
    </article>
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
      <div
        className="flex flex-1 items-center justify-center"
        data-testid="network-channel-loading"
      >
        <Loader2 className="size-5 animate-spin text-[color:var(--color-text-tertiary)]" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex flex-1 items-center justify-center" data-testid="network-channel-error">
        <div className="flex flex-col items-center gap-2 text-center">
          <AlertCircle className="size-6 text-[color:var(--color-danger)]" />
          <p className="text-sm text-[color:var(--color-text-tertiary)]">
            {error.message ?? "Failed to load network channel"}
          </p>
        </div>
      </div>
    );
  }

  if (!channel) {
    return (
      <div className="flex flex-1 items-center justify-center" data-testid="network-channel-empty">
        <p className="max-w-md text-center text-sm leading-relaxed text-[color:var(--color-text-tertiary)]">
          Select a channel to inspect the read-only message timeline.
        </p>
      </div>
    );
  }

  return (
    <section
      className="flex min-h-0 flex-1 flex-col overflow-hidden"
      data-testid="network-channel-detail-panel"
    >
      <div className="border-b border-[color:var(--color-divider)] px-6 py-4">
        <div className="flex flex-wrap items-center gap-3">
          <div className="flex size-8 items-center justify-center rounded-lg border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] text-[color:var(--color-text-secondary)]">
            <Hash className="size-4" />
          </div>
          <h2 className="text-xl font-semibold text-[color:var(--color-text-primary)]">
            {channel.channel}
          </h2>
          <Pill emphasis="strong" kind="state" tone="green">
            active
          </Pill>
          <Pill kind="tag" tone="neutral">
            {formatChannelMemberCount(channel)}
          </Pill>
        </div>
        <p className="mt-2 text-sm text-[color:var(--color-text-secondary)]">
          {getChannelDetailDescription(channel)}
        </p>
      </div>

      <div className="flex items-center gap-2 border-b border-[color:var(--color-divider)] bg-[color:var(--color-surface-panel)] px-6 py-3 text-sm text-[color:var(--color-text-secondary)]">
        <Info className="size-4 text-[color:var(--color-text-tertiary)]" />
        <span>This channel is read-only. Use the CLI to send messages.</span>
      </div>

      <div className="min-h-0 flex-1 overflow-y-auto px-6 py-4">
        {isMessagesLoading && messages.length === 0 ? (
          <div
            className="flex h-full items-center justify-center"
            data-testid="network-channel-messages-loading"
          >
            <Loader2 className="size-5 animate-spin text-[color:var(--color-text-tertiary)]" />
          </div>
        ) : messages.length === 0 ? (
          <div
            className="flex h-full items-center justify-center"
            data-testid="network-channel-no-messages"
          >
            <div className="max-w-md space-y-3 text-center">
              <div className="mx-auto flex size-12 items-center justify-center rounded-2xl border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] text-[color:var(--color-text-tertiary)]">
                <Hash className="size-5" />
              </div>
              <p className="text-base font-medium text-[color:var(--color-text-primary)]">
                No messages yet
              </p>
              <p className="text-sm leading-relaxed text-[color:var(--color-text-secondary)]">
                This channel exists and can accept members, but no read-only timeline messages have
                been recorded yet.
              </p>
            </div>
          </div>
        ) : (
          <div className="space-y-3">
            {messages.map(message => (
              <MessageItem key={message.message_id} message={message} />
            ))}
          </div>
        )}
      </div>
    </section>
  );
}
