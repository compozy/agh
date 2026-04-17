import type { ReactNode } from "react";

import { cn } from "@/lib/utils";

interface SettingsFieldRowProps {
  label: string;
  description?: ReactNode;
  hint?: ReactNode;
  control: ReactNode;
  className?: string;
  "data-testid"?: string;
}

function SettingsFieldRow({
  label,
  description,
  hint,
  control,
  className,
  "data-testid": testId,
}: SettingsFieldRowProps) {
  return (
    <div
      className={cn(
        "flex items-start justify-between gap-6 border-t border-[color:var(--color-divider)] pt-4 first:border-t-0 first:pt-0",
        className
      )}
      data-testid={testId}
    >
      <div className="flex min-w-0 flex-col gap-1">
        <span className="text-sm font-medium text-[color:var(--color-text-primary)]">{label}</span>
        {description ? (
          <span className="text-xs text-[color:var(--color-text-tertiary)]">{description}</span>
        ) : null}
      </div>
      <div className="flex shrink-0 items-center gap-3">
        {control}
        {hint ? (
          <span className="hidden font-mono text-[0.58rem] uppercase tracking-[0.18em] text-[color:var(--color-text-label)] md:inline">
            {hint}
          </span>
        ) : null}
      </div>
    </div>
  );
}

export { SettingsFieldRow };
