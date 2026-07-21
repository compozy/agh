import { useRef, useSyncExternalStore } from "react";

import { FieldRow, PageShell, Switch, cn } from "@agh/ui";

import { SettingsTopbarPublisher } from "@/systems/settings";

import { useDesktop } from "../../hooks/use-desktop";
import { useOsShell } from "../../hooks/use-os-shell";
import { getSystemReducedMotion, subscribeSystemReducedMotion } from "../../lib/reduced-motion";
import type { OsWallpaper } from "../../lib/os-types";

const WALLPAPERS: Array<{ id: OsWallpaper; label: string }> = [
  { id: "ember", label: "Ember" },
  { id: "mesh", label: "Mesh" },
  { id: "carbon", label: "Carbon" },
];

const THUMB_BACKGROUND: Record<OsWallpaper, string> = {
  ember: "var(--wallpaper-thumb-ember)",
  mesh: "var(--wallpaper-thumb-mesh)",
  carbon: "var(--wallpaper-thumb-carbon)",
};

/**
 * Wallpaper choice as an APG radio group: one tab stop, arrows move AND
 * select (automatic activation — switching wallpapers is cheap and instant).
 */
function WallpaperPicker({
  value,
  onChange,
}: {
  value: OsWallpaper;
  onChange: (wallpaper: OsWallpaper) => void;
}) {
  const groupRef = useRef<HTMLDivElement>(null);

  const moveSelection = (offset: number) => {
    const index = WALLPAPERS.findIndex(option => option.id === value);
    const next = WALLPAPERS[(index + offset + WALLPAPERS.length) % WALLPAPERS.length];
    onChange(next.id);
    groupRef.current
      ?.querySelector<HTMLButtonElement>(`[data-wallpaper-option="${next.id}"]`)
      ?.focus();
  };

  const handleKeyDown = (event: React.KeyboardEvent) => {
    if (event.key === "ArrowRight" || event.key === "ArrowDown") {
      event.preventDefault();
      moveSelection(1);
    } else if (event.key === "ArrowLeft" || event.key === "ArrowUp") {
      event.preventDefault();
      moveSelection(-1);
    }
  };

  return (
    <div
      ref={groupRef}
      role="radiogroup"
      aria-label="Wallpaper"
      className="flex flex-wrap gap-3"
      onKeyDown={handleKeyDown}
    >
      {WALLPAPERS.map(option => {
        const selected = option.id === value;
        return (
          <button
            key={option.id}
            type="button"
            role="radio"
            aria-checked={selected}
            tabIndex={selected ? 0 : -1}
            data-wallpaper-option={option.id}
            data-testid={`os-wallpaper-option-${option.id}`}
            className={cn(
              "group flex w-28 flex-col overflow-hidden rounded-md border text-left",
              "transition-colors duration-base",
              "focus-visible:shadow-focus-ring focus-visible:outline-none",
              selected ? "border-accent" : "border-line-strong hover:border-line-focus"
            )}
            onClick={() => onChange(option.id)}
          >
            <span
              aria-hidden="true"
              className="block h-14 w-full border-b border-line"
              style={{ backgroundImage: THUMB_BACKGROUND[option.id] }}
            />
            <span
              className={cn(
                "flex items-center justify-between px-2 py-1.5 text-small-body",
                selected ? "font-semibold text-fg-strong" : "text-muted"
              )}
            >
              {option.label}
              {selected ? (
                <span aria-hidden="true" className="size-1.5 rounded-full bg-accent" />
              ) : null}
            </span>
          </button>
        );
      })}
    </div>
  );
}

/**
 * The Appearance pane (US-015): wallpaper per space, dock magnification, and
 * the in-product reduced-motion preference — all desktop-doc state persisted
 * with the active space and applied live. The system reduced-motion
 * preference always wins over the toggle (US-015.EC-1).
 */
export function AppearanceSettingsPane() {
  const { store } = useOsShell();
  const wallpaper = useDesktop(state => state.wallpaper);
  const dockMagnify = useDesktop(state => state.dockMagnify);
  const reduceMotion = useDesktop(state => state.reduceMotion);
  const systemReducedMotion = useSyncExternalStore(
    subscribeSystemReducedMotion,
    getSystemReducedMotion,
    () => false
  );

  return (
    <PageShell slug="appearance" head={<SettingsTopbarPublisher slug="appearance" />}>
      <div className="flex max-w-2xl flex-col gap-8" data-testid="os-appearance-pane">
        <FieldRow
          label="Wallpaper"
          description="Backdrop for this workspace's space. Each space keeps its own."
          control={
            <WallpaperPicker
              value={wallpaper}
              onChange={next => store.getState().setWallpaper(next)}
            />
          }
        />
        <FieldRow
          label="Dock magnification"
          description="Dock icons lift and scale under the pointer."
          htmlFor="os-appearance-magnify"
          control={
            <Switch
              id="os-appearance-magnify"
              data-testid="os-appearance-magnify"
              checked={dockMagnify}
              onCheckedChange={checked => store.getState().setDockMagnify(checked === true)}
            />
          }
        />
        <FieldRow
          label="Reduce motion"
          description={
            systemReducedMotion
              ? "Your system already prefers reduced motion — that preference wins while it is on."
              : "Window, dock, and minimize animations become instant."
          }
          htmlFor="os-appearance-reduce-motion"
          control={
            <Switch
              id="os-appearance-reduce-motion"
              data-testid="os-appearance-reduce-motion"
              checked={reduceMotion}
              onCheckedChange={checked => store.getState().setReduceMotion(checked === true)}
            />
          }
        />
      </div>
    </PageShell>
  );
}
