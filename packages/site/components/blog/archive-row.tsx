import type { Post } from "#site/content";
import { ArrowUpRight } from "lucide-react";
import Link from "next/link";
import { categoryLabel, formatDateCompact, formatReadingTime } from "./format";

export interface ArchiveRowProps {
  post: Post;
}

export function ArchiveRow({ post }: ArchiveRowProps) {
  return (
    <Link
      href={post.permalink}
      className="group grid grid-cols-[88px_minmax(0,1fr)_140px_70px_16px] items-start gap-6 border-b border-(--color-divider) py-5 transition-colors hover:bg-(--color-surface)"
    >
      <span className="font-mono text-[11px] uppercase tracking-[0.06em] text-(--color-text-tertiary)">
        {formatDateCompact(post.date)}
      </span>
      <div>
        <p className="font-sans text-[19px] font-medium leading-[1.3] tracking-[-0.02em] text-(--color-text-primary) group-hover:text-(--color-accent)">
          {post.title}
        </p>
        <p className="mt-1.5 text-sm leading-[1.55] text-(--color-text-secondary)">
          {post.description}
        </p>
        {post.tags.length > 0 && (
          <div className="mt-2.5 flex flex-wrap gap-1.5">
            <span className="rounded-[5px] bg-(--color-surface-elevated) px-1.5 py-0.5 font-mono text-[10px] uppercase tracking-[0.06em] text-(--color-text-tertiary)">
              {categoryLabel(post.category)}
            </span>
            {post.tags.map(tag => (
              <span
                key={tag}
                className="rounded-[5px] bg-(--color-surface-elevated) px-1.5 py-0.5 font-mono text-[10px] tracking-[0.04em] text-(--color-text-tertiary)"
              >
                {tag}
              </span>
            ))}
          </div>
        )}
      </div>
      <span className="font-mono text-[11px] uppercase tracking-[0.06em] text-(--color-text-label)">
        {post.author}
      </span>
      <span className="font-mono text-[11px] tracking-[0.04em] text-(--color-text-tertiary)">
        {formatReadingTime(post.metadata.readingTime)}
      </span>
      <ArrowUpRight
        size={16}
        aria-hidden
        className="self-center text-(--color-text-tertiary) group-hover:text-(--color-accent)"
      />
    </Link>
  );
}
