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

import {
  Button,
  Empty,
  KIND_DOT_COLORS,
  KindChip,
  MonoBadge,
  MonoChip,
  Pills,
  SearchInput,
  SidebarSectionLabel,
  StatusDot,
  Textarea,
  WireCard,
  WireChip,
} from "@agh/ui";
import { cn } from "@/lib/utils";

import {
  NETWORK_KIND_FILTERS,
  formatNetworkDateTime,
  formatNetworkKindLabel,
  formatNetworkRelativeTime,
  getMessageAuthorInitial,
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
  onTogglePresence: () => void;
  onToggleStarChannel: (channel: string) => void;
  roomError: Error | null;
  selectedRoomKey: string | null;
  sidebarQuery: string;
  showPresence: boolean;
  status: NetworkStatus;
  onSidebarQueryChange: (value: string) => void;
  starredChannelRooms: NetworkRoomListItem[];
}

const AVATAR_PALETTE: ReadonlyArray<readonly [string, string]> = [
  ["var(--color-accent-tint)", "var(--color-accent)"],
  ["var(--color-info-tint)", "var(--color-info)"],
  ["var(--color-success-tint)", "var(--color-success)"],
  ["var(--color-warning-tint)", "var(--color-warning)"],
  ["var(--color-danger-tint)", "var(--color-danger)"],
  ["var(--color-neutral-tint)", "var(--color-text-label)"],
  ["var(--color-surface-elevated)", "var(--color-text-primary)"],
  ["var(--color-surface-panel)", "var(--color-text-secondary)"],
];

function pickAvatarColors(seed: string): readonly [string, string] {
  let hash = 0;
  for (let i = 0; i < seed.length; i += 1) {
    hash = (hash * 31 + seed.charCodeAt(i)) | 0;
  }
  const index = Math.abs(hash) % AVATAR_PALETTE.length;
  return AVATAR_PALETTE[index]!;
}

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

function NetworkShellSection({ children, title }: { children: ReactNode; title: string }) {
  return (
    <section className="space-y-1">
      <SidebarSectionLabel>{title}</SidebarSectionLabel>
      <div>{children}</div>
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
  const unread = item.unreadCount > 0;

  return (
    <div
      className={cn(
        "group relative mx-1.5 flex items-center gap-2 rounded-[6px] pr-1.5 transition-colors",
        active ? "bg-[color:var(--color-surface)]" : "hover:bg-[color:var(--color-hover)]"
      )}
      data-testid={`network-room-${item.roomType}-${item.id}`}
    >
      {active ? (
        <span
          aria-hidden="true"
          className="pointer-events-none absolute top-1.5 bottom-1.5 -left-1.5 w-[2px] rounded-r-[2px] bg-[color:var(--color-accent)]"
        />
      ) : null}
      <button
        aria-current={active ? "page" : undefined}
        className="flex min-w-0 flex-1 items-center gap-2.5 rounded-[6px] px-2 py-1.5 text-left focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-[color:var(--color-accent)]"
        onClick={() => onSelect(item)}
        type="button"
      >
        <span className="flex shrink-0 items-center">
          {isChannel ? (
            <Hash
              aria-hidden="true"
              className={cn(
                "size-[13px]",
                active
                  ? "text-[color:var(--color-text-primary)]"
                  : "text-[color:var(--color-text-tertiary)]"
              )}
            />
          ) : (
            <StatusDot tone={item.tone} />
          )}
        </span>
        <span
          className={cn(
            "truncate font-mono text-[12px] tracking-[-0.01em]",
            active
              ? "text-[color:var(--color-text-primary)] font-medium"
              : "text-[color:var(--color-text-secondary)]",
            unread && "font-semibold text-[color:var(--color-text-primary)]"
          )}
        >
          {item.title}
        </span>
      </button>
      {unread ? (
        <MonoBadge tone="solid-accent" uppercase={false}>
          {item.unreadCount}
        </MonoBadge>
      ) : null}
      {isChannel ? (
        <button
          aria-label={item.isStarred ? "Unstar channel" : "Star channel"}
          className={cn(
            "rounded-[4px] p-1 text-[color:var(--color-text-tertiary)] opacity-0 transition-opacity focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-[color:var(--color-accent)] group-hover:opacity-100",
            (item.isStarred || active) && "opacity-100"
          )}
          onClick={event => {
            event.stopPropagation();
            onToggleStar?.(item.id);
          }}
          type="button"
        >
          <Sparkles
            className={cn("size-3", item.isStarred && "text-[color:var(--color-accent)]")}
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
        <WireCard className="w-full max-w-none space-y-3 p-3.5">
          <div className="flex flex-wrap items-center gap-1.5">
            {readString(capability, "id") ? (
              <MonoChip>{readString(capability, "id")}</MonoChip>
            ) : null}
            {readString(capability, "version") ? (
              <MonoChip>{readString(capability, "version")}</MonoChip>
            ) : null}
          </div>
          <div className="space-y-1">
            <p className="text-[13.5px] leading-[1.55] text-[color:var(--color-text-primary)]">
              {readString(capability, "summary") ?? getNetworkMessagePrimaryText(message)}
            </p>
            {readString(capability, "outcome") ? (
              <p className="text-[12.5px] leading-5 text-[color:var(--color-text-secondary)]">
                {readString(capability, "outcome")}
              </p>
            ) : null}
          </div>
          {readStringList(capability, "execution_outline").length > 0 ? (
            <div className="space-y-1.5">
              <SidebarSectionLabel className="px-0 pt-0 pb-0">
                Execution Outline
              </SidebarSectionLabel>
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
        </WireCard>
      );
    case "receipt":
      return (
        <WireCard className="w-full max-w-none space-y-1.5 p-3.5">
          <div className="flex flex-wrap items-center gap-1.5">
            {readString(body, "status") ? <MonoChip>{readString(body, "status")}</MonoChip> : null}
            {readString(body, "for_id") ? <MonoChip>{readString(body, "for_id")}</MonoChip> : null}
          </div>
          <p className="text-[13.5px] leading-[1.55] text-[color:var(--color-text-primary)]">
            {readString(body, "detail") ?? getNetworkMessagePrimaryText(message)}
          </p>
        </WireCard>
      );
    case "greet":
      return (
        <WireCard className="w-full max-w-none space-y-2.5 p-3.5">
          <div className="flex flex-wrap items-center gap-1.5">
            {readString(peerCard, "display_name") ? (
              <span className="font-mono text-[12px] font-medium text-[color:var(--color-text-primary)]">
                {readString(peerCard, "display_name")}
              </span>
            ) : null}
            {message.presence_count && message.presence_count > 1 ? (
              <MonoChip>{message.presence_count} heartbeats</MonoChip>
            ) : null}
          </div>
          <p className="text-[13.5px] leading-[1.55] text-[color:var(--color-text-primary)]">
            {readString(body, "summary") ?? getNetworkMessagePrimaryText(message)}
          </p>
          {message.presence_started_at && message.presence_last_seen_at ? (
            <p className="font-mono text-[10px] uppercase tracking-[0.08em] text-[color:var(--color-text-tertiary)]">
              {formatNetworkDateTime(message.presence_started_at)} to{" "}
              {formatNetworkDateTime(message.presence_last_seen_at)}
            </p>
          ) : null}
          {readStringList(peerCard, "capabilities").length > 0 ? (
            <div className="flex flex-wrap gap-1.5">
              {readStringList(peerCard, "capabilities").map(capabilityName => (
                <MonoChip key={capabilityName}>{capabilityName}</MonoChip>
              ))}
            </div>
          ) : null}
        </WireCard>
      );
    case "whois":
      return (
        <WireCard className="w-full max-w-none space-y-1.5 p-3.5">
          {readString(body, "type") ? (
            <div className="flex flex-wrap items-center gap-1.5">
              <MonoChip>{readString(body, "type")}</MonoChip>
            </div>
          ) : null}
          <p className="text-[13.5px] leading-[1.55] text-[color:var(--color-text-primary)]">
            {readString(body, "query") ?? getNetworkMessagePrimaryText(message)}
          </p>
          {readString(peerCard, "peer_id") ? (
            <p className="font-mono text-[11px] text-[color:var(--color-text-tertiary)]">
              {readString(peerCard, "peer_id")}
            </p>
          ) : null}
        </WireCard>
      );
    case "trace":
      return (
        <WireCard className="w-full max-w-none space-y-1.5 p-3.5">
          {readString(body, "state") ? (
            <div className="flex flex-wrap items-center gap-1.5">
              <MonoChip>{readString(body, "state")}</MonoChip>
            </div>
          ) : null}
          <p className="text-[13.5px] leading-[1.55] text-[color:var(--color-text-primary)]">
            {readString(body, "message") ?? getNetworkMessagePrimaryText(message)}
          </p>
        </WireCard>
      );
    default:
      return (
        <div className="space-y-1.5">
          {intent ? <MonoChip>{intent}</MonoChip> : null}
          <p className="whitespace-pre-wrap text-[13.5px] leading-[1.55] text-[color:var(--color-text-primary)]">
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
    <div className="px-5 pb-8" data-testid="network-message-list">
      {messages.map((message, index) => {
        const grouped = isGroupedWithPrevious(messages[index - 1], message);
        const senderSeed = message.display_name ?? message.peer_from;
        const [avatarBg, avatarFg] = pickAvatarColors(senderSeed);
        const kindLabel =
          message.kind === "greet" && (message.presence_count ?? 0) > 1
            ? "presence"
            : formatNetworkKindLabel(message.kind);

        return (
          <article
            className={cn(
              "group/msg flex gap-3.5 transition-colors hover:bg-[color:var(--color-hover)]",
              grouped ? "px-2 py-1" : "px-2 pt-3.5 pb-2"
            )}
            data-testid={`network-message-${message.message_id}`}
            key={message.message_id}
          >
            <div className="w-9 shrink-0">
              {grouped ? (
                <span className="flex h-[18px] items-center justify-center font-mono text-[10px] text-[color:var(--color-text-tertiary)] opacity-0 transition-opacity group-hover/msg:opacity-100">
                  {message.timestamp.slice(11, 16)}
                </span>
              ) : (
                <div
                  aria-hidden="true"
                  className="flex size-9 items-center justify-center rounded-[6px] font-mono text-[10px] font-semibold tracking-[-0.02em]"
                  style={{ background: avatarBg, color: avatarFg }}
                >
                  {getMessageAuthorInitial(message)}
                </div>
              )}
            </div>
            <div className="min-w-0 flex-1 space-y-1.5">
              {grouped ? null : (
                <div className="flex flex-wrap items-center gap-2">
                  <span className="font-mono text-[13px] font-semibold text-[color:var(--color-text-primary)]">
                    {message.display_name ?? message.peer_from}
                  </span>
                  <KindChip kind={message.kind} label={kindLabel} />
                  <span className="font-mono text-[10.5px] text-[color:var(--color-text-tertiary)]">
                    {formatNetworkRelativeTime(message.timestamp)}
                  </span>
                  {message.direction === "sent" ? (
                    <MonoBadge tone="accent" uppercase={false}>
                      {message.direction}
                    </MonoBadge>
                  ) : null}
                </div>
              )}
              <NetworkMessageBody message={message} />
              {(message.peer_to || message.trace_id || message.reply_to) && !grouped ? (
                <div className="flex flex-wrap gap-1.5">
                  {message.peer_to ? <MonoChip>to {message.peer_to}</MonoChip> : null}
                  {message.trace_id ? <MonoChip>trace {message.trace_id}</MonoChip> : null}
                  {message.reply_to ? <MonoChip>reply {message.reply_to}</MonoChip> : null}
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
  onTogglePresence,
  onToggleStarChannel,
  roomError,
  selectedRoomKey,
  sidebarQuery,
  showPresence,
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
                className="flex flex-wrap items-center gap-1.5"
                data-testid="network-kind-filter-bar"
                role="group"
              >
                <span className="mr-1 font-mono text-[9px] font-semibold uppercase tracking-[0.14em] text-[color:var(--color-text-label)]">
                  Filter by kind
                </span>
                <WireChip active={activeKind === "all"} onClick={() => onSelectKind("all")}>
                  all
                </WireChip>
                {NETWORK_KIND_FILTERS.map(kind => {
                  const count =
                    activeRoom?.kindCounts.find(metric => metric.kind === kind)?.count ?? 0;

                  return (
                    <WireChip
                      active={activeKind === kind}
                      dotColor={KIND_DOT_COLORS[kind.toLowerCase()]}
                      key={kind}
                      onClick={() => onSelectKind(kind)}
                    >
                      {formatNetworkKindLabel(kind).toLowerCase()}
                      {count > 0 ? ` ${count}` : ""}
                    </WireChip>
                  );
                })}
                <WireChip
                  active={showPresence}
                  dotColor="var(--color-success)"
                  onClick={onTogglePresence}
                >
                  presence
                  {(activeRoom?.presenceCount ?? 0) > 0 ? ` ${activeRoom?.presenceCount ?? 0}` : ""}
                </WireChip>
              </div>
            </div>

            <div className="flex min-h-0 flex-1 flex-col overflow-hidden">
              <div className="flex-1 overflow-y-auto">
                <div className="px-5 py-5" data-testid="network-room-intro">
                  <WireCard className="w-full max-w-none p-5">
                    <div className="flex flex-wrap items-center gap-1.5">
                      <KindChip kind={activeRoom?.roomType === "channel" ? "channel" : "direct"} />
                      {activeRoom?.purpose ? <MonoChip>{activeRoom.purpose}</MonoChip> : null}
                    </div>
                    <h2 className="mt-3 text-[16px] font-semibold text-[color:var(--color-text-primary)]">
                      {activeRoom?.introTitle ?? "Loading room"}
                    </h2>
                    <p className="mt-2 max-w-3xl text-[13.5px] leading-[1.55] text-[color:var(--color-text-secondary)]">
                      {activeRoom?.introBody ?? "Resolving room metadata and timeline context."}
                    </p>
                  </WireCard>
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
            <div className="mt-4">
              <Pills<NetworkDetailsTab>
                aria-label="Room detail tabs"
                items={[
                  { value: "about", label: "About" },
                  { value: "members", label: "Members" },
                  { value: "wire", label: "Wire" },
                ]}
                onChange={onSelectDetailsTab}
                value={detailsTab}
              />
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
                  <div className="space-y-2">
                    <SidebarSectionLabel className="px-0 pt-0 pb-0">
                      Capabilities
                    </SidebarSectionLabel>
                    <div className="space-y-2">
                      {activeRoom.capabilities.map(capability => (
                        <WireCard className="w-full max-w-none p-3.5" key={capability.id}>
                          <div className="flex flex-wrap items-center gap-1.5">
                            <MonoChip>{capability.id}</MonoChip>
                            {capability.detail?.version ? (
                              <MonoChip>{capability.detail.version}</MonoChip>
                            ) : null}
                          </div>
                          <p className="mt-2 text-[12.5px] leading-5 text-[color:var(--color-text-secondary)]">
                            {capability.summary}
                          </p>
                        </WireCard>
                      ))}
                    </div>
                  </div>
                ) : null}
              </div>
            ) : detailsTab === "members" ? (
              <div className="space-y-2" data-testid="network-details-members">
                {activeRoom.members.length === 0 ? (
                  <p className="text-[13px] leading-6 text-[color:var(--color-text-tertiary)]">
                    No visible members were returned for this room yet.
                  </p>
                ) : (
                  activeRoom.members.map(member => (
                    <WireCard className="w-full max-w-none p-3.5" key={member.id}>
                      <div className="flex items-center gap-3">
                        <StatusDot tone={member.tone} />
                        <div className="min-w-0 flex-1">
                          <p className="truncate font-mono text-[12px] font-medium text-[color:var(--color-text-primary)]">
                            {member.title}
                          </p>
                          <p className="truncate text-[11.5px] text-[color:var(--color-text-secondary)]">
                            {member.subtitle}
                          </p>
                        </div>
                        <MonoChip>{member.local ? "local" : "remote"}</MonoChip>
                      </div>
                      {member.lastSeen ? (
                        <p className="mt-3 font-mono text-[10px] uppercase tracking-[0.08em] text-[color:var(--color-text-tertiary)]">
                          seen {formatNetworkRelativeTime(member.lastSeen)}
                        </p>
                      ) : null}
                    </WireCard>
                  ))
                )}
              </div>
            ) : (
              <div className="space-y-5" data-testid="network-details-wire">
                <NetworkDetailFieldList fields={activeRoom.wireFields} />
                {activeRoom.kindCounts.length > 0 ? (
                  <div className="space-y-2">
                    <SidebarSectionLabel className="px-0 pt-0 pb-0">
                      Timeline Kinds
                    </SidebarSectionLabel>
                    <div className="flex flex-wrap gap-1.5">
                      {activeRoom.kindCounts.map(metric => (
                        <KindChip
                          key={metric.kind}
                          kind={metric.kind}
                          label={`${formatNetworkKindLabel(metric.kind)} ${metric.count}`}
                        />
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
