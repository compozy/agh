import type { Release } from "#site/content";
import Link from "next/link";
import { formatDateCompact } from "./format";
import { MonoBadge } from "./mono-badge";
import { MonoEyebrow } from "./mono-eyebrow";

export interface ChangelogRailProps {
  releases: Release[];
}

export function ChangelogRail({ releases }: ChangelogRailProps) {
  const items = releases.slice(0, 4);
  return (
    <aside className="rounded-xl border border-(--color-divider) bg-(--color-surface) p-5">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <span className="inline-block h-1.5 w-1.5 rounded-full bg-(--color-success)" />
          <MonoEyebrow tracking="wide">Changelog</MonoEyebrow>
        </div>
        <Link
          href="/changelog"
          className="text-xs text-(--color-text-tertiary) hover:text-(--color-text-primary)"
        >
          all versions →
        </Link>
      </div>
      <ul className="mt-5 flex flex-col">
        {items.map((release, idx) => (
          <li
            key={release.version}
            className={`flex items-start gap-3 py-3 ${
              idx < items.length - 1 ? "border-b border-(--color-divider)" : ""
            }`}
          >
            <span className="w-20 shrink-0">
              <MonoBadge tone="success">{release.version}</MonoBadge>
            </span>
            <div className="flex-1 min-w-0">
              <p className="font-sans text-[13px] leading-[1.4] text-(--color-text-primary)">
                {release.summary}
              </p>
              <p className="mt-1 font-mono text-[10px] uppercase tracking-[0.06em] text-(--color-text-tertiary)">
                {formatDateCompact(release.date)} · {new Date(release.date).getFullYear()}
              </p>
            </div>
          </li>
        ))}
      </ul>
      <Link
        href="/changelog"
        className="mt-4 inline-flex h-8 w-full items-center justify-center rounded-lg border border-(--color-divider) font-sans text-xs font-medium text-(--color-text-primary) transition-colors hover:bg-(--color-hover)"
      >
        Open the changelog
      </Link>
    </aside>
  );
}
