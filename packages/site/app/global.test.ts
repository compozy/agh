import { readFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";

const appDir = dirname(fileURLToPath(import.meta.url));
const globalCSS = readFileSync(resolve(appDir, "global.css"), "utf8");
const layoutSource = readFileSync(resolve(appDir, "layout.tsx"), "utf8");

describe("site global styles", () => {
  it("maps the display font token from a dedicated Playfair source variable", () => {
    expect(layoutSource).toContain('variable: "--font-playfair"');
    expect(globalCSS).toContain('--font-display: var(--font-playfair), "Playfair Display", serif;');
    expect(globalCSS).not.toContain("--font-display: var(--font-display),");
  });

  it("does not globally suppress box-shadow-based focus rings", () => {
    expect(globalCSS).not.toContain("box-shadow: none !important;");
  });

  it("limits inline code chrome to textual content instead of fenced code blocks", () => {
    expect(globalCSS).toContain(
      ".site-doc-body :is(p, li, td, th, blockquote, h1, h2, h3, h4, h5, h6) code {"
    );
    expect(globalCSS).not.toContain(".site-doc-body :not(pre) > code {");
  });
});
