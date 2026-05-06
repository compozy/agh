import { existsSync, readdirSync, readFileSync, statSync } from "node:fs";
import { dirname, relative, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";

const siteRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..", "..");
const publicRoot = resolve(siteRoot, "public");
const checkedSourceRoots = ["app", "components", "content", "lib"].map(root =>
  resolve(siteRoot, root)
);
const checkedPublicTextFiles = ["favicon.svg", "site.webmanifest"].map(file =>
  resolve(publicRoot, file)
);
const publicAssetPattern =
  /\/(?:images|static|fonts)\/[A-Za-z0-9._~!$&*+,;=:@%/-]+\.(?:png|jpe?g|webp|gif|svg|ico|woff2?|ttf|mp4|webm)|\/(?:hero-bg\.webp|favicon\.svg|favicon\.ico|apple-touch-icon\.png|site\.webmanifest|icon-192\.png|icon-512\.png|install\.sh)\b/g;
const staleAccentToken = "#E857" + "2B";

function listFiles(dir: string): string[] {
  const files: string[] = [];
  for (const entry of readdirSync(dir)) {
    const fullPath = resolve(dir, entry);
    const stat = statSync(fullPath);
    if (stat.isDirectory()) {
      if (entry === "__tests__") {
        continue;
      }
      files.push(...listFiles(fullPath));
      continue;
    }
    if (
      stat.isFile() &&
      /\.(?:css|mdx?|tsx?|json|svg)$/.test(fullPath) &&
      !fullPath.endsWith(".test.ts") &&
      !fullPath.endsWith(".test.tsx")
    ) {
      files.push(fullPath);
    }
  }
  return files;
}

function sourceFiles(): string[] {
  return [...checkedSourceRoots.flatMap(root => listFiles(root)), ...checkedPublicTextFiles].sort(
    (left, right) => left.localeCompare(right)
  );
}

function publicPathExists(publicPath: string): boolean {
  return existsSync(resolve(publicRoot, publicPath.replace(/^\//, "")));
}

describe("site public assets", () => {
  it("keeps source references pointed at files under public", () => {
    const missing = sourceFiles().flatMap(file => {
      const content = readFileSync(file, "utf8");
      const paths = [...content.matchAll(publicAssetPattern)].map(match => match[0]);
      return [...new Set(paths)]
        .filter(path => !publicPathExists(path))
        .map(path => `${relative(siteRoot, file)} -> ${path}`);
    });

    expect(missing).toEqual([]);
  });

  it("keeps installable metadata aligned with the AGH design tokens", () => {
    const staleAccentRefs = sourceFiles().flatMap(file => {
      const content = readFileSync(file, "utf8");
      return content.includes(staleAccentToken) ? [relative(siteRoot, file)] : [];
    });
    const manifest = JSON.parse(readFileSync(resolve(publicRoot, "site.webmanifest"), "utf8")) as {
      icons: Array<{ src: string }>;
      theme_color: string;
      background_color: string;
    };
    const missingManifestIcons = manifest.icons
      .map(icon => icon.src)
      .filter(src => !publicPathExists(src));

    expect(staleAccentRefs).toEqual([]);
    expect(manifest.theme_color).toBe("#E8572A");
    expect(manifest.background_color).toBe("#141312");
    expect(missingManifestIcons).toEqual([]);
  });
});
