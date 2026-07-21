import { cn } from "@/lib/utils";

/**
 * The three OS window controls, matching the OpenDesign prototype geometry:
 * 12px glyphs, 7px gap. Signal tones (accent/muted/success) appear only on
 * real buttons — never on inert presentation — and focus is a distinct ring,
 * not a restatement of hover. Interactive controls keep the 12px glyph inside
 * a ≥24px target. Action association comes from the wiring in Task 04.
 *
 * Compact (<960px, os-v2.css mobile block): the zoom control disappears
 * (meaningless in a stack), glyphs grow to 15px, gaps to 12px, and each
 * control gets a 36px touch target.
 */
export type OsTrafficLightAction = "close" | "minimize" | "zoom";

const ACTION_LABEL: Record<OsTrafficLightAction, string> = {
  close: "Close window",
  minimize: "Minimize window",
  zoom: "Zoom window",
};

const ACTION_ORDER: OsTrafficLightAction[] = ["close", "minimize", "zoom"];

// Tone applies to the 12px glyph on button hover/focus, per the prototype.
const ACTION_TONE: Record<OsTrafficLightAction, string> = {
  close:
    "group-hover/traffic:bg-accent group-hover/traffic:border-accent group-focus-visible/traffic:bg-accent group-focus-visible/traffic:border-accent",
  minimize:
    "group-hover/traffic:bg-muted group-hover/traffic:border-muted group-focus-visible/traffic:bg-muted group-focus-visible/traffic:border-muted",
  zoom: "group-hover/traffic:bg-success group-hover/traffic:border-success group-focus-visible/traffic:bg-success group-focus-visible/traffic:border-success",
};

export interface OsTrafficLightsProps extends Omit<React.ComponentProps<"div">, "onSelect"> {
  /**
   * Called with the control's action when activated. When omitted the controls
   * render as non-interactive presentation (truthful chrome, no dead buttons).
   */
  onSelect?: (action: OsTrafficLightAction) => void;
  /** Compact presentation: zoom hidden, enlarged glyphs and hit areas. */
  compact?: boolean;
}

function Light({
  action,
  onSelect,
  compact,
}: {
  action: OsTrafficLightAction;
  onSelect?: (action: OsTrafficLightAction) => void;
  compact: boolean;
}) {
  const glyph = cn(
    "rounded-xs border border-line-strong bg-btn-default-fill transition-colors duration-base",
    compact ? "size-traffic-light-compact" : "size-traffic-light"
  );

  if (!onSelect) {
    return <span aria-hidden="true" data-action={action} className={glyph} />;
  }
  return (
    <button
      type="button"
      aria-label={ACTION_LABEL[action]}
      data-action={action}
      className={cn(
        "group/traffic grid place-items-center rounded-xs focus-visible:shadow-focus-ring focus-visible:outline-none",
        compact
          ? // Prototype mobile hit expansion (os-v2.css `.wc::after{inset:-9px}`):
            // a 15px glyph inside a 33px effective target (≥24px WCAG floor).
            "relative size-traffic-light-compact after:absolute after:-inset-[9px] after:content-['']"
          : // ≥24px target around the 12px glyph; -mx-1.5 (-6px each side) so
            // adjacent button starts are 19px apart and the glyph edge gap is 7px.
            "-mx-1.5 size-6"
      )}
      onClick={() => onSelect(action)}
    >
      <span aria-hidden="true" className={cn(glyph, ACTION_TONE[action])} />
    </button>
  );
}

export function OsTrafficLights({
  onSelect,
  compact = false,
  className,
  ...props
}: OsTrafficLightsProps) {
  const actions = compact ? ACTION_ORDER.filter(action => action !== "zoom") : ACTION_ORDER;
  return (
    <div
      data-slot="os-traffic-lights"
      data-presentation={compact ? "compact" : undefined}
      className={cn(
        "flex items-center",
        // Prototype mobile: `.win-controls{gap:12px}` between the 15px glyphs.
        compact ? "gap-3" : "gap-traffic-light-gap",
        className
      )}
      {...props}
    >
      {actions.map(action => (
        <Light key={action} action={action} onSelect={onSelect} compact={compact} />
      ))}
    </div>
  );
}
