import { ArrowRight, Plus } from "lucide-react";

import {
  Button,
  Dialog,
  DialogContent,
  DialogDescription,
  DialogTitle,
  Eyebrow,
  OwnerAvatar,
  Pill,
  cn,
} from "@agh/ui";

import type { WorkspacePayload } from "@/systems/workspace";

import { useOsReducedMotion } from "../hooks/use-os-reduced-motion";
import { useSpaceArrangements, type OsSpaceCardModel } from "../hooks/use-space-arrangements";
import { useSpaceDetails, type OsSpaceDetailModel } from "../hooks/use-space-details";
import { useDesktop } from "../hooks/use-desktop";

export interface OsSpacesOverviewProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  workspaces: WorkspacePayload[];
  activeWorkspaceId: string | null;
  onSelectWorkspace: (workspaceId: string) => void;
  onNewSpace: () => void;
}

const MEMBER_AVATAR_MAX = 5;

function workspaceMonogram(name: string): string {
  return name.trim().slice(0, 2).toUpperCase() || "WS";
}

function windowCountLabel(card: OsSpaceCardModel | undefined): string | null {
  if (!card || card.status === "loading") return null;
  if (card.status === "error" || card.arrangement === null) return null;
  const count = card.arrangement.windows.length;
  return `${count} window${count === 1 ? "" : "s"}`;
}

function spaceMetaLine(detail: OsSpaceDetailModel | undefined, windowLabel: string | null): string {
  if (!detail || detail.status === "loading") return "Loading…";
  if (detail.status === "error") return windowLabel ?? "Details unavailable";
  const parts = [
    `${detail.agentCount} agent${detail.agentCount === 1 ? "" : "s"}`,
    `${detail.sessionCount} session${detail.sessionCount === 1 ? "" : "s"}`,
  ];
  if (windowLabel) parts.push(windowLabel);
  return parts.join(" · ");
}

function SpaceMembers({ detail }: { detail: OsSpaceDetailModel | undefined }) {
  if (!detail || detail.status !== "ready" || detail.agentNames.length === 0) return null;
  const visible = detail.agentNames.slice(0, MEMBER_AVATAR_MAX);
  const overflow = detail.agentNames.length - visible.length;

  return (
    <span className="flex flex-col gap-1.5" data-slot="os-space-members">
      <Eyebrow className="text-faint">Members</Eyebrow>
      <span className="flex items-center">
        <span className="flex -space-x-1.5">
          {visible.map(name => (
            <OwnerAvatar
              className="ring-2 ring-canvas-soft"
              key={name}
              name={name}
              ownerId={name}
              ownerKind="agent"
              size="sm"
            />
          ))}
        </span>
        {overflow > 0 ? (
          <span className="ml-1.5 font-mono text-mono-id text-subtle">+{overflow}</span>
        ) : null}
      </span>
    </span>
  );
}

/**
 * The ⇧⌘S Spaces overview: page-style header (space count, agent rollup, New
 * space) over one card per workspace — identity, real agent/session counts,
 * member monograms, and the Enter affordance. Counts come from the workspace
 * detail endpoint; nothing rendered here is invented.
 */
export function OsSpacesOverview({
  open,
  onOpenChange,
  workspaces,
  activeWorkspaceId,
  onSelectWorkspace,
  onNewSpace,
}: OsSpacesOverviewProps) {
  const presentation = useDesktop(state => state.presentation);
  const reducedMotion = useOsReducedMotion();
  const workspaceIds = workspaces.map(workspace => workspace.id);
  const cards = useSpaceArrangements(workspaceIds, activeWorkspaceId, { enabled: open });
  const details = useSpaceDetails(workspaceIds, { enabled: open });

  const selectWorkspace = (workspaceId: string) => {
    if (workspaceId !== activeWorkspaceId) onSelectWorkspace(workspaceId);
    onOpenChange(false);
  };

  const spaceCount = workspaces.length;
  const subtitle =
    details.totalAgents !== null
      ? `${spaceCount} space${spaceCount === 1 ? "" : "s"} · ${details.totalAgents} agent${details.totalAgents === 1 ? "" : "s"} distributed`
      : `${spaceCount} space${spaceCount === 1 ? "" : "s"}`;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        unframed
        showCloseButton={false}
        data-testid="os-spaces-overview"
        className={cn(
          "top-0 left-0 h-full w-full max-w-none translate-x-0 translate-y-0 overflow-hidden",
          "flex flex-col items-center justify-center gap-2 rounded-none bg-transparent",
          "shadow-none backdrop-blur-shell-scrim"
        )}
      >
        <div className="flex w-full max-w-[calc(3*var(--width-space-card)+2*var(--spacing-spaces-row-gap))] items-start justify-between gap-4 px-6">
          <div className="flex min-w-0 flex-col gap-1">
            <DialogTitle className="text-compact-h1 font-semibold text-fg-strong">
              Spaces
            </DialogTitle>
            <DialogDescription
              className="text-small-body text-muted"
              data-testid="os-spaces-subtitle"
            >
              {subtitle}
            </DialogDescription>
          </div>
          <Button
            data-testid="os-spaces-new-space"
            onClick={() => {
              onOpenChange(false);
              onNewSpace();
            }}
            size="sm"
            type="button"
          >
            <Plus aria-hidden="true" className="size-3" />
            New space
          </Button>
        </div>
        <div
          data-slot="os-spaces-row"
          className={cn(
            "mt-spaces-subtitle-gap flex max-w-full justify-center",
            presentation === "compact"
              ? "max-h-[64vh] flex-col items-center gap-3 overflow-y-auto px-4 py-1"
              : "max-h-[70vh] flex-wrap gap-spaces-row-gap overflow-y-auto px-6 py-1"
          )}
        >
          {workspaces.map(workspace => {
            const active = workspace.id === activeWorkspaceId;
            const detail = details.byWorkspaceId.get(workspace.id);
            const windowLabel = windowCountLabel(cards.get(workspace.id));
            const meta = spaceMetaLine(detail, windowLabel);
            return (
              <button
                key={workspace.id}
                type="button"
                data-slot="os-space-card"
                data-workspace-id={workspace.id}
                data-current={active ? "true" : undefined}
                aria-label={`${active ? "Current workspace" : "Switch to"} ${workspace.name}. ${meta}`}
                className={cn(
                  "group flex w-space-card shrink-0 flex-col gap-3 overflow-hidden rounded-xl border bg-canvas-soft p-4 text-left",
                  "transition-[transform,border-color] duration-shell-fast ease-spring",
                  "focus-visible:shadow-focus-ring focus-visible:outline-none",
                  !reducedMotion && "hover:-translate-y-1",
                  active ? "border-accent" : "border-line-strong hover:border-line-focus"
                )}
                onClick={() => selectWorkspace(workspace.id)}
              >
                <span className="flex items-center gap-2.5">
                  <span
                    aria-hidden="true"
                    className="grid size-7 shrink-0 place-items-center rounded-sm border border-line-strong bg-elevated font-mono text-space-avatar font-semibold text-muted"
                  >
                    {workspaceMonogram(workspace.name)}
                  </span>
                  <span className="min-w-0 flex-1 truncate text-small-body font-semibold text-fg-strong">
                    {workspace.name}
                  </span>
                  {active ? (
                    <Pill data-slot="os-space-current" size="xs" tone="neutral">
                      Current
                    </Pill>
                  ) : null}
                </span>
                <span
                  className="text-form-label text-subtle"
                  data-testid={`os-space-meta-${workspace.id}`}
                >
                  {meta}
                </span>
                <SpaceMembers detail={detail} />
                <span className="flex items-center justify-between border-t border-line-soft pt-2.5">
                  <span className="truncate font-mono text-mono-id text-faint">
                    {workspace.root_dir}
                  </span>
                  <span
                    className={cn(
                      "inline-flex shrink-0 items-center gap-1 text-form-label font-medium text-muted",
                      "transition-colors duration-base group-hover:text-fg"
                    )}
                  >
                    Enter
                    <ArrowRight aria-hidden="true" className="size-3" />
                  </span>
                </span>
              </button>
            );
          })}
        </div>
      </DialogContent>
    </Dialog>
  );
}
