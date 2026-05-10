import { ArrowUpRight, Rss } from "lucide-react";
import Link from "next/link";
import { MonoEyebrow } from "./mono-eyebrow";
import { Eyebrow } from "@agh/ui";

export function SubscribeRail() {
  return (
    <aside
      aria-label="Blog subscription links"
      className="rounded-xl border border-(--color-divider) bg-(--color-canvas-deep) p-5"
    >
      <MonoEyebrow tone="accent" tracking="wide">
        Stay current
      </MonoEyebrow>
      <h4 className="mt-3 font-sans text-lg font-medium leading-tight tracking-tight text-(--color-text-primary)">
        Follow releases on the wire.
      </h4>
      <p className="mt-3 text-small-body leading-6 text-(--color-text-secondary)">
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
          href="/changelog"
          aria-label="Read the changelog"
          className="inline-flex items-center justify-between rounded-lg border border-(--color-divider) bg-(--color-surface-elevated) px-3 py-2.5 text-sm font-medium text-(--color-text-primary) transition-colors hover:bg-(--color-hover)"
        >
          <span className="inline-flex items-center gap-2">
            <ArrowUpRight size={14} aria-hidden />
            <span>Read the changelog</span>
          </span>
          <ArrowUpRight size={14} aria-hidden className="text-(--color-text-tertiary)" />
        </Link>
      </div>
      <Eyebrow case="upper" tone="muted" className="mt-5 text-(--color-text-tertiary)">
        /blog/feed.xml
      </Eyebrow>
    </aside>
  );
}
