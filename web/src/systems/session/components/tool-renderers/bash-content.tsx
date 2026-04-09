import { useMemo } from "react";
import { ChevronsUpDown } from "lucide-react";
import { useState } from "react";

import type { UIMessage } from "../../types";

const MAX_OUTPUT_LINES = 200;

/** Format non-stderr output (stderr is rendered separately with error styling). */
function formatBashOutput(result: NonNullable<UIMessage["toolResult"]>): string {
  const parts: string[] = [];
  if (result.stdout) parts.push(result.stdout);
  if (result.content && !result.stdout) parts.push(result.content);
  if (result.error) parts.push(result.error);
  return parts.join("\n");
}

export function BashContent({ message }: { message: UIMessage }) {
  const command = message.toolInput?.command;
  const result = message.toolResult;
  const [expanded, setExpanded] = useState(false);

  const formattedResult = useMemo(() => (result ? formatBashOutput(result) : ""), [result]);

  const { displayText, totalLines, isTruncated } = useMemo(() => {
    if (!formattedResult) return { displayText: "", totalLines: 0, isTruncated: false };
    const lines = formattedResult.split("\n");
    const total = lines.length;
    if (expanded || total <= MAX_OUTPUT_LINES) {
      return { displayText: formattedResult, totalLines: total, isTruncated: false };
    }
    return {
      displayText: lines.slice(0, MAX_OUTPUT_LINES).join("\n"),
      totalLines: total,
      isTruncated: true,
    };
  }, [formattedResult, expanded]);

  return (
    <div className="space-y-1.5 text-xs" data-testid="bash-content">
      {!!command && (
        <div className="rounded-md bg-[color:var(--color-surface)] px-3 py-2 font-mono text-[11px] whitespace-pre-wrap break-words">
          <span className="text-[color:var(--color-text-tertiary)]/40 select-none">$ </span>
          <span className="text-[color:var(--color-text-secondary)]">{String(command)}</span>
        </div>
      )}
      {result && (
        <div>
          {result.stderr && (
            <pre className="max-h-48 overflow-auto rounded-md bg-red-500/5 px-3 py-2 font-mono text-[11px] text-red-400/80 whitespace-pre-wrap break-words">
              {result.stderr}
            </pre>
          )}
          {displayText && (
            <pre className="max-h-48 overflow-auto rounded-md bg-[color:var(--color-surface)] px-3 py-2 font-mono text-[11px] text-[color:var(--color-text-tertiary)] whitespace-pre-wrap break-words">
              {displayText}
            </pre>
          )}
          {isTruncated && (
            <button
              type="button"
              onClick={() => setExpanded(true)}
              className="mt-1 flex items-center gap-1 text-[10px] font-medium text-[color:var(--color-text-tertiary)]/40 hover:text-[color:var(--color-text-tertiary)]/70 transition-colors"
            >
              <ChevronsUpDown className="size-3" />
              Show full output ({totalLines} lines)
            </button>
          )}
          {expanded && totalLines > MAX_OUTPUT_LINES && (
            <button
              type="button"
              onClick={() => setExpanded(false)}
              className="mt-1 flex items-center gap-1 text-[10px] font-medium text-[color:var(--color-text-tertiary)]/40 hover:text-[color:var(--color-text-tertiary)]/70 transition-colors"
            >
              <ChevronsUpDown className="size-3" />
              Collapse
            </button>
          )}
        </div>
      )}
    </div>
  );
}
