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
        "grid gap-3 border-t border-[color:var(--color-divider)] pt-5 first:border-t-0 first:pt-0 md:grid-cols-[minmax(0,17rem)_minmax(0,1fr)] md:gap-x-8 md:gap-y-0",
        className
      )}
      data-testid={testId}
    >
      <div className="flex min-w-0 flex-col gap-1.5">
        <div className="flex flex-wrap items-center gap-2">
          <span className="text-sm font-medium text-[color:var(--color-text-primary)]">
            {label}
          </span>
          {hint ? (
            <span className="font-mono text-[0.58rem] uppercase tracking-[0.18em] text-[color:var(--color-text-label)] md:hidden">
              {hint}
            </span>
          ) : null}
        </div>
        {description ? (
          <span className="max-w-[34rem] text-xs leading-5 text-[color:var(--color-text-secondary)]">
            {description}
          </span>
        ) : null}
      </div>
      <div className="flex min-w-0 items-start md:justify-self-start">
        <div className="flex min-w-0 max-w-full flex-wrap items-center gap-3 [&_input]:max-w-full [&_select]:max-w-full">
          {control}
          {hint ? (
            <span className="hidden font-mono text-[0.58rem] uppercase tracking-[0.18em] text-[color:var(--color-text-label)] md:inline">
              {hint}
            </span>
          ) : null}
        </div>
      </div>
    </div>
  );
}

export { SettingsFieldRow };
