import { readdirSync, readFileSync, statSync } from "node:fs";
import { relative, resolve } from "node:path";
import { describe, expect, it } from "vitest";
import { BLOG_CATEGORIES } from "@/lib/blog";
import { siteRoot } from "./content-test-utils";

const contentRoot = resolve(siteRoot, "content");
const checkedRoots = ["app", "components", "lib"].map(root => resolve(siteRoot, root));
const ignoredPathSegments = ["/.source/", "/.velite/", "/node_modules/"];
const sourceExtensions = [".ts", ".tsx"];
const appRoutes = new Set([
  "/",
  "/api/search",
  "/blog",
  "/blog/feed.xml",
  "/changelog",
  "/opengraph-image",
  "/robots.txt",
  "/sitemap.xml",
]);
const ignoredPrefixes = ["/api/", "/images/", "/fonts/"];

function listSourceFiles(dir: string): string[] {
  const files: string[] = [];
  for (const entry of readdirSync(dir)) {
    const fullPath = resolve(dir, entry);
    const normalizedPath = fullPath.replaceAll("\\", "/");
    if (ignoredPathSegments.some(segment => normalizedPath.includes(segment))) {
      continue;
    }

    const stat = statSync(fullPath);
    if (stat.isDirectory()) {
      files.push(...listSourceFiles(fullPath));
      continue;
    }

    if (
      stat.isFile() &&
      sourceExtensions.some(extension => fullPath.endsWith(extension)) &&
      !/\.test\.[cm]?tsx?$/.test(fullPath)
    ) {
      files.push(fullPath);
    }
  }

  return files.sort();
}

function listContentRoutes(dir: string): string[] {
  const routes: string[] = [];
  for (const entry of readdirSync(dir)) {
    const fullPath = resolve(dir, entry);
    const stat = statSync(fullPath);
    if (stat.isDirectory()) {
      routes.push(...listContentRoutes(fullPath));
      continue;
    }
    if (!stat.isFile() || !fullPath.endsWith(".mdx")) {
      continue;
    }

    const routeParts = relative(contentRoot, fullPath)
      .replace(/\.mdx$/, "")
      .split("/");
    if (routeParts.at(-1) === "index") {
      routeParts.pop();
    }
    routes.push(normalizeRoute(`/${routeParts.join("/")}`));
  }

  return routes.sort();
}

function listBlogPostRoutes(): string[] {
  const postsRoot = resolve(contentRoot, "blog/posts");
  return readdirSync(postsRoot)
    .filter(file => file.endsWith(".mdx"))
    .map(file => normalizeRoute(`/blog/${file.replace(/\.mdx$/, "")}`))
    .sort();
}

function normalizeRoute(route: string): string {
  const withoutHashOrQuery = route.split(/[?#]/, 1)[0] ?? route;
  return withoutHashOrQuery.replace(/\/$/, "") || "/";
}

function cleanRoute(route: string): string {
  return normalizeRoute(route.replace(/^<|>$/g, ""));
}

function extractConcreteInternalHrefTargets(content: string): string[] {
  const targets = new Set<string>();
  const patterns = [
    /\bhref=["'](\/[^"']+)["']/g,
    /\bhref=\{["'](\/[^"']+)["']\}/g,
    /\bhref=\{`(\/[^`$]+)`\}/g,
  ];

  for (const pattern of patterns) {
    for (const match of content.matchAll(pattern)) {
      const rawTarget = match[1] ?? "";
      targets.add(cleanRoute(rawTarget));
    }
  }

  return [...targets].sort();
}

function routePatternExists(route: string): boolean {
  if (/^\/blog\/categories\/[^/]+$/.test(route)) {
    const category = route.split("/").at(-1);
    return BLOG_CATEGORIES.some(item => item === category);
  }
  if (/^\/changelog#[^#]+$/.test(route)) {
    return true;
  }
  if (/^#[A-Za-z0-9_-]+$/.test(route)) {
    return true;
  }

  return false;
}

describe("public internal links", () => {
  it("points literal internal hrefs in public TSX/TS source at existing routes", () => {
    const routes = new Set([
      ...appRoutes,
      ...listContentRoutes(contentRoot),
      ...listBlogPostRoutes(),
      ...BLOG_CATEGORIES.map(category => normalizeRoute(`/blog/categories/${category}`)),
    ]);

    const missing = checkedRoots.flatMap(listSourceFiles).flatMap(file => {
      const relativePath = relative(siteRoot, file);
      return extractConcreteInternalHrefTargets(readFileSync(file, "utf8"))
        .filter(route => !ignoredPrefixes.some(prefix => route.startsWith(prefix)))
        .filter(route => !routes.has(route) && !routePatternExists(route))
        .map(route => `${relativePath}: ${route}`);
    });

    expect(missing).toEqual([]);
  });

  it("does not publish Markdown source filenames as internal hrefs", () => {
    const markdownLinks = checkedRoots.flatMap(listSourceFiles).flatMap(file => {
      const relativePath = relative(siteRoot, file);
      return extractConcreteInternalHrefTargets(readFileSync(file, "utf8"))
        .filter(route => /\.mdx?(?:$|[?#])/.test(route))
        .map(route => `${relativePath}: ${route}`);
    });

    expect(markdownLinks).toEqual([]);
  });
});
