import { describe, expect, it } from "vitest";
import { listManualDocs } from "../content-test-utils";

function mermaidBlocks(content: string): string[] {
  const blocks: string[] = [];
  const lines = content.split("\n");
  for (let index = 0; index < lines.length; index += 1) {
    if (!lines[index]?.includes("<Mermaid")) {
      continue;
    }

    const blockLines: string[] = [];
    for (let blockIndex = index; blockIndex < lines.length; blockIndex += 1) {
      blockLines.push(lines[blockIndex] ?? "");
      if (lines[blockIndex]?.trim() === "/>") {
        index = blockIndex;
        break;
      }
    }
    blocks.push(blockLines.join("\n"));
  }
  return blocks;
}

function textLikeBlocks(content: string): string[] {
  return [
    ...content.matchAll(
      /```(?:text|txt|ascii|plain|plaintext|output|console)\b[^\n]*\n([\s\S]*?)```/g
    ),
  ].map(match => match[1] ?? "");
}

describe("content diagram quality", () => {
  it("uses the themed Mermaid component instead of raw mermaid fences", () => {
    const rawFenceViolations = listManualDocs()
      .filter(doc => /```mermaid\b/.test(doc.content))
      .map(doc => doc.path);

    expect(rawFenceViolations).toEqual([]);
  });

  it("gives every manual Mermaid diagram an explanatory caption", () => {
    const missingCaptions = listManualDocs().flatMap(doc =>
      mermaidBlocks(doc.content)
        .filter(block => !/\bcaption=/.test(block))
        .map(() => doc.path)
    );

    expect(missingCaptions).toEqual([]);
  });

  it("keeps flow arrows out of raw text fences", () => {
    const rawFlowViolations = listManualDocs().flatMap(doc =>
      textLikeBlocks(doc.content)
        .filter(block => /(?:\w|`)[\w`]*\s*(?:->|=>)\s*(?:\w|`)/.test(block))
        .map(() => doc.path)
    );

    expect(rawFlowViolations).toEqual([]);
  });
});
