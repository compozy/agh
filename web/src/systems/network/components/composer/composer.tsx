import { Button, Textarea } from "@agh/ui";

import { cn } from "@/lib/utils";

import { ComposerSlashPopover } from "./composer-slash-popover";
import { ComposerToolbar } from "./composer-toolbar";
import {
  useComposerState,
  type ComposerSubmitArgs,
  type UseComposerStateResult,
} from "./use-composer-state";

const TEXTAREA_MIN_ROWS = 2;
const TEXTAREA_MAX_ROWS = 8;

export type { ComposerSubmitArgs };

export interface ComposerProps {
  placeholder: string;
  /** Stable suffix for `data-testid` attributes (e.g. `channel`, `thread`, `direct`). */
  testIdSuffix: string;
  /** Tooltip + aria-label for the Send button (`Send to #ops` etc). */
  sendLabel: string;
  /** Currently submitting? Disables the textarea + Send button. */
  isSending?: boolean;
  /** Disabled completely (e.g. network down). */
  disabled?: boolean;
  disabledReason?: string;
  onSubmit: (args: ComposerSubmitArgs) => void;
  className?: string;
}

interface ComposerViewProps extends ComposerProps {
  state: UseComposerStateResult;
}

function ComposerView({
  placeholder,
  testIdSuffix,
  sendLabel,
  isSending = false,
  disabled = false,
  disabledReason,
  className,
  state,
}: ComposerViewProps) {
  return (
    <form
      aria-label={sendLabel}
      className={cn(
        "relative flex flex-col gap-2 border-t border-line bg-canvas px-4 py-3",
        className
      )}
      data-testid={`network-composer-${testIdSuffix}`}
      onSubmit={state.handleSubmit}
    >
      <Textarea
        aria-label={placeholder}
        className={cn(
          "min-h-[64px] resize-none rounded border-0 bg-input-fill px-3 py-2 text-sm focus-visible:ring-0",
          "focus:bg-canvas focus-visible:bg-canvas focus-visible:shadow-[0_0_0_1px_var(--line-strong)]",
          disabled && "cursor-not-allowed opacity-60"
        )}
        data-testid={`network-composer-textarea-${testIdSuffix}`}
        disabled={disabled || isSending}
        onChange={state.handleChange}
        onKeyDown={state.handleKeyDown}
        placeholder={disabled ? (disabledReason ?? placeholder) : placeholder}
        ref={state.textareaRef}
        rows={TEXTAREA_MIN_ROWS}
        style={{ maxHeight: `${TEXTAREA_MAX_ROWS * 1.5}rem` }}
        value={state.value}
      />

      <div className="flex items-end justify-between gap-2">
        <ComposerToolbar onSlash={state.handleToolbarSlash} testIdSuffix={testIdSuffix} />
        <Button
          aria-label={sendLabel}
          data-testid={`network-composer-send-${testIdSuffix}`}
          disabled={state.sendDisabled}
          size="sm"
          title={sendLabel}
          type="submit"
          variant="default"
        >
          Send
        </Button>
      </div>

      <ComposerSlashPopover
        filterValue={state.slashFilter}
        onClose={state.handleSlashClose}
        onSelect={state.handleSlashSelect}
        open={state.slashOpen}
      />
    </form>
  );
}

export function Composer(props: ComposerProps) {
  const state = useComposerState({
    disabled: props.disabled ?? false,
    isSending: props.isSending ?? false,
    onSubmit: props.onSubmit,
  });
  return <ComposerView {...props} state={state} />;
}
