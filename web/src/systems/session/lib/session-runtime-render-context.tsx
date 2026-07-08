import { createContext, type ReactNode } from "react";

interface SessionRuntimeRenderContextValue {
  sessionId: string;
  workspaceId: string;
}

export const SessionRuntimeRenderContext = createContext<SessionRuntimeRenderContextValue | null>(
  null
);

export function SessionRuntimeRenderProvider({
  children,
  sessionId,
  workspaceId,
}: SessionRuntimeRenderContextValue & { children: ReactNode }) {
  return (
    <SessionRuntimeRenderContext.Provider value={{ sessionId, workspaceId }}>
      {children}
    </SessionRuntimeRenderContext.Provider>
  );
}
