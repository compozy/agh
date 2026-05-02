import { ArrowUpRight, Rss } from "lucide-react";
import Link from "next/link";
import { GithubLogo } from "@agh/ui/logos";
import { siteConfig } from "@/lib/site-config";
import { MonoEyebrow } from "./mono-eyebrow";

export function SubscribeRail() {
  const releasesUrl = `${siteConfig.githubUrl}/releases`;
  return (
    <aside className="rounded-xl border border-(--color-divider) bg-(--color-canvas-deep) p-5">
      <MonoEyebrow tone="accent" tracking="wide">
        Stay current
      </MonoEyebrow>
      <h4 className="mt-3 font-sans text-lg font-medium leading-[1.25] tracking-[-0.01em] text-(--color-text-primary)">
        Follow releases on the wire.
      </h4>
      <p className="mt-3 text-[13px] leading-[1.55] text-(--color-text-secondary)">
        Protocol changes, runtime drops, and the occasional engineering note. Pick your channel.
      </p>
      <div className="mt-5 flex flex-col gap-2">
        <Link
          href="/blog/feed.xml"
          className="inline-flex items-center justify-between rounded-lg border border-(--color-divider) bg-(--color-surface-elevated) px-3 py-2.5 text-sm font-medium text-(--color-text-primary) transition-colors hover:bg-(--color-hover)"
        >
          <span className="inline-flex items-center gap-2">
            <Rss size={14} aria-hidden />
            <span>RSS feed</span>
          </span>
          <ArrowUpRight size={14} aria-hidden className="text-(--color-text-tertiary)" />
        </Link>
        <Link
          href={releasesUrl}
          target="_blank"
          rel="noreferrer noopener"
          className="inline-flex items-center justify-between rounded-lg border border-(--color-divider) bg-(--color-surface-elevated) px-3 py-2.5 text-sm font-medium text-(--color-text-primary) transition-colors hover:bg-(--color-hover)"
        >
          <span className="inline-flex items-center gap-2">
            <GithubLogo aria-hidden className="h-3.5 w-3.5" />
            <span>Watch releases on GitHub</span>
          </span>
          <ArrowUpRight size={14} aria-hidden className="text-(--color-text-tertiary)" />
        </Link>
      </div>
      <p className="mt-5 font-mono text-[11px] uppercase tracking-[0.06em] text-(--color-text-tertiary)">
        /blog/feed.xml
      </p>
    </aside>
  );
}
