import { describe, expect, it } from "vitest";
import { listManualDocs, stripFencedCode } from "../content-test-utils";

type LinkText = {
  text: string;
  target: string;
};

const weakLinkText = new Set([
  "click here",
  "documentation",
  "docs",
  "here",
  "learn more",
  "link",
  "more",
  "read more",
  "this page",
]);

function normalizeText(text: string): string {
  return text
    .replace(/<[^>]+>/g, "")
    .replace(/[`*_]/g, "")
    .replace(/\s+/g, " ")
    .trim()
    .toLowerCase();
}

function markdownLinks(content: string): LinkText[] {
  return [...stripFencedCode(content).matchAll(/\[([^\]]*)\]\(([^)\s]+)(?:\s+"[^"]*")?\)/g)].map(
    match => ({
      text: normalizeText(match[1] ?? ""),
      target: match[2] ?? "",
    })
  );
}

function mdxAnchorLinks(content: string): LinkText[] {
  return [
    ...stripFencedCode(content).matchAll(/<a\b[^>]*href=["']([^"']+)["'][^>]*>([\s\S]*?)<\/a>/g),
  ].map(match => ({
    text: normalizeText(match[2] ?? ""),
    target: match[1] ?? "",
  }));
}

function linksFor(content: string): LinkText[] {
  return [...markdownLinks(content), ...mdxAnchorLinks(content)];
}

describe("manual content link text quality", () => {
  it("uses descriptive link text instead of generic calls to action", () => {
    const violations = listManualDocs().flatMap(doc =>
      linksFor(doc.content)
        .filter(link => weakLinkText.has(link.text))
        .map(link => `${doc.path}: weak link text "${link.text}" -> ${link.target}`)
    );

    expect(violations).toEqual([]);
  });

  it("does not use raw URLs as link text", () => {
    const violations = listManualDocs().flatMap(doc =>
      linksFor(doc.content)
        .filter(link => /^https?:\/\//i.test(link.text))
        .map(link => `${doc.path}: raw URL link text -> ${link.target}`)
    );

    expect(violations).toEqual([]);
  });

  it("does not publish empty manual link text", () => {
    const violations = listManualDocs().flatMap(doc =>
      linksFor(doc.content)
        .filter(link => link.text.length === 0)
        .map(link => `${doc.path}: empty link text -> ${link.target}`)
    );

    expect(violations).toEqual([]);
  });
});
