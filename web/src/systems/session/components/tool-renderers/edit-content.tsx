import { useState } from "react";
import { ChevronsUpDown } from "lucide-react";

import { CodeBlock } from "@agh/ui";

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
      {filePath && <div className="font-mono text-eyebrow text-(--subtle)">{filePath}</div>}
      {(oldStr || newStr) && (
        <div className="overflow-hidden rounded-md border border-(--line)">
          {oldStr ? (
            <CodeBlock
              code={`${displayOld}${!showFull && oldStr.length > TRUNCATE_THRESHOLD ? "\u2026" : ""}`}
              className="rounded-none"
              copyable={false}
              showPrompt={false}
              tone="danger"
              truncateLines={12}
            />
          ) : null}
          {oldStr && newStr ? <div className="border-t border-(--line)" /> : null}
          {newStr ? (
            <CodeBlock
              code={`${displayNew}${!showFull && newStr.length > TRUNCATE_THRESHOLD ? "\u2026" : ""}`}
              className="rounded-none"
              copyable={false}
              showPrompt={false}
              tone="success"
              truncateLines={12}
            />
          ) : null}
        </div>
      )}
      {isTruncated && (
        <button
          type="button"
          onClick={() => setShowFull(true)}
          className="flex items-center gap-1 text-eyebrow text-(--subtle) hover:text-(--muted) transition-colors"
        >
          <ChevronsUpDown className="size-3" />
          Show full content
        </button>
      )}
    </div>
  );
}
