import type { ReactNode } from "react";

import {
  AgentCreateHostContext,
  type AgentCreateHostValue,
} from "@/systems/agent/lib/agent-create-host-context";

export type { AgentCreateHostValue };

export interface AgentCreateHostProviderProps {
  openDialog: () => void;
  children: ReactNode;
}

export function AgentCreateHostProvider({ openDialog, children }: AgentCreateHostProviderProps) {
  return <AgentCreateHostContext value={{ openDialog }}>{children}</AgentCreateHostContext>;
}
