import type { Post } from "#site/content";
import { Clock } from "lucide-react";
import Link from "next/link";
import { DateStamp } from "./date-stamp";
import { BulletDivider } from "./divider";
import { categoryLabel, formatReadingTime } from "./format";
import { MonoEyebrow } from "./mono-eyebrow";
import { Eyebrow } from "@agh/ui";

export interface PostCardProps {
  post: Post;
}

export function PostCard({ post }: PostCardProps) {
  return (
    <article className="group flex min-h-[230px] flex-col rounded-xl border border-(--color-divider) bg-(--color-surface) p-5">
      <div className="flex items-center gap-2.5">
        <MonoEyebrow tone="accent">{categoryLabel(post.category)}</MonoEyebrow>
        <BulletDivider />
        <DateStamp date={post.date} />
      </div>
      <h3 className="mt-4 font-sans text-xl font-medium leading-tight tracking-tight text-(--color-text-primary) transition-colors group-hover:text-accent">
        <Link href={post.permalink}>{post.title}</Link>
      </h3>
      <p className="mt-3 text-sm leading-7 text-(--color-text-secondary)">{post.description}</p>
      <div className="mt-auto flex items-center justify-between border-t border-(--color-divider) pt-3.5">
        <Eyebrow case="upper" tone="muted" className="text-(--color-text-label)">
          {post.author}
        </Eyebrow>
        <span className="inline-flex items-center gap-1.5 text-eyebrow text-(--color-text-tertiary)">
          <Clock size={11} aria-hidden />
          <span className="font-mono tracking-mono">
            {formatReadingTime(post.metadata.readingTime)}
          </span>
        </span>
      </div>
    </article>
  );
}
