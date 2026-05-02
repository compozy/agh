import type { Post } from "#site/content";
import { ArrowUpRight, Clock } from "lucide-react";
import Image from "next/image";
import Link from "next/link";
import { AuthorMeta } from "./author-meta";
import { DateStamp } from "./date-stamp";
import { BulletDivider } from "./divider";
import { categoryLabel, formatReadingTime } from "./format";
import { KindChip, type WireKind } from "./kind-chip";
import { MonoBadge } from "./mono-badge";
import { MonoEyebrow } from "./mono-eyebrow";
import { blogPostCover } from "@/lib/blog";

export interface FeaturedPostProps {
  post: Post;
  authorInitial: string;
}

export function FeaturedPost({ post, authorInitial }: FeaturedPostProps) {
  const readingTime = formatReadingTime(post.metadata.readingTime);
  const cover = resolveFeaturedCover(post);

  return (
    <article className="grid gap-10 rounded-xl border border-(--color-divider) bg-(--color-surface) p-7 lg:grid-cols-[minmax(0,1.05fr)_minmax(0,1fr)] lg:items-center">
      <div className="order-2 lg:order-1">
        <div className="flex flex-wrap items-center gap-3">
          <MonoBadge tone="accent">FEATURED</MonoBadge>
          <BulletDivider />
          <MonoEyebrow>{categoryLabel(post.category)}</MonoEyebrow>
          <BulletDivider />
          <DateStamp date={post.date} />
        </div>
        <h2 className="mt-6 max-w-[20ch] font-display text-[clamp(2rem,3.4vw,2.8rem)] font-normal leading-[1.02] tracking-[-0.03em] text-(--color-text-primary)">
          <Link href={post.permalink} className="transition-colors hover:text-(--color-accent)">
            {post.title}
          </Link>
        </h2>
        <p className="mt-5 max-w-[54ch] text-base leading-[1.6] text-(--color-text-secondary)">
          {post.description}
        </p>
        {post.kinds.length > 0 && (
          <div className="mt-7 flex flex-wrap items-center gap-2">
            {post.kinds.map(kind => (
              <KindChip key={kind} kind={kind as WireKind} />
            ))}
          </div>
        )}
        <div className="mt-7 flex flex-wrap items-center gap-4">
          <AuthorMeta handle={post.author} initial={authorInitial} size="sm" />
          <BulletDivider />
          <span className="inline-flex items-center gap-1.5 text-xs text-(--color-text-tertiary)">
            <Clock size={12} aria-hidden />
            <span>{readingTime} read</span>
          </span>
          <Link
            href={post.permalink}
            className="ml-auto inline-flex items-center gap-1.5 text-sm font-medium text-(--color-accent)"
          >
            Read post <ArrowUpRight size={14} aria-hidden />
          </Link>
        </div>
      </div>
      <div className="order-1 lg:order-2">
        {cover ? (
          <FeaturedCover
            src={cover.src}
            alt={cover.alt}
            width={cover.width}
            height={cover.height}
          />
        ) : (
          <FeaturedVisual kinds={post.kinds.length > 0 ? (post.kinds as WireKind[]) : undefined} />
        )}
      </div>
    </article>
  );
}

function resolveFeaturedCover(
  post: Post
): { src: string; alt: string; width: number; height: number } | null {
  return blogPostCover(post);
}

interface FeaturedCoverProps {
  src: string;
  alt: string;
  width: number;
  height: number;
}

function FeaturedCover({ src, alt, width, height }: FeaturedCoverProps) {
  return (
    <div className="overflow-hidden rounded-xl border border-(--color-divider) bg-(--color-canvas-deep)">
      <Image
        src={src}
        alt={alt}
        width={width}
        height={height}
        priority
        sizes="(min-width: 1024px) 38vw, 100vw"
        className="block h-auto w-full"
      />
    </div>
  );
}

interface FeaturedVisualProps {
  kinds?: WireKind[];
}

function FeaturedVisual({ kinds }: FeaturedVisualProps) {
  const wire = (
    kinds && kinds.length >= 4 ? kinds.slice(0, 4) : ["greet", "direct", "receipt", "trace"]
  ) as WireKind[];
  const trace: { kind: WireKind; from: string; to: string; t: string }[] = [
    { kind: wire[0], from: "alpha", to: "bravo", t: "00:00.041" },
    { kind: wire[1], from: "alpha", to: "bravo", t: "00:00.108" },
    { kind: wire[2], from: "bravo", to: "alpha", t: "00:00.382" },
    { kind: wire[3], from: "bravo", to: "*", t: "00:00.384" },
  ];
  return (
    <div className="relative min-h-[340px] rounded-xl border border-(--color-divider) bg-(--color-canvas-deep) p-6">
      <div className="flex items-center justify-between">
        <MonoEyebrow tracking="wide">agh-network/v0</MonoEyebrow>
        <span className="inline-flex items-center gap-1.5">
          <span className="inline-block h-1.5 w-1.5 rounded-full bg-(--color-accent)" />
          <MonoEyebrow tone="accent" tracking="wide">
            ALPHA
          </MonoEyebrow>
        </span>
      </div>
      <div className="mt-10 grid grid-cols-3 gap-4">
        {[
          { id: "agent.alpha", role: "planner", highlight: false },
          { id: "agent.bravo", role: "executor", highlight: true },
          { id: "agent.charlie", role: "guardian", highlight: false },
        ].map(node => (
          <div
            key={node.id}
            className={`rounded-lg border bg-(--color-surface) px-3 py-2.5 ${
              node.highlight ? "border-(--color-accent)" : "border-(--color-divider)"
            }`}
          >
            <p
              className={`font-mono text-[11px] ${
                node.highlight ? "text-(--color-accent)" : "text-(--color-text-primary)"
              }`}
            >
              {node.id}
            </p>
            <p className="mt-1 font-mono text-[9.5px] uppercase tracking-[0.08em] text-(--color-text-tertiary)">
              {node.role}
            </p>
          </div>
        ))}
      </div>
      <div className="mt-6 rounded-lg border border-(--color-divider) bg-(--color-surface) px-3 py-2.5">
        <div className="flex items-center justify-between border-b border-(--color-divider) pb-2">
          <MonoEyebrow tracking="wide">WIRE TRACE</MonoEyebrow>
          <MonoEyebrow tone="neutral">{trace.length} events</MonoEyebrow>
        </div>
        <ul className="mt-3 flex flex-col gap-2">
          {trace.map((row, idx) => (
            <li key={idx} className="flex items-center gap-3">
              <span className="w-16 font-mono text-[10px] text-(--color-text-tertiary)">
                {row.t}
              </span>
              <KindChip kind={row.kind} />
              <span className="font-mono text-[11px] text-(--color-text-secondary)">
                {row.from} <span className="text-(--color-accent)">→</span> {row.to}
              </span>
            </li>
          ))}
        </ul>
      </div>
    </div>
  );
}
