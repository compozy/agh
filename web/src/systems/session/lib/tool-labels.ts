import {
  Terminal,
  FileText,
  FileEdit,
  Search,
  FolderSearch,
  Globe,
  Wrench,
  ListChecks,
  Lightbulb,
  Map,
  MessageCircleQuestion,
  PackageSearch,
  Sparkles,
  NotebookPen,
  Hammer,
} from "lucide-react";
import type { LucideIcon } from "lucide-react";
import type { UIMessage } from "../types";

// --- Tool Tone System ---

export type ToolTone = "tool" | "error" | "thinking" | "info";

const THINKING_TOOLS = new Set(["Think", "Agent", "Task"]);
const INFO_TOOLS = new Set(["EnterPlanMode", "ExitPlanMode", "TodoWrite", "ToolSearch", "Skill"]);

export function getToolTone(message: UIMessage): ToolTone {
  if (message.toolError) return "error";
  const name = message.toolName ?? "";
  if (THINKING_TOOLS.has(name)) return "thinking";
  if (INFO_TOOLS.has(name)) return "info";
  return "tool";
}

export function toolToneClass(tone: ToolTone): string {
  switch (tone) {
    case "error":
      return "text-danger/50";
    case "tool":
      return "text-subtle/70";
    case "thinking":
      return "text-subtle/50";
    case "info":
      return "text-subtle/40";
  }
}

// --- Tool Icons ---

const TOOL_ICONS: Record<string, LucideIcon> = {
  Bash: Terminal,
  Read: FileText,
  Write: FileEdit,
  Edit: FileEdit,
  Grep: Search,
  Glob: FolderSearch,
  WebSearch: Globe,
  WebFetch: Globe,
  Task: Hammer,
  Agent: Hammer,
  Think: Lightbulb,
  TodoWrite: ListChecks,
  NotebookEdit: NotebookPen,
  EnterPlanMode: Lightbulb,
  ExitPlanMode: Map,
  AskUserQuestion: MessageCircleQuestion,
  ToolSearch: PackageSearch,
  Skill: Sparkles,
};

/**
 * Resolve tool icon by name, with semantic fallbacks for unknown/MCP tools.
 */
export function getToolIcon(toolName: string, toolInput?: Record<string, unknown>): LucideIcon {
  const direct = TOOL_ICONS[toolName];
  if (direct) return direct;

  // Semantic fallbacks for unknown tools (MCP, dynamic, etc.)
  if (toolInput) {
    if ("command" in toolInput) return Terminal;
    if ("file_path" in toolInput || "filePath" in toolInput) return FileText;
    if ("pattern" in toolInput) return Search;
    if ("url" in toolInput || "query" in toolInput) return Globe;
  }

  return Wrench;
}

// --- Tool Labels ---

export type ToolLabelTense = "active" | "past" | "failure";

interface ToolLabels {
  active: string;
  past: string;
  failure: string;
}

const TOOL_LABELS: Record<string, ToolLabels> = {
  Bash: { active: "Running...", past: "Ran command", failure: "run command" },
  Read: { active: "Reading...", past: "Read file", failure: "read file" },
  Write: { active: "Writing...", past: "Wrote file", failure: "write file" },
  Edit: { active: "Editing...", past: "Edited file", failure: "edit file" },
  Grep: { active: "Searching...", past: "Searched content", failure: "search content" },
  Glob: { active: "Finding files...", past: "Found files", failure: "find files" },
  WebSearch: { active: "Searching web...", past: "Searched web", failure: "search web" },
  WebFetch: { active: "Fetching page...", past: "Fetched page", failure: "fetch page" },
  Task: { active: "Running task...", past: "Ran task", failure: "run task" },
  Agent: { active: "Running agent...", past: "Ran agent", failure: "run agent" },
  Think: { active: "Thinking...", past: "Thought", failure: "think" },
  TodoWrite: { active: "Updating tasks...", past: "Updated tasks", failure: "update tasks" },
  NotebookEdit: {
    active: "Editing notebook...",
    past: "Edited notebook",
    failure: "edit notebook",
  },
  EnterPlanMode: {
    active: "Entering plan mode...",
    past: "Entered plan mode",
    failure: "enter plan mode",
  },
  ExitPlanMode: {
    active: "Preparing plan...",
    past: "Presented plan",
    failure: "prepare plan",
  },
  AskUserQuestion: { active: "Asking...", past: "Asked question", failure: "ask question" },
  ToolSearch: { active: "Loading tools...", past: "Loaded tools", failure: "load tools" },
  Skill: { active: "Loading skill...", past: "Loaded skill", failure: "load skill" },
};

export function getToolLabel(toolName: string, tense: ToolLabelTense): string {
  const labels = TOOL_LABELS[toolName];
  if (labels) return labels[tense];

  // Fallback for unknown tools
  switch (tense) {
    case "active":
      return `Running ${toolName}...`;
    case "past":
      return `Used ${toolName}`;
    case "failure":
      return `use ${toolName}`;
  }
}

// --- Compact Summary Extractors ---

/**
 * Extract a short summary string from tool input for display in collapsed tool cards.
 * Returns undefined if no meaningful summary can be extracted.
 */
export function getToolCompactSummary(
  toolName: string,
  toolInput?: Record<string, unknown>
): string | undefined {
  const fullSummary = getToolFullSummary(toolName, toolInput);
  if (fullSummary === undefined) return undefined;

  return truncate(fullSummary, getToolSummaryMaxLength(toolName));
}

function truncate(str: string, maxLen: number): string {
  if (!str) return "";
  if (str.length <= maxLen) return str;
  return str.slice(0, maxLen - 1) + "\u2026";
}

export function getToolFullSummary(
  toolName: string,
  toolInput?: Record<string, unknown>
): string | undefined {
  if (!toolInput) return undefined;

  switch (toolName) {
    case "Bash":
      return String(toolInput.command ?? "");
    case "Read":
    case "Write":
    case "Edit":
      return String(toolInput.file_path ?? toolInput.filePath ?? "");
    case "Grep":
    case "Glob":
      return String(toolInput.pattern ?? "");
    case "WebSearch":
      return String(toolInput.query ?? "");
    case "WebFetch":
      return String(toolInput.url ?? "");
    default:
      return undefined;
  }
}

function getToolSummaryMaxLength(toolName: string): number {
  return toolName === "Bash" ? 80 : 60;
}
