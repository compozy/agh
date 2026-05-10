import { FileText } from "lucide-react";

import type { UIMessage } from "../../types";

function shortenPath(filePath: string): string {
  const parts = filePath.split("/");
  if (parts.length <= 3) return filePath;
  return parts.slice(-3).join("/");
}

export function SearchContent({ message }: { message: UIMessage }) {
  const pattern = String(message.toolInput?.pattern ?? "");
  const glob = message.toolInput?.glob ? String(message.toolInput.glob) : "";
  const path = message.toolInput?.path ? String(message.toolInput.path) : "";
  const result = message.toolResult;

  const resultText = result?.stdout ?? result?.content ?? "";
  const lines = resultText ? resultText.split("\n").filter(Boolean) : [];

  return (
    <div className="space-y-1.5 text-xs" data-testid="search-content">
      {pattern && (
        <div className="font-mono text-eyebrow text-(--subtle)">
          {pattern}
          {glob && <span className="text-(--subtle)/30 ms-1.5">in {glob}</span>}
          {!glob && path && (
            <span className="text-(--subtle)/30 ms-1.5">in {shortenPath(path)}</span>
          )}
        </div>
      )}
      {lines.length > 0 ? (
        <div className="rounded-md border border-(--line) overflow-hidden">
          {lines.slice(0, 20).map(line => (
            <div
              key={line}
              className="flex items-center gap-2 border-t border-(--line) px-3 py-1 text-eyebrow first:border-t-0"
            >
              <FileText className="size-3 shrink-0 text-(--subtle)/20" />
              <span className="truncate text-(--subtle) font-mono" title={line}>
                {shortenPath(line)}
              </span>
            </div>
          ))}
          {lines.length > 20 && (
            <div className="px-3 py-1 border-t border-(--line) text-badge text-(--subtle)/40">
              +{lines.length - 20} more
            </div>
          )}
        </div>
      ) : result ? (
        <span className="text-badge text-(--subtle)/30 italic">No matches</span>
      ) : null}
    </div>
  );
}
