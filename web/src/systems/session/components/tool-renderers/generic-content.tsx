import type { UIMessage } from "../../types";

function formatInput(input: Record<string, unknown>): string {
  try {
    return JSON.stringify(input, null, 2);
  } catch {
    return String(input);
  }
}

function formatResult(result: NonNullable<UIMessage["toolResult"]>): string {
  if (result.error) return result.error;
  if (result.stdout) return result.stdout;
  if (result.content) return result.content;
  try {
    return JSON.stringify(result, null, 2);
  } catch {
    return String(result);
  }
}

export function GenericContent({ message }: { message: UIMessage }) {
  const result = message.toolResult;
  const hasResult = result && (result.stdout || result.content || result.error);

  return (
    <div className="space-y-1.5 text-xs">
      {message.toolInput && (
        <pre className="max-h-32 overflow-auto rounded-md bg-[color:var(--color-surface)] px-3 py-2 font-mono text-[11px] text-[color:var(--color-text-tertiary)] whitespace-pre-wrap break-words">
          {formatInput(message.toolInput)}
        </pre>
      )}
      {hasResult && (
        <pre className="max-h-48 overflow-auto rounded-md bg-[color:var(--color-surface)] px-3 py-2 font-mono text-[11px] text-[color:var(--color-text-tertiary)] whitespace-pre-wrap break-words">
          {formatResult(result)}
        </pre>
      )}
    </div>
  );
}
