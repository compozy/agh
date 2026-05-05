import { useEffect, useRef } from "react";

import { cn } from "@/lib/utils";

export interface SlashCommandEntry {
  /** The literal command keyword without the leading `/`. */
  command: string;
  description: string;
  disabled?: boolean;
  disabledReason?: string;
}

export interface ComposerSlashPopoverProps {
  open: boolean;
  filterValue: string;
  onSelect: (entry: SlashCommandEntry) => void;
  onClose: () => void;
  className?: string;
}

const SLASH_COMMANDS: ReadonlyArray<SlashCommandEntry> = [
  {
    command: "run",
    description: "Run a capability for this conversation.",
  },
  {
    command: "mention",
    description: "Mention a peer to draw their attention.",
  },
  {
    command: "attach",
    description: "Attach context (file, URL, capability ref).",
    disabled: true,
    disabledReason: "Post-MVP",
  },
];

export function getSlashCommandEntries(): ReadonlyArray<SlashCommandEntry> {
  return SLASH_COMMANDS;
}

function filterEntries(filterValue: string): ReadonlyArray<SlashCommandEntry> {
  const trimmed = filterValue.trim().toLowerCase();
  if (trimmed === "") {
    return SLASH_COMMANDS;
  }
  return SLASH_COMMANDS.filter(entry => entry.command.toLowerCase().startsWith(trimmed));
}

export function ComposerSlashPopover({
  open,
  filterValue,
  onSelect,
  onClose,
  className,
}: ComposerSlashPopoverProps) {
  const containerRef = useRef<HTMLUListElement | null>(null);

  useEffect(() => {
    if (!open) {
      return undefined;
    }
    function handleKey(event: KeyboardEvent) {
      if (event.key === "Escape") {
        event.preventDefault();
        onClose();
      }
    }
    window.addEventListener("keydown", handleKey);
    return () => window.removeEventListener("keydown", handleKey);
  }, [open, onClose]);

  if (!open) {
    return null;
  }

  const entries = filterEntries(filterValue);

  return (
    <ul
      aria-label="Slash commands"
      className={cn(
        "absolute bottom-full left-0 mb-2 w-72 rounded-[6px] border border-[color:var(--color-divider)] bg-[color:var(--color-canvas)] p-1 text-[13px]",
        className
      )}
      data-testid="network-composer-slash-popover"
      ref={containerRef}
      role="listbox"
    >
      {entries.length === 0 ? (
        <li className="px-3 py-2 text-[12px] text-[color:var(--color-text-tertiary)]">
          No matching commands.
        </li>
      ) : (
        entries.map(entry => (
          <li
            key={entry.command}
            role="option"
            aria-disabled={entry.disabled ? "true" : "false"}
            aria-selected="false"
            data-testid={`network-composer-slash-option-${entry.command}`}
            data-disabled={entry.disabled ? "true" : "false"}
            title={entry.disabled ? entry.disabledReason : undefined}
          >
            <button
              className={cn(
                "flex w-full items-baseline gap-2 rounded-[4px] px-3 py-2 text-left",
                entry.disabled
                  ? "cursor-not-allowed text-[color:var(--color-text-tertiary)]"
                  : "hover:bg-[color:var(--color-hover)] focus-visible:bg-[color:var(--color-hover)] focus-visible:outline-none"
              )}
              disabled={entry.disabled}
              onClick={() => {
                if (entry.disabled) {
                  return;
                }
                onSelect(entry);
              }}
              type="button"
            >
              <span
                className={cn(
                  "font-mono text-[12px] tracking-[0.04em]",
                  entry.disabled
                    ? "text-[color:var(--color-text-tertiary)]"
                    : "text-[color:var(--color-text-primary)]"
                )}
              >
                /{entry.command}
              </span>
              <span className="truncate text-[12px] text-[color:var(--color-text-tertiary)]">
                {entry.description}
              </span>
              {entry.disabled ? (
                <span className="ml-auto font-mono text-[10px] uppercase tracking-[0.06em] text-[color:var(--color-text-tertiary)]">
                  Post-MVP
                </span>
              ) : null}
            </button>
          </li>
        ))
      )}
    </ul>
  );
}
