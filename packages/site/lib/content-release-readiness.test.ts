import { describe, expect, it } from "vitest";
import { listManualDocs, stripFencedCode } from "./content-test-utils";

const unfinishedContentPatterns = [
  /\bTODO\b/i,
  /\bTBD\b/i,
  /\bFIXME\b/i,
  /lorem ipsum/i,
  /coming soon/i,
  /under construction/i,
  /to be written/i,
];

describe("manual content release readiness", () => {
  it("does not ship unfinished documentation markers in manual pages", () => {
    const violations = listManualDocs().flatMap(doc =>
      stripFencedCode(doc.content)
        .split("\n")
        .flatMap((line, index) => {
          const pattern = unfinishedContentPatterns.find(candidate => candidate.test(line));
          if (!pattern) {
            return [];
          }
          return [`${doc.path}:${index + 1}: ${line.trim()}`];
        })
    );

    expect(violations).toEqual([]);
  });
});
