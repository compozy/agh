import { useCallback, useRef, type KeyboardEvent } from "react";
import { SendHorizontal } from "lucide-react";

import { cn } from "@/lib/utils";

export interface MessageComposerProps {
  onSend: (text: string) => void;
  disabled?: boolean;
  className?: string;
}

export function MessageComposer({ onSend, disabled, className }: MessageComposerProps) {
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  const handleSend = useCallback(() => {
    const el = textareaRef.current;
    if (!el) return;
    const text = el.value.trim();
    if (!text) return;
    onSend(text);
    el.value = "";
    el.style.height = "auto";
  }, [onSend]);

  const handleKeyDown = useCallback(
    (e: KeyboardEvent<HTMLTextAreaElement>) => {
      if (e.key === "Enter" && !e.shiftKey) {
        e.preventDefault();
        if (!disabled) {
          handleSend();
        }
      }
    },
    [disabled, handleSend]
  );

  const handleInput = useCallback(() => {
    const el = textareaRef.current;
    if (!el) return;
    el.style.height = "auto";
    el.style.height = `${Math.min(el.scrollHeight, 200)}px`;
  }, []);

  return (
    <div className={cn("px-4 py-3", className)} data-testid="message-composer">
      <div
        className={cn(
          "flex items-end gap-2 rounded-xl border px-4 py-2.5",
          "border-[color:var(--color-divider)] bg-[color:var(--color-surface)]",
          "focus-within:border-[color:var(--color-accent)]",
          "transition-colors"
        )}
        data-testid="composer-container"
      >
        <textarea
          ref={textareaRef}
          placeholder="Send a message..."
          disabled={disabled}
          onKeyDown={handleKeyDown}
          onInput={handleInput}
          rows={1}
          className={cn(
            "flex-1 resize-none bg-transparent text-sm",
            "text-[color:var(--color-text-primary)] placeholder:text-[color:var(--color-text-tertiary)]",
            "outline-none disabled:cursor-not-allowed disabled:opacity-50",
            "max-h-[200px] py-1"
          )}
          data-testid="composer-textarea"
        />
        <button
          type="button"
          disabled={disabled}
          onClick={handleSend}
          className={cn(
            "flex size-9 shrink-0 items-center justify-center rounded-full",
            "bg-[color:var(--color-accent)] text-white",
            "hover:bg-[color:var(--color-accent-hover)] transition-colors",
            "disabled:opacity-50 disabled:cursor-not-allowed"
          )}
          data-testid="composer-send-button"
        >
          <SendHorizontal className="size-4" />
        </button>
      </div>
    </div>
  );
}
