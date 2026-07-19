import { Plus, Settings2 } from "lucide-react";

import {
  Button,
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
  Pill,
  TopbarOverflowIcon,
} from "@agh/ui";

import { formatAgentOriginLabel, formatCategoryMetaSegment } from "../lib/agent-fleet-projection";
import type { AgentPayload } from "../types";

export interface AgentPageStatusPillProps {
  activeCount: number;
}

export function AgentPageStatusPill({ activeCount }: AgentPageStatusPillProps) {
  const status =
    activeCount > 0
      ? { label: "Active", tone: "success" as const }
      : { label: "Idle", tone: "neutral" as const };
  return (
    <Pill tone={status.tone} data-testid="agent-page-header-status">
      <Pill.Dot tone={status.tone} size="sm" />
      {status.label}
    </Pill>
  );
}

export interface AgentPageMetaProps {
  agent: AgentPayload;
}

export function AgentPageMeta({ agent }: AgentPageMetaProps) {
  const category = formatCategoryMetaSegment(agent.category_path);
  const origin = formatAgentOriginLabel(agent.origin);
  const parts = [category, origin].filter(Boolean);
  if (parts.length === 0) return null;
  return (
    <span
      className="truncate font-mono text-badge tracking-mono text-muted"
      data-testid="agent-page-meta"
    >
      {parts.join(" · ")}
    </span>
  );
}

export interface AgentPageActionsProps {
  onEditSettings: () => void;
  onNewSession: () => void;
  isCreatingSession: boolean;
  newSessionDisabled: boolean;
}

/** Leading topbar actions: New session (accent) + Edit settings (secondary). */
export function AgentPageActions({
  onEditSettings,
  onNewSession,
  isCreatingSession,
  newSessionDisabled,
}: AgentPageActionsProps) {
  return (
    <div className="flex items-center gap-2" data-testid="agent-page-toolbar">
      <Button
        type="button"
        variant="default"
        size="sm"
        onClick={onNewSession}
        disabled={newSessionDisabled}
        aria-busy={isCreatingSession}
        data-testid="agent-page-new-session"
      >
        <Plus aria-hidden="true" className="size-3" />
        New session
      </Button>
      <Button
        type="button"
        variant="neutral"
        size="sm"
        onClick={onEditSettings}
        data-testid="agent-page-edit-settings"
      >
        <Settings2 aria-hidden="true" className="size-3" />
        Edit settings
      </Button>
    </div>
  );
}

export interface AgentPageOverflowProps {
  onDuplicate: () => void;
  onDelete: () => void;
}

/** Trailing topbar overflow: Duplicate + Delete. */
export function AgentPageOverflow({ onDuplicate, onDelete }: AgentPageOverflowProps) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        aria-label="More agent actions"
        data-testid="agent-page-overflow"
        render={<Button type="button" variant="ghost" size="icon-sm" />}
      >
        <TopbarOverflowIcon aria-hidden="true" className="size-3" />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" data-testid="agent-page-overflow-menu">
        <DropdownMenuItem
          data-testid="agent-page-duplicate"
          onClick={() => {
            onDuplicate();
          }}
        >
          Duplicate
        </DropdownMenuItem>
        <DropdownMenuItem
          variant="destructive"
          data-testid="agent-page-delete"
          onClick={() => {
            onDelete();
          }}
        >
          Delete…
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
