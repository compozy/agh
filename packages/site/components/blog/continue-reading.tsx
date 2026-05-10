import type { Post } from "#site/content";
import { Eyebrow } from "@agh/ui";
import { ArrowUpRight, Clock } from "lucide-react";
import Link from "next/link";
import { BulletDivider } from "./divider";
import { BlogEmptyState } from "./empty-state";
import { DateStamp } from "./date-stamp";
import { categoryLabel, formatReadingTime } from "./format";
import { MonoEyebrow } from "./mono-eyebrow";

export interface ContinueReadingProps {
  posts: Post[];
}

export function ContinueReading({ posts }: ContinueReadingProps) {
  return (
    <section className="border-t border-(--color-divider) bg-(--color-surface) px-4 py-16 lg:py-20">
      <div className="mx-auto max-w-(--site-layout-width)">
        <div className="flex flex-wrap items-baseline justify-between gap-3">
          <div className="flex items-center gap-3">
            <MonoEyebrow tracking="wide">Continue reading</MonoEyebrow>
            <span className="inline-block h-px w-9 bg-(--color-divider)" />
            <span className="font-sans text-small-body text-(--color-text-tertiary)">
              Picked for this post
            </span>
          </div>
          <Link href="/blog" className="text-small-body text-(--color-text-secondary)">
            All posts →
          </Link>
        </div>
        {posts.length === 0 ? (
          <div className="mt-6">
            <BlogEmptyState
              eyebrow="Reading queue pending"
              title="More field notes are being prepared."
              description="This post is the full archive for now. Subscribe to the feed or read the release log while the next runtime note, protocol note, or release receipt is prepared."
              primaryAction={{ href: "/blog/feed.xml", label: "Subscribe via RSS" }}
              secondaryAction={{ href: "/changelog", label: "Read the release log" }}
            />
          </div>
        ) : (
          <div className="mt-6 grid gap-5 sm:grid-cols-2 lg:grid-cols-3">
            {posts.map(post => (
              <article
                key={post.slug}
                className="group rounded-xl border border-(--color-divider) bg-(--color-canvas) p-5"
              >
                <div className="flex items-center gap-2.5">
                  <MonoEyebrow tone="accent">{categoryLabel(post.category)}</MonoEyebrow>
                  <BulletDivider />
                  <DateStamp date={post.date} format="compact" />
                </div>
                <h3 className="mt-4 font-sans text-lg font-medium leading-tight tracking-tight text-(--color-text-primary) group-hover:text-accent">
                  <Link href={post.permalink}>{post.title}</Link>
                </h3>
                <div className="mt-5 flex items-center justify-between">
                  <span className="inline-flex items-center gap-1.5 text-eyebrow text-(--color-text-tertiary)">
                    <Clock size={11} aria-hidden />
                    <Eyebrow case="upper">{formatReadingTime(post.metadata.readingTime)}</Eyebrow>
                  </span>
                  <Link
                    href={post.permalink}
                    aria-label={`Read ${post.title}`}
                    className="text-accent"
                  >
                    <ArrowUpRight size={14} aria-hidden />
                  </Link>
                </div>
              </article>
            ))}
          </div>
        )}
      </div>
    </section>
  );
}
