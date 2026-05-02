import { describe, expect, it } from "vitest";

import { listManualDocs, stripFencedCode } from "./content-test-utils";

const guideIntroPattern = /<OperatorNote\b/;
const nextStepsHeadingPattern = /^## Next steps\s*$/m;
const workflowPattern = /<Workflow\b/;
const mermaidPattern = /<Mermaid\b/;
const numberedStepHeadingPattern = /^## \d+\.\s+\S/gm;

const requiredUseCaseHeadings = ["Setup", "Flow", "Evidence to keep", "Failure path"];

function nonIndexDocs(prefix: string) {
  return listManualDocs([prefix]).filter(doc => !doc.path.endsWith("/index.mdx"));
}

function hasHeading(content: string, heading: string): boolean {
  const escapedHeading = heading.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
  return new RegExp(`^## ${escapedHeading}\\s*$`, "m").test(stripFencedCode(content));
}

function hasProceduralFlow(content: string): boolean {
  const body = stripFencedCode(content);
  return (
    workflowPattern.test(content) || [...body.matchAll(numberedStepHeadingPattern)].length >= 2
  );
}

function hasFlowModel(content: string): boolean {
  return workflowPattern.test(content) || mermaidPattern.test(content);
}

describe("runtime outcome documentation quality", () => {
  it("keeps runtime guides framed as problem-solving guides", () => {
    const failures = nonIndexDocs("runtime/guides/").flatMap(doc => {
      const checks = [
        {
          ok: guideIntroPattern.test(doc.content),
          message: "missing OperatorNote context block",
        },
        {
          ok: hasProceduralFlow(doc.content),
          message: "missing procedural flow",
        },
        {
          ok: nextStepsHeadingPattern.test(stripFencedCode(doc.content)),
          message: "missing Next steps section",
        },
      ];

      return checks.filter(check => !check.ok).map(check => `${doc.path}: ${check.message}`);
    });

    expect(failures).toEqual([]);
  });

  it("keeps runtime use cases outcome-oriented", () => {
    const failures = nonIndexDocs("runtime/use-cases/").flatMap(doc => {
      const headingFailures = requiredUseCaseHeadings
        .filter(heading => !hasHeading(doc.content, heading))
        .map(heading => `${doc.path}: missing ${heading} section`);

      if (!hasFlowModel(doc.content)) {
        headingFailures.push(`${doc.path}: missing flow model`);
      }

      return headingFailures;
    });

    expect(failures).toEqual([]);
  });
});
