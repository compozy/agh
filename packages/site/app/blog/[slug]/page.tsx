import type { Metadata } from "next";
import Link from "next/link";
import { notFound } from "next/navigation";
import { ArrowLeft, Clock } from "lucide-react";
import { AuthorMeta } from "@/components/blog/author-meta";
import { ContinueReading } from "@/components/blog/continue-reading";
import { BulletDivider } from "@/components/blog/divider";
import { categoryLabel, formatDate, formatReadingTime } from "@/components/blog/format";
import { MdxContent } from "@/components/blog/mdx-content";
import { MonoEyebrow } from "@/components/blog/mono-eyebrow";
import { TocRail } from "@/components/blog/toc-rail";
import { flattenToc } from "@/components/blog/toc-utils";
import { allPosts, authorByHandle, authorInitial, postBySlug, relatedPosts } from "@/lib/blog";
import { createPageMetadata } from "@/lib/site-config";

interface PageProps {
  params: Promise<{ slug: string }>;
}

export function generateStaticParams() {
  return allPosts().map(post => ({ slug: post.slug.replace(/^posts\//, "") }));
}

export async function generateMetadata({ params }: PageProps): Promise<Metadata> {
  const { slug } = await params;
  const post = postBySlug(slug);
  if (!post) return {};
  return createPageMetadata({
    title: post.title,
    description: post.description,
    path: post.permalink,
  });
}

export default async function BlogPostPage({ params }: PageProps) {
  const { slug } = await params;
  const post = postBySlug(slug);
  if (!post) notFound();

  const author = authorByHandle(post.author);
  const initial = authorInitial(post.author);
  const related = relatedPosts(post);
  const readingTime = formatReadingTime(post.metadata.readingTime);

  return (
    <>
      <section className="border-b border-(--color-divider) px-4 pt-14 pb-9">
        <div className="mx-auto max-w-(--site-layout-width)">
          <div className="max-w-[760px]">
            <Link
              href="/blog"
              className="inline-flex items-center gap-1.5 text-[13px] text-(--color-text-tertiary) hover:text-(--color-text-primary)"
            >
              <ArrowLeft size={13} aria-hidden />
              <span>Back to blog</span>
            </Link>
            <div className="mt-7 flex flex-wrap items-center gap-3">
              <MonoEyebrow tone="accent">BLOG</MonoEyebrow>
              <BulletDivider />
              <MonoEyebrow>{categoryLabel(post.category)}</MonoEyebrow>
              <BulletDivider />
              <MonoEyebrow>{formatDate(post.date)}</MonoEyebrow>
              <BulletDivider />
              <span className="inline-flex items-center gap-1.5 text-[11px] text-(--color-text-tertiary)">
                <Clock size={11} aria-hidden />
                <span className="font-mono uppercase tracking-[0.06em]">{readingTime} read</span>
              </span>
            </div>
            <h1 className="mt-7 font-display text-[clamp(2.4rem,4.4vw,3.6rem)] font-normal leading-[1] tracking-[-0.035em] text-(--color-text-primary)">
              {post.title}
            </h1>
            <p className="mt-6 max-w-[58ch] text-[19px] leading-[1.5] text-(--color-text-secondary)">
              {post.description}
            </p>
            <div className="mt-9 flex items-center justify-between gap-4">
              <AuthorMeta
                handle={author?.name ?? post.author}
                initial={initial}
                role={author?.role}
                size="md"
                layout="stacked"
              />
            </div>
          </div>
        </div>
      </section>

      <section className="px-4 pt-8 pb-16">
        <div className="mx-auto grid max-w-(--site-layout-width) gap-12 lg:grid-cols-[minmax(0,760px)_220px]">
          <article className="prose-blog max-w-none">
            <MdxContent code={post.body} />
          </article>
          <TocRail items={flattenToc(post.toc)} />
        </div>
      </section>

      <ContinueReading posts={related} />
    </>
  );
}
