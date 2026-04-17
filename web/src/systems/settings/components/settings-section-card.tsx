import type { ComponentProps, ReactNode } from "react";

import { cn } from "@/lib/utils";

interface SettingsSectionCardProps extends ComponentProps<"section"> {
  eyebrow: string;
  note?: ReactNode;
  headerAction?: ReactNode;
  children: ReactNode;
}

function SettingsSectionCard({
  eyebrow,
  note,
  headerAction,
  children,
  className,
  ...props
}: SettingsSectionCardProps) {
  return (
    <section className={cn("flex flex-col gap-4", className)} {...props}>
      <div className="flex items-center justify-between gap-4">
        <div className="flex flex-wrap items-baseline gap-3">
          <span className="font-mono text-[0.64rem] uppercase tracking-[0.2em] text-[color:var(--color-text-label)]">
            {eyebrow}
          </span>
          {note ? (
            <span className="text-xs text-[color:var(--color-text-tertiary)]">{note}</span>
          ) : null}
        </div>
        {headerAction ? <div className="shrink-0">{headerAction}</div> : null}
      </div>
      <div className="flex flex-col gap-3">{children}</div>
    </section>
  );
}

export { SettingsSectionCard };
