import { readFileSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, it } from "vitest";

const PROVIDER_CARD_PATH = join(__dirname, "..", "provider-card.tsx");

describe("provider-card source", () => {
  const source = readFileSync(PROVIDER_CARD_PATH, "utf-8");

  it("Should not paint a hover accent ring", () => {
    expect(source).not.toMatch(/hover:ring-accent/);
  });

  it("Should hover on the canvas-soft surface tone", () => {
    expect(source).toMatch(/hover:bg-hover\b/);
  });
});
