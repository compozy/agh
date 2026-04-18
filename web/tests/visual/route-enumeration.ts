import { readdirSync, statSync } from "node:fs";
import path from "node:path";

export type RouteEntry = {
  /** Canonical URL path, e.g. "/", "/tasks", "/tasks/$id", "/settings/general". */
  urlPath: string;
  /** Absolute filesystem path to the `.tsx` route module. */
  filePath: string;
  /** The route is reachable without authentication (outside `_app/`). */
  isPublic: boolean;
};

const STORY_FILE_PATTERN = /\.stories\.(tsx|ts)$/;

function isExcludedName(name: string): boolean {
  // TanStack convention: `-prefixed` files and folders are excluded from the route tree.
  return name.startsWith("-");
}

function isLayoutName(base: string): boolean {
  // Pathless layout routes (e.g. `_app`) are not user-navigable URLs.
  return base.startsWith("_");
}

function joinUrl(prefix: string, segment: string): string {
  const safePrefix = prefix.endsWith("/") ? prefix.slice(0, -1) : prefix;
  if (segment === "") return safePrefix === "" ? "/" : safePrefix;
  const safeSegment = segment.startsWith("/") ? segment : `/${segment}`;
  return `${safePrefix}${safeSegment}`;
}

function segmentsFromFileBase(base: string): string[] {
  // Flat-routes: dots split segments. `tasks.$id.edit` → ["tasks", "$id", "edit"].
  return base
    .split(".")
    .map(part => part.trim())
    .filter(Boolean);
}

function readEntries(dir: string): string[] {
  try {
    return readdirSync(dir).sort();
  } catch {
    return [];
  }
}

function collectRouteFiles(dir: string, urlPrefix: string, acc: RouteEntry[]): void {
  for (const entry of readEntries(dir)) {
    if (isExcludedName(entry)) continue;
    if (entry === "stories" || entry === "__snapshots__") continue;
    const full = path.join(dir, entry);
    const stats = statSync(full);
    if (stats.isDirectory()) {
      if (isLayoutName(entry)) {
        // Pathless layout folder — descend without changing the URL prefix.
        collectRouteFiles(full, urlPrefix, acc);
      } else {
        collectRouteFiles(full, joinUrl(urlPrefix, entry), acc);
      }
      continue;
    }
    if (!entry.endsWith(".tsx")) continue;
    if (STORY_FILE_PATTERN.test(entry)) continue;
    if (entry.endsWith(".test.tsx")) continue;
    const base = entry.slice(0, -".tsx".length);
    if (base.startsWith("__")) continue; // `__root`
    if (isLayoutName(base)) {
      if (base === "_app") continue; // root layout folder handles pathless nesting
      // Other layout files: descend into matching folder only via directory step above.
      continue;
    }
    const segments = segmentsFromFileBase(base);
    if (segments.length === 0) continue;
    // Treat `index` as the directory default.
    const effective = segments[segments.length - 1] === "index" ? segments.slice(0, -1) : segments;
    const urlPath = effective.reduce<string>((prev, seg) => joinUrl(prev, seg), urlPrefix) || "/";
    acc.push({
      urlPath,
      filePath: full,
      isPublic: !urlPrefix.startsWith("/_") && !full.includes(`${path.sep}_app${path.sep}`),
    });
  }
}

/**
 * Enumerate public routes under `web/src/routes/`.
 *
 * - Skips `-prefixed` files/folders (excluded from the route tree).
 * - Expands flat-routes (`tasks.$id.edit`) into URL segments.
 * - Descends into pathless layouts (`_app`) without contributing to the URL path.
 * - Ignores `stories`, `__snapshots__`, `*.stories.tsx`, `*.test.tsx`, `__root.tsx`.
 */
export function enumerateRoutes(routesRoot: string): RouteEntry[] {
  const acc: RouteEntry[] = [];
  collectRouteFiles(routesRoot, "", acc);
  // Deduplicate by URL (index + layout collapsing can produce duplicates).
  const seen = new Map<string, RouteEntry>();
  for (const entry of acc) {
    if (!seen.has(entry.urlPath)) {
      seen.set(entry.urlPath, entry);
    }
  }
  return Array.from(seen.values()).sort((a, b) => a.urlPath.localeCompare(b.urlPath));
}

/**
 * Enumerate `.stories.tsx` files under `web/src/`.
 * Returns absolute file paths sorted alphabetically.
 */
export function enumerateStoryFiles(srcRoot: string): string[] {
  const out: string[] = [];
  function walk(dir: string): void {
    for (const entry of readEntries(dir)) {
      if (entry === "node_modules" || entry === "__snapshots__") continue;
      const full = path.join(dir, entry);
      const stats = statSync(full);
      if (stats.isDirectory()) {
        walk(full);
        continue;
      }
      if (STORY_FILE_PATTERN.test(entry)) {
        out.push(full);
      }
    }
  }
  walk(srcRoot);
  return out.sort();
}

/**
 * Derive a storybook id candidate from a `*.stories.tsx` file by reading the
 * default export's `title` field. This is a purely heuristic read and returns
 * `null` if no title is found — callers should fall back to the storybook
 * index for authoritative ids.
 */
export function readStoryTitle(fileSource: string): string | null {
  const match = fileSource.match(/title\s*:\s*["']([^"']+)["']/);
  return match ? match[1] : null;
}
