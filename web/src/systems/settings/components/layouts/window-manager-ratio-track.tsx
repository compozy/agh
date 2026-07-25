import { useRef, useState, type Dispatch, type SetStateAction } from "react";

import { cn } from "@agh/ui";

import type { WindowManagerConfig } from "@/systems/os";

import { usePointerDrag } from "../../hooks/use-pointer-drag";
import { fractionLabel, nearestDetent } from "../../lib/window-manager-layout-detents";
import { WINDOW_MANAGER_RANGES } from "../../lib/window-manager-snap-geometry";

const EDGE_PAD = 10;
const KEYBOARD_STEP = 0.01;

function clamp(value: number): number {
  const { min, max } = WINDOW_MANAGER_RANGES.repeatRatio;
  return Math.min(Math.max(value, min), max);
}

function round(value: number): number {
  return Math.round(value * 1_000_000) / 1_000_000;
}

/**
 * Repeat widths as stops along the axis they describe. Snapping to the same edge
 * again cycles through them, so the set is ordered and comparable at a glance —
 * which a comma-separated string of decimals never was.
 */
export function WindowManagerRatioTrack({
  draft,
  setDraft,
}: {
  draft: WindowManagerConfig;
  setDraft: Dispatch<SetStateAction<WindowManagerConfig>>;
}) {
  const trackRef = useRef<HTMLDivElement>(null);
  const [selected, setSelected] = useState<number | null>(null);
  const ratios = draft.snap.repeatRatios;
  const limits = WINDOW_MANAGER_RANGES.repeatStops;

  const setRatios = (next: number[]) =>
    setDraft(current => ({ ...current, snap: { ...current.snap, repeatRatios: next } }));

  const positionFromClientX = (clientX: number): number | null => {
    const rect = trackRef.current?.getBoundingClientRect();
    if (!rect || rect.width <= EDGE_PAD * 2) return null;
    return clamp((clientX - rect.left - EDGE_PAD) / (rect.width - EDGE_PAD * 2));
  };

  const preview = ratios[selected ?? 0] ?? 0.5;

  return (
    <div className="flex flex-col gap-3">
      <div
        className="relative h-20 overflow-hidden rounded-md border border-line-strong bg-canvas select-none"
        data-testid="window-manager-ratio-track"
        ref={trackRef}
        onPointerDown={event => {
          if (event.target !== event.currentTarget) return;
          if (ratios.length >= limits.max) return;
          const position = positionFromClientX(event.clientX);
          if (position === null) return;
          const snapped = nearestDetent(position, 0.015) ?? position;
          if (ratios.some(ratio => Math.abs(ratio - snapped) < 0.01)) return;
          setRatios([...ratios, round(snapped)]);
          setSelected(ratios.length);
        }}
      >
        <span
          aria-hidden="true"
          className="pointer-events-none absolute top-2.5 bottom-2.5 left-2.5 rounded-xxs border border-line-soft bg-surface-glaze"
          style={{ width: `calc((100% - ${EDGE_PAD * 2}px) * ${preview})` }}
        />
        <span
          aria-hidden="true"
          className="pointer-events-none absolute right-2.5 bottom-2.5 left-2.5 h-0.5 rounded-pill bg-line-strong"
        />
        <span className="pointer-events-none absolute top-2 right-3 text-form-hint text-faint">
          {ratios.length} of {limits.max} stops
        </span>

        {ratios.map((ratio, index) => (
          <RatioStop
            index={index}
            key={ratio}
            ratio={ratio}
            removable={ratios.length > limits.min}
            selected={selected === index}
            onChange={next => setRatios(ratios.map((value, at) => (at === index ? next : value)))}
            onRemove={() => {
              setRatios(ratios.filter((_, at) => at !== index));
              setSelected(null);
            }}
            onSelect={() => setSelected(index)}
            positionFromClientX={positionFromClientX}
          />
        ))}
      </div>

      <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-form-hint text-subtle">
        <span className="inline-flex items-baseline gap-1">
          Cycle
          <b className="font-mono text-form-hint font-medium tabular-nums text-fg">
            {[...ratios]
              .sort((a, b) => b - a)
              .map(ratio => fractionLabel(ratio))
              .join(" → ")}
          </b>
        </span>
        {ratios.length >= limits.max ? (
          <span className="text-warning">{limits.max} stops is the maximum</span>
        ) : null}
      </div>
    </div>
  );
}

function RatioStop({
  ratio,
  index,
  selected,
  removable,
  positionFromClientX,
  onChange,
  onRemove,
  onSelect,
}: {
  ratio: number;
  index: number;
  selected: boolean;
  removable: boolean;
  positionFromClientX: (clientX: number) => number | null;
  onChange: (next: number) => void;
  onRemove: () => void;
  onSelect: () => void;
}) {
  const drag = usePointerDrag<null>({
    onStart: () => {
      onSelect();
      return null;
    },
    onMove: event => {
      const position = positionFromClientX(event.clientX);
      if (position === null) return;
      onChange(round(nearestDetent(position, 0.015) ?? position));
    },
  });

  return (
    <button
      aria-label={`Repeat width ${Math.round(ratio * 100)} percent`}
      aria-valuemax={Math.round(WINDOW_MANAGER_RANGES.repeatRatio.max * 100)}
      aria-valuemin={Math.round(WINDOW_MANAGER_RANGES.repeatRatio.min * 100)}
      aria-valuenow={Math.round(ratio * 100)}
      className={cn(
        "group/stop absolute bottom-0.5 z-5 flex -translate-x-1/2 cursor-ew-resize flex-col items-center gap-0.5 px-1",
        "text-muted focus-visible:outline-none focus-visible:shadow-focus-ring",
        selected && "text-fg-strong",
        drag.dragging && "text-accent-strong"
      )}
      data-testid={`window-manager-ratio-stop-${index}`}
      role="slider"
      style={{ left: `calc(${EDGE_PAD}px + (100% - ${EDGE_PAD * 2}px) * ${ratio})` }}
      type="button"
      onKeyDown={event => {
        if ((event.key === "Backspace" || event.key === "Delete") && removable) {
          event.preventDefault();
          onRemove();
          return;
        }
        const forward = event.key === "ArrowRight";
        const backward = event.key === "ArrowLeft";
        if (!forward && !backward) return;
        event.preventDefault();
        onChange(round(clamp(ratio + (forward ? KEYBOARD_STEP : -KEYBOARD_STEP))));
      }}
      onPointerDown={drag.onPointerDown}
    >
      <b className="font-mono text-form-hint font-medium tabular-nums">{fractionLabel(ratio)}</b>
      <span
        aria-hidden="true"
        className={cn(
          "block h-3.5 w-0.5 rounded-pill bg-subtle transition-colors duration-base ease-out",
          "group-hover/stop:bg-accent group-focus-visible/stop:bg-accent",
          drag.dragging && "bg-accent"
        )}
      />
    </button>
  );
}
