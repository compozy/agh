import {
  type ChangeEvent,
  type KeyboardEvent,
  type RefObject,
  useCallback,
  useEffect,
  useRef,
  useState,
} from "react";

import type {
  MessageComposerAttachment,
  MessageComposerPayload,
} from "../components/message-composer";
import { useSessionStore } from "./use-session-store";

export interface UseMessageComposerOptions {
  sessionId?: string | null;
  disabled?: boolean;
  onSend: (payload: MessageComposerPayload) => void;
}

export interface UseMessageComposerReturn {
  textareaRef: RefObject<HTMLTextAreaElement | null>;
  text: string;
  channel: string | null;
  attachments: MessageComposerAttachment[];
  handleChange: (event: ChangeEvent<HTMLTextAreaElement>) => void;
  handleKeyDown: (event: KeyboardEvent<HTMLTextAreaElement>) => void;
  handleSend: () => void;
  handleChannelChange: (next: string | null) => void;
  handleAttach: (item: MessageComposerAttachment) => void;
  handleRemoveAttachment: (id: string) => void;
}

export const MESSAGE_COMPOSER_AUTOGROW_CAP_PX = 200;

export function useMessageComposer({
  sessionId,
  disabled,
  onSend,
}: UseMessageComposerOptions): UseMessageComposerReturn {
  const textareaRef = useRef<HTMLTextAreaElement | null>(null);
  const draft = useSessionStore(state => (sessionId ? state.drafts[sessionId] : undefined));

  const [text, setText] = useState<string>(draft?.text ?? "");
  const [channel, setChannel] = useState<string | null>(draft?.channel ?? null);
  const [attachments, setAttachments] = useState<MessageComposerAttachment[]>([]);

  const lastHydratedRef = useRef<string | null>(null);
  useEffect(() => {
    if (!sessionId || lastHydratedRef.current === sessionId) return;
    lastHydratedRef.current = sessionId;
    setText(draft?.text ?? "");
    setChannel(draft?.channel ?? null);
  }, [sessionId, draft?.channel, draft?.text]);

  const persistDraft = useCallback(
    (patch: { text?: string; channel?: string | null }) => {
      if (!sessionId) return;
      useSessionStore.getState().setDraft(sessionId, {
        ...(patch.text !== undefined ? { text: patch.text } : {}),
        ...(patch.channel !== undefined ? { channel: patch.channel ?? undefined } : {}),
      });
    },
    [sessionId]
  );

  const handleChange = useCallback(
    (event: ChangeEvent<HTMLTextAreaElement>) => {
      const next = event.target.value;
      setText(next);
      persistDraft({ text: next });
      const el = textareaRef.current;
      if (el) {
        el.style.height = "auto";
        el.style.height = `${Math.min(el.scrollHeight, MESSAGE_COMPOSER_AUTOGROW_CAP_PX)}px`;
      }
    },
    [persistDraft]
  );

  const handleSend = useCallback(() => {
    const trimmed = text.trim();
    if (!trimmed) return;
    onSend({
      text: trimmed,
      channel: channel ?? undefined,
      attachments: attachments.length > 0 ? attachments : undefined,
    });
    setText("");
    setAttachments([]);
    if (sessionId) {
      useSessionStore.getState().clearDraft(sessionId);
    }
    const el = textareaRef.current;
    if (el) {
      el.style.height = "auto";
    }
  }, [attachments, channel, onSend, sessionId, text]);

  const handleKeyDown = useCallback(
    (event: KeyboardEvent<HTMLTextAreaElement>) => {
      if (event.key === "Enter" && !event.shiftKey) {
        event.preventDefault();
        if (!disabled) {
          handleSend();
        }
      }
    },
    [disabled, handleSend]
  );

  const handleChannelChange = useCallback(
    (next: string | null) => {
      setChannel(next);
      persistDraft({ channel: next });
    },
    [persistDraft]
  );

  const handleAttach = useCallback((item: MessageComposerAttachment) => {
    setAttachments(previous =>
      previous.some(att => att.id === item.id) ? previous : [...previous, item]
    );
  }, []);

  const handleRemoveAttachment = useCallback((id: string) => {
    setAttachments(previous => previous.filter(att => att.id !== id));
  }, []);

  return {
    textareaRef,
    text,
    channel,
    attachments,
    handleChange,
    handleKeyDown,
    handleSend,
    handleChannelChange,
    handleAttach,
    handleRemoveAttachment,
  };
}
