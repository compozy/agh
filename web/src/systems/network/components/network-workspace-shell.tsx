import type { ReactNode } from "react";
import {
  Bot,
  Hash,
  PanelRightClose,
  PanelRightOpen,
  Plus,
  SendHorizontal,
  Sparkles,
  Users,
  Workflow,
} from "lucide-react";

import { Button, Empty, MonoBadge, SearchInput, StatusDot, Textarea } from "@agh/ui";
import { cn } from "@/lib/utils";

import {
  NETWORK_KIND_FILTERS,
  formatNetworkDateTime,
  formatNetworkKindLabel,
  formatNetworkRelativeTime,
  getMessageAuthorInitial,
  getNetworkKindTone,
  getNetworkMessagePrimaryText,
  getNetworkStatusTone,
} from "../lib/network-formatters";
import type {
  NetworkActiveRoom,
  NetworkDetailsTab,
  NetworkKindFilter,
  NetworkRoomField,
  NetworkRoomListItem,
  NetworkStatus,
  NetworkTimelineMessage,
} from "../types";

interface NetworkWorkspaceShellProps {
  activeKind: NetworkKindFilter;
  activeRoom: NetworkActiveRoom | null;
  channelRooms: NetworkRoomListItem[];
  composeDraft: string;
  directRooms: NetworkRoomListItem[];
  detailsTab: NetworkDetailsTab;
  isComposePending: boolean;
  isDetailsOpen: boolean;
  isRoomLoading: boolean;
  isTimelineLoading: boolean;
  onComposeDraftChange: (value: string) => void;
  onComposeSubmit: () => void;
  onOpenCreateDialog: () => void;
  onSelectDetailsTab: (tab: NetworkDetailsTab) => void;
  onSelectKind: (kind: NetworkKindFilter) => void;
  onSelectRoom: (room: NetworkRoomListItem) => void;
  onToggleDetails: () => void;
  onToggleStarChannel: (channel: string) => void;
  roomError: Error | null;
  selectedRoomKey: string | null;
  sidebarQuery: string;
  status: NetworkStatus;
  onSidebarQueryChange: (value: string) => void;
  starredChannelRooms: NetworkRoomListItem[];
}

const toneClasses = {
  accent: "bg-[color:var(--color-accent-tint)] text-[color:var(--color-accent)]",
  danger: "bg-[color:var(--color-danger-tint)] text-[color:var(--color-danger)]",
  info: "bg-[color:var(--color-info-tint)] text-[color:var(--color-info)]",
  neutral: "bg-[color:var(--color-neutral-tint)] text-[color:var(--color-text-label)]",
  success: "bg-[color:var(--color-success-tint)] text-[color:var(--color-success)]",
  warning: "bg-[color:var(--color-warning-tint)] text-[color:var(--color-warning)]",
} as const;

function fieldToneToBadgeTone(field?: NetworkRoomField["tone"]) {
  switch (field) {
    case "accent":
    case "danger":
    case "info":
    case "success":
    case "warning":
      return field;
    default:
      return "default";
  }
}

function readRecord(value: unknown): Record<string, unknown> | null {
  if (typeof value !== "object" || value === null || Array.isArray(value)) {
    return null;
  }

  return value as Record<string, unknown>;
}

function readString(record: Record<string, unknown> | null, key: string): string | null {
  const value = record?.[key];
  return typeof value === "string" && value.trim() !== "" ? value : null;
}

function readStringList(record: Record<string, unknown> | null, key: string): string[] {
  const value = record?.[key];
  if (!Array.isArray(value)) {
    return [];
  }

  return value.filter(item => typeof item === "string" && item.trim() !== "") as string[];
}

function hasMessages(messages: NetworkTimelineMessage[]) {
  return messages.length > 0;
}

function isGroupedWithPrevious(
  previous: NetworkTimelineMessage | undefined,
  current: NetworkTimelineMessage
) {
  if (!previous) {
    return false;
  }

  if (previous.peer_from !== current.peer_from) {
    return false;
  }

  const previousAt = new Date(previous.timestamp).getTime();
  const currentAt = new Date(current.timestamp).getTime();
  if (Number.isNaN(previousAt) || Number.isNaN(currentAt)) {
    return false;
  }

  return currentAt - previousAt <= 5 * 60_000;
}

function NetworkRoomKindPill({ kind }: { kind: string }) {
  const tone = getNetworkKindTone(kind);

  return (
    <span
      className={cn(
        "inline-flex items-center rounded-[var(--radius-chip)] px-1.5 py-0.5 font-mono text-[10px] font-medium lowercase",
        toneClasses[tone]
      )}
    >
      {formatNetworkKindLabel(kind)}
    </span>
  );
}

function NetworkShellSection({ children, title }: { children: ReactNode; title: string }) {
  return (
    <section className="space-y-2">
      <div className="px-3">
        <span className="font-mono text-[10px] uppercase tracking-[0.08em] text-[color:var(--color-text-tertiary)]">
          {title}
        </span>
      </div>
      <div className="space-y-1">{children}</div>
    </section>
  );
}

function NetworkSidebarRow({
  active,
  item,
  onSelect,
  onToggleStar,
}: {
  active: boolean;
  item: NetworkRoomListItem;
  onSelect: (item: NetworkRoomListItem) => void;
  onToggleStar?: (channel: string) => void;
}) {
  const isChannel = item.roomType === "channel";

  return (
    <div
      className={cn(
        "flex w-full items-start gap-1 rounded-[var(--radius-md)] border text-left transition-colors focus-within:border-[color:var(--color-accent-dim)]",
        active
          ? "border-[color:var(--color-accent-dim)] bg-[color:var(--color-accent-tint)]"
          : "border-transparent hover:border-[color:var(--color-divider)] hover:bg-[color:var(--color-surface)]"
      )}
      data-testid={`network-room-${item.roomType}-${item.id}`}
    >
      <button
        aria-current={active ? "page" : undefined}
        className="flex min-w-0 flex-1 items-start gap-3 px-3 py-2.5 text-left focus-visible:outline-none"
        onClick={() => onSelect(item)}
        type="button"
      >
        <div className="mt-0.5 flex items-center gap-2">
          {isChannel ? (
            <Hash className="size-3.5 text-[color:var(--color-text-secondary)]" />
          ) : (
            <StatusDot tone={item.tone} />
          )}
        </div>
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2">
            <span
              className={cn(
                "truncate text-[13px] font-medium",
                item.unreadCount > 0
                  ? "text-[color:var(--color-text-primary)]"
                  : "text-[color:var(--color-text-secondary)]"
              )}
            >
              {item.title}
            </span>
            {item.unreadCount > 0 ? (
              <MonoBadge className="ml-auto" tone="accent">
                {item.unreadCount}
              </MonoBadge>
            ) : null}
          </div>
          <div className="mt-1 flex items-center gap-2">
            <span className="truncate text-[12px] text-[color:var(--color-text-tertiary)]">
              {item.preview}
            </span>
          </div>
          <div className="mt-1 flex items-center gap-2">
            <span className="font-mono text-[10px] uppercase tracking-[0.08em] text-[color:var(--color-text-tertiary)]">
              {item.meta}
            </span>
            <span className="text-[color:var(--color-text-tertiary)]">·</span>
            <span className="truncate font-mono text-[10px] uppercase tracking-[0.08em] text-[color:var(--color-text-tertiary)]">
              {item.subtitle}
            </span>
          </div>
        </div>
      </button>
      {isChannel ? (
        <button
          aria-label={item.isStarred ? "Unstar channel" : "Star channel"}
          className="mt-2 mr-2 rounded-[var(--radius-md)] p-1 text-[color:var(--color-text-tertiary)] transition-colors hover:bg-[color:var(--color-surface-elevated)] hover:text-[color:var(--color-accent)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[color:var(--color-accent)]"
          onClick={event => {
            event.stopPropagation();
            onToggleStar?.(item.id);
          }}
          type="button"
        >
          <Sparkles
            className={cn("size-3.5", item.isStarred && "text-[color:var(--color-accent)]")}
          />
        </button>
      ) : null}
    </div>
  );
}

function NetworkMessageBody({ message }: { message: NetworkTimelineMessage }) {
  const body = readRecord(message.body);
  const intent = message.intent?.trim() || readString(body, "intent");
  const capability = readRecord(body?.capability);
  const peerCard = readRecord(body?.peer_card);

  switch (message.kind) {
    case "capability":
      return (
        <div className="space-y-3 rounded-[var(--radius-md)] border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] p-4">
          <div className="flex flex-wrap items-center gap-2">
            <MonoBadge tone="info">capability</MonoBadge>
            {readString(capability, "id") ? (
              <MonoBadge uppercase={false}>{readString(capability, "id")}</MonoBadge>
            ) : null}
            {readString(capability, "version") ? (
              <MonoBadge uppercase={false}>{readString(capability, "version")}</MonoBadge>
            ) : null}
          </div>
          <div className="space-y-1">
            <p className="text-[14px] font-medium text-[color:var(--color-text-primary)]">
              {readString(capability, "summary") ?? getNetworkMessagePrimaryText(message)}
            </p>
            {readString(capability, "outcome") ? (
              <p className="text-[13px] leading-6 text-[color:var(--color-text-secondary)]">
                {readString(capability, "outcome")}
              </p>
            ) : null}
          </div>
          {readStringList(capability, "execution_outline").length > 0 ? (
            <div className="space-y-2">
              <span className="font-mono text-[10px] uppercase tracking-[0.08em] text-[color:var(--color-text-tertiary)]">
                Execution Outline
              </span>
              <div className="space-y-1">
                {readStringList(capability, "execution_outline").map((step, stepIndex) => (
                  <p
                    className="text-[12px] leading-5 text-[color:var(--color-text-secondary)]"
                    key={`${stepIndex}-${step}`}
                  >
                    {step}
                  </p>
                ))}
              </div>
            </div>
          ) : null}
        </div>
      );
    case "receipt":
      return (
        <div className="space-y-2 rounded-[var(--radius-md)] border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] p-4">
          <div className="flex flex-wrap items-center gap-2">
            <MonoBadge tone="warning">receipt</MonoBadge>
            {readString(body, "status") ? (
              <MonoBadge uppercase={false} tone="warning">
                {readString(body, "status")}
              </MonoBadge>
            ) : null}
            {readString(body, "for_id") ? (
              <MonoBadge uppercase={false}>{readString(body, "for_id")}</MonoBadge>
            ) : null}
          </div>
          <p className="text-[13px] leading-6 text-[color:var(--color-text-secondary)]">
            {readString(body, "detail") ?? getNetworkMessagePrimaryText(message)}
          </p>
        </div>
      );
    case "greet":
      return (
        <div className="space-y-3 rounded-[var(--radius-md)] border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] p-4">
          <div className="flex flex-wrap items-center gap-2">
            <MonoBadge tone="success">greet</MonoBadge>
            {readString(peerCard, "display_name") ? (
              <span className="text-[13px] font-medium text-[color:var(--color-text-primary)]">
                {readString(peerCard, "display_name")}
              </span>
            ) : null}
          </div>
          <p className="text-[13px] leading-6 text-[color:var(--color-text-secondary)]">
            {readString(body, "summary") ?? getNetworkMessagePrimaryText(message)}
          </p>
          {readStringList(peerCard, "capabilities").length > 0 ? (
            <div className="flex flex-wrap gap-2">
              {readStringList(peerCard, "capabilities").map(capabilityName => (
                <MonoBadge key={capabilityName} uppercase={false}>
                  {capabilityName}
                </MonoBadge>
              ))}
            </div>
          ) : null}
        </div>
      );
    case "whois":
      return (
        <div className="space-y-2 rounded-[var(--radius-md)] border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] p-4">
          <div className="flex flex-wrap items-center gap-2">
            <MonoBadge tone="warning">whois</MonoBadge>
            {readString(body, "type") ? (
              <MonoBadge uppercase={false}>{readString(body, "type")}</MonoBadge>
            ) : null}
          </div>
          <p className="text-[13px] leading-6 text-[color:var(--color-text-secondary)]">
            {readString(body, "query") ?? getNetworkMessagePrimaryText(message)}
          </p>
          {readString(peerCard, "peer_id") ? (
            <p className="font-mono text-[11px] text-[color:var(--color-text-tertiary)]">
              {readString(peerCard, "peer_id")}
            </p>
          ) : null}
        </div>
      );
    case "trace":
      return (
        <div className="space-y-2 rounded-[var(--radius-md)] border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] p-4">
          <div className="flex flex-wrap items-center gap-2">
            <MonoBadge tone="info">trace</MonoBadge>
            {readString(body, "state") ? (
              <MonoBadge uppercase={false} tone="info">
                {readString(body, "state")}
              </MonoBadge>
            ) : null}
          </div>
          <p className="text-[13px] leading-6 text-[color:var(--color-text-secondary)]">
            {readString(body, "message") ?? getNetworkMessagePrimaryText(message)}
          </p>
        </div>
      );
    default:
      return (
        <div className="space-y-2">
          {intent ? (
            <MonoBadge uppercase={false} tone="neutral">
              {intent}
            </MonoBadge>
          ) : null}
          <p className="whitespace-pre-wrap text-[14px] leading-6 text-[color:var(--color-text-primary)]">
            {getNetworkMessagePrimaryText(message)}
          </p>
        </div>
      );
  }
}

function NetworkMessageList({
  isTimelineLoading,
  messages,
}: {
  isTimelineLoading: boolean;
  messages: NetworkTimelineMessage[];
}) {
  if (isTimelineLoading && !hasMessages(messages)) {
    return (
      <div className="flex min-h-40 items-center justify-center" data-testid="network-room-loading">
        <p className="font-mono text-[11px] uppercase tracking-[0.08em] text-[color:var(--color-text-tertiary)]">
          Loading timeline
        </p>
      </div>
    );
  }

  if (!hasMessages(messages)) {
    return (
      <div className="px-5 pb-8" data-testid="network-room-empty">
        <Empty
          className="max-w-md"
          description="This room has no persisted traffic yet. Send the first message to materialize the next step."
          icon={Workflow}
          title="No timeline events yet"
        />
      </div>
    );
  }

  return (
    <div className="space-y-1 px-5 pb-8" data-testid="network-message-list">
      {messages.map((message, index) => {
        const grouped = isGroupedWithPrevious(messages[index - 1], message);

        return (
          <article
            className={cn("flex gap-3 rounded-[var(--radius-md)] px-3 py-3", grouped && "pt-1")}
            data-testid={`network-message-${message.message_id}`}
            key={message.message_id}
          >
            <div className="w-10 shrink-0">
              {grouped ? null : (
                <div className="flex size-10 items-center justify-center rounded-[var(--radius-md)] border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] font-mono text-[12px] font-medium text-[color:var(--color-text-primary)]">
                  {getMessageAuthorInitial(message)}
                </div>
              )}
            </div>
            <div className="min-w-0 flex-1 space-y-2">
              {grouped ? null : (
                <div className="flex flex-wrap items-center gap-2">
                  <span className="text-[14px] font-medium text-[color:var(--color-text-primary)]">
                    {message.display_name ?? message.peer_from}
                  </span>
                  <span className="font-mono text-[10px] uppercase tracking-[0.08em] text-[color:var(--color-text-tertiary)]">
                    {formatNetworkRelativeTime(message.timestamp)}
                  </span>
                  <NetworkRoomKindPill kind={message.kind} />
                  <MonoBadge tone={message.direction === "sent" ? "accent" : "default"}>
                    {message.direction}
                  </MonoBadge>
                </div>
              )}
              <NetworkMessageBody message={message} />
              {(message.peer_to || message.trace_id || message.reply_to) && !grouped ? (
                <div className="flex flex-wrap gap-2">
                  {message.peer_to ? (
                    <MonoBadge uppercase={false} tone="default">
                      to {message.peer_to}
                    </MonoBadge>
                  ) : null}
                  {message.trace_id ? (
                    <MonoBadge uppercase={false} tone="default">
                      trace {message.trace_id}
                    </MonoBadge>
                  ) : null}
                  {message.reply_to ? (
                    <MonoBadge uppercase={false} tone="default">
                      reply {message.reply_to}
                    </MonoBadge>
                  ) : null}
                </div>
              ) : null}
            </div>
          </article>
        );
      })}
    </div>
  );
}

function NetworkDetailFieldList({ fields }: { fields: NetworkRoomField[] }) {
  if (fields.length === 0) {
    return (
      <p className="text-[13px] leading-6 text-[color:var(--color-text-tertiary)]">
        No additional detail is available for this room yet.
      </p>
    );
  }

  return (
    <div className="space-y-3">
      {fields.map(field => (
        <div className="space-y-1" key={`${field.label}-${field.value}`}>
          <div className="font-mono text-[10px] uppercase tracking-[0.08em] text-[color:var(--color-text-tertiary)]">
            {field.label}
          </div>
          {field.mono ? (
            <MonoBadge tone={fieldToneToBadgeTone(field.tone)} uppercase={false}>
              {field.value}
            </MonoBadge>
          ) : (
            <p className="text-[13px] leading-6 text-[color:var(--color-text-primary)]">
              {field.value}
            </p>
          )}
        </div>
      ))}
    </div>
  );
}

export function NetworkWorkspaceShell({
  activeKind,
  activeRoom,
  channelRooms,
  composeDraft,
  detailsTab,
  directRooms,
  isComposePending,
  isDetailsOpen,
  isRoomLoading,
  isTimelineLoading,
  onComposeDraftChange,
  onComposeSubmit,
  onOpenCreateDialog,
  onSelectDetailsTab,
  onSelectKind,
  onSelectRoom,
  onSidebarQueryChange,
  onToggleDetails,
  onToggleStarChannel,
  roomError,
  selectedRoomKey,
  sidebarQuery,
  starredChannelRooms,
  status,
}: NetworkWorkspaceShellProps) {
  const selectedTitle =
    activeRoom?.roomType === "channel" ? `#${activeRoom.title}` : activeRoom?.title;

  return (
    <div
      className={cn(
        "grid min-h-0 flex-1 bg-[color:var(--color-canvas)]",
        isDetailsOpen
          ? "grid-cols-1 lg:grid-cols-[18rem_minmax(0,1fr)_20rem]"
          : "grid-cols-1 lg:grid-cols-[18rem_minmax(0,1fr)]"
      )}
      data-testid="network-workspace"
    >
      <aside className="flex min-h-0 flex-col border-b border-[color:var(--color-divider)] bg-[color:var(--color-canvas-deep)] lg:border-r lg:border-b-0">
        <div className="border-b border-[color:var(--color-divider)] px-4 py-4">
          <div className="flex items-start justify-between gap-3">
            <div className="space-y-2">
              <div className="flex items-center gap-2">
                <Workflow className="size-4 text-[color:var(--color-text-secondary)]" />
                <span className="font-mono text-[13px] font-semibold text-[color:var(--color-text-primary)]">
                  agh-network
                </span>
              </div>
              <div className="flex items-center gap-2">
                <StatusDot
                  pulse={status.status === "running" || status.status === "online"}
                  tone={getNetworkStatusTone(status.status)}
                />
                <span className="font-mono text-[10px] uppercase tracking-[0.08em] text-[color:var(--color-text-tertiary)]">
                  {(status.local_peers ?? 0) + (status.remote_peers ?? 0)} peers
                </span>
                <span className="text-[color:var(--color-text-tertiary)]">·</span>
                <span className="font-mono text-[10px] uppercase tracking-[0.08em] text-[color:var(--color-text-tertiary)]">
                  {status.channels ?? channelRooms.length} channels
                </span>
              </div>
            </div>
            <Button
              data-testid="network-open-create-dialog"
              onClick={onOpenCreateDialog}
              size="sm"
              type="button"
              variant="outline"
            >
              <Plus className="size-3.5" />
              Channel
            </Button>
          </div>
          <div className="mt-4">
            <SearchInput
              data-testid="network-sidebar-search"
              kbd="jump"
              onChange={onSidebarQueryChange}
              placeholder="Jump to room"
              value={sidebarQuery}
            />
          </div>
        </div>

        <div className="flex-1 space-y-6 overflow-y-auto px-3 py-4">
          {starredChannelRooms.length > 0 ? (
            <NetworkShellSection title="Starred">
              {starredChannelRooms.map(room => (
                <NetworkSidebarRow
                  active={selectedRoomKey === room.key}
                  item={room}
                  key={room.key}
                  onSelect={onSelectRoom}
                  onToggleStar={onToggleStarChannel}
                />
              ))}
            </NetworkShellSection>
          ) : null}

          <NetworkShellSection title="Channels">
            {channelRooms.length === 0 ? (
              <p className="px-3 text-[12px] leading-5 text-[color:var(--color-text-tertiary)]">
                No channels matched the current search.
              </p>
            ) : (
              channelRooms.map(room => (
                <NetworkSidebarRow
                  active={selectedRoomKey === room.key}
                  item={room}
                  key={room.key}
                  onSelect={onSelectRoom}
                  onToggleStar={onToggleStarChannel}
                />
              ))
            )}
          </NetworkShellSection>

          <NetworkShellSection title="Direct Messages">
            {directRooms.length === 0 ? (
              <p className="px-3 text-[12px] leading-5 text-[color:var(--color-text-tertiary)]">
                No peers matched the current search.
              </p>
            ) : (
              directRooms.map(room => (
                <NetworkSidebarRow
                  active={selectedRoomKey === room.key}
                  item={room}
                  key={room.key}
                  onSelect={onSelectRoom}
                />
              ))
            )}
          </NetworkShellSection>
        </div>
      </aside>

      <section className="flex min-h-0 flex-col border-b border-[color:var(--color-divider)] lg:border-b-0">
        {roomError ? (
          <div className="flex min-h-0 flex-1 items-center justify-center px-6 py-10">
            <Empty
              className="max-w-lg"
              description={roomError.message}
              icon={Workflow}
              title="Unable to load this room"
            />
          </div>
        ) : activeRoom == null && !isRoomLoading ? (
          <div className="flex min-h-0 flex-1 items-center justify-center px-6 py-10">
            <Empty
              className="max-w-lg"
              description="Choose a channel or direct-message room to inspect the live network timeline."
              icon={Workflow}
              title="Select a room"
            />
          </div>
        ) : (
          <>
            <header
              className="flex flex-wrap items-center gap-3 border-b border-[color:var(--color-divider)] px-5 py-4"
              data-testid="network-room-header"
            >
              <div className="flex min-w-0 flex-1 items-center gap-3">
                {activeRoom?.roomType === "channel" ? (
                  <Hash className="size-4 text-[color:var(--color-text-secondary)]" />
                ) : (
                  <Bot className="size-4 text-[color:var(--color-text-secondary)]" />
                )}
                <div className="min-w-0">
                  <div className="flex flex-wrap items-center gap-2">
                    <h1 className="truncate text-[18px] font-semibold text-[color:var(--color-text-primary)]">
                      {selectedTitle ?? "Loading room"}
                    </h1>
                    {activeRoom?.lastActivityAt ? (
                      <span className="font-mono text-[10px] uppercase tracking-[0.08em] text-[color:var(--color-text-tertiary)]">
                        {formatNetworkRelativeTime(activeRoom.lastActivityAt)}
                      </span>
                    ) : null}
                  </div>
                  <p className="truncate text-[13px] text-[color:var(--color-text-secondary)]">
                    {activeRoom?.subtitle ?? "Resolving room details"}
                  </p>
                </div>
              </div>

              <div className="flex items-center gap-2">
                {activeRoom?.canStar ? (
                  <Button
                    onClick={() => onToggleStarChannel(activeRoom.id)}
                    size="sm"
                    type="button"
                    variant="outline"
                  >
                    <Sparkles className="size-3.5" />
                    {activeRoom.isStarred ? "Starred" : "Star"}
                  </Button>
                ) : null}
                <Button
                  aria-label={isDetailsOpen ? "Close room details" : "Open room details"}
                  data-testid="network-toggle-details"
                  onClick={onToggleDetails}
                  size="icon-sm"
                  type="button"
                  variant="outline"
                >
                  {isDetailsOpen ? (
                    <PanelRightClose className="size-4" />
                  ) : (
                    <PanelRightOpen className="size-4" />
                  )}
                </Button>
              </div>
            </header>

            <div className="border-b border-[color:var(--color-divider)] px-5 py-3">
              <div
                aria-label="Timeline kind filters"
                className="flex flex-wrap items-center gap-2"
                data-testid="network-kind-filter-bar"
                role="group"
              >
                <button
                  aria-pressed={activeKind === "all"}
                  className={cn(
                    "rounded-[var(--radius-chip)] border px-3 py-1.5 font-mono text-[10px] uppercase tracking-[0.08em] transition-colors",
                    activeKind === "all"
                      ? "border-[color:var(--color-accent)] bg-[color:var(--color-accent)] text-[color:var(--color-accent-ink)]"
                      : "border-[color:var(--color-divider)] text-[color:var(--color-text-secondary)] hover:border-[color:var(--color-accent-dim)] hover:text-[color:var(--color-text-primary)]"
                  )}
                  onClick={() => onSelectKind("all")}
                  type="button"
                >
                  All
                </button>
                {NETWORK_KIND_FILTERS.map(kind => {
                  const count =
                    activeRoom?.kindCounts.find(metric => metric.kind === kind)?.count ?? 0;

                  return (
                    <button
                      aria-pressed={activeKind === kind}
                      className={cn(
                        "rounded-[var(--radius-chip)] border px-3 py-1.5 font-mono text-[10px] uppercase tracking-[0.08em] transition-colors",
                        activeKind === kind
                          ? "border-[color:var(--color-accent)] bg-[color:var(--color-accent)] text-[color:var(--color-accent-ink)]"
                          : "border-[color:var(--color-divider)] text-[color:var(--color-text-secondary)] hover:border-[color:var(--color-accent-dim)] hover:text-[color:var(--color-text-primary)]"
                      )}
                      key={kind}
                      onClick={() => onSelectKind(kind)}
                      type="button"
                    >
                      {formatNetworkKindLabel(kind)}
                      {count > 0 ? ` ${count}` : ""}
                    </button>
                  );
                })}
              </div>
            </div>

            <div className="flex min-h-0 flex-1 flex-col overflow-hidden">
              <div className="flex-1 overflow-y-auto">
                <div className="px-5 py-5" data-testid="network-room-intro">
                  <div className="rounded-[var(--radius-md)] border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] p-5">
                    <div className="flex flex-wrap items-center gap-2">
                      <MonoBadge tone="accent">
                        {activeRoom?.roomType === "channel" ? "channel" : "direct"}
                      </MonoBadge>
                      {activeRoom?.purpose ? (
                        <MonoBadge uppercase={false}>{activeRoom.purpose}</MonoBadge>
                      ) : null}
                    </div>
                    <h2 className="mt-3 text-[16px] font-semibold text-[color:var(--color-text-primary)]">
                      {activeRoom?.introTitle ?? "Loading room"}
                    </h2>
                    <p className="mt-2 max-w-3xl text-[14px] leading-6 text-[color:var(--color-text-secondary)]">
                      {activeRoom?.introBody ?? "Resolving room metadata and timeline context."}
                    </p>
                  </div>
                </div>

                <NetworkMessageList
                  isTimelineLoading={isTimelineLoading || isRoomLoading}
                  messages={activeRoom?.messages ?? []}
                />
              </div>

              <div className="border-t border-[color:var(--color-divider)] bg-[color:var(--color-canvas)] px-5 py-4">
                <div className="space-y-3" data-testid="network-composer">
                  <Textarea
                    aria-label="Network message composer"
                    className="min-h-24 border-[color:var(--color-divider)] bg-[color:var(--color-surface)] px-3 py-3 text-[14px] leading-6 text-[color:var(--color-text-primary)]"
                    data-testid="network-composer-input"
                    disabled={!activeRoom?.canCompose}
                    onChange={event => onComposeDraftChange(event.target.value)}
                    placeholder={
                      activeRoom?.composePlaceholder ?? "Select a room to start composing"
                    }
                    value={composeDraft}
                  />
                  <div className="flex flex-wrap items-center justify-between gap-3">
                    <div className="space-y-1">
                      <div className="font-mono text-[10px] uppercase tracking-[0.08em] text-[color:var(--color-text-tertiary)]">
                        {activeRoom?.composeHint ??
                          "Messages poll automatically every few seconds."}
                      </div>
                      {activeRoom?.lastActivityAt ? (
                        <div className="text-[12px] text-[color:var(--color-text-secondary)]">
                          Last room activity {formatNetworkDateTime(activeRoom.lastActivityAt)}
                        </div>
                      ) : null}
                    </div>
                    <Button
                      data-testid="network-composer-submit"
                      disabled={
                        !activeRoom?.canCompose || composeDraft.trim() === "" || isComposePending
                      }
                      onClick={onComposeSubmit}
                      type="button"
                    >
                      <SendHorizontal className="size-4" />
                      Send
                    </Button>
                  </div>
                </div>
              </div>
            </div>
          </>
        )}
      </section>

      {isDetailsOpen ? (
        <aside
          className="flex min-h-0 flex-col border-t border-[color:var(--color-divider)] bg-[color:var(--color-surface-panel)] lg:border-l lg:border-t-0"
          data-testid="network-details-panel"
        >
          <div className="border-b border-[color:var(--color-divider)] px-4 py-4">
            <div className="flex items-center gap-2">
              <Users className="size-4 text-[color:var(--color-text-secondary)]" />
              <span className="text-[14px] font-medium text-[color:var(--color-text-primary)]">
                Room details
              </span>
            </div>
            <div aria-label="Room detail tabs" className="mt-4 flex gap-2" role="tablist">
              {(["about", "members", "wire"] as const).map(tab => (
                <button
                  aria-selected={detailsTab === tab}
                  className={cn(
                    "rounded-[var(--radius-chip)] border px-3 py-1.5 font-mono text-[10px] uppercase tracking-[0.08em] transition-colors",
                    detailsTab === tab
                      ? "border-[color:var(--color-accent)] bg-[color:var(--color-accent)] text-[color:var(--color-accent-ink)]"
                      : "border-[color:var(--color-divider)] text-[color:var(--color-text-secondary)] hover:border-[color:var(--color-accent-dim)] hover:text-[color:var(--color-text-primary)]"
                  )}
                  key={tab}
                  onClick={() => onSelectDetailsTab(tab)}
                  role="tab"
                  type="button"
                >
                  {tab}
                </button>
              ))}
            </div>
          </div>

          <div className="min-h-0 flex-1 overflow-y-auto px-4 py-5">
            {activeRoom == null ? (
              <p className="text-[13px] leading-6 text-[color:var(--color-text-tertiary)]">
                Select a room to inspect its metadata.
              </p>
            ) : detailsTab === "about" ? (
              <div className="space-y-5" data-testid="network-details-about">
                <div className="space-y-2">
                  <p className="text-[14px] font-medium text-[color:var(--color-text-primary)]">
                    {activeRoom.description}
                  </p>
                  <p className="text-[13px] leading-6 text-[color:var(--color-text-secondary)]">
                    Room activity last changed{" "}
                    {activeRoom.lastActivityAt
                      ? formatNetworkDateTime(activeRoom.lastActivityAt)
                      : "recently"}
                    .
                  </p>
                </div>
                <NetworkDetailFieldList fields={activeRoom.aboutFields} />
                {activeRoom.capabilities.length > 0 ? (
                  <div className="space-y-3">
                    <div className="font-mono text-[10px] uppercase tracking-[0.08em] text-[color:var(--color-text-tertiary)]">
                      Capabilities
                    </div>
                    <div className="space-y-3">
                      {activeRoom.capabilities.map(capability => (
                        <div
                          className="rounded-[var(--radius-md)] border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] p-4"
                          key={capability.id}
                        >
                          <div className="flex flex-wrap items-center gap-2">
                            <MonoBadge uppercase={false}>{capability.id}</MonoBadge>
                            {capability.detail?.version ? (
                              <MonoBadge uppercase={false} tone="info">
                                {capability.detail.version}
                              </MonoBadge>
                            ) : null}
                          </div>
                          <p className="mt-2 text-[13px] leading-6 text-[color:var(--color-text-secondary)]">
                            {capability.summary}
                          </p>
                        </div>
                      ))}
                    </div>
                  </div>
                ) : null}
              </div>
            ) : detailsTab === "members" ? (
              <div className="space-y-3" data-testid="network-details-members">
                {activeRoom.members.length === 0 ? (
                  <p className="text-[13px] leading-6 text-[color:var(--color-text-tertiary)]">
                    No visible members were returned for this room yet.
                  </p>
                ) : (
                  activeRoom.members.map(member => (
                    <div
                      className="rounded-[var(--radius-md)] border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] p-4"
                      key={member.id}
                    >
                      <div className="flex items-center gap-3">
                        <StatusDot tone={member.tone} />
                        <div className="min-w-0 flex-1">
                          <p className="truncate text-[13px] font-medium text-[color:var(--color-text-primary)]">
                            {member.title}
                          </p>
                          <p className="truncate text-[12px] text-[color:var(--color-text-secondary)]">
                            {member.subtitle}
                          </p>
                        </div>
                        <MonoBadge tone={member.local ? "accent" : "default"}>
                          {member.local ? "local" : "remote"}
                        </MonoBadge>
                      </div>
                      {member.lastSeen ? (
                        <p className="mt-3 font-mono text-[10px] uppercase tracking-[0.08em] text-[color:var(--color-text-tertiary)]">
                          seen {formatNetworkRelativeTime(member.lastSeen)}
                        </p>
                      ) : null}
                    </div>
                  ))
                )}
              </div>
            ) : (
              <div className="space-y-5" data-testid="network-details-wire">
                <NetworkDetailFieldList fields={activeRoom.wireFields} />
                {activeRoom.kindCounts.length > 0 ? (
                  <div className="space-y-3">
                    <div className="font-mono text-[10px] uppercase tracking-[0.08em] text-[color:var(--color-text-tertiary)]">
                      Timeline Kinds
                    </div>
                    <div className="flex flex-wrap gap-2">
                      {activeRoom.kindCounts.map(metric => (
                        <MonoBadge
                          key={metric.kind}
                          tone={fieldToneToBadgeTone(getNetworkKindTone(metric.kind))}
                        >
                          {formatNetworkKindLabel(metric.kind)} {metric.count}
                        </MonoBadge>
                      ))}
                    </div>
                  </div>
                ) : null}
              </div>
            )}
          </div>
        </aside>
      ) : null}
    </div>
  );
}
