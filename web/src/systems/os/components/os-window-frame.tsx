import { Topbar, TopbarSlotProvider } from "@agh/ui";

import { cn } from "@/lib/utils";

import { OsTrafficLights, type OsTrafficLightAction } from "./os-traffic-lights";

/**
 * Floating window frame: the shell chrome around one app's route subtree.
 * The window head IS the route's `<Topbar>` at its 48px three-zone anatomy —
 * traffic lights injected leading, route identity centered, published route
 * actions trailing — wrapped in a per-window `<TopbarSlotProvider>` so the
 * app's routes publish actions into this window's head via `useTopbarSlot`.
 * Frame depth (border + cast shadow) is the sanctioned shell carve-out;
 * window-body content stays on the flat ramp/hairline model.
 *
 * Presentational only — drag, z-order, and focus come from the window
 * manager in Task 04. `focused` selects the focused/unfocused depth and
 * head-dim states; it is not interactive by itself.
 */
export interface OsWindowFrameProps extends Omit<React.ComponentProps<"section">, "title"> {
  /** Current page identity in the breadcrumb (`agh / <title>`). */
  title: React.ReactNode;
  /** Workspace/root crumb shown before the page title. Defaults to `agh`. */
  rootCrumb?: React.ReactNode;
  /** Focused (sharp border, cast shadow) vs unfocused (dimmed head, lighter shadow). */
  focused?: boolean;
  /** Traffic-light activation. Omit to render the controls as presentation. */
  onTrafficLight?: (action: OsTrafficLightAction) => void;
}

export function OsWindowFrame({
  title,
  rootCrumb = "agh",
  focused = true,
  onTrafficLight,
  className,
  children,
  ...props
}: OsWindowFrameProps) {
  return (
    <section
      data-slot="os-window-frame"
      data-focused={focused ? "" : undefined}
      className={cn(
        "flex min-h-0 flex-col overflow-hidden rounded-window border bg-canvas",
        focused ? "border-line-focus shadow-window" : "border-line-strong shadow-window-unfocused",
        className
      )}
      {...props}
    >
      <TopbarSlotProvider>
        <Topbar
          data-slot="os-window-head"
          leading={<OsTrafficLights onSelect={onTrafficLight} />}
          breadcrumb={
            <span className="flex items-center gap-1.5">
              <span className="font-mono text-eyebrow text-subtle">{rootCrumb}</span>
              <span aria-hidden="true" className="text-eyebrow text-faint">
                /
              </span>
            </span>
          }
          title={title}
          className={cn(
            // Unfocused dims head foreground only (prototype `.win:not(.is-focused)
            // .win-head{color:subtle}`) — background and hairline border stay. Title
            // is targeted directly; zones use local opacity (explicit-color
            // descendants would ignore a wrapper's inherited color).
            !focused &&
              "[&_[data-slot=topbar-title]]:text-subtle [&_[data-slot=topbar-breadcrumb]]:opacity-60 [&_[data-slot=topbar-trailing]]:opacity-60"
          )}
        />
        <div data-slot="os-window-body" className="min-h-0 flex-1 overflow-auto bg-canvas">
          {children}
        </div>
      </TopbarSlotProvider>
    </section>
  );
}
