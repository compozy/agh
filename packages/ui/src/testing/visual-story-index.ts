export type StorybookIndexEntry = {
  id: string;
  title: string;
  name: string;
  type: "story" | "docs";
  tags?: string[];
};

export type StorybookIndex = {
  v: number;
  entries: Record<string, StorybookIndexEntry>;
};

export type VisualStoryTarget = {
  id: string;
  title: string;
  name: string;
  snapshotName: string;
  storyUrl: string;
};

export class StorybookIndexError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "StorybookIndexError";
  }
}

export function assertStorybookIndex(payload: unknown): asserts payload is StorybookIndex {
  if (!payload || typeof payload !== "object") {
    throw new StorybookIndexError("Storybook index payload is not an object");
  }
  const entries = (payload as { entries?: unknown }).entries;
  if (!entries || typeof entries !== "object") {
    throw new StorybookIndexError("Storybook index is missing 'entries'");
  }
}

const SAFE_ID = /^[a-z0-9-]+(?:--[a-z0-9-]+)?$/i;

export type CollectOptions = {
  excludeTags?: readonly string[];
};

export function collectVisualTargets(
  index: StorybookIndex,
  baseUrl: string,
  options: CollectOptions = {}
): VisualStoryTarget[] {
  const excluded = new Set(options.excludeTags ?? []);
  const base = baseUrl.replace(/\/$/, "");
  const targets: VisualStoryTarget[] = [];
  for (const entry of Object.values(index.entries)) {
    if (entry.type !== "story") continue;
    if (!SAFE_ID.test(entry.id)) continue;
    const tags = entry.tags ?? [];
    if (tags.some(tag => excluded.has(tag))) continue;
    const params = new URLSearchParams({
      id: entry.id,
      viewMode: "story",
      globals: "backgrounds:!undefined",
    });
    targets.push({
      id: entry.id,
      title: entry.title,
      name: entry.name,
      snapshotName: `${entry.id}.png`,
      storyUrl: `${base}/iframe.html?${params.toString()}`,
    });
  }
  targets.sort((a, b) => a.id.localeCompare(b.id));
  return targets;
}

export async function fetchStorybookIndex(baseUrl: string): Promise<StorybookIndex> {
  const base = baseUrl.replace(/\/$/, "");
  const res = await fetch(`${base}/index.json`);
  if (!res.ok) {
    throw new StorybookIndexError(
      `Failed to fetch Storybook index at ${base}/index.json — ${res.status} ${res.statusText}`
    );
  }
  const payload = (await res.json()) as unknown;
  assertStorybookIndex(payload);
  return payload;
}
