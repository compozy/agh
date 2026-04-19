import type { ComponentProps, ReactNode } from "react";

import { cn } from "@agh/ui";

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
    <section
      className={cn(
        "flex flex-col gap-5 border-t border-[color:var(--color-divider)] pt-6 first:border-t-0 first:pt-0",
        className
      )}
      {...props}
    >
      <div className="flex flex-col gap-3 md:flex-row md:items-start md:justify-between">
        <div className="flex min-w-0 flex-col gap-2">
          <span className="font-mono text-[11px] font-semibold uppercase tracking-[0.2em] text-[color:var(--color-text-label)]">
            {eyebrow}
          </span>
          {note ? (
            <span className="max-w-[38rem] text-xs leading-5 text-[color:var(--color-text-secondary)]">
              {note}
            </span>
          ) : null}
        </div>
        {headerAction ? <div className="shrink-0 self-start">{headerAction}</div> : null}
      </div>
      <div className="flex flex-col gap-4">{children}</div>
    </section>
  );
}

export { SettingsSectionCard };
