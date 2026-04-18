import type { ReactNode } from "react";

import { cn } from "@/lib/utils";

interface SettingsStatGridProps {
  children: ReactNode;
  className?: string;
}

interface SettingsStatItemProps {
  label: string;
  value: ReactNode;
  detail?: ReactNode;
  className?: string;
  testId?: string;
  "data-testid"?: string;
}

function SettingsStatGrid({ children, className }: SettingsStatGridProps) {
  return (
    <div
      className={cn(
        "grid gap-3 sm:grid-cols-2 xl:grid-cols-4",
        "[&>*]:min-h-[5.5rem] [&>*]:rounded-lg [&>*]:border [&>*]:border-[color:var(--color-divider)]",
        "[&>*]:bg-[color:var(--color-surface-elevated)] [&>*]:px-4 [&>*]:py-3",
        className
      )}
    >
      {children}
    </div>
  );
}

function SettingsStatItem({
  label,
  value,
  detail,
  className,
  testId,
  "data-testid": dataTestId,
}: SettingsStatItemProps) {
  return (
    <div
      className={cn("flex flex-col justify-between gap-3", className)}
      data-testid={dataTestId ?? testId}
    >
      <span className="font-mono text-[0.6rem] uppercase tracking-[0.18em] text-[color:var(--color-text-label)]">
        {label}
      </span>
      <div className="flex flex-col gap-1">
        <span className="text-base font-medium tracking-[-0.01em] text-[color:var(--color-text-primary)]">
          {value}
        </span>
        {detail ? (
          <span className="text-xs text-[color:var(--color-text-tertiary)]">{detail}</span>
        ) : null}
      </div>
    </div>
  );
}

export { SettingsStatGrid, SettingsStatItem };
