import { readdirSync, readFileSync, statSync } from "node:fs";
import { dirname, relative, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";

const siteRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const repoRoot = resolve(siteRoot, "../..");
const contentRoot = resolve(siteRoot, "content");

type ManualDoc = {
  path: string;
  content: string;
};

function readRepoFile(...parts: string[]): string {
  return readFileSync(resolve(repoRoot, ...parts), "utf8");
}

function listManualDocs(dir: string): ManualDoc[] {
  const docs: ManualDoc[] = [];
  for (const entry of readdirSync(dir)) {
    const fullPath = resolve(dir, entry);
    const relPath = relative(contentRoot, fullPath);
    if (
      relPath === "runtime/cli-reference" ||
      relPath.startsWith("runtime/cli-reference/") ||
      relPath === "runtime/api-reference" ||
      relPath.startsWith("runtime/api-reference/")
    ) {
      continue;
    }

    const stat = statSync(fullPath);
    if (stat.isDirectory()) {
      docs.push(...listManualDocs(fullPath));
      continue;
    }
    if (stat.isFile() && fullPath.endsWith(".mdx")) {
      docs.push({ path: relPath, content: readFileSync(fullPath, "utf8") });
    }
  }
  return docs.sort((left, right) => left.path.localeCompare(right.path));
}

function manualContent(): string {
  return listManualDocs(contentRoot)
    .map(doc => `\n--- ${doc.path} ---\n${doc.content}`)
    .join("\n");
}

function extractGoStringConstants(source: string, typeName: string): Set<string> {
  const constants = new Set<string>();
  const matcher = new RegExp(`\\b\\w+\\s+${typeName}\\s*=\\s*"([^"]+)"`, "g");
  for (const match of source.matchAll(matcher)) {
    constants.add(match[1] ?? "");
  }
  return constants;
}

describe("runtime docs truth", () => {
  it("uses the canonical MCP server resource kind from the runtime codec", () => {
    const mcpResourceSource = readRepoFile("internal/config/mcp_resource.go");
    const resourceDoc = readRepoFile("packages/site/content/runtime/core/resources/index.mdx");
    const kindMatch = mcpResourceSource.match(
      /MCPServerResourceKind\s+resources\.ResourceKind\s*=\s*"([^"]+)"/
    );

    expect(kindMatch?.[1]).toBe("mcp_server");
    expect(resourceDoc).toContain("`mcp_server`");
    expect(resourceDoc).not.toContain("`mcp.server`");
  });

  it("documents resource validation failures with the status used by the API error mapper", () => {
    const errorSource = readRepoFile("internal/api/core/errors.go");
    const resourceDoc = readRepoFile("packages/site/content/runtime/core/resources/index.mdx");

    expect(errorSource).toContain("StatusForResourceError");
    expect(errorSource).toContain("http.StatusUnprocessableEntity");
    expect(resourceDoc).toContain("| `400` on write");
    expect(resourceDoc).toContain("malformed JSON");
    expect(resourceDoc).toContain("| `422` on write");
    expect(resourceDoc).toContain(
      "Invalid kind, scope binding, missing codec, or codec spec validation"
    );
    expect(resourceDoc).not.toMatch(/\| `400` on write\s+\|\s+Invalid kind/);
  });

  it("does not route session SSE examples through the replay events endpoint", () => {
    const content = manualContent().replaceAll("\\\n", " ");

    expect(content).not.toMatch(/curl\s+-N\b[\s\S]{0,240}\/api\/sessions\/[^/\s]+\/events\b/);
    expect(content).toContain("/api/sessions/sess_1234/stream");
  });

  it("declares the API reference as built from the canonical OpenAPI spec on every site build", () => {
    const content = manualContent();
    const apiReference = readRepoFile("packages/site/content/runtime/api-reference/index.mdx");

    expect(apiReference).toMatch(/built from\s+`openapi\/agh\.json`/);
    expect(apiReference).toContain("make codegen-check");
    expect(apiReference).not.toMatch(
      /does not yet cover every implemented\s+streaming and bundle route/
    );
    expect(content).toContain("The API route map lists the implemented route families");
  });

  it("keeps concrete tool invocation examples tied to compiled builtin tool IDs", () => {
    const toolSource = readRepoFile("internal/tools/builtin_ids.go");
    const builtinToolIDs = extractGoStringConstants(toolSource, "ToolID");
    const content = manualContent();
    const concreteInvocations = [...content.matchAll(/\bagh tool invoke\s+(agh__[a-z0-9_]+)/g)].map(
      match => match[1] ?? ""
    );

    expect(content).not.toContain("agh__example_tool");
    expect(concreteInvocations.length).toBeGreaterThan(0);
    expect(concreteInvocations.filter(id => !builtinToolIDs.has(id))).toEqual([]);
  });
});
