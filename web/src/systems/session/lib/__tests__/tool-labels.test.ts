import { describe, it, expect } from "vitest";
import { getToolIcon, getToolLabel, getToolCompactSummary } from "../tool-labels";
import { Terminal, FileText, FileEdit, Search, FolderSearch, Globe, Wrench } from "lucide-react";

describe("getToolIcon", () => {
  it("returns Terminal for Bash", () => {
    expect(getToolIcon("Bash")).toBe(Terminal);
  });

  it("returns FileText for Read", () => {
    expect(getToolIcon("Read")).toBe(FileText);
  });

  it("returns FileEdit for Write", () => {
    expect(getToolIcon("Write")).toBe(FileEdit);
  });

  it("returns FileEdit for Edit", () => {
    expect(getToolIcon("Edit")).toBe(FileEdit);
  });

  it("returns Search for Grep", () => {
    expect(getToolIcon("Grep")).toBe(Search);
  });

  it("returns FolderSearch for Glob", () => {
    expect(getToolIcon("Glob")).toBe(FolderSearch);
  });

  it("returns Globe for WebSearch", () => {
    expect(getToolIcon("WebSearch")).toBe(Globe);
  });

  it("returns fallback Wrench icon for unknown tool name", () => {
    expect(getToolIcon("SomeUnknownTool")).toBe(Wrench);
    expect(getToolIcon("")).toBe(Wrench);
  });

  it("uses semantic fallbacks for unknown tools based on tool input", () => {
    expect(getToolIcon("SomeUnknownTool", { command: "ls -la" })).toBe(Terminal);
    expect(getToolIcon("SomeUnknownTool", { file_path: "/tmp/file.txt" })).toBe(FileText);
    expect(getToolIcon("SomeUnknownTool", { filePath: "/tmp/file.txt" })).toBe(FileText);
    expect(getToolIcon("SomeUnknownTool", { pattern: "TODO" })).toBe(Search);
    expect(getToolIcon("SomeUnknownTool", { url: "https://example.com" })).toBe(Globe);
    expect(getToolIcon("SomeUnknownTool", { query: "search term" })).toBe(Globe);
    expect(getToolIcon("SomeUnknownTool", { other: true })).toBe(Wrench);
  });
});

describe("getToolLabel", () => {
  it("returns active label for known tools", () => {
    expect(getToolLabel("Read", "active")).toBe("Reading...");
    expect(getToolLabel("Bash", "active")).toBe("Running...");
    expect(getToolLabel("Edit", "active")).toBe("Editing...");
    expect(getToolLabel("Write", "active")).toBe("Writing...");
  });

  it("returns past label for known tools", () => {
    expect(getToolLabel("Read", "past")).toBe("Read file");
    expect(getToolLabel("Bash", "past")).toBe("Ran command");
    expect(getToolLabel("Grep", "past")).toBe("Searched content");
  });

  it("returns failure label for known tools", () => {
    expect(getToolLabel("Read", "failure")).toBe("read file");
    expect(getToolLabel("Bash", "failure")).toBe("run command");
    expect(getToolLabel("WebSearch", "failure")).toBe("search web");
  });

  it("returns fallback for unknown tool - active", () => {
    expect(getToolLabel("CustomTool", "active")).toBe("Running CustomTool...");
  });

  it("returns fallback for unknown tool - past", () => {
    expect(getToolLabel("CustomTool", "past")).toBe("Used CustomTool");
  });

  it("returns fallback for unknown tool - failure", () => {
    expect(getToolLabel("CustomTool", "failure")).toBe("use CustomTool");
  });
});

describe("getToolCompactSummary", () => {
  it("extracts command from Bash input", () => {
    expect(getToolCompactSummary("Bash", { command: "ls -la" })).toBe("ls -la");
  });

  it("extracts file_path from Read input", () => {
    expect(getToolCompactSummary("Read", { file_path: "/src/index.ts" })).toBe("/src/index.ts");
  });

  it("extracts pattern from Grep input", () => {
    expect(getToolCompactSummary("Grep", { pattern: "TODO|FIXME" })).toBe("TODO|FIXME");
  });

  it("extracts pattern from Glob input", () => {
    expect(getToolCompactSummary("Glob", { pattern: "**/*.ts" })).toBe("**/*.ts");
  });

  it("extracts query from WebSearch input", () => {
    expect(getToolCompactSummary("WebSearch", { query: "React hooks" })).toBe("React hooks");
  });

  it("truncates long strings", () => {
    const longCommand = "a".repeat(100);
    const result = getToolCompactSummary("Bash", { command: longCommand });
    expect(result!.length).toBeLessThanOrEqual(80);
    expect(result!.endsWith("\u2026")).toBe(true);
  });

  it("returns undefined for unknown tools", () => {
    expect(getToolCompactSummary("UnknownTool", { data: "stuff" })).toBeUndefined();
  });

  it("returns undefined when toolInput is undefined", () => {
    expect(getToolCompactSummary("Bash")).toBeUndefined();
  });
});
