import { Eyebrow } from "@agh/ui";

import { SettingsNumberInput } from "@/systems/settings";
import { formatBytes } from "./-observability-format";

interface NumberFieldProps {
  label: string;
  testId: string;
  value: number;
  suffix?: string;
  errorMessage?: string;
  onValidityChange: (message: string | null) => void;
  onChange: (value: number) => void;
}

export function ObservabilityNumberField({
  label,
  testId,
  value,
  suffix,
  errorMessage,
  onValidityChange,
  onChange,
}: NumberFieldProps) {
  return (
    <div className="flex flex-col gap-1">
      <Eyebrow className="text-muted">{label}</Eyebrow>
      <div className="flex items-center gap-2">
        <SettingsNumberInput
          className="w-full"
          min={0}
          data-testid={testId}
          value={value}
          onValidityChange={onValidityChange}
          onValueChange={onChange}
        />
        {suffix ? <Eyebrow className="text-muted">{suffix}</Eyebrow> : null}
      </div>
      {errorMessage ? <span className="text-xs text-danger">{errorMessage}</span> : null}
    </div>
  );
}

export function UsageBreakdown({
  globalBytes,
  sessionBytes,
  cap,
}: {
  globalBytes: number;
  sessionBytes: number;
  cap: number;
}) {
  const total = Math.max(1, cap);
  const globalPct = Math.min(100, (globalBytes / total) * 100);
  const sessionPct = Math.min(100, (sessionBytes / total) * 100);

  return (
    <div className="flex flex-col gap-2" data-testid="settings-page-observability-usage-breakdown">
      <div className="flex items-center justify-between text-xs text-subtle">
        <Eyebrow className="text-muted">Usage breakdown</Eyebrow>
      </div>
      <div className="relative h-2 w-full overflow-hidden rounded-full bg-canvas-soft">
        <div
          className="absolute inset-y-0 left-0 bg-accent-tint-strong"
          style={{ width: `${globalPct}%` }}
          data-testid="settings-page-observability-usage-bar-global"
        />
        <div
          className="absolute inset-y-0 bg-info-tint"
          style={{ left: `${globalPct}%`, width: `${sessionPct}%` }}
          data-testid="settings-page-observability-usage-bar-sessions"
        />
      </div>
      <div className="flex flex-wrap gap-4 text-xs text-muted">
        <span className="inline-flex items-center gap-1.5">
          <span aria-hidden="true" className="size-2 rounded-full bg-accent-tint-strong" />
          global DB {formatBytes(globalBytes)}
        </span>
        <span className="inline-flex items-center gap-1.5">
          <span aria-hidden="true" className="size-2 rounded-full bg-info-tint" />
          session DB {formatBytes(sessionBytes)}
        </span>
      </div>
    </div>
  );
}
