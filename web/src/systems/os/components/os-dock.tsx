import { Plus } from "lucide-react";

import { Icon } from "@agh/ui";

import { cn } from "@/lib/utils";

import { isOsDockSeparator, type OsDockEntry, type OsDockItemData } from "./os-dock-types";

export type { OsDockEntry, OsDockItemData, OsDockSeparator } from "./os-dock-types";

/**
 * The dock: a centered glass strip of app launchers floating over the desktop,
 * with an optional detached New Session control in its own glass segment
 * (OpenDesign `dock-zone` anatomy). Presentational — WM wiring is Task 04.
 */

export interface OsDockProps extends Omit<React.ComponentProps<"nav">, "onSelect"> {
  items: OsDockEntry[];
  /** Item activation. Omit to render items as presentation. */
  onSelect?: (id: string) => void;
}

export interface OsDockNewSessionProps extends Omit<
  React.ComponentProps<"button">,
  "onClick" | "children"
> {
  /** When omitted the control renders as non-interactive presentation. */
  onNewSession?: () => void;
}

/** Counts cap at "9+" without collapsing the zero/non-zero distinction. */
function formatBadge(count: number): string {
  return count > 9 ? "9+" : String(count);
}

function DockItem({ item, onSelect }: { item: OsDockItemData; onSelect?: (id: string) => void }) {
  const body = (
    <>
      <span
        className={cn(
          "grid place-items-center text-muted transition-colors duration-base",
          item.minimized && "opacity-55"
        )}
      >
        <item.icon className="size-dock-icon" />
      </span>
      {item.badge ? (
        <span
          data-slot="os-dock-badge"
          className="absolute top-0.5 right-0.5 grid h-dock-badge min-w-dock-badge place-items-center rounded-lg bg-accent px-1 font-mono text-micro font-bold text-accent-ink"
        >
          {formatBadge(item.badge)}
        </span>
      ) : null}
      <span
        data-slot="os-dock-indicator"
        aria-hidden="true"
        className={cn(
          "absolute bottom-0 left-1/2 -translate-x-1/2 rounded-full transition-opacity duration-base",
          item.running || item.minimized ? "opacity-100" : "opacity-0",
          item.minimized
            ? "size-dock-indicator-min border border-muted bg-transparent"
            : "size-dock-indicator bg-muted"
        )}
      />
    </>
  );

  const base =
    "relative grid size-dock-item place-items-center rounded-dock-item transition-[transform,background-color,color] duration-shell-fast ease-spring";
  const interactive =
    "hover:bg-btn-default-fill focus-visible:shadow-focus-ring focus-visible:outline-none";
  const classes = cn(base, onSelect && interactive);

  if (!onSelect) {
    return (
      <span data-slot="os-dock-item" data-app={item.id} className={classes} title={item.name}>
        {body}
      </span>
    );
  }
  return (
    <button
      type="button"
      data-slot="os-dock-item"
      data-app={item.id}
      aria-label={item.name}
      className={classes}
      onClick={() => onSelect(item.id)}
    >
      {body}
    </button>
  );
}

const DOCK_SEG =
  "flex items-end gap-dock-gap rounded-dock border border-line bg-shell-glass p-dock-pad shadow-dock backdrop-blur-shell";

export function OsDock({ items, onSelect, className, ...props }: OsDockProps) {
  return (
    <nav data-slot="os-dock" aria-label="Dock" className={cn(DOCK_SEG, className)} {...props}>
      {items.map(entry =>
        isOsDockSeparator(entry) ? (
          <span
            key={entry.id}
            data-slot="os-dock-sep"
            aria-hidden="true"
            className="mb-dock-pad h-8 w-px shrink-0 self-end bg-line-strong"
          />
        ) : (
          <DockItem key={entry.id} item={entry} onSelect={onSelect} />
        )
      )}
    </nav>
  );
}

/**
 * Detached New Session control — its own glass segment beside the dock strip
 * (OpenDesign `dock-actions` / `dock-new`).
 */
export function OsDockNewSession({ onNewSession, className, ...props }: OsDockNewSessionProps) {
  const inner = (
    <span
      className={cn(
        "grid size-dock-item place-items-center rounded-dock-item bg-accent text-accent-ink shadow-highlight",
        onNewSession &&
          "transition-transform duration-shell-fast ease-spring hover:-translate-y-0.5 hover:bg-accent-hover"
      )}
    >
      <Icon as={Plus} className="size-dock-new-icon" />
    </span>
  );

  if (!onNewSession) {
    return (
      <div data-slot="os-dock-actions" className={cn(DOCK_SEG, className)}>
        <span
          data-slot="os-dock-new"
          title="New session"
          {...(props as React.ComponentProps<"span">)}
        >
          {inner}
        </span>
      </div>
    );
  }

  return (
    <div data-slot="os-dock-actions" className={cn(DOCK_SEG, className)}>
      <button
        type="button"
        data-slot="os-dock-new"
        aria-label="New session"
        title="New session"
        className="rounded-dock-item focus-visible:shadow-focus-ring focus-visible:outline-none"
        onClick={onNewSession}
        {...props}
      >
        {inner}
      </button>
    </div>
  );
}

/**
 * Full dock zone: centered strip + detached New Session, matching OpenDesign
 * `dock-zone` (flex spacers keep the pair centered).
 */
export function OsDockZone({
  items,
  onSelect,
  onNewSession,
  className,
  ...props
}: {
  items: OsDockEntry[];
  onSelect?: (id: string) => void;
  onNewSession?: () => void;
  className?: string;
} & Omit<React.ComponentProps<"div">, "children">) {
  return (
    <div
      data-slot="os-dock-zone"
      className={cn(
        "pointer-events-none absolute inset-x-0 bottom-2.5 z-10 flex items-end gap-2.5 px-4",
        className
      )}
      {...props}
    >
      <span className="min-w-0 flex-1" aria-hidden="true" />
      <OsDock items={items} onSelect={onSelect} className="pointer-events-auto" />
      <OsDockNewSession onNewSession={onNewSession} className="pointer-events-auto" />
      <span className="min-w-0 flex-1" aria-hidden="true" />
    </div>
  );
}
