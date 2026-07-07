import type { LoopRunMeter, LoopMeterTone } from "../../lib/loop-run-meters";

interface LoopRunMetersProps {
  meters: LoopRunMeter[];
}

/** Bar fill class by tone — warn/danger only tint NEAR the ceiling (§4.4). */
const BAR_FILL_CLASS: Record<LoopMeterTone, string> = {
  neutral: "bg-subtle",
  warn: "bg-warning",
  danger: "bg-danger",
};

function LoopRunMeterCard({ meter }: { meter: LoopRunMeter }) {
  return (
    <div
      className="rounded-md border border-line-soft bg-canvas-tint px-3 py-2.5"
      data-testid={`loop-run-meter-${meter.key}`}
      data-tone={meter.tone}
      data-display-only={meter.displayOnly ? "true" : undefined}
    >
      <div className="mb-2 flex items-baseline justify-between gap-1.5">
        <span className="eyebrow text-faint">{meter.label}</span>
        <span className="font-mono text-[13px] font-semibold tabular-nums text-fg-strong">
          {meter.value}
          {meter.max ? (
            <span className="text-[11px] font-medium text-faint"> {meter.max}</span>
          ) : null}
        </span>
      </div>
      {meter.percent !== null ? (
        <div className="h-[3px] overflow-hidden rounded-full bg-fg-strong/10">
          <div
            className={`h-full rounded-full transition-[width] duration-700 ${BAR_FILL_CLASS[meter.tone]}`}
            style={{ width: `${meter.percent}%` }}
            data-testid={`loop-run-meter-bar-${meter.key}`}
          />
        </div>
      ) : null}
      <div className="mt-2 flex flex-wrap items-center gap-1.5">
        {meter.tags.map(tag => (
          <span
            key={tag}
            className="rounded-xs bg-badge-fill px-1.5 py-0.5 font-mono text-[9px] text-subtle"
          >
            {tag}
          </span>
        ))}
      </div>
    </div>
  );
}

/**
 * The 5 run-page meters (Attempts / Tokens / Wall / Cost / Breadth, §4.4). Each bar
 * warn-tints only near its ceiling; Cost is display-only with no bar and no cap
 * control (truthful UI — AGH enforces no USD cap, ADR-017 §9.5.2).
 */
export function LoopRunMeters({ meters }: LoopRunMetersProps) {
  return (
    <div
      className="grid grid-cols-2 gap-2.5 sm:grid-cols-3 xl:grid-cols-5"
      data-testid="loop-run-meters"
    >
      {meters.map(meter => (
        <LoopRunMeterCard key={meter.key} meter={meter} />
      ))}
    </div>
  );
}
