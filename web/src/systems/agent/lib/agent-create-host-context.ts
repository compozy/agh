import { createContext } from "react";

export interface AgentCreateHostValue {
  openDialog: () => void;
}

export const AgentCreateHostContext = createContext<AgentCreateHostValue | null>(null);
