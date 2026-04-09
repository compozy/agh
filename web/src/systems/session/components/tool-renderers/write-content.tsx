import type { UIMessage } from "../../types";
import { GenericContent } from "./generic-content";

export function WriteContent({ message }: { message: UIMessage }) {
  const filePath = String(
    message.toolInput?.file_path ??
      message.toolInput?.filePath ??
      message.toolResult?.filePath ??
      ""
  );
  const content = String(message.toolInput?.content ?? message.toolResult?.content ?? "");

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
          {content.length > 2000 ? `${content.slice(0, 2000)}\u2026` : content}
        </pre>
      )}
    </div>
  );
}
