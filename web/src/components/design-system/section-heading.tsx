import type { ComponentProps, ReactNode } from "react";

import { cn } from "@/lib/utils";

interface SectionHeadingProps extends ComponentProps<"header"> {
  action?: ReactNode;
  description?: string;
  eyebrow?: string;
  title: string;
}

function SectionHeading({
  action,
  className,
  description,
  eyebrow,
  title,
  ...props
}: SectionHeadingProps) {
  return (
    <header
      className={cn(
        "flex flex-col gap-5 lg:flex-row lg:items-end lg:justify-between lg:gap-8",
        className
      )}
      {...props}
    >
      <div className="max-w-3xl space-y-3">
        {eyebrow ? (
          <p className="font-mono text-[0.68rem] uppercase tracking-[0.18em] text-[color:var(--ds-text-mono)]">
            {eyebrow}
          </p>
        ) : null}
        <h1 className="max-w-4xl text-balance font-display text-4xl leading-[1.08] font-semibold tracking-[-0.04em] text-[color:var(--ds-text-primary)] sm:text-5xl lg:text-6xl">
          {title}
        </h1>
        {description ? (
          <p className="max-w-2xl text-base leading-7 text-[color:var(--ds-text-secondary)] sm:text-lg">
            {description}
          </p>
        ) : null}
      </div>
      {action ? <div className="shrink-0">{action}</div> : null}
    </header>
  );
}

export { SectionHeading };
