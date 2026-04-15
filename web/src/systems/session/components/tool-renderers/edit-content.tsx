import { useState } from "react";
import { ChevronsUpDown } from "lucide-react";

import type { UIMessage } from "../../types";
import { GenericContent } from "./generic-content";

const TRUNCATE_THRESHOLD = 1500;

export function EditContent({ message }: { message: UIMessage }) {
  const [showFull, setShowFull] = useState(false);
  const filePath = String(
    message.toolInput?.file_path ??
      message.toolInput?.filePath ??
      message.toolResult?.filePath ??
      ""
  );
  const rawOld = message.toolInput?.old_string;
  const rawNew = message.toolInput?.new_string;
  const oldStr = rawOld != null ? String(rawOld) : "";
  const newStr = rawNew != null ? String(rawNew) : "";
  const isTruncated =
    !showFull && (oldStr.length > TRUNCATE_THRESHOLD || newStr.length > TRUNCATE_THRESHOLD);

  if (!filePath && !oldStr && !newStr) {
    return <GenericContent message={message} />;
  }

  const displayOld = showFull ? oldStr : oldStr.slice(0, TRUNCATE_THRESHOLD);
  const displayNew = showFull ? newStr : newStr.slice(0, TRUNCATE_THRESHOLD);

  return (
    <div className="space-y-1.5 text-xs" data-testid="edit-content">
      {filePath && (
        <div className="font-mono text-[11px] text-[color:var(--color-text-tertiary)]">
          {filePath}
        </div>
      )}
      {(oldStr || newStr) && (
        <div className="rounded-md border border-[color:var(--color-divider)] overflow-hidden font-mono text-[11px]">
          {oldStr ? (
            <pre className="bg-red-500/5 px-3 py-2 text-red-400/70 whitespace-pre-wrap break-words max-h-40 overflow-auto">
              {displayOld}
              {!showFull && oldStr.length > TRUNCATE_THRESHOLD ? "\u2026" : ""}
            </pre>
          ) : null}
          {oldStr && newStr ? (
            <div className="border-t border-[color:var(--color-divider)]" />
          ) : null}
          {newStr ? (
            <pre className="bg-green-500/5 px-3 py-2 text-green-400/70 whitespace-pre-wrap break-words max-h-40 overflow-auto">
              {displayNew}
              {!showFull && newStr.length > TRUNCATE_THRESHOLD ? "\u2026" : ""}
            </pre>
          ) : null}
        </div>
      )}
      {isTruncated && (
        <button
          type="button"
          onClick={() => setShowFull(true)}
          className="flex items-center gap-1 text-[11px] text-[color:var(--color-text-tertiary)] hover:text-[color:var(--color-text-secondary)] transition-colors"
        >
          <ChevronsUpDown className="size-3" />
          Show full content
        </button>
      )}
    </div>
  );
}
