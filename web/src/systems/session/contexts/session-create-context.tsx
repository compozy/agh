import { createContext } from "react";

export interface SessionCreateContextValue {
  openForAgent: (agentName: string) => void;
  isCreating: boolean;
  pendingAgentName: string | null;
  hasActiveWorkspace: boolean;
}

export const SessionCreateContext = createContext<SessionCreateContextValue | null>(null);

export function SessionCreateProvider({
  value,
  children,
}: {
  value: SessionCreateContextValue;
  children: React.ReactNode;
}) {
  return <SessionCreateContext.Provider value={value}>{children}</SessionCreateContext.Provider>;
}
