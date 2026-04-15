import { useState } from "react";
import { ChevronsUpDown } from "lucide-react";

import type { UIMessage } from "../../types";
import { GenericContent } from "./generic-content";

const TRUNCATE_THRESHOLD = 2000;

export function WriteContent({ message }: { message: UIMessage }) {
  const [showFull, setShowFull] = useState(false);
  const filePath = String(
    message.toolInput?.file_path ??
      message.toolInput?.filePath ??
      message.toolResult?.filePath ??
      ""
  );
  const content = String(message.toolInput?.content ?? message.toolResult?.content ?? "");
  const isTruncated = !showFull && content.length > TRUNCATE_THRESHOLD;
  const displayContent = showFull ? content : content.slice(0, TRUNCATE_THRESHOLD);

  if (!filePath && !content) {
    return <GenericContent message={message} />;
  }

  return (
    <div className="space-y-1.5 text-xs" data-testid="write-content">
      {filePath && (
        <div className="font-mono text-[11px] text-[color:var(--color-text-tertiary)]">
          {filePath}
        </div>
      )}
      {content && (
        <pre className="max-h-48 overflow-auto rounded-md bg-[color:var(--color-surface)] px-3 py-2 font-mono text-[11px] text-[color:var(--color-text-tertiary)] whitespace-pre-wrap break-words">
          {displayContent}
          {isTruncated ? "\u2026" : ""}
        </pre>
      )}
      {isTruncated && (
        <button
          type="button"
          onClick={() => setShowFull(true)}
          className="flex items-center gap-1 text-[11px] text-[color:var(--color-text-tertiary)] hover:text-[color:var(--color-text-secondary)] transition-colors"
        >
          <ChevronsUpDown className="size-3" />
          Show full content ({content.length.toLocaleString()} chars)
        </button>
      )}
    </div>
  );
}
