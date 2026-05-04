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
        "flex flex-col gap-4 border-t border-[color:var(--color-divider)] pt-5 first:border-t-0 first:pt-0 md:gap-5 md:pt-6",
        className
      )}
      {...props}
    >
      <div className="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
        <div className="flex min-w-0 flex-col gap-2">
          <span className="font-mono text-[11px] font-semibold uppercase tracking-[var(--tracking-mono)] text-[color:var(--color-text-label)]">
            {eyebrow}
          </span>
          {note ? (
            <span className="max-w-[38rem] text-xs leading-5 text-[color:var(--color-text-secondary)]">
              {note}
            </span>
          ) : null}
        </div>
        {headerAction ? (
          <div className="w-full self-start lg:w-auto lg:shrink-0">{headerAction}</div>
        ) : null}
      </div>
      <div className="flex flex-col gap-4">{children}</div>
    </section>
  );
}

export { SettingsSectionCard };
