import Link from "next/link";
import { MonoEyebrow } from "./mono-eyebrow";

export interface BlogEmptyStateAction {
  href: string;
  label: string;
}

export interface BlogEmptyStateProps {
  eyebrow: string;
  title: string;
  description: string;
  primaryAction: BlogEmptyStateAction;
  secondaryAction?: BlogEmptyStateAction;
}

export function BlogEmptyState({
  eyebrow,
  title,
  description,
  primaryAction,
  secondaryAction,
}: BlogEmptyStateProps) {
  return (
    <section className="rounded-xl border border-(--color-divider) bg-(--color-surface) p-6">
      <MonoEyebrow tone="accent">{eyebrow}</MonoEyebrow>
      <h2 className="mt-4 max-w-[26ch] font-sans text-[clamp(1.45rem,3vw,1.9rem)] font-semibold leading-[1.1] tracking-[-0.025em] text-(--color-text-primary)">
        {title}
      </h2>
      <p className="mt-4 max-w-[58ch] text-sm leading-[1.7] text-(--color-text-secondary)">
        {description}
      </p>
      <div className="mt-6 flex flex-wrap gap-3">
        <Link
          href={primaryAction.href}
          className="inline-flex h-9 items-center justify-center rounded-lg border border-(--color-divider) px-3.5 font-sans text-sm font-medium text-(--color-text-primary) transition-colors hover:bg-(--color-hover)"
        >
          {primaryAction.label}
        </Link>
        {secondaryAction && (
          <Link
            href={secondaryAction.href}
            className="inline-flex h-9 items-center justify-center rounded-lg border border-(--color-divider) px-3.5 font-sans text-sm font-medium text-(--color-text-primary) transition-colors hover:bg-(--color-hover)"
          >
            {secondaryAction.label}
          </Link>
        )}
      </div>
    </section>
  );
}
