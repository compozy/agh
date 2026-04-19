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
  skillId: string | null;
  channel: string | null;
  attachments: MessageComposerAttachment[];
  handleChange: (event: ChangeEvent<HTMLTextAreaElement>) => void;
  handleKeyDown: (event: KeyboardEvent<HTMLTextAreaElement>) => void;
  handleSend: () => void;
  handleSkillChange: (next: string | null) => void;
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
  const [skillId, setSkillId] = useState<string | null>(draft?.skillId ?? null);
  const [channel, setChannel] = useState<string | null>(draft?.channel ?? null);
  const [attachments, setAttachments] = useState<MessageComposerAttachment[]>([]);

  // Hydrate local state when the session changes (covers route-driven remounts).
  const lastHydratedRef = useRef<string | null>(null);
  useEffect(() => {
    if (!sessionId || lastHydratedRef.current === sessionId) return;
    lastHydratedRef.current = sessionId;
    setText(draft?.text ?? "");
    setSkillId(draft?.skillId ?? null);
    setChannel(draft?.channel ?? null);
  }, [sessionId, draft?.text, draft?.skillId, draft?.channel]);

  const persistDraft = useCallback(
    (patch: { text?: string; skillId?: string | null; channel?: string | null }) => {
      if (!sessionId) return;
      useSessionStore.getState().setDraft(sessionId, {
        ...(patch.text !== undefined ? { text: patch.text } : {}),
        ...(patch.skillId !== undefined ? { skillId: patch.skillId ?? undefined } : {}),
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
      skillId: skillId ?? undefined,
      channel: channel ?? undefined,
      attachments: attachments.length > 0 ? attachments : undefined,
    });
    setText("");
    setAttachments([]);
    if (sessionId) {
      useSessionStore.getState().clearDraft(sessionId);
    }
    const el = textareaRef.current;
    if (el) el.style.height = "auto";
  }, [attachments, channel, onSend, sessionId, skillId, text]);

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

  const handleSkillChange = useCallback(
    (next: string | null) => {
      setSkillId(next);
      persistDraft({ skillId: next });
    },
    [persistDraft]
  );

  const handleChannelChange = useCallback(
    (next: string | null) => {
      setChannel(next);
      persistDraft({ channel: next });
    },
    [persistDraft]
  );

  const handleAttach = useCallback((item: MessageComposerAttachment) => {
    setAttachments(prev => (prev.some(a => a.id === item.id) ? prev : [...prev, item]));
  }, []);

  const handleRemoveAttachment = useCallback((id: string) => {
    setAttachments(prev => prev.filter(a => a.id !== id));
  }, []);

  return {
    textareaRef,
    text,
    skillId,
    channel,
    attachments,
    handleChange,
    handleKeyDown,
    handleSend,
    handleSkillChange,
    handleChannelChange,
    handleAttach,
    handleRemoveAttachment,
  };
}
