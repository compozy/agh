import type { ReactNode } from "react";

import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuShortcut,
  DropdownMenuTrigger,
} from "@agh/ui";

import { cn } from "@/lib/utils";

import { useOsZoomMenu } from "../hooks/use-os-zoom-menu";
import { OS_ARRANGE_COMMANDS, OS_SNAP_COMMANDS } from "../lib/os-snap-commands";
import { OS_SNAP_ZONES } from "../lib/os-snap-zones";
import type { OsSnapZone } from "../lib/os-types";

/**
 * macOS-style zoom-button menu (Sequoia green-button posture): hovering the
 * zoom traffic light opens Move & Resize (halves + quarters as zone glyphs,
 * restore while snapped) and Fill & Arrange (fill, 2-up, grid). Click stays
 * `toggleZoom`; every action here also lives in the palette — the guaranteed
 * keyboard path — so the menu is discoverability, never the only route.
 * The hidden trigger span only anchors the Radix content; hover intent lives
 * on the wrapper so the real button keeps its own semantics.
 */

const HALF_IDS = ["left", "right", "top", "bottom"] as const;
const QUARTER_IDS = ["top-left", "top-right", "bottom-left", "bottom-right"] as const;

function ZoneGlyph({ zones, className }: { zones: readonly OsSnapZone[]; className?: string }) {
  return (
    <svg
      aria-hidden="true"
      viewBox="0 0 16 12"
      className={cn("size-4 text-current", className)}
      fill="none"
    >
      <rect
        x="0.5"
        y="0.5"
        width="15"
        height="11"
        rx="2"
        stroke="currentColor"
        strokeOpacity="0.55"
      />
      {zones.map((zone, index) => (
        <rect
          key={index}
          x={1.5 + 13 * zone.fx}
          y={1.5 + 9 * zone.fy}
          width={Math.max(13 * zone.fw - 1, 1.5)}
          height={Math.max(9 * zone.fh - 1, 1.5)}
          rx="0.75"
          fill="currentColor"
          fillOpacity={index === 0 ? 0.85 : 0.4}
        />
      ))}
    </svg>
  );
}

export interface OsZoomMenuProps {
  windowId: string;
  /** The real zoom traffic-light button (keeps its own click = toggleZoom). */
  children: ReactNode;
}

export function OsZoomMenu({ windowId, children }: OsZoomMenuProps) {
  const menu = useOsZoomMenu(windowId);
  const restore = OS_SNAP_COMMANDS.find(command => command.zoneId === null);
  return (
    <DropdownMenu open={menu.open} onOpenChange={menu.onOpenChange} modal={false}>
      <span
        data-slot="os-zoom-menu-anchor"
        className="relative inline-flex"
        onPointerEnter={menu.onHoverEnter}
        onPointerLeave={menu.onHoverLeave}
      >
        {children}
        <DropdownMenuTrigger
          nativeButton={false}
          render={
            <span
              aria-hidden="true"
              tabIndex={-1}
              className="pointer-events-none absolute inset-0"
            />
          }
        />
      </span>
      <DropdownMenuContent
        align="start"
        sideOffset={6}
        data-testid="os-zoom-menu"
        onPointerEnter={menu.onContentEnter}
        onPointerLeave={menu.onHoverLeave}
      >
        <DropdownMenuGroup>
          <DropdownMenuLabel>Move &amp; Resize</DropdownMenuLabel>
          <div className="grid grid-cols-4">
            {[...HALF_IDS, ...QUARTER_IDS].map(zoneId => {
              const command = OS_SNAP_COMMANDS.find(row => row.zoneId === zoneId);
              if (!command) return null;
              return (
                <DropdownMenuItem
                  key={zoneId}
                  aria-label={command.label}
                  title={command.label}
                  data-testid={`os-zoom-menu-${zoneId}`}
                  className="justify-center px-2 py-1.5"
                  onClick={() => menu.dispatchSnap(command)}
                >
                  <ZoneGlyph zones={[OS_SNAP_ZONES[zoneId]]} />
                </DropdownMenuItem>
              );
            })}
          </div>
          {menu.snapped && restore ? (
            <DropdownMenuItem
              data-testid="os-zoom-menu-restore"
              onClick={() => menu.dispatchSnap(restore)}
            >
              {restore.label}
              {restore.keys ? <DropdownMenuShortcut>{restore.keys}</DropdownMenuShortcut> : null}
            </DropdownMenuItem>
          ) : null}
        </DropdownMenuGroup>
        <DropdownMenuSeparator />
        <DropdownMenuGroup>
          <DropdownMenuLabel>Fill &amp; Arrange</DropdownMenuLabel>
          <div className="grid grid-cols-4">
            <DropdownMenuItem
              aria-label="Fill window"
              title="Fill window"
              data-testid="os-zoom-menu-fill"
              className="justify-center px-2 py-1.5"
              onClick={() => menu.dispatchFill()}
            >
              <ZoneGlyph zones={[{ fx: 0, fy: 0, fw: 1, fh: 1 }]} />
            </DropdownMenuItem>
            {OS_ARRANGE_COMMANDS.map(command => (
              <DropdownMenuItem
                key={command.preset}
                aria-label={command.label}
                title={command.label}
                data-testid={`os-zoom-menu-${command.preset}`}
                className="justify-center px-2 py-1.5"
                disabled={!menu.arrangeEnabled}
                onClick={() => menu.dispatchArrange(command.preset)}
              >
                <ZoneGlyph
                  zones={
                    command.preset === "two-up"
                      ? [OS_SNAP_ZONES.left, OS_SNAP_ZONES.right]
                      : [
                          OS_SNAP_ZONES["top-left"],
                          OS_SNAP_ZONES["top-right"],
                          OS_SNAP_ZONES["bottom-left"],
                          OS_SNAP_ZONES["bottom-right"],
                        ]
                  }
                />
              </DropdownMenuItem>
            ))}
          </div>
        </DropdownMenuGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
