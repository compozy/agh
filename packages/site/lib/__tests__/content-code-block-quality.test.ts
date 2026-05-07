import { describe, expect, it } from "vitest";
import { fencedCodeBlocks, listManualDocs } from "../content-test-utils";

const shellLanguages = new Set(["bash", "sh", "shell", "zsh"]);
const supportedHighlightLanguages = new Set([
  "bash",
  "dotenv",
  "go",
  "http",
  "ini",
  "js",
  "json",
  "markdown",
  "md",
  "sql",
  "text",
  "toml",
  "ts",
  "xml",
  "yaml",
]);
const shellCommandPatterns = [
  /^(agh|curl|go|bun|npm|pnpm|yarn|cat|mkdir|tail|mv)\s+/,
  /^command\s+-v\s+/,
  /^export\s+[A-Z_][A-Z0-9_]*=/,
];

function hasCopyPasteableShellCommand(body: string): boolean {
  return body
    .split("\n")
    .some(line => shellCommandPatterns.some(pattern => pattern.test(line.trimEnd())));
}

describe("manual content code block quality", () => {
  it("labels every manual fenced code block with a language", () => {
    const violations = listManualDocs().flatMap(doc =>
      fencedCodeBlocks(doc.content)
        .filter(block => block.language.length === 0)
        .map(() => `${doc.path}: unlabeled fenced code block`)
    );

    expect(violations).toEqual([]);
  });

  it("uses known highlighting languages for manual fenced code blocks", () => {
    const violations = listManualDocs().flatMap(doc =>
      fencedCodeBlocks(doc.content)
        .filter(block => block.language.length > 0)
        .filter(block => !supportedHighlightLanguages.has(block.language))
        .map(block => `${doc.path}: unsupported code block language ${block.language}`)
    );

    expect(violations).toEqual([]);
  });

  it("uses shell highlighting for copy-pasteable shell commands", () => {
    const violations = listManualDocs().flatMap(doc =>
      fencedCodeBlocks(doc.content)
        .filter(block => hasCopyPasteableShellCommand(block.body))
        .filter(block => !shellLanguages.has(block.language))
        .map(block => `${doc.path}: shell command is labelled as ${block.language}`)
    );

    expect(violations).toEqual([]);
  });
});
