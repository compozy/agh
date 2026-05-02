import type { Release } from "#site/content";
import { cn } from "@agh/ui";
import Link from "next/link";
import { MonoEyebrow } from "./mono-eyebrow";

export interface ChangelogTocRailProps {
  releases: Release[];
  activeVersion?: string;
}

export function ChangelogTocRail({ releases, activeVersion }: ChangelogTocRailProps) {
  return (
    <aside aria-label="Changelog versions" className="sticky top-20 flex flex-col gap-9 self-start">
      <div>
        <MonoEyebrow tracking="wide">All versions</MonoEyebrow>
        <ul className="mt-4 flex flex-col gap-2.5">
          {releases.map(release => {
            const isActive = activeVersion ? release.version === activeVersion : false;
            return (
              <li key={release.version}>
                <Link
                  href={`#${release.version}`}
                  aria-current={isActive ? "location" : undefined}
                  className={cn(
                    "block font-mono text-[13px] tracking-[0.02em]",
                    isActive
                      ? "text-(--color-accent)"
                      : "text-(--color-text-secondary) hover:text-(--color-text-primary)"
                  )}
                >
                  {release.version}
                </Link>
              </li>
            );
          })}
        </ul>
      </div>
      <div className="rounded-xl border border-(--color-divider) bg-(--color-surface) p-5">
        <MonoEyebrow tracking="wide" tone="accent">
          Upgrade
        </MonoEyebrow>
        <p className="mt-3 text-[13px] leading-[1.55] text-(--color-text-secondary)">
          One binary, one daemon. Pull the latest from Go and restart the process.
        </p>
        <Link
          href="/runtime/core/getting-started/installation"
          className="mt-4 inline-flex items-center gap-1.5 text-xs font-medium text-(--color-accent)"
        >
          Install instructions →
        </Link>
      </div>
    </aside>
  );
}
