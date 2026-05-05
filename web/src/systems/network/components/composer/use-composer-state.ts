import { useCallback, useEffect, useRef, useState, type RefObject } from "react";

import type { SlashCommandEntry } from "./composer-slash-popover";

const SLASH_PREFIX = /(^|\s)\/([\w-]*)$/u;

export interface ComposerSubmitArgs {
  text: string;
  /** Reset the textarea after a successful send. */
  reset: () => void;
  /** Restore the textarea to the value the user typed (used when the send fails). */
  restore: () => void;
}

export interface UseComposerStateArgs {
  disabled: boolean;
  isSending: boolean;
  onSubmit: (args: ComposerSubmitArgs) => void;
}

export interface UseComposerStateResult {
  value: string;
  trimmed: string;
  slashOpen: boolean;
  slashFilter: string;
  textareaRef: RefObject<HTMLTextAreaElement | null>;
  sendDisabled: boolean;
  handleChange: (event: React.ChangeEvent<HTMLTextAreaElement>) => void;
  handleSubmit: (event: React.FormEvent<HTMLFormElement>) => void;
  handleKeyDown: (event: React.KeyboardEvent<HTMLTextAreaElement>) => void;
  handleSlashSelect: (entry: SlashCommandEntry) => void;
  handleToolbarSlash: () => void;
  handleSlashClose: () => void;
}

/**
 * Drives composer textarea state, slash command detection, and Cmd/Ctrl+Enter
 * submission. Extracted from `<Composer>` so it stays under the
 * `compozy-react(max-component-complexity)` cap and can be re-used by future
 * composer variants.
 */
export function useComposerState({
  disabled,
  isSending,
  onSubmit,
}: UseComposerStateArgs): UseComposerStateResult {
  const [value, setValue] = useState("");
  const [slashOpen, setSlashOpen] = useState(false);
  const [slashFilter, setSlashFilter] = useState("");
  const textareaRef = useRef<HTMLTextAreaElement | null>(null);

  const reset = useCallback(() => {
    setValue("");
    setSlashOpen(false);
    setSlashFilter("");
  }, []);

  const restore = useCallback(() => {
    // Caller invokes after a failed send if the optimistic message was
    // discarded. Currently a no-op since we keep the textarea contents in
    // `value` so the user can retry without retyping.
  }, []);

  useEffect(() => {
    if (disabled) {
      reset();
    }
  }, [disabled, reset]);

  const updateSlashState = useCallback((next: string) => {
    const match = SLASH_PREFIX.exec(next);
    if (match == null) {
      setSlashOpen(false);
      setSlashFilter("");
      return;
    }
    setSlashOpen(true);
    setSlashFilter(match[2] ?? "");
  }, []);

  const handleChange = useCallback(
    (event: React.ChangeEvent<HTMLTextAreaElement>) => {
      const next = event.target.value;
      setValue(next);
      updateSlashState(next);
    },
    [updateSlashState]
  );

  const submitInternal = useCallback(
    (text: string) => {
      onSubmit({ text, reset, restore });
    },
    [onSubmit, reset, restore]
  );

  const handleSubmit = useCallback(
    (event: React.FormEvent<HTMLFormElement>) => {
      event.preventDefault();
      if (disabled || isSending) {
        return;
      }
      const text = value.trim();
      if (text.length === 0) {
        return;
      }
      submitInternal(text);
    },
    [disabled, isSending, submitInternal, value]
  );

  const handleKeyDown = useCallback(
    (event: React.KeyboardEvent<HTMLTextAreaElement>) => {
      if (event.key === "Enter" && (event.metaKey || event.ctrlKey)) {
        event.preventDefault();
        const text = value.trim();
        if (text.length === 0 || disabled || isSending) {
          return;
        }
        submitInternal(text);
      }
    },
    [disabled, isSending, submitInternal, value]
  );

  const handleSlashSelect = useCallback((entry: SlashCommandEntry) => {
    setValue(prev =>
      prev.replace(SLASH_PREFIX, (_match, leading: string) => `${leading}/${entry.command} `)
    );
    setSlashOpen(false);
    setSlashFilter("");
    textareaRef.current?.focus();
  }, []);

  const handleToolbarSlash = useCallback(() => {
    setSlashOpen(true);
    setSlashFilter("");
    textareaRef.current?.focus();
  }, []);

  const handleSlashClose = useCallback(() => {
    setSlashOpen(false);
  }, []);

  const trimmed = value.trim();
  const sendDisabled = disabled || isSending || trimmed.length === 0;

  return {
    value,
    trimmed,
    slashOpen,
    slashFilter,
    textareaRef,
    sendDisabled,
    handleChange,
    handleSubmit,
    handleKeyDown,
    handleSlashSelect,
    handleToolbarSlash,
    handleSlashClose,
  };
}
