import { existsSync, readFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";

const siteRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const runtimeRoot = resolve(siteRoot, "content/runtime");
const protocolRoot = resolve(siteRoot, "content/protocol");

function readRuntimeJSON<T>(...parts: string[]): T {
  return JSON.parse(readFileSync(resolve(runtimeRoot, ...parts), "utf8")) as T;
}

function readProtocolJSON<T>(...parts: string[]): T {
  return JSON.parse(readFileSync(resolve(protocolRoot, ...parts), "utf8")) as T;
}

function runtimePageExists(...parts: string[]): boolean {
  return existsSync(resolve(runtimeRoot, ...parts));
}

function protocolPageExists(...parts: string[]): boolean {
  return existsSync(resolve(protocolRoot, ...parts));
}

describe("runtime docs discovery", () => {
  it("exposes orientation, guides, use cases, generated references, and core concepts from runtime meta", () => {
    const runtimeMeta = readRuntimeJSON<{ pages: string[] }>("meta.json");

    expect(runtimeMeta.pages).toEqual([
      "index",
      "how-to-use-these-docs",
      "core",
      "guides",
      "use-cases",
      "cli-reference",
      "api-reference",
    ]);
    expect(runtimePageExists("how-to-use-these-docs.mdx")).toBe(true);
  });

  it("keeps newly added guide and use-case sections discoverable", () => {
    const guidesMeta = readRuntimeJSON<{ pages: string[] }>("guides/meta.json");
    const useCasesMeta = readRuntimeJSON<{ pages: string[] }>("use-cases/meta.json");

    expect(guidesMeta.pages).toEqual([
      "index",
      "choose-an-operator-surface",
      "debug-a-failed-session",
      "coordinate-agents-over-network",
    ]);
    expect(useCasesMeta.pages).toEqual([
      "index",
      "prepare-a-project-workspace",
      "review-a-change",
      "release-readiness-sweep",
      "handoff-between-agents",
    ]);

    for (const page of guidesMeta.pages) {
      expect(runtimePageExists("guides", `${page}.mdx`)).toBe(true);
    }
    for (const page of useCasesMeta.pages) {
      expect(runtimePageExists("use-cases", `${page}.mdx`)).toBe(true);
    }
  });

  it("keeps resources and tools reachable from core concepts", () => {
    const coreMeta = readRuntimeJSON<{ pages: string[] }>("core/meta.json");
    const resourcesMeta = readRuntimeJSON<{ pages: string[] }>("core/resources/meta.json");
    const toolsMeta = readRuntimeJSON<{ pages: string[] }>("core/tools/meta.json");

    expect(coreMeta.pages).toContain("resources");
    expect(coreMeta.pages).toContain("tools");
    expect(resourcesMeta.pages).toEqual(["index", "bundles"]);
    expect(toolsMeta.pages).toEqual(["index", "toolsets", "policy-and-invocation"]);
  });

  it("keeps protocol implementation status reachable from protocol meta", () => {
    const protocolMeta = readProtocolJSON<{ pages: string[] }>("meta.json");

    expect(protocolMeta.pages).toContain("implementation-status");
    expect(protocolPageExists("implementation-status.mdx")).toBe(true);
  });
});
