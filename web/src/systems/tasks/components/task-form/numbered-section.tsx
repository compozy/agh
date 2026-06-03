import type { ReactNode } from "react";

import { cn } from "@agh/ui";

interface NumberedSectionProps {
  /** Mono section index (e.g. `01`). */
  index: string;
  title: ReactNode;
  subtitle?: ReactNode;
  /** Suppress the top hairline + top padding on the first section in a form. */
  first?: boolean;
  children: ReactNode;
}

/**
 * Numbered form section — a baseline-aligned header (mono index, title,
 * subtitle) followed by the section body. Every section except the first
 * carries a top hairline + spacing, matching the `.sec` grammar in the
 * task-create redesign.
 */
export function NumberedSection({ index, title, subtitle, first, children }: NumberedSectionProps) {
  return (
    <section className={cn(!first && "mt-7 border-t border-line pt-5")}>
      <div className="flex items-baseline gap-2.5">
        <span className="font-mono text-form-hint font-semibold text-faint">{index}</span>
        <span className="text-small-body font-semibold tracking-tight text-fg-strong">{title}</span>
        {subtitle ? <span className="text-form-hint text-subtle">{subtitle}</span> : null}
      </div>
      <div className="mt-3">{children}</div>
    </section>
  );
}
