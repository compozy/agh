import type { ReactNode } from "react";
import type { LucideIcon } from "lucide-react";

import { cn, Eyebrow } from "@agh/ui";

interface FlowStepProps {
  icon: LucideIcon;
  kicker: string;
  title: ReactNode;
  subtitle?: ReactNode;
  active?: boolean;
  last?: boolean;
  children: ReactNode;
}

/** One node on the trigger flow spine: numbered icon well + kicker/title/subtitle + body. */
export function FlowStep({
  icon: Icon,
  kicker,
  title,
  subtitle,
  active,
  last,
  children,
}: FlowStepProps) {
  return (
    <div className="flex gap-3.5">
      <div className="flex flex-col items-center">
        <div
          className={cn(
            "flex size-7 shrink-0 items-center justify-center rounded-full border",
            active
              ? "border-accent-dim bg-accent-tint text-accent-strong"
              : "border-line bg-canvas-soft text-muted"
          )}
        >
          <Icon aria-hidden="true" className="size-4" />
        </div>
        {last ? null : (
          <div className="mt-1 w-px flex-1 bg-gradient-to-b from-line-strong to-line-soft" />
        )}
      </div>
      <div className={cn("min-w-0 flex-1", last ? "pb-1" : "pb-6")}>
        <Eyebrow className="text-subtle">{kicker}</Eyebrow>
        <div className="mt-0.5 text-card-title font-semibold tracking-tight text-fg-strong">
          {title}
        </div>
        {subtitle ? <p className="mt-0.5 text-small-body text-muted">{subtitle}</p> : null}
        <div className="mt-3">{children}</div>
      </div>
    </div>
  );
}
