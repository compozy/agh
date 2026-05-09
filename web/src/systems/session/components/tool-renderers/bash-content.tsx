import { useMemo } from "react";
import { ChevronsUpDown } from "lucide-react";
import { useState } from "react";

import { CodeBlock } from "@agh/ui";

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
      {!!command && <CodeBlock code={String(command)} copyable={false} truncateLines={4} />}
      {result && (
        <div>
          {result.stderr && (
            <CodeBlock code={result.stderr} copyable={false} showPrompt={false} tone="danger" />
          )}
          {displayText && (
            <CodeBlock code={displayText} copyable={false} showPrompt={false} truncateLines={20} />
          )}
          {isTruncated && (
            <button
              type="button"
              onClick={() => setExpanded(true)}
              className="mt-1 flex items-center gap-1 text-badge font-medium text-(--color-text-tertiary)/40 hover:text-(--color-text-tertiary)/70 transition-colors"
            >
              <ChevronsUpDown className="size-3" />
              Show full output ({totalLines} lines)
            </button>
          )}
          {expanded && totalLines > MAX_OUTPUT_LINES && (
            <button
              type="button"
              onClick={() => setExpanded(false)}
              className="mt-1 flex items-center gap-1 text-badge font-medium text-(--color-text-tertiary)/40 hover:text-(--color-text-tertiary)/70 transition-colors"
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
