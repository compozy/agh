import { existsSync, readdirSync, readFileSync, statSync } from "node:fs";
import { dirname, relative, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";

const siteRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..", "..");
const repoRoot = resolve(siteRoot, "../..");
const qaRoot = resolve(repoRoot, ".compozy/tasks/mem-v2/qa");
const testPlansRoot = resolve(qaRoot, "test-plans");
const testCasesRoot = resolve(qaRoot, "test-cases");

function readFile(path: string): string {
  return readFileSync(path, "utf8");
}

function listMarkdownFiles(dir: string): string[] {
  const files: string[] = [];
  for (const entry of readdirSync(dir)) {
    const fullPath = resolve(dir, entry);
    const stat = statSync(fullPath);
    if (stat.isDirectory()) {
      files.push(...listMarkdownFiles(fullPath));
      continue;
    }
    if (stat.isFile() && entry.endsWith(".md")) {
      files.push(fullPath);
    }
  }
  return files.sort((left, right) => left.localeCompare(right));
}

function readDossier(): string {
  return [...listMarkdownFiles(testPlansRoot), ...listMarkdownFiles(testCasesRoot)]
    .map(path => `\n--- ${relative(qaRoot, path)} ---\n${readFile(path)}`)
    .join("\n");
}

describe("Memory v2 QA artifacts", () => {
  it("ships the required plan, regression, traceability, and scenario files", () => {
    for (const path of [
      resolve(testPlansRoot, "memory-v2-test-plan.md"),
      resolve(testPlansRoot, "memory-v2-regression.md"),
      resolve(testPlansRoot, "memory-v2-traceability.md"),
      resolve(testCasesRoot, "TC-SCEN-001.md"),
      resolve(testCasesRoot, "TC-SCEN-002.md"),
      resolve(testCasesRoot, "TC-INT-001.md"),
      resolve(testCasesRoot, "TC-INT-002.md"),
      resolve(testCasesRoot, "TC-INT-003.md"),
      resolve(testCasesRoot, "TC-INT-004.md"),
      resolve(testCasesRoot, "TC-INT-005.md"),
      resolve(testCasesRoot, "TC-UI-001.md"),
      resolve(testCasesRoot, "TC-UI-002.md"),
      resolve(testCasesRoot, "TC-UI-003.md"),
      resolve(testCasesRoot, "TC-SEC-001.md"),
      resolve(testCasesRoot, "TC-REG-001.md"),
    ]) {
      expect(existsSync(path), `${relative(repoRoot, path)} should exist`).toBe(true);
    }
  });

  it("maps every completed implementation task and public Memory v2 surface", () => {
    const dossier = readDossier();

    for (let index = 1; index <= 24; index += 1) {
      const taskID = `task_${String(index).padStart(2, "0")}`;
      expect(dossier, `${taskID} should be traceable`).toContain(taskID);
    }

    for (const required of [
      "controller-backed write",
      "CLI",
      "HTTP",
      "UDS",
      "native tool",
      "extension host",
      "MemoryProvider",
      "workspace_id",
      "memory_decisions",
      "memory_events",
      "memory_recall_signals",
      "frozen snapshot",
      "_inbox",
      "_system/extractor/failures",
      "_system/dreaming",
      "ledger.jsonl",
      "Knowledge",
      "Memory Settings",
      "Session Inspector",
      "generated CLI/API",
      "config lifecycle",
    ]) {
      expect(dossier, `${required} should be covered`).toContain(required);
    }
  });

  it("promotes the search-visibility risk into an explicit P0 execution scenario", () => {
    const scenario = readFile(resolve(testCasesRoot, "TC-SCEN-001.md"));

    expect(scenario).toContain("Controller-Backed Write Is Searchable");
    expect(scenario).toContain("**Priority:** P0");
    expect(scenario).toContain("without reindex");
    expect(scenario).toContain("CLI");
    expect(scenario).toContain("UDS");
    expect(scenario).toContain("HTTP");
    expect(scenario).toContain("memory_decisions");
    expect(scenario).toContain("memory_events");
  });

  it("keeps cases execution-ready rather than thin shells", () => {
    const forbiddenTerms = [
      "TB" + "D",
      "TO" + "DO",
      "place" + "holder",
      "smoke" + "-only",
      "fr" + "aco",
    ];
    const thinShellPattern = new RegExp(`\\b(${forbiddenTerms.join("|")})\\b`, "i");

    for (const path of listMarkdownFiles(testCasesRoot)) {
      const content = readFile(path);
      const relPath = relative(qaRoot, path);

      expect(content, relPath).toContain("**Priority:**");
      expect(content, relPath).toContain("**Status:** Not Run");
      expect(content, relPath).toContain("## Preconditions");
      expect(content, relPath).toMatch(/\*\*Expected:\*\*/);
      expect(content, relPath).toMatch(/## (Required Evidence|Evidence To Capture)/);
      expect(content, relPath).not.toMatch(thinShellPattern);
    }
  });
});
