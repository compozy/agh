import { readdirSync, readFileSync, statSync } from "node:fs";
import { relative, resolve } from "node:path";
import { describe, expect, it } from "vitest";

import { siteRoot, stripFencedCode } from "./content-test-utils";

const publicSourceRoots = ["app", "components", "content", "lib"];
const publicSourceExtensions = [".md", ".mdx", ".ts", ".tsx"];
const ignoredPathSegments = ["/.source/", "/.velite/", "/node_modules/"];
const ignoredFilePatterns = [/\.test\.[cm]?[tj]sx?$/];

const burnedOutMarketingPhrases = [
  "AI-powered",
  "revolutionary",
  "game-changing",
  "next-generation",
  "supercharge",
  "unleash",
  "seamless",
  "effortless",
  "10x",
  "cutting-edge",
  "state-of-the-art",
  "state of the art",
];

const capabilitySynonyms = [
  "recipe",
  "recipes",
  "workflow",
  "workflows",
  "procedure",
  "procedures",
  "playbook",
  "playbooks",
];
const capabilityPattern = /\bcapabilit(?:y|ies)\b/i;
const capabilityBoundaryAllowedPattern = /\b(no|not|without|never)\b/i;
const staleCanonicalAgentExamplePatterns = [
  /\bClaude Code,\s*Codex\b/i,
  /\bClaude Code,\s*Gemini CLI\b/i,
  /\bClaude Code,\s*Pi\b/i,
  /\bCodex or Claude\b/i,
  /\bClaude or Codex\b/i,
];
const timeRelativeShippingClaimPatterns = [
  /\bshipping today\b/i,
  /\bavailable today\b/i,
  /\blive today\b/i,
  /\bworks? in main today\b/i,
];
const marketingDateRelativeClaimPatterns = [/\btoday\b/i];
const marketingInternalClaimPatterns = [
  /\bimplemented in[\s\S]{0,160}\bmain\b/i,
  /\blabel:\s*["']internal\//i,
];
const marketingAvailabilityClaimPatterns = [/\blive\b/i];

function publicSourceFiles(dir: string): string[] {
  const files: string[] = [];
  for (const entry of readdirSync(dir)) {
    const fullPath = resolve(dir, entry);
    const normalizedPath = fullPath.replaceAll("\\", "/");
    if (ignoredPathSegments.some(segment => normalizedPath.includes(segment))) {
      continue;
    }

    const stat = statSync(fullPath);
    if (stat.isDirectory()) {
      files.push(...publicSourceFiles(fullPath));
      continue;
    }

    if (
      stat.isFile() &&
      publicSourceExtensions.some(extension => fullPath.endsWith(extension)) &&
      !ignoredFilePatterns.some(pattern => pattern.test(fullPath))
    ) {
      files.push(fullPath);
    }
  }

  return files.sort();
}

function phrasePattern(phrase: string): RegExp {
  const escapedPhrase = phrase.replace(/[.*+?^${}()|[\]\\]/g, "\\$&").replaceAll("\\-", "[- ]");
  return new RegExp(`\\b${escapedPhrase}\\b`, "i");
}

function stripAccessibilityAttributes(content: string): string {
  return content.replace(/\saria-live=(?:"[^"]*"|'[^']*'|\{[^}]*\})/g, "");
}

describe("public copy quality", () => {
  const sourceFiles = publicSourceRoots.flatMap(root => publicSourceFiles(resolve(siteRoot, root)));
  const marketingFiles = sourceFiles.filter(file => {
    const relativePath = relative(siteRoot, file);
    return (
      relativePath.startsWith("app/(home)/") ||
      relativePath.startsWith("components/landing/") ||
      relativePath.startsWith("content/blog/")
    );
  });

  it("keeps burned-out marketing phrases out of public site copy", () => {
    const failures = sourceFiles.flatMap(file => {
      const relativePath = relative(siteRoot, file);
      const content = stripFencedCode(readFileSync(file, "utf8"));

      return burnedOutMarketingPhrases
        .filter(phrase => phrasePattern(phrase).test(content))
        .map(phrase => `${relativePath}: remove burned-out marketing phrase "${phrase}"`);
    });

    expect(failures).toEqual([]);
  });

  it("does not rename capabilities with forbidden synonyms", () => {
    const failures = sourceFiles.flatMap(file => {
      const relativePath = relative(siteRoot, file);
      const content = stripFencedCode(readFileSync(file, "utf8"));

      return content.split("\n").flatMap((line, index) => {
        const synonym = capabilitySynonyms.find(term => phrasePattern(term).test(line));
        if (
          !synonym ||
          !capabilityPattern.test(line) ||
          capabilityBoundaryAllowedPattern.test(line)
        ) {
          return [];
        }

        return `${relativePath}:${index + 1}: capability copy uses forbidden synonym "${synonym}"`;
      });
    });

    expect(failures).toEqual([]);
  });

  it("uses the canonical agent trio for public example lists", () => {
    const failures = sourceFiles.flatMap(file => {
      const relativePath = relative(siteRoot, file);
      const content = stripFencedCode(readFileSync(file, "utf8"));

      return staleCanonicalAgentExamplePatterns
        .filter(pattern => pattern.test(content))
        .map(pattern => `${relativePath}: replace stale agent example list ${pattern}`);
    });

    expect(failures).toEqual([]);
  });

  it("avoids time-relative shipping claims in release copy", () => {
    const failures = sourceFiles.flatMap(file => {
      const relativePath = relative(siteRoot, file);
      const content = stripFencedCode(readFileSync(file, "utf8"));

      return timeRelativeShippingClaimPatterns
        .filter(pattern => pattern.test(content))
        .map(pattern => `${relativePath}: replace time-relative shipping claim ${pattern}`);
    });

    expect(failures).toEqual([]);
  });

  it("avoids date-relative claims in marketing and release copy", () => {
    const failures = marketingFiles.flatMap(file => {
      const relativePath = relative(siteRoot, file);
      const content = stripFencedCode(readFileSync(file, "utf8"));

      return marketingDateRelativeClaimPatterns
        .filter(pattern => pattern.test(content))
        .map(pattern => `${relativePath}: replace date-relative release claim ${pattern}`);
    });

    expect(failures).toEqual([]);
  });

  it("keeps marketing proof labels reader-facing", () => {
    const failures = marketingFiles.flatMap(file => {
      const relativePath = relative(siteRoot, file);
      const content = stripFencedCode(readFileSync(file, "utf8"));

      return marketingInternalClaimPatterns
        .filter(pattern => pattern.test(content))
        .map(pattern => `${relativePath}: replace internal marketing proof ${pattern}`);
    });

    expect(failures).toEqual([]);
  });

  it("avoids broad live-availability claims in marketing and release copy", () => {
    const failures = marketingFiles.flatMap(file => {
      const relativePath = relative(siteRoot, file);
      const content = stripAccessibilityAttributes(stripFencedCode(readFileSync(file, "utf8")));

      return marketingAvailabilityClaimPatterns
        .filter(pattern => pattern.test(content))
        .map(pattern => `${relativePath}: replace broad live-availability claim ${pattern}`);
    });

    expect(failures).toEqual([]);
  });
});
