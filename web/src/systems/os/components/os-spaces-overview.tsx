import { Dialog, DialogContent, DialogDescription, DialogTitle, Pill, cn } from "@agh/ui";

import type { WorkspacePayload } from "@/systems/workspace";

import { useOsReducedMotion } from "../hooks/use-os-reduced-motion";
import { useSpaceArrangements, type OsSpaceCardModel } from "../hooks/use-space-arrangements";
import { useDesktop } from "../hooks/use-desktop";
import type { OsDesktopBounds, OsWallpaper, OsWindow } from "../lib/os-types";

export interface OsSpacesOverviewProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  workspaces: WorkspacePayload[];
  activeWorkspaceId: string | null;
  onSelectWorkspace: (workspaceId: string) => void;
}

/** Prototype thumb geometry (os-v2.js `openSpaces`): card-scale factors with viewport floors. */
const SPACE_CARD_WIDTH = 264;
const SPACE_THUMB_HEIGHT = 148;
const THUMB_MIN_DESK_WIDTH = 1000;
const THUMB_MIN_DESK_HEIGHT = 600;
const THUMB_MAX_WINDOWS = 4;
const THUMB_TOP_OFFSET = 8;

const THUMB_WALLPAPER: Record<OsWallpaper, string> = {
  ember: "var(--wallpaper-thumb-ember)",
  mesh: "var(--wallpaper-thumb-mesh)",
  carbon: "var(--wallpaper-thumb-carbon)",
};

function workspaceMonogram(name: string): string {
  return name.trim().slice(0, 2).toUpperCase() || "WS";
}

function countLabel(count: number): string {
  return `${count} window${count === 1 ? "" : "s"}`;
}

function cardDetail(card: OsSpaceCardModel | undefined): string {
  if (!card || card.status === "loading") return "Loading…";
  if (card.status === "error" || card.arrangement === null) return "Arrangement unavailable";
  return countLabel(card.arrangement.windows.length);
}

function SpaceThumb({
  card,
  bounds,
}: {
  card: OsSpaceCardModel | undefined;
  bounds: OsDesktopBounds | null;
}) {
  const arrangement = card?.arrangement ?? null;
  const wallpaper = arrangement?.wallpaper ?? "ember";
  const deskWidth = Math.max(bounds?.width ?? 0, THUMB_MIN_DESK_WIDTH);
  const deskHeight = Math.max(bounds?.height ?? 0, THUMB_MIN_DESK_HEIGHT);
  const sx = SPACE_CARD_WIDTH / deskWidth;
  const sy = SPACE_THUMB_HEIGHT / deskHeight;
  const minis: OsWindow[] = arrangement ? arrangement.windows.slice(-THUMB_MAX_WINDOWS) : [];

  return (
    <span
      data-slot="os-space-thumb"
      data-wallpaper={wallpaper}
      aria-hidden="true"
      className="relative block h-space-thumb w-full overflow-hidden border-b border-line"
      style={{ backgroundImage: THUMB_WALLPAPER[wallpaper] }}
    >
      {minis.map(win => (
        <span
          key={win.id}
          data-slot="os-space-mini-win"
          className="absolute rounded-xs border border-line-strong bg-line-strong"
          style={{
            left: win.rect.x * sx,
            top: win.rect.y * sy + THUMB_TOP_OFFSET,
            width: win.rect.w * sx,
            height: win.rect.h * sy,
          }}
        />
      ))}
    </span>
  );
}

/**
 * The ⇧⌘S Spaces overview (US-010.AC-2): one card per workspace with a
 * mini-window thumbnail of its arrangement — the active space from the live
 * store, the rest from persisted desktop state (delta #4: real workspaces).
 * A desktop-level overlay from the closed set (Modal & Overlay Policy).
 */
export function OsSpacesOverview({
  open,
  onOpenChange,
  workspaces,
  activeWorkspaceId,
  onSelectWorkspace,
}: OsSpacesOverviewProps) {
  const desktopBounds = useDesktop(state => state.desktopBounds);
  const presentation = useDesktop(state => state.presentation);
  const reducedMotion = useOsReducedMotion();
  const cards = useSpaceArrangements(
    workspaces.map(workspace => workspace.id),
    activeWorkspaceId,
    { enabled: open }
  );

  const selectWorkspace = (workspaceId: string) => {
    if (workspaceId !== activeWorkspaceId) onSelectWorkspace(workspaceId);
    onOpenChange(false);
  };

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
        <DialogTitle className="text-compact-h1 font-semibold text-fg-strong">Spaces</DialogTitle>
        <DialogDescription className="mb-spaces-subtitle-gap text-small-body text-muted">
          Each workspace is a space — windows stay where you left them.
        </DialogDescription>
        <div
          data-slot="os-spaces-row"
          className={cn(
            "flex max-w-full justify-center",
            presentation === "compact"
              ? "max-h-[64vh] flex-col items-center gap-3 overflow-y-auto px-4 py-1"
              : "max-h-[70vh] flex-wrap gap-spaces-row-gap overflow-y-auto px-6 py-1"
          )}
        >
          {workspaces.map(workspace => {
            const active = workspace.id === activeWorkspaceId;
            const card = cards.get(workspace.id);
            const detail = cardDetail(card);
            return (
              <button
                key={workspace.id}
                type="button"
                data-slot="os-space-card"
                data-workspace-id={workspace.id}
                data-current={active ? "true" : undefined}
                aria-label={`${active ? "Current workspace" : "Switch to"} ${workspace.name}. ${detail}`}
                className={cn(
                  "w-space-card shrink-0 overflow-hidden rounded-xl border text-left",
                  "transition-[transform,border-color] duration-shell-fast ease-spring",
                  "focus-visible:shadow-focus-ring focus-visible:outline-none",
                  !reducedMotion && "hover:-translate-y-1",
                  active ? "border-accent" : "border-line-strong bg-canvas-soft"
                )}
                onClick={() => selectWorkspace(workspace.id)}
              >
                <SpaceThumb card={card} bounds={desktopBounds} />
                <span data-slot="os-space-foot" className="flex items-center gap-2 px-3 py-2.5">
                  <span
                    aria-hidden="true"
                    className="grid size-5 shrink-0 place-items-center rounded-xs border border-line-strong bg-elevated font-mono text-space-avatar font-semibold text-muted"
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
                  <span className="shrink-0 font-mono text-micro text-subtle">{detail}</span>
                </span>
              </button>
            );
          })}
        </div>
      </DialogContent>
    </Dialog>
  );
}
