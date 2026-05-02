import type { Post } from "#site/content";
import { ArrowUpRight, Clock } from "lucide-react";
import Link from "next/link";
import { BulletDivider } from "./divider";
import { categoryLabel, formatDateCompact, formatReadingTime } from "./format";
import { MonoEyebrow } from "./mono-eyebrow";

export interface ContinueReadingProps {
  posts: Post[];
}

export function ContinueReading({ posts }: ContinueReadingProps) {
  if (posts.length === 0) return null;

  return (
    <section className="border-t border-(--color-divider) bg-(--color-surface) px-4 py-16 lg:py-20">
      <div className="mx-auto max-w-(--site-layout-width)">
        <div className="flex flex-wrap items-baseline justify-between gap-3">
          <div className="flex items-center gap-3">
            <MonoEyebrow tracking="wide">Continue reading</MonoEyebrow>
            <span className="inline-block h-px w-9 bg-(--color-divider)" />
            <span className="font-sans text-[13px] text-(--color-text-tertiary)">
              Picked for this post
            </span>
          </div>
          <Link href="/blog" className="text-[13px] text-(--color-text-secondary)">
            All posts →
          </Link>
        </div>
        <div className="mt-6 grid gap-5 sm:grid-cols-2 lg:grid-cols-3">
          {posts.map(post => (
            <article
              key={post.slug}
              className="group rounded-xl border border-(--color-divider) bg-(--color-canvas) p-5"
            >
              <div className="flex items-center gap-2.5">
                <MonoEyebrow tone="accent">{categoryLabel(post.category)}</MonoEyebrow>
                <BulletDivider />
                <MonoEyebrow tone="neutral">{formatDateCompact(post.date)}</MonoEyebrow>
              </div>
              <h3 className="mt-4 font-sans text-[18px] font-medium leading-[1.25] tracking-[-0.02em] text-(--color-text-primary) group-hover:text-(--color-accent)">
                <Link href={post.permalink}>{post.title}</Link>
              </h3>
              <div className="mt-5 flex items-center justify-between">
                <span className="inline-flex items-center gap-1.5 text-[11px] text-(--color-text-tertiary)">
                  <Clock size={11} aria-hidden />
                  <span className="font-mono uppercase tracking-[0.06em]">
                    {formatReadingTime(post.metadata.readingTime)}
                  </span>
                </span>
                <Link
                  href={post.permalink}
                  aria-label={`Read ${post.title}`}
                  className="text-(--color-accent)"
                >
                  <ArrowUpRight size={14} aria-hidden />
                </Link>
              </div>
            </article>
          ))}
        </div>
      </div>
    </section>
  );
}
