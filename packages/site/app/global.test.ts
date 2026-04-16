import { readFileSync } from "node:fs";
import path from "node:path";
import { describe, expect, it } from "vitest";

const globalCSS = readFileSync(path.join(process.cwd(), "app", "global.css"), "utf8");
const layoutSource = readFileSync(path.join(process.cwd(), "app", "layout.tsx"), "utf8");

describe("site global styles", () => {
  it("maps the display font token from a dedicated Playfair source variable", () => {
    expect(layoutSource).toContain('variable: "--font-playfair"');
    expect(globalCSS).toContain('--font-display: var(--font-playfair), "Playfair Display", serif;');
    expect(globalCSS).not.toContain("--font-display: var(--font-display),");
  });

  it("does not globally suppress box-shadow-based focus rings", () => {
    expect(globalCSS).not.toContain("box-shadow: none !important;");
  });
});
