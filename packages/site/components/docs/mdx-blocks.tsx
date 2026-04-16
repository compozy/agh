import Link from "next/link";
import { ArrowRight } from "lucide-react";
import type { ReactNode } from "react";

interface RouteRowProps {
  href: string;
  label: string;
  title: string;
  description: string;
  meta?: string;
}

export function OperatorNote({
  label = "Operator note",
  children,
}: {
  label?: string;
  children: ReactNode;
}) {
  return (
    <aside className="not-prose rounded-[28px] border border-(--color-divider) bg-(--color-surface) px-5 py-5 md:px-6">
      <p className="font-mono text-[10px] font-semibold uppercase tracking-[0.14em] text-(--color-accent)">
        {label}
      </p>
      <div className="mt-3 text-[0.98rem] leading-7 text-(--color-text-secondary)">{children}</div>
    </aside>
  );
}

export function RouteList({ children }: { children: ReactNode }) {
  return (
    <div className="not-prose overflow-hidden rounded-[28px] border border-(--color-divider) bg-(--color-surface)">
      {children}
    </div>
  );
}

export function RouteRow({ href, label, title, description, meta }: RouteRowProps) {
  return (
    <Link
      href={href}
      className="group grid gap-3 border-t border-(--color-divider) px-5 py-5 transition-colors first:border-t-0 hover:bg-[rgba(44,44,46,0.55)] md:grid-cols-[132px_minmax(0,1fr)_150px] md:items-center md:px-6"
    >
      <p className="font-mono text-[10px] font-semibold uppercase tracking-[0.14em] text-(--color-text-tertiary)">
        {label}
      </p>

      <div className="min-w-0">
        <p className="text-lg font-semibold tracking-[-0.02em] text-(--color-text-primary)">
          {title}
        </p>
        <p className="mt-1 text-sm leading-6 text-(--color-text-secondary)">{description}</p>
      </div>

      <div className="flex items-center gap-2 text-sm text-(--color-text-secondary) md:justify-end">
        {meta ? (
          <span className="hidden md:inline">{meta}</span>
        ) : (
          <span className="hidden md:inline">Open section</span>
        )}
        <ArrowRight className="size-4 text-(--color-accent) transition-transform group-hover:translate-x-0.5" />
      </div>
    </Link>
  );
}
