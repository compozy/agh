import { useState } from "react";

import { useBundleActivations, useExtensionInventory } from "./use-extensions";
import type { ExtensionEntry } from "../types";

export function useExtensionInventoryState() {
  const inventory = useExtensionInventory();
  const activations = useBundleActivations();
  const [query, setQuery] = useState("");
  const [provenance, setProvenance] = useState<ExtensionEntry | null>(null);
  const [removing, setRemoving] = useState<ExtensionEntry | null>(null);
  const visible = inventory.data.filter(item =>
    item.extension.name.toLowerCase().includes(query.toLowerCase())
  );
  return {
    activations,
    inventory,
    provenance,
    query,
    removing,
    setProvenance,
    setQuery,
    setRemoving,
    visible,
  };
}
