import type { ReactNode } from "react";

import { Metric, cn } from "@agh/ui";

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
    <div className={cn("grid gap-3 sm:grid-cols-2 xl:grid-cols-4", className)}>{children}</div>
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
    <Metric
      label={label}
      value={value}
      subtext={detail}
      className={className}
      data-testid={dataTestId ?? testId}
    />
  );
}

export { SettingsStatGrid, SettingsStatItem };
