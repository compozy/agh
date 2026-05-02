import { describe, expect, it } from "vitest";
import { listManualDocs, stripFencedCode } from "./content-test-utils";

const localHttpHosts = new Set(["localhost", "127.0.0.1", "::1"]);

function externalUrls(content: string): string[] {
  return [...stripFencedCode(content).matchAll(/\bhttps?:\/\/[^\s<>"'`)]+/g)]
    .map(match => match[0].replace(/[.,;:]+$/g, ""))
    .sort();
}

function proseBareUrls(content: string): string[] {
  const prose = stripFencedCode(content)
    .replace(/`[^`\n]*`/g, "")
    .replace(/\[[^\]]*\]\([^)]*\)/g, "")
    .replace(/\b(?:href|src)=["'][^"']*["']/g, "");
  return [...prose.matchAll(/\bhttps?:\/\/[^\s<>"'`)]+/g)]
    .map(match => match[0].replace(/[.,;:]+$/g, ""))
    .sort();
}

describe("manual content external links", () => {
  it("uses HTTPS for external URLs and reserves HTTP for local daemon examples", () => {
    const violations = listManualDocs().flatMap(doc =>
      externalUrls(doc.content).flatMap(url => {
        const parsed = new URL(url);
        if (parsed.protocol !== "http:") {
          return [];
        }
        if (localHttpHosts.has(parsed.hostname)) {
          return [];
        }
        return [`${doc.path}: ${url} uses http:// outside a local daemon example`];
      })
    );

    expect(violations).toEqual([]);
  });

  it("formats prose URLs as links or inline code examples", () => {
    const violations = listManualDocs().flatMap(doc =>
      proseBareUrls(doc.content).map(url => `${doc.path}: bare prose URL ${url}`)
    );

    expect(violations).toEqual([]);
  });
});
