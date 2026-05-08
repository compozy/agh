import type { ReactNode } from "react";
import Link from "next/link";
import { ArrowUpRight } from "lucide-react";
import { cn } from "@agh/ui/lib/utils";

interface FeatureCardProps {
  eyebrow?: string;
  title: ReactNode;
  description: ReactNode;
  icon?: ReactNode;
  /** Optional doc path that backs this claim. Renders as a subtle "source" link. */
  cite?: { href: string; label?: string };
  className?: string;
}

export function FeatureCard({
  eyebrow,
  title,
  description,
  icon,
  cite,
  className,
}: FeatureCardProps) {
  return (
    <article
      className={cn(
        "flex flex-col gap-3 rounded-(--radius-diagram) border border-(--color-divider) bg-(--color-surface) p-6 transition-colors hover:border-accent/40",
        className
      )}
    >
      {icon ? (
        <div className="flex h-10 w-10 items-center justify-center rounded-icon-well bg-(--color-surface-elevated) text-accent">
          {icon}
        </div>
      ) : null}
      {eyebrow ? (
        <p className="font-mono text-badge font-semibold uppercase tracking-mono text-accent">
          {eyebrow}
        </p>
      ) : null}
      <h3 className="text-base font-medium leading-snug text-(--color-text-primary)">{title}</h3>
      <p className="text-sm leading-relaxed text-(--color-text-secondary)">{description}</p>
      {cite ? (
        <Link
          href={cite.href}
          className="mt-auto inline-flex items-center gap-1 pt-2 font-mono text-badge uppercase tracking-mono text-(--color-text-tertiary) transition-colors hover:text-accent"
        >
          {cite.label ?? "source"}
          <ArrowUpRight aria-hidden className="h-3 w-3" />
        </Link>
      ) : null}
    </article>
  );
}
