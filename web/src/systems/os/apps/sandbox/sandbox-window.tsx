import { SandboxPage, validateSandboxSearch } from "@/systems/sandbox";
import { useDesktop } from "../../hooks/use-desktop";

/** Sandbox app controller driven exclusively by the logical window's WM location. */
export function SandboxWindow({ windowId }: { windowId: string }) {
  const search = useDesktop(state =>
    validateSandboxSearch(state.windows[windowId]?.location.search ?? {})
  );
  return <SandboxPage search={search} />;
}
