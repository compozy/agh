import type { UIMessage } from "../../types";
import { GenericContent } from "./generic-content";

export function EditContent({ message }: { message: UIMessage }) {
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

  if (!filePath && !oldStr && !newStr) {
    return <GenericContent message={message} />;
  }

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
              {oldStr.length > 1500 ? `${oldStr.slice(0, 1500)}\u2026` : oldStr}
            </pre>
          ) : null}
          {oldStr && newStr ? (
            <div className="border-t border-[color:var(--color-divider)]" />
          ) : null}
          {newStr ? (
            <pre className="bg-green-500/5 px-3 py-2 text-green-400/70 whitespace-pre-wrap break-words max-h-40 overflow-auto">
              {newStr.length > 1500 ? `${newStr.slice(0, 1500)}\u2026` : newStr}
            </pre>
          ) : null}
        </div>
      )}
    </div>
  );
}
