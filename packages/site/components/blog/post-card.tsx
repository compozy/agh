import type { Post } from "#site/content";
import { Clock } from "lucide-react";
import Link from "next/link";
import { BulletDivider } from "./divider";
import { categoryLabel, formatDate, formatReadingTime } from "./format";
import { MonoEyebrow } from "./mono-eyebrow";

export interface PostCardProps {
  post: Post;
}

export function PostCard({ post }: PostCardProps) {
  return (
    <article className="group flex min-h-[230px] flex-col rounded-xl border border-(--color-divider) bg-(--color-surface) p-5">
      <div className="flex items-center gap-2.5">
        <MonoEyebrow tone="accent">{categoryLabel(post.category)}</MonoEyebrow>
        <BulletDivider />
        <MonoEyebrow tone="neutral">{formatDate(post.date)}</MonoEyebrow>
      </div>
      <h3 className="mt-4 font-sans text-[20px] font-medium leading-[1.25] tracking-[-0.02em] text-(--color-text-primary) transition-colors group-hover:text-(--color-accent)">
        <Link href={post.permalink}>{post.title}</Link>
      </h3>
      <p className="mt-3 text-sm leading-[1.6] text-(--color-text-secondary)">{post.description}</p>
      <div className="mt-auto flex items-center justify-between border-t border-(--color-divider) pt-3.5">
        <span className="font-mono text-[11px] uppercase tracking-[0.06em] text-(--color-text-label)">
          {post.author}
        </span>
        <span className="inline-flex items-center gap-1.5 text-[11px] text-(--color-text-tertiary)">
          <Clock size={11} aria-hidden />
          <span className="font-mono tracking-[0.04em]">
            {formatReadingTime(post.metadata.readingTime)}
          </span>
        </span>
      </div>
    </article>
  );
}
