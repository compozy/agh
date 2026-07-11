import { createContext, type ReactNode } from "react";

export type SessionComposerPrefill = (text: string) => void;

export const SessionComposerPrefillContext = createContext<SessionComposerPrefill | null>(null);

export function SessionComposerPrefillProvider({
  children,
  setComposerText,
}: {
  children: ReactNode;
  setComposerText: SessionComposerPrefill;
}) {
  return (
    <SessionComposerPrefillContext.Provider value={setComposerText}>
      {children}
    </SessionComposerPrefillContext.Provider>
  );
}
