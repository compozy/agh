import { readdirSync, readFileSync } from "node:fs";
import { extname, join, relative } from "node:path";
import { describe, expect, it } from "vitest";

const WEB_SRC_ROOT = join(__dirname, "..");

const COMPONENT_NAME_PATTERN =
  /\b(?:Soul|Heartbeat|SessionHealth|HeartbeatStatus|HeartbeatWake)(?:Editor|Form|Composer|Settings|Panel|Inspector|Workbench|Builder)\b/;

const FILE_NAME_PATTERN =
  /(?:soul|heartbeat|session-health)-(?:editor|form|composer|settings|panel|inspector|workbench|builder)\.(?:tsx|ts)$/i;

const ALLOWED_FORBIDDEN_FILES = new Set<string>(["lib/agent-authored-context-no-ui.test.ts"]);

function walkSource(root: string): string[] {
  const collected: string[] = [];
  const stack = [root];
  while (stack.length > 0) {
    const current = stack.pop();
    if (current === undefined) {
      continue;
    }
    for (const entry of readdirSync(current, { withFileTypes: true })) {
      if (entry.name === "node_modules" || entry.name.startsWith(".")) {
        continue;
      }
      const absolute = join(current, entry.name);
      if (entry.isDirectory()) {
        stack.push(absolute);
        continue;
      }
      const ext = extname(entry.name);
      if (ext === ".ts" || ext === ".tsx") {
        collected.push(absolute);
      }
    }
  }
  return collected;
}

describe("authored context UI guard", () => {
  it("does not introduce Soul/Heartbeat editor or fake status control components", () => {
    const files = walkSource(WEB_SRC_ROOT);
    const violations: { file: string; reason: string }[] = [];
    for (const file of files) {
      const relativePath = relative(WEB_SRC_ROOT, file);
      if (ALLOWED_FORBIDDEN_FILES.has(relativePath)) {
        continue;
      }
      if (FILE_NAME_PATTERN.test(relativePath)) {
        violations.push({ file: relativePath, reason: "filename" });
        continue;
      }
      const content = readFileSync(file, "utf8");
      if (COMPONENT_NAME_PATTERN.test(content)) {
        violations.push({ file: relativePath, reason: "component-name" });
      }
    }
    expect(violations).toEqual([]);
  });
});
