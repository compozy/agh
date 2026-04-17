import type { ReactNode } from "react";

import { StatusDot } from "@/components/design-system";

interface SettingsStatusLineProps {
  daemonAvailable: boolean;
  daemonLabel?: string;
  items: ReactNode[];
  "data-testid"?: string;
}

function SettingsStatusLine({
  daemonAvailable,
  daemonLabel,
  items,
  "data-testid": testId,
}: SettingsStatusLineProps) {
  const label = daemonLabel ?? (daemonAvailable ? "daemon running" : "daemon unavailable");
  return (
    <div className="flex flex-wrap items-center gap-x-4 gap-y-1" data-testid={testId}>
      <span className="flex items-center gap-2">
        <StatusDot tone={daemonAvailable ? "green" : "amber"} />
        <span>{label}</span>
      </span>
      {items.map((item, index) => (
        // biome-ignore lint/suspicious/noArrayIndexKey: ordered static items from caller
        <span key={index} className="flex items-center gap-1">
          <span aria-hidden="true" className="text-[color:var(--color-text-tertiary)]">
            ·
          </span>
          {item}
        </span>
      ))}
    </div>
  );
}

export { SettingsStatusLine };
