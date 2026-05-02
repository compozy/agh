import type { Metadata } from "next";
import { notFound } from "next/navigation";
import { ArchiveRow } from "@/components/blog/archive-row";
import { CategoryPill } from "@/components/blog/category-pill";
import { BlogEmptyState } from "@/components/blog/empty-state";
import { categoryLabel } from "@/components/blog/format";
import { MonoEyebrow } from "@/components/blog/mono-eyebrow";
import {
  BLOG_CATEGORIES,
  type BlogCategory,
  allPosts,
  categoryCounts,
  postsByCategory,
} from "@/lib/blog";
import { createPageMetadata } from "@/lib/site-config";

interface PageProps {
  params: Promise<{ category: string }>;
}

export function generateStaticParams() {
  return BLOG_CATEGORIES.map(category => ({ category }));
}

function isCategory(slug: string): slug is BlogCategory {
  return (BLOG_CATEGORIES as readonly string[]).includes(slug);
}

export async function generateMetadata({ params }: PageProps): Promise<Metadata> {
  const { category } = await params;
  if (!isCategory(category)) return {};
  return createPageMetadata({
    title: `${categoryLabel(category)} posts`,
    description: `Posts filed under ${categoryLabel(category)}.`,
    path: `/blog/categories/${category}`,
  });
}

export default async function CategoryArchivePage({ params }: PageProps) {
  const { category } = await params;
  if (!isCategory(category)) notFound();

  const posts = postsByCategory(category);
  const counts = categoryCounts();
  const total = allPosts().length;

  return (
    <>
      <section className="border-b border-(--color-divider) px-4 pt-14 pb-12">
        <div className="mx-auto max-w-(--site-layout-width)">
          <div className="flex items-center gap-3">
            <MonoEyebrow tone="accent">CATEGORY</MonoEyebrow>
            <span className="inline-block h-px w-9 bg-(--color-divider)" />
            <MonoEyebrow>{categoryLabel(category)}</MonoEyebrow>
          </div>
          <h1 className="mt-6 font-display text-[clamp(2.2rem,4vw,3.4rem)] font-normal leading-[1.02] tracking-[-0.035em] text-(--color-text-primary)">
            {categoryLabel(category)} posts
          </h1>
          <p className="mt-4 max-w-[58ch] text-base leading-[1.6] text-(--color-text-secondary)">
            {posts.length === 0
              ? "Nothing filed here yet. Subscribe to the feed to catch the next one."
              : `${posts.length} ${posts.length === 1 ? "post" : "posts"} in this category.`}
          </p>
          <div className="mt-7 flex flex-wrap items-center gap-2">
            <CategoryPill label="All" count={total} href="/blog" />
            {BLOG_CATEGORIES.map(slug => (
              <CategoryPill
                key={slug}
                label={categoryLabel(slug)}
                count={counts[slug]}
                href={`/blog/categories/${slug}`}
                active={slug === category}
              />
            ))}
          </div>
        </div>
      </section>

      <section className="px-4 pt-10 pb-20">
        <div className="mx-auto max-w-(--site-layout-width)">
          {posts.length === 0 ? (
            <BlogEmptyState
              eyebrow="Category pending"
              title={`${categoryLabel(category)} posts are not published yet.`}
              description="This category is part of the public archive, but no post has been filed here yet. Browse the full blog archive or subscribe to the feed to catch the next entry."
              primaryAction={{ href: "/blog", label: "Browse all posts" }}
              secondaryAction={{ href: "/blog/feed.xml", label: "Subscribe via RSS" }}
            />
          ) : (
            <div className="border-t border-(--color-divider)">
              {posts.map(post => (
                <ArchiveRow key={post.slug} post={post} />
              ))}
            </div>
          )}
        </div>
      </section>
    </>
  );
}
