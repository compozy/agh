import { readdirSync, readFileSync, statSync } from "node:fs";
import { dirname, posix, relative, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";

const siteRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..", "..");
const contentRoot = resolve(siteRoot, "content");
const appRoutes = new Set(["/", "/blog", "/changelog"]);

type ContentDoc = {
  path: string;
  route: string;
  content: string;
};

type ResolvedLink = {
  route: string;
  hash: string | null;
};

type RawLinkTarget = {
  docPath: string;
  target: string;
};

function listContentDocs(dir: string): ContentDoc[] {
  const docs: ContentDoc[] = [];
  for (const entry of readdirSync(dir)) {
    const fullPath = resolve(dir, entry);
    const stat = statSync(fullPath);
    if (stat.isDirectory()) {
      docs.push(...listContentDocs(fullPath));
      continue;
    }
    if (!stat.isFile() || !fullPath.endsWith(".mdx")) {
      continue;
    }

    const relPath = relative(contentRoot, fullPath);
    const routeParts = relPath.replace(/\.mdx$/, "").split("/");
    if (routeParts.at(-1) === "index") {
      routeParts.pop();
    }
    docs.push({
      path: relPath,
      route: normalizeRoute(`/${routeParts.join("/")}`),
      content: readFileSync(fullPath, "utf8"),
    });
  }
  return docs.sort((left, right) => left.path.localeCompare(right.path));
}

function normalizeRoute(route: string): string {
  const withoutHashOrQuery = route.split(/[?#]/, 1)[0] ?? route;
  return withoutHashOrQuery.replace(/\/$/, "") || "/";
}

function stripCodeFences(content: string): string {
  return content.replace(/```[\s\S]*?```/g, "");
}

function routeFromContentPath(contentPath: string): string {
  const routeParts = contentPath.replace(/\.(?:mdx?|MDX?)$/, "").split("/");
  if (routeParts.at(-1) === "index") {
    routeParts.pop();
  }
  return normalizeRoute(`/${routeParts.join("/")}`);
}

function cleanRawTarget(rawTarget: string): string {
  return rawTarget.trim().replace(/^<|>$/g, "");
}

function resolveLinkTarget(doc: ContentDoc, target: string): ResolvedLink | null {
  const rawTarget = cleanRawTarget(target);
  const [targetWithoutHash = "", rawHash] = rawTarget.split("#", 2);
  const normalized = targetWithoutHash.split("?", 1)[0]?.replace(/\/$/, "") ?? "";
  if (
    rawTarget.startsWith("http://") ||
    rawTarget.startsWith("https://") ||
    rawTarget.startsWith("mailto:") ||
    rawTarget.startsWith("tel:")
  ) {
    return null;
  }
  const hash = rawHash ? rawHash.replace(/^\/+/, "") : null;
  if (!normalized && hash) {
    return { route: doc.route, hash };
  }
  if (!normalized) {
    return null;
  }
  if (normalized.startsWith("/images/") || normalized.startsWith("/api/")) {
    return null;
  }
  if (normalized === "/robots.txt" || normalized === "/sitemap.xml") {
    return null;
  }
  if (normalized === "/opengraph-image") {
    return null;
  }
  if (normalized.startsWith("/")) {
    return { route: normalizeRoute(normalized), hash };
  }

  const docDir = posix.dirname(doc.path);
  const resolvedPath = posix.normalize(posix.join(docDir, normalized));
  return { route: routeFromContentPath(resolvedPath), hash };
}

function extractInternalLinks(doc: ContentDoc): ResolvedLink[] {
  const content = stripCodeFences(doc.content);
  const links: ResolvedLink[] = [];
  for (const match of content.matchAll(/\]\(([^)\s]+)(?:\s+"[^"]*")?\)/g)) {
    const resolved = resolveLinkTarget(doc, match[1] ?? "");
    if (resolved) {
      links.push(resolved);
    }
  }
  for (const match of content.matchAll(/\bhref=["']([^"']+)["']/g)) {
    const resolved = resolveLinkTarget(doc, match[1] ?? "");
    if (resolved) {
      links.push(resolved);
    }
  }
  return links;
}

function extractRawInternalLinkTargets(doc: ContentDoc): RawLinkTarget[] {
  const content = stripCodeFences(doc.content);
  const targets: RawLinkTarget[] = [];
  for (const match of content.matchAll(/\]\(([^)\s]+)(?:\s+"[^"]*")?\)/g)) {
    const target = cleanRawTarget(match[1] ?? "");
    if (resolveLinkTarget(doc, target)) {
      targets.push({ docPath: doc.path, target });
    }
  }
  for (const match of content.matchAll(/\bhref=["']([^"']+)["']/g)) {
    const target = cleanRawTarget(match[1] ?? "");
    if (resolveLinkTarget(doc, target)) {
      targets.push({ docPath: doc.path, target });
    }
  }
  return targets;
}

function slugifyHeading(heading: string): string {
  return heading
    .replace(/<[^>]+>/g, "")
    .replace(/`([^`]+)`/g, "$1")
    .toLowerCase()
    .trim()
    .replace(/[^\p{L}\p{N}\s-]/gu, "")
    .replace(/\s+/g, "-");
}

function anchorCountsFor(doc: ContentDoc): Map<string, number> {
  const anchors = new Map<string, number>();
  const content = stripCodeFences(doc.content);
  for (const match of content.matchAll(/^#{1,6}\s+(.+)$/gm)) {
    const anchor = slugifyHeading(match[1] ?? "");
    anchors.set(anchor, (anchors.get(anchor) ?? 0) + 1);
  }
  for (const match of content.matchAll(/\bid=["']([^"']+)["']/g)) {
    const anchor = match[1] ?? "";
    anchors.set(anchor, (anchors.get(anchor) ?? 0) + 1);
  }
  return anchors;
}

describe("content internal links", () => {
  it("points Markdown and MDX component links at existing site routes", () => {
    const docs = listContentDocs(contentRoot);
    const routes = new Set([...appRoutes, ...docs.map(doc => doc.route)]);
    const missing = docs.flatMap(doc =>
      extractInternalLinks(doc)
        .filter(link => !routes.has(link.route))
        .map(link => `${doc.path} -> ${link.route}`)
    );

    expect(missing).toEqual([]);
  });

  it("points hash links at headings or explicit ids on the target page", () => {
    const docs = listContentDocs(contentRoot);
    const docsByRoute = new Map(docs.map(doc => [doc.route, doc]));
    const anchorsByRoute = new Map(docs.map(doc => [doc.route, anchorCountsFor(doc)]));
    const missing = docs.flatMap(doc =>
      extractInternalLinks(doc)
        .filter(link => link.hash && docsByRoute.has(link.route))
        .filter(link => !anchorsByRoute.get(link.route)?.has(link.hash ?? ""))
        .map(link => `${doc.path} -> ${link.route}#${link.hash}`)
    );

    expect(missing).toEqual([]);
  });

  it("points hash links at unambiguous anchors on the target page", () => {
    const docs = listContentDocs(contentRoot);
    const docsByRoute = new Map(docs.map(doc => [doc.route, doc]));
    const anchorsByRoute = new Map(docs.map(doc => [doc.route, anchorCountsFor(doc)]));
    const ambiguous = docs.flatMap(doc =>
      extractInternalLinks(doc)
        .filter(link => link.hash && docsByRoute.has(link.route))
        .filter(link => (anchorsByRoute.get(link.route)?.get(link.hash ?? "") ?? 0) > 1)
        .map(link => `${doc.path} -> ${link.route}#${link.hash}`)
    );

    expect(ambiguous).toEqual([]);
  });

  it("uses route links instead of Markdown source filenames", () => {
    const markdownSourceLinks = listContentDocs(contentRoot)
      .flatMap(extractRawInternalLinkTargets)
      .filter(link => /\.mdx?(?:$|[?#])/.test(link.target))
      .map(link => `${link.docPath} -> ${link.target}`);

    expect(markdownSourceLinks).toEqual([]);
  });
});
