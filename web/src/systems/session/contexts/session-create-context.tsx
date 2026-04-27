import { createContext } from "react";
import type { ReactNode } from "react";

export interface SessionCreateContextValue {
  openForAgent: (agentName: string) => void;
  isCreating: boolean;
  pendingAgentName: string | null;
  hasActiveWorkspace: boolean;
}

export const SessionCreateContext = createContext<SessionCreateContextValue | null>(null);

interface SessionCreateProviderProps {
  value: SessionCreateContextValue;
  children: ReactNode;
}

export function SessionCreateProvider({ value, children }: SessionCreateProviderProps) {
  return <SessionCreateContext.Provider value={value}>{children}</SessionCreateContext.Provider>;
}
