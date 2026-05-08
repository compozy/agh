import type { ReactNode } from "react";

import { cn } from "@agh/ui";

interface SettingsCollectionHeaderProps {
  eyebrow: string;
  summary?: ReactNode;
  action?: ReactNode;
  className?: string;
  "data-testid"?: string;
}

function SettingsCollectionHeader({
  eyebrow,
  summary,
  action,
  className,
  "data-testid": testId,
}: SettingsCollectionHeaderProps) {
  return (
    <div
      className={cn("flex flex-wrap items-center justify-between gap-4", className)}
      data-testid={testId}
    >
      <div className="flex flex-wrap items-baseline gap-3">
        <span className="font-mono text-eyebrow font-semibold uppercase tracking-mono text-(--color-text-label)">
          {eyebrow}
        </span>
        {summary ? <span className="text-xs text-(--color-text-tertiary)">{summary}</span> : null}
      </div>
      {action ? <div className="shrink-0">{action}</div> : null}
    </div>
  );
}

export { SettingsCollectionHeader };
