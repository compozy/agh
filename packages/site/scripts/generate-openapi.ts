import path from "node:path";
import { promises as fs } from "node:fs";
import { fileURLToPath } from "node:url";
import { generateFiles } from "fumadocs-openapi";
import { openapi, AGH_OPENAPI_PATH } from "../lib/openapi";
import { API_SECTIONS } from "../lib/runtime-navigation";

const HERE = path.dirname(fileURLToPath(import.meta.url));
const OUT_DIR = path.resolve(HERE, "../content/runtime/api-reference");
const PRESERVE = new Set(["index.mdx"]);

type OpenAPIDocument = {
  paths?: Record<string, Record<string, { tags?: string[] }>>;
};

async function cleanGenerated(): Promise<void> {
  const entries = await fs.readdir(OUT_DIR);
  await Promise.all(
    entries
      .filter(entry => entry.endsWith(".mdx") && !PRESERVE.has(entry))
      .map(entry => fs.rm(path.join(OUT_DIR, entry), { force: true }))
  );
}

async function readUsedTags(): Promise<string[]> {
  const raw = await fs.readFile(AGH_OPENAPI_PATH, "utf8");
  const doc = JSON.parse(raw) as OpenAPIDocument;
  const tags = new Set<string>();
  for (const ops of Object.values(doc.paths ?? {})) {
    for (const op of Object.values(ops)) {
      for (const tag of op.tags ?? []) tags.add(tag);
    }
  }
  return [...tags];
}

function tagSlug(tag: string): string {
  return tag.toLowerCase().replace(/\s+/g, "-");
}

function buildMetaPages(usedTagSlugs: Set<string>): string[] {
  const placed = new Set<string>();
  const pages: string[] = ["index"];
  for (const section of API_SECTIONS) {
    const present = section.ids.filter(id => usedTagSlugs.has(id));
    if (present.length === 0) continue;
    pages.push(`---${section.label}---`);
    for (const id of present) {
      pages.push(id);
      placed.add(id);
    }
  }
  const trailing = [...usedTagSlugs].filter(slug => !placed.has(slug)).sort();
  if (trailing.length > 0) {
    pages.push("---More---", ...trailing);
  }
  return pages;
}

async function writeMeta(usedTagSlugs: Set<string>): Promise<void> {
  const meta = {
    title: "API Reference",
    icon: "FileCode",
    root: true,
    pages: buildMetaPages(usedTagSlugs),
  };
  await fs.writeFile(path.join(OUT_DIR, "meta.json"), `${JSON.stringify(meta, null, 4)}\n`, "utf8");
}

const TAG_ICONS: Record<string, string> = {
  agent: "MessageSquare",
  agents: "FileText",
  automation: "Activity",
  bridges: "Layers",
  daemon: "Activity",
  extensions: "Plug",
  hooks: "Waypoints",
  memory: "Brain",
  network: "Network",
  observe: "Compass",
  resources: "Database",
  sessions: "Send",
  settings: "Settings",
  skills: "FileCode",
  tasks: "Workflow",
  tools: "Plug",
  toolsets: "Layers",
  vault: "Key",
  workspaces: "FolderTree",
};

function iconForTitle(title: string): string | undefined {
  return TAG_ICONS[tagSlug(title)];
}

async function main(): Promise<void> {
  await cleanGenerated();
  await generateFiles({
    input: openapi,
    output: OUT_DIR,
    per: "tag",
    includeDescription: true,
    addGeneratedComment: true,
    frontmatter: (title, description) => {
      const frontmatter: Record<string, unknown> = {
        title,
        description: description ?? `AGH ${title} HTTP endpoints.`,
        full: true,
        _generated: "fumadocs-openapi",
      };
      const icon = iconForTitle(title);
      if (icon) frontmatter.icon = icon;
      return frontmatter;
    },
  });
  const usedTags = (await readUsedTags()).map(tagSlug);
  await writeMeta(new Set(usedTags));
}

main().catch(err => {
  console.error("[generate-openapi] failed:", err);
  process.exit(1);
});
