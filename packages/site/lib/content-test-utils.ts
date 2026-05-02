import { readdirSync, readFileSync, statSync } from "node:fs";
import { dirname, relative, resolve } from "node:path";
import { fileURLToPath } from "node:url";

export const siteRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..");
export const contentRoot = resolve(siteRoot, "content");
export const publicRoot = resolve(siteRoot, "public");

export type ManualDoc = {
  path: string;
  content: string;
};

export type FencedCodeBlock = {
  info: string;
  language: string;
  body: string;
};

const defaultManualDocPrefixes = ["runtime/", "protocol/", "blog/"];
const generatedPrefixes = ["runtime/cli-reference/", "runtime/api-reference/"];

function listMDXFiles(dir: string): string[] {
  const files: string[] = [];
  for (const entry of readdirSync(dir)) {
    const fullPath = resolve(dir, entry);
    const stat = statSync(fullPath);
    if (stat.isDirectory()) {
      files.push(...listMDXFiles(fullPath));
      continue;
    }
    if (stat.isFile() && fullPath.endsWith(".mdx")) {
      files.push(fullPath);
    }
  }
  return files.sort();
}

export function listManualDocs(prefixes: string[] = defaultManualDocPrefixes): ManualDoc[] {
  return listMDXFiles(contentRoot)
    .map(file => ({
      path: relative(contentRoot, file),
      content: readFileSync(file, "utf8"),
    }))
    .filter(doc => prefixes.some(prefix => doc.path.startsWith(prefix)))
    .filter(doc => !generatedPrefixes.some(prefix => doc.path.startsWith(prefix)));
}

export function stripFencedCode(content: string): string {
  return content.replace(/```[\s\S]*?```/g, "");
}

export function fencedCodeBlocks(content: string): FencedCodeBlock[] {
  return [...content.matchAll(/```([^\n]*)\n([\s\S]*?)```/g)].map(match => {
    const info = (match[1] ?? "").trim();
    return {
      info,
      language: info.split(/\s+/)[0] ?? "",
      body: match[2] ?? "",
    };
  });
}

export function mdxAttribute(tag: string, name: string): string | null {
  const escapedName = name.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
  return tag.match(new RegExp(`\\b${escapedName}=["']([^"']*)["']`))?.[1] ?? null;
}
