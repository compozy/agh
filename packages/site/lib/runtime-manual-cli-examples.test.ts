import { readdirSync, readFileSync, statSync } from "node:fs";
import { dirname, relative, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";

const siteRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const contentRoot = resolve(siteRoot, "content");
const cliReferenceRoot = resolve(contentRoot, "runtime/cli-reference");

type ManualDoc = {
  path: string;
  content: string;
};

type CodeBlock = {
  language: string;
  body: string;
};

function listManualDocs(dir: string): ManualDoc[] {
  const docs: ManualDoc[] = [];
  for (const entry of readdirSync(dir)) {
    const fullPath = resolve(dir, entry);
    const relPath = relative(contentRoot, fullPath);
    if (relPath === "runtime/cli-reference" || relPath.startsWith("runtime/cli-reference/")) {
      continue;
    }

    const stat = statSync(fullPath);
    if (stat.isDirectory()) {
      docs.push(...listManualDocs(fullPath));
      continue;
    }
    if (stat.isFile() && fullPath.endsWith(".mdx")) {
      docs.push({
        path: relPath,
        content: readFileSync(fullPath, "utf8"),
      });
    }
  }
  return docs.sort((left, right) => left.path.localeCompare(right.path));
}

function extractBashBlocks(doc: ManualDoc): string[] {
  const blocks: string[] = [];
  const matcher = /```(?:bash|sh|shell)\n([\s\S]*?)```/g;
  for (const match of doc.content.matchAll(matcher)) {
    blocks.push(match[1] ?? "");
  }
  return blocks;
}

function extractCodeBlocks(doc: ManualDoc): CodeBlock[] {
  const blocks: CodeBlock[] = [];
  const matcher = /```([^\n]*)\n([\s\S]*?)```/g;
  for (const match of doc.content.matchAll(matcher)) {
    blocks.push({
      language: (match[1] ?? "").trim(),
      body: match[2] ?? "",
    });
  }
  return blocks;
}

function commandBlocks(command: string): Array<{ path: string; block: string }> {
  return listManualDocs(contentRoot).flatMap(doc =>
    extractBashBlocks(doc)
      .filter(block => block.includes(command))
      .map(block => ({ path: doc.path, block }))
  );
}

function listCLIReferenceDocs(dir: string): string[] {
  const docs: string[] = [];
  for (const entry of readdirSync(dir)) {
    const fullPath = resolve(dir, entry);
    const stat = statSync(fullPath);
    if (stat.isDirectory()) {
      docs.push(...listCLIReferenceDocs(fullPath));
      continue;
    }
    if (stat.isFile() && fullPath.endsWith(".mdx")) {
      docs.push(readFileSync(fullPath, "utf8"));
    }
  }
  return docs;
}

function generatedCLICommands(): Set<string> {
  const commands = new Set<string>();
  for (const doc of listCLIReferenceDocs(cliReferenceRoot)) {
    const match = doc.match(/^## (agh(?: [^\n]+)?)/m);
    if (match?.[1]) {
      commands.add(match[1].trim());
    }
  }
  return commands;
}

function stalePatternViolations(pattern: RegExp): string[] {
  return listManualDocs(contentRoot)
    .filter(doc => pattern.test(doc.content))
    .map(doc => doc.path);
}

function extractManualAghCommandPrefixes(line: string, generatedCommands: Set<string>): string[] {
  const commands: string[] = [];
  const tokens = line
    .replace(/^[\s$>]+/, "")
    .split(/\s+/)
    .map(token => token.replace(/^[("'`]+|[)"'`,;]+$/g, ""))
    .filter(Boolean);

  for (let index = 0; index < tokens.length; index += 1) {
    if (tokens[index] !== "agh") {
      continue;
    }

    let longest = "";
    for (let end = index + 1; end <= tokens.length; end += 1) {
      const candidate = tokens.slice(index, end).join(" ");
      if (generatedCommands.has(candidate)) {
        longest = candidate;
      }
    }
    commands.push(longest);
  }
  return commands;
}

function manualAghCommandViolations(): string[] {
  const generatedCommands = generatedCLICommands();
  return listManualDocs(contentRoot).flatMap(doc =>
    extractBashBlocks(doc).flatMap(block =>
      block
        .replaceAll("\\\n", " ")
        .split("\n")
        .flatMap(line =>
          extractManualAghCommandPrefixes(line, generatedCommands).map(command => ({
            command,
            line,
          }))
        )
        .filter(({ command }) => command === "")
        .map(({ line }) => `${doc.path}: ${line.trim()}`)
    )
  );
}

describe("manual site CLI examples", () => {
  it("labels manual shell examples that contain agh commands", () => {
    const shellLanguages = new Set(["bash", "sh", "shell"]);
    const violations = listManualDocs(contentRoot).flatMap(doc =>
      extractCodeBlocks(doc)
        .filter(block => /^\s*(?:[$>]\s*)?agh(?:\s|$)/m.test(block.body))
        .filter(block => !shellLanguages.has(block.language))
        .map(block => `${doc.path}: ${block.language || "<unlabeled>"}`)
    );

    expect(violations).toEqual([]);
  });

  it("uses command names that exist in the generated CLI reference", () => {
    expect(manualAghCommandViolations()).toEqual([]);
  });

  it("does not document stale command forms that are not implemented by cobra", () => {
    expect(stalePatternViolations(/\bagh session get\b/)).toEqual([]);
    expect(stalePatternViolations(/\bagh network peers\s+--channel\b/)).toEqual([]);
    expect(stalePatternViolations(/\bagh spawn\b[\s\S]{0,240}--prompt(?!-overlay)\b/)).toEqual([]);
  });

  it("uses the implemented flag shape for network send examples", () => {
    const violations = commandBlocks("agh network send")
      .filter(({ block }) => {
        const normalized = block.replaceAll("\\\n", " ");
        return (
          !normalized.includes("--session ") ||
          !normalized.includes("--channel ") ||
          !normalized.includes("--kind ") ||
          !normalized.includes("--body ")
        );
      })
      .map(({ path }) => path);

    expect(violations).toEqual([]);
  });

  it("uses the implemented flag shape for network inbox examples", () => {
    const violations = commandBlocks("agh network inbox")
      .filter(({ block }) => !block.includes("--session "))
      .map(({ path }) => path);

    expect(violations).toEqual([]);
  });

  it("keeps manual spawn examples explicit about bounded child session TTL", () => {
    const violations = commandBlocks("agh spawn")
      .filter(({ block }) => !block.includes("--ttl-seconds "))
      .map(({ path }) => path);

    expect(violations).toEqual([]);
  });
});
