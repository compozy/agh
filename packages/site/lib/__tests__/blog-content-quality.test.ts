import { existsSync } from "node:fs";
import { resolve } from "node:path";
import { authors, posts, releases } from "#site/content";
import { describe, expect, it } from "vitest";
import {
  BLOG_CATEGORIES,
  allPosts,
  allReleases,
  authorByHandle,
  blogPostCover,
  categoryCounts,
  featuredPost,
  postBySlug,
  postsByCategory,
} from "../blog";
import { publicRoot } from "../content-test-utils";

const categorySet = new Set<string>(BLOG_CATEGORIES);

function expectPublishedDate(value: string, label: string) {
  const parsed = new Date(value);
  expect(Number.isNaN(parsed.getTime()), label).toBe(false);
  expect(parsed.getTime(), label).toBeLessThanOrEqual(Date.now());
}

function expectUnique(values: string[], label: string) {
  expect(new Set(values).size, label).toBe(values.length);
}

describe("blog content quality", () => {
  it("keeps posts discoverable, dated, and attributed", () => {
    expect(posts.length).toBeGreaterThan(0);
    expectUnique(
      posts.map(post => post.slug),
      "post slugs"
    );
    expectUnique(
      posts.map(post => post.permalink),
      "post permalinks"
    );

    for (const post of posts) {
      expect(post.slug.startsWith("posts/"), post.slug).toBe(true);
      expect(post.permalink, post.slug).toBe(`/blog/${post.slug.replace(/^posts\//, "")}`);
      expect(post.permalink.endsWith(".mdx"), post.slug).toBe(false);
      expect(categorySet.has(post.category), post.slug).toBe(true);
      expect(authorByHandle(post.author), post.slug).toBeTruthy();
      expect(post.description.length, post.slug).toBeGreaterThanOrEqual(80);
      expectPublishedDate(post.date, `${post.slug} date`);
      if (post.updated) {
        expectPublishedDate(post.updated, `${post.slug} updated`);
        expect(new Date(post.updated).getTime(), post.slug).toBeGreaterThanOrEqual(
          new Date(post.date).getTime()
        );
      }
      expect(post.tags.length, post.slug).toBeGreaterThanOrEqual(3);
      expectUnique(post.tags, `${post.slug} tags`);
      for (const tag of post.tags) {
        expect(tag, post.slug).toMatch(/^[a-z0-9]+(?:-[a-z0-9]+)*$/);
      }
    }
  });

  it("keeps blog helpers aligned with generated content", () => {
    const sorted = [...posts].sort(
      (a, b) => new Date(b.date).getTime() - new Date(a.date).getTime()
    );

    expect(allPosts().map(post => post.slug)).toEqual(sorted.map(post => post.slug));
    for (const post of posts) {
      expect(postBySlug(post.slug)).toBe(post);
      expect(postBySlug(post.slug.replace(/^posts\//, ""))).toBe(post);
    }

    const counts = categoryCounts();
    for (const category of BLOG_CATEGORIES) {
      const expected = posts.filter(post => post.category === category);
      expect(postsByCategory(category).map(post => post.slug)).toEqual(
        expected
          .sort((a, b) => new Date(b.date).getTime() - new Date(a.date).getTime())
          .map(post => post.slug)
      );
      expect(counts[category]).toBe(expected.length);
    }

    const featured = featuredPost();
    expect(featured).toBeTruthy();
    expect(featured ? posts.includes(featured) : false).toBe(true);
  });

  it("keeps blog covers backed by public assets", () => {
    for (const post of posts) {
      const cover = blogPostCover(post);
      if (!cover) {
        continue;
      }

      expect(cover.src.startsWith("/"), post.slug).toBe(true);
      expect(existsSync(resolve(publicRoot, cover.src.slice(1))), post.slug).toBe(true);
      expect(cover.alt.length, post.slug).toBeGreaterThanOrEqual(20);
      expect(cover.width, post.slug).toBeGreaterThan(0);
      expect(cover.height, post.slug).toBeGreaterThan(0);
    }
  });

  it("keeps fallback featured cover alt text aligned with public copy", () => {
    const post = postBySlug("posts/introducing-agh-the-first-agent-network-protocol");
    expect(post).toBeTruthy();
    expect(post ? blogPostCover(post)?.alt : null).toBe(
      "agh-network/v0, three peers exchanging direct, receipt, and trace envelopes"
    );
  });

  it("keeps authors and releases internally consistent", () => {
    expectUnique(
      authors.map(author => author.handle),
      "author handles"
    );
    for (const author of authors) {
      expect(author.avatar.length, author.handle).toBe(1);
      if (author.github) {
        const github = new URL(author.github);
        expect(github.protocol, author.handle).toBe("https:");
        expect(github.hostname, author.handle).toBe("github.com");
      }
    }

    expectUnique(
      releases.map(release => release.version),
      "release versions"
    );
    expect(allReleases().map(release => release.version)).toEqual(
      [...releases]
        .sort((a, b) => new Date(b.date).getTime() - new Date(a.date).getTime())
        .map(release => release.version)
    );
    for (const release of releases) {
      expectPublishedDate(release.date, `${release.version} date`);
      if (release.compareUrl) {
        expect(new URL(release.compareUrl).protocol, release.version).toBe("https:");
      }
    }
  });
});
