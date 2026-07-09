import { useContext } from "react";

import { SessionRuntimeRenderContext } from "../lib/session-runtime-render-context";

export function useSessionRuntimeRenderContext() {
  return useContext(SessionRuntimeRenderContext);
}
