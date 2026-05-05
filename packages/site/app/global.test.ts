import { readFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";

const appDir = dirname(fileURLToPath(import.meta.url));
const globalCSS = readFileSync(resolve(appDir, "global.css"), "utf8");
const layoutSource = readFileSync(resolve(appDir, "layout.tsx"), "utf8");

describe("site global styles", () => {
  it("maps the display font token from a dedicated Playfair source variable without swap-based first paint", () => {
    expect(layoutSource).toContain('variable: "--font-playfair"');
    expect(layoutSource).toMatch(
      /const playfairDisplay = Playfair_Display\(\{[\s\S]*?display: "block"[\s\S]*?weight: \["400", "500"\]/
    );
    expect(globalCSS).toContain('--font-display: var(--font-playfair), "Playfair Display", serif;');
    expect(globalCSS).not.toContain("--font-display: var(--font-display),");
  });

  it("configures the default Fumadocs search dialog to fetch live results from /api/search", () => {
    expect(layoutSource).toMatch(
      /<RootProvider[\s\S]*search=\{\{[\s\S]*type: "fetch"[\s\S]*api: "\/api\/search"[\s\S]*\}\}/
    );
  });

  it("does not carry an unused site-local wordmark font-face declaration", () => {
    expect(globalCSS).not.toContain('@font-face {\n  font-family: "NuixyberNext";');
  });

  it("does not globally suppress box-shadow-based focus rings", () => {
    expect(globalCSS).not.toContain("box-shadow: none !important;");
  });

  it("maps Fumadocs semantic tokens to the AGH signal palette in dark mode", () => {
    expect(globalCSS).toContain("--color-fd-info: var(--color-info);");
    expect(globalCSS).toContain("--color-fd-warning: var(--color-warning);");
    expect(globalCSS).toContain("--color-fd-error: var(--color-danger);");
    expect(globalCSS).toContain("--color-fd-success: var(--color-success);");
    expect(globalCSS).toContain("--color-fd-idea: var(--color-accent);");
    expect(globalCSS).toContain("--color-fd-diff-remove: var(--color-danger-tint);");
    expect(globalCSS).toContain("--color-fd-diff-add: var(--color-success-tint);");
  });

  it("limits inline code chrome to textual content instead of fenced code blocks", () => {
    expect(globalCSS).toContain(
      ".site-doc-body :is(p, li, td, th, blockquote, h1, h2, h3, h4, h5, h6) code {"
    );
    expect(globalCSS).not.toContain(".site-doc-body :not(pre) > code {");
  });
});
