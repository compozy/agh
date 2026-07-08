import type { ThreadMessage } from "@assistant-ui/react";
import { useRef } from "react";

export function useThreadProviderIdentity(messages: readonly ThreadMessage[]): string {
  const identityRef = useRef<{ ids: string[]; value: string }>({ ids: [], value: "" });
  const previous = identityRef.current;
  const idsChanged =
    previous.ids.length !== messages.length ||
    messages.some((message, index) => previous.ids[index] !== message.id);
  if (idsChanged) {
    const ids = messages.map(message => message.id);
    identityRef.current = {
      ids,
      value: ids.map(id => `${id.length}:${id}`).join("|"),
    };
  }
  return identityRef.current.value;
}
