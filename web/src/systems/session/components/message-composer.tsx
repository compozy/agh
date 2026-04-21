import { useCallback, useMemo, useState } from "react";
import { Combobox as ComboboxPrimitive } from "@base-ui/react";
import { ChevronDown, Hash, Paperclip, SendHorizontal, X } from "lucide-react";

import {
  Button,
  Combobox,
  ComboboxCollection,
  ComboboxContent,
  ComboboxEmpty,
  ComboboxItem,
  ComboboxList,
  Popover,
  PopoverContent,
  PopoverTrigger,
  Textarea,
  cn,
} from "@agh/ui";

import {
  MESSAGE_COMPOSER_AUTOGROW_CAP_PX,
  useMessageComposer,
} from "../hooks/use-message-composer";

export interface MessageComposerAttachment {
  id: string;
  name: string;
  size?: number;
}

export interface MessageComposerChannel {
  id: string;
  name: string;
  description?: string;
}

export interface MessageComposerPayload {
  text: string;
  channel?: string;
  attachments?: MessageComposerAttachment[];
}

export interface MessageComposerProps {
  onSend: (payload: MessageComposerPayload) => void;
  /** Session id used to scope persisted drafts. When omitted, drafts are not persisted. */
  sessionId?: string | null;
  disabled?: boolean;
  inert?: boolean;
  channels?: MessageComposerChannel[];
  /** Attach menu items rendered inside the Popover. */
  attachOptions?: MessageComposerAttachment[];
  className?: string;
}

const pillTriggerClass = cn(
  "inline-flex h-[22px] items-center gap-1 rounded-[var(--radius-mono-badge)] border border-[color:var(--color-divider)]",
  "bg-transparent px-2 font-mono text-[10px] font-semibold tracking-[0.08em] uppercase",
  "text-[color:var(--color-text-secondary)] transition-colors duration-150 ease-out cursor-pointer",
  "hover:border-[color:var(--color-text-label)] hover:bg-[color:var(--color-hover)] hover:text-[color:var(--color-text-primary)]",
  "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[color:var(--color-accent)]",
  "data-[popup-open]:border-[color:var(--color-accent)] data-[popup-open]:text-[color:var(--color-accent)]",
  "disabled:cursor-not-allowed disabled:opacity-50",
  "[&_svg]:size-2.5 [&_svg]:shrink-0"
);

export function MessageComposer({
  onSend,
  sessionId,
  disabled,
  inert = false,
  channels,
  attachOptions,
  className,
}: MessageComposerProps) {
  const composer = useMessageComposer({ sessionId, disabled, onSend });
  const interactiveDisabled = disabled || inert;

  const attachmentMap = useMemo(() => {
    const map = new Map<string, MessageComposerAttachment>();
    for (const item of composer.attachments) map.set(item.id, item);
    return map;
  }, [composer.attachments]);

  const hasChannels = (channels?.length ?? 0) > 0;
  const activeChannel = hasChannels ? channels?.find(c => c.id === composer.channel) : undefined;

  return (
    <div className={cn("px-4 py-3", className)} data-testid="message-composer">
      <div
        aria-busy={disabled || undefined}
        aria-disabled={interactiveDisabled || undefined}
        className={cn(
          "flex flex-col gap-2 rounded-xl border px-3 pt-2.5 pb-2",
          "border-[color:var(--color-divider)] bg-[color:var(--color-surface)]",
          "focus-within:border-[color:var(--color-accent)]",
          "transition-colors",
          inert && "pointer-events-none opacity-60"
        )}
        data-testid="composer-container"
        inert={inert || undefined}
      >
        {composer.attachments.length > 0 && (
          <div className="flex flex-wrap gap-1.5" data-testid="composer-attachments">
            {composer.attachments.map(att => (
              <span
                key={att.id}
                data-slot="composer-attachment"
                data-testid={`composer-attachment-${att.id}`}
                className={cn(
                  "inline-flex items-center gap-1 rounded-[var(--radius-mono-badge)] bg-[color:var(--color-surface-elevated)] px-2 py-0.5",
                  "font-mono text-[10px] font-medium tracking-[0.04em] text-[color:var(--color-text-primary)]"
                )}
              >
                <Paperclip className="size-2.5 text-[color:var(--color-text-tertiary)]" />
                <span data-testid={`composer-attachment-name-${att.id}`}>{att.name}</span>
                <button
                  type="button"
                  aria-label={`Remove ${att.name}`}
                  className="flex size-3.5 items-center justify-center rounded-full text-[color:var(--color-text-tertiary)] hover:text-[color:var(--color-text-primary)]"
                  onClick={() => composer.handleRemoveAttachment(att.id)}
                >
                  <X className="size-2.5" />
                </button>
              </span>
            ))}
          </div>
        )}

        <Textarea
          ref={composer.textareaRef}
          value={composer.text}
          onChange={composer.handleChange}
          onKeyDown={composer.handleKeyDown}
          placeholder="Send a message..."
          disabled={interactiveDisabled}
          rows={1}
          aria-label="Message composer"
          className={cn(
            "min-h-6 w-full resize-none border-none bg-transparent p-0 text-sm leading-relaxed",
            "text-[color:var(--color-text-primary)] placeholder:text-[color:var(--color-text-tertiary)]",
            "shadow-none outline-none focus-visible:border-transparent focus-visible:ring-0",
            "dark:bg-transparent"
          )}
          style={{ maxHeight: `${MESSAGE_COMPOSER_AUTOGROW_CAP_PX}px` }}
          data-testid="composer-textarea"
        />

        <div className="flex items-center gap-1.5" data-testid="composer-action-row">
          <ComposerAttachPill
            options={attachOptions ?? []}
            attachmentMap={attachmentMap}
            onAttach={composer.handleAttach}
            disabled={interactiveDisabled}
          />

          {hasChannels && (
            <ComposerChannelPill
              channels={channels ?? []}
              value={composer.channel}
              activeChannel={activeChannel}
              onChange={composer.handleChannelChange}
              disabled={interactiveDisabled}
            />
          )}

          <div className="ml-auto">
            <Button
              type="button"
              aria-label="Send message"
              disabled={interactiveDisabled}
              onClick={composer.handleSend}
              className={cn(
                "flex size-9 shrink-0 items-center justify-center rounded-full p-0",
                "bg-[color:var(--color-accent)] text-white",
                "hover:bg-[color:var(--color-accent-hover)] hover:text-white",
                "focus-visible:ring-[color:var(--color-accent)]/50",
                "disabled:opacity-50 disabled:cursor-not-allowed"
              )}
              data-testid="composer-send-button"
            >
              <SendHorizontal className="size-4" />
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
}

interface ComposerAttachPillProps {
  options: MessageComposerAttachment[];
  attachmentMap: Map<string, MessageComposerAttachment>;
  onAttach: (item: MessageComposerAttachment) => void;
  disabled?: boolean;
}

function ComposerAttachPill({
  options,
  attachmentMap,
  onAttach,
  disabled,
}: ComposerAttachPillProps) {
  const [open, setOpen] = useState(false);

  const handleSelect = useCallback(
    (option: MessageComposerAttachment) => {
      onAttach(option);
      setOpen(false);
    },
    [onAttach]
  );

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger
        type="button"
        disabled={disabled}
        data-testid="composer-attach-pill"
        className={pillTriggerClass}
      >
        <Paperclip />
        <span>attach</span>
      </PopoverTrigger>
      <PopoverContent
        align="start"
        side="top"
        className="w-60 gap-1 p-1.5"
        data-testid="composer-attach-popover"
      >
        <div
          className={cn(
            "px-2 py-1 font-mono text-[10px] tracking-[0.08em] uppercase",
            "text-[color:var(--color-text-tertiary)]"
          )}
        >
          Attach files
        </div>
        {options.length === 0 ? (
          <div
            className="px-2 py-4 text-center text-xs text-[color:var(--color-text-tertiary)]"
            data-testid="composer-attach-empty"
          >
            No items to attach.
          </div>
        ) : (
          <ul className="flex flex-col gap-0.5">
            {options.map(option => {
              const alreadyAttached = attachmentMap.has(option.id);
              return (
                <li key={option.id}>
                  <button
                    type="button"
                    disabled={alreadyAttached}
                    onClick={() => handleSelect(option)}
                    data-testid={`composer-attach-option-${option.id}`}
                    className={cn(
                      "flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-sm",
                      "text-[color:var(--color-text-primary)]",
                      "hover:bg-[color:var(--color-hover)]",
                      "disabled:cursor-not-allowed disabled:opacity-50"
                    )}
                  >
                    <Paperclip className="size-3.5 text-[color:var(--color-text-tertiary)]" />
                    <span className="flex-1 truncate">{option.name}</span>
                    {alreadyAttached && (
                      <span
                        className={cn(
                          "font-mono text-[10px] tracking-[0.08em] uppercase",
                          "text-[color:var(--color-text-tertiary)]"
                        )}
                      >
                        added
                      </span>
                    )}
                  </button>
                </li>
              );
            })}
          </ul>
        )}
      </PopoverContent>
    </Popover>
  );
}

interface ComposerChannelPillProps {
  channels: MessageComposerChannel[];
  value: string | null;
  activeChannel?: MessageComposerChannel;
  onChange: (next: string | null) => void;
  disabled?: boolean;
}

function ComposerChannelPill({
  channels,
  value,
  activeChannel,
  onChange,
  disabled,
}: ComposerChannelPillProps) {
  const handleValueChange = useCallback(
    (next: unknown) => {
      if (typeof next === "string") {
        onChange(next.length > 0 ? next : null);
        return;
      }
      onChange(null);
    },
    [onChange]
  );

  return (
    <Combobox
      items={channels.map(c => c.id)}
      value={value ?? ""}
      onValueChange={handleValueChange}
      autoHighlight
    >
      <ComboboxPrimitive.Trigger
        type="button"
        disabled={disabled}
        data-testid="composer-channel-pill"
        className={pillTriggerClass}
      >
        <Hash />
        <span>{activeChannel ? `#${activeChannel.name}` : "channel"}</span>
        <ChevronDown className="opacity-60" />
      </ComboboxPrimitive.Trigger>
      <ComboboxContent
        align="start"
        side="top"
        className="w-64 overflow-hidden p-0"
        data-testid="composer-channel-combobox"
      >
        <div className="border-b border-[color:var(--color-divider)] p-1.5">
          <ComboboxPrimitive.Input
            data-testid="composer-channel-search"
            placeholder="Search channels"
            className={cn(
              "h-7 w-full rounded-md bg-transparent px-2 text-sm",
              "text-[color:var(--color-text-primary)] placeholder:text-[color:var(--color-text-tertiary)]",
              "outline-none focus:ring-0"
            )}
          />
        </div>
        <ComboboxList className="max-h-56">
          <ComboboxEmpty className="py-3 text-xs">No channels match.</ComboboxEmpty>
          <ComboboxCollection>
            {(id: string) => {
              const ch = channels.find(c => c.id === id);
              return (
                <ComboboxItem
                  key={id}
                  value={id}
                  data-testid={`composer-channel-item-${id}`}
                  className="flex-col items-start gap-0 px-2 py-1.5 pr-2"
                >
                  <span className="text-sm text-[color:var(--color-text-primary)]">
                    #{ch?.name ?? id}
                  </span>
                  {ch?.description && (
                    <span className="text-[11px] text-[color:var(--color-text-tertiary)]">
                      {ch.description}
                    </span>
                  )}
                </ComboboxItem>
              );
            }}
          </ComboboxCollection>
        </ComboboxList>
      </ComboboxContent>
    </Combobox>
  );
}
