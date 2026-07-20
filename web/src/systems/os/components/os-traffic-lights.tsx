import { cn } from "@/lib/utils";

/**
 * The three OS window controls, matching the OpenDesign prototype geometry:
 * 12px glyphs, 7px gap. Signal tones (accent/muted/success) appear only on
 * real buttons — never on inert presentation — and focus is a distinct ring,
 * not a restatement of hover. Interactive controls keep the 12px glyph inside
 * a ≥24px target. Action association comes from the wiring in Task 04.
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
}

function Light({
  action,
  onSelect,
}: {
  action: OsTrafficLightAction;
  onSelect?: (action: OsTrafficLightAction) => void;
}) {
  const glyph =
    "size-traffic-light rounded-xs border border-line-strong bg-btn-default-fill transition-colors duration-base";

  if (!onSelect) {
    return <span aria-hidden="true" data-action={action} className={glyph} />;
  }
  return (
    <button
      type="button"
      aria-label={ACTION_LABEL[action]}
      data-action={action}
      className={cn(
        // ≥24px target around the 12px glyph; -mx-1.5 (-6px each side) so
        // adjacent button starts are 19px apart and the glyph edge gap is 7px.
        "group/traffic -mx-1.5 grid size-6 place-items-center rounded-xs focus-visible:shadow-focus-ring focus-visible:outline-none"
      )}
      onClick={() => onSelect(action)}
    >
      <span aria-hidden="true" className={cn(glyph, ACTION_TONE[action])} />
    </button>
  );
}

export function OsTrafficLights({ onSelect, className, ...props }: OsTrafficLightsProps) {
  return (
    <div
      data-slot="os-traffic-lights"
      className={cn("flex items-center gap-traffic-light-gap", className)}
      {...props}
    >
      {ACTION_ORDER.map(action => (
        <Light key={action} action={action} onSelect={onSelect} />
      ))}
    </div>
  );
}
