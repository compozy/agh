import { validateVaultSearch, VaultPage } from "@/systems/vault";
import { useDesktop } from "../../hooks/use-desktop";

/** Vault app controller driven exclusively by the logical window's WM location. */
export function VaultWindow({ windowId }: { windowId: string }) {
  const search = useDesktop(state =>
    validateVaultSearch(state.windows[windowId]?.location.search ?? {})
  );
  return <VaultPage search={search} />;
}
