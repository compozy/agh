import { useEffect, useEffectEvent, useRef } from "react";

import { Command, CommandEmpty, CommandItem, CommandList, CommandShortcut } from "@agh/ui";

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
  const containerRef = useRef<HTMLDivElement | null>(null);
  const closePopover = useEffectEvent(onClose);

  useEffect(() => {
    if (!open) {
      return undefined;
    }
    function handleKey(event: KeyboardEvent) {
      if (event.key === "Escape") {
        event.preventDefault();
        closePopover();
      }
    }
    window.addEventListener("keydown", handleKey);
    return () => window.removeEventListener("keydown", handleKey);
  }, [open]);

  if (!open) {
    return null;
  }

  const entries = filterEntries(filterValue);

  return (
    <Command
      aria-label="Slash commands"
      className={cn(
        "absolute bottom-full left-0 mb-2 w-72 rounded-mono-badge border border-(--line) bg-(--canvas) p-1 text-small-body",
        className
      )}
      data-testid="network-composer-slash-popover"
      ref={containerRef}
    >
      <CommandList>
        {entries.length === 0 ? <CommandEmpty>No matching commands.</CommandEmpty> : null}
        {entries.map(entry => (
          <CommandItem
            aria-disabled={entry.disabled ? "true" : "false"}
            className={cn(
              "items-baseline rounded-chip px-3 py-2",
              entry.disabled ? "cursor-not-allowed text-(--subtle)" : null
            )}
            data-disabled={entry.disabled ? "true" : "false"}
            data-testid={`network-composer-slash-option-${entry.command}`}
            disabled={entry.disabled}
            key={entry.command}
            onSelect={() => {
              if (!entry.disabled) {
                onSelect(entry);
              }
            }}
            title={entry.disabled ? entry.disabledReason : undefined}
            value={entry.command}
          >
            <span
              className={cn(
                "font-mono text-xs tracking-mono",
                entry.disabled ? "text-(--subtle)" : "text-(--fg)"
              )}
            >
              /{entry.command}
            </span>
            <span className="truncate text-xs text-(--subtle)">{entry.description}</span>
            {entry.disabled ? <CommandShortcut>Post-MVP</CommandShortcut> : null}
          </CommandItem>
        ))}
      </CommandList>
    </Command>
  );
}
