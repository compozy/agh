import { readdirSync, readFileSync, statSync } from "node:fs";
import { dirname, relative, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";
import { PROVIDERS } from "@/components/landing/supported-agents";

const siteRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..", "..");
const repoRoot = resolve(siteRoot, "../..");
const landingRoot = resolve(siteRoot, "components/landing");
const runtimeRoot = resolve(siteRoot, "content/runtime");
const providerSourcePath = resolve(repoRoot, "internal/config/provider.go");

const deepCitationTargets = new Map([
  ["hooks catalog", "/runtime/core/hooks"],
  ["skills guide", "/runtime/core/skills"],
  ["automation", "/runtime/core/automation"],
  ["sandbox profiles", "/runtime/core/sandbox/profiles"],
  ["sessions lifecycle", "/runtime/core/sessions/lifecycle"],
  ["daemon surfaces", "/runtime/core/operations/daemon"],
  ["permissions", "/runtime/core/sessions/permissions"],
]);

function listFiles(dir: string, suffix: string): string[] {
  const files: string[] = [];
  for (const entry of readdirSync(dir)) {
    const fullPath = resolve(dir, entry);
    const stat = statSync(fullPath);
    if (stat.isDirectory()) {
      if (entry === "__tests__") {
        continue;
      }
      files.push(...listFiles(fullPath, suffix));
      continue;
    }
    if (stat.isFile() && fullPath.endsWith(suffix)) {
      files.push(fullPath);
    }
  }
  return files.sort();
}

function builtinProviderNames(): Map<string, string> {
  const source = readFileSync(providerSourcePath, "utf8");
  const mapSource =
    source.match(/var builtinProviders = map\[string\]ProviderConfig\{([\s\S]*?)\n\}/)?.[1] ?? "";
  const providers = new Map<string, string>();
  for (const match of mapSource.matchAll(/"([^"]+)":\s*\{[\s\S]*?DisplayName:\s*"([^"]+)"/g)) {
    providers.set(match[1] ?? "", match[2] ?? "");
  }
  return providers;
}

function runtimeRouteExists(route: string): boolean {
  const relativeRoute = route.replace(/^\/runtime\/?/, "");
  if (!relativeRoute) {
    return true;
  }
  return (
    statSync(resolve(runtimeRoot, `${relativeRoute}.mdx`), { throwIfNoEntry: false })?.isFile() ===
      true ||
    statSync(resolve(runtimeRoot, relativeRoute, "index.mdx"), {
      throwIfNoEntry: false,
    })?.isFile() === true
  );
}

describe("landing truth", () => {
  it("keeps provider names aligned with the runtime built-in registry", () => {
    const runtimeProviders = builtinProviderNames();
    const landingProviders = new Map(PROVIDERS.map(provider => [provider.id, provider.name]));

    expect(Object.fromEntries(landingProviders)).toEqual(Object.fromEntries(runtimeProviders));
  });

  it("does not imply v0 signature verification before the runtime verifies trust proofs", () => {
    const violations = listFiles(landingRoot, ".tsx").flatMap(file => {
      const source = readFileSync(file, "utf8");
      return [...source.matchAll(/\b(?:signed|verified identity|Ed25519)\b/gi)].map(
        match => `${relative(siteRoot, file)}: ${match[0]}`
      );
    });

    expect(violations).toEqual([]);
  });

  it("points landing source citations at the specific docs they name", () => {
    const violations = listFiles(landingRoot, ".tsx").flatMap(file => {
      const source = readFileSync(file, "utf8");
      return [...source.matchAll(/cite:\s*\{\s*href:\s*"([^"]+)",\s*label:\s*"([^"]+)"/g)]
        .map(match => ({
          href: match[1] ?? "",
          label: match[2] ?? "",
        }))
        .filter(cite => deepCitationTargets.has(cite.label))
        .filter(cite => cite.href !== deepCitationTargets.get(cite.label))
        .map(cite => `${relative(siteRoot, file)}: ${cite.label} -> ${cite.href}`);
    });
    const missingTargets = [...deepCitationTargets.values()]
      .filter(route => !runtimeRouteExists(route))
      .map(route => `missing route: ${route}`);

    expect([...violations, ...missingTargets]).toEqual([]);
  });
});
