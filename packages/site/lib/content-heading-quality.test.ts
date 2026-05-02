import { describe, expect, it } from "vitest";
import { listManualDocs, stripFencedCode } from "./content-test-utils";

type Heading = {
  level: number;
  text: string;
  anchor: string;
};

function slugifyHeading(heading: string): string {
  return heading
    .replace(/<[^>]+>/g, "")
    .replace(/`([^`]+)`/g, "$1")
    .toLowerCase()
    .trim()
    .replace(/[^\p{L}\p{N}\s-]/gu, "")
    .replace(/\s+/g, "-")
    .replace(/^-|-$/g, "");
}

function headingsFor(content: string): Heading[] {
  return [...stripFencedCode(content).matchAll(/^(#{1,6})\s+(.+)$/gm)].map(match => {
    const text = (match[2] ?? "").replace(/\s+#+\s*$/, "").trim();
    return {
      level: match[1]?.length ?? 0,
      text,
      anchor: slugifyHeading(text),
    };
  });
}

describe("manual content heading quality", () => {
  it("uses frontmatter titles instead of manual h1 headings", () => {
    const violations = listManualDocs().flatMap(doc =>
      headingsFor(doc.content)
        .filter(heading => heading.level === 1)
        .map(heading => `${doc.path}: h1 heading ${heading.text}`)
    );

    expect(violations).toEqual([]);
  });

  it("does not skip heading levels", () => {
    const violations = listManualDocs().flatMap(doc => {
      const headings = headingsFor(doc.content);
      return headings.slice(1).flatMap((heading, index) => {
        const previous = headings[index];
        if (!previous || heading.level <= previous.level + 1) {
          return [];
        }
        return [
          `${doc.path}: ${previous.text} h${previous.level} -> ${heading.text} h${heading.level}`,
        ];
      });
    });

    expect(violations).toEqual([]);
  });

  it("keeps generated heading anchors unique within each manual page", () => {
    const violations = listManualDocs().flatMap(doc => {
      const seen = new Map<string, number>();
      for (const heading of headingsFor(doc.content)) {
        seen.set(heading.anchor, (seen.get(heading.anchor) ?? 0) + 1);
      }
      return [...seen.entries()]
        .filter(([, count]) => count > 1)
        .map(([anchor, count]) => `${doc.path}: duplicate #${anchor} (${count})`);
    });

    expect(violations).toEqual([]);
  });
});
