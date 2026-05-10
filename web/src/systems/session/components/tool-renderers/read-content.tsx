import type { UIMessage } from "../../types";
import { GenericContent } from "./generic-content";

export function ReadContent({ message }: { message: UIMessage }) {
  const result = message.toolResult;
  const filePath = String(message.toolInput?.file_path ?? message.toolInput?.filePath ?? "");

  if (filePath && typeof result?.stdout === "string") {
    const lineCount = result.stdout.split("\n").length;
    return (
      <div
        className="flex items-center gap-1.5 text-xs font-mono text-eyebrow text-(--subtle)"
        data-testid="read-content"
      >
        <span className="truncate">{filePath}</span>
        <span className="text-(--subtle)/30 shrink-0">{lineCount} lines</span>
      </div>
    );
  }

  if (filePath && typeof result?.content === "string") {
    const lineCount = result.content.split("\n").length;
    return (
      <div
        className="flex items-center gap-1.5 text-xs font-mono text-eyebrow text-(--subtle)"
        data-testid="read-content"
      >
        <span className="truncate">{filePath}</span>
        <span className="text-(--subtle)/30 shrink-0">{lineCount} lines</span>
      </div>
    );
  }

  // Path-only fallback
  if (filePath) {
    return (
      <div
        className="flex items-center gap-1.5 text-xs font-mono text-eyebrow text-(--subtle)"
        data-testid="read-content"
      >
        <span className="truncate">{filePath}</span>
      </div>
    );
  }

  return <GenericContent message={message} />;
}
