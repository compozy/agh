import { useEffect, useRef } from "react";

import type { ProviderInspectorState } from "./use-settings-providers-page";

type ProviderInspectorMode = ProviderInspectorState["mode"];

export function useCreateProviderFocusRestore(inspectorMode: ProviderInspectorMode) {
  const createProviderButtonRef = useRef<HTMLButtonElement>(null);
  const inspectorOpen = inspectorMode !== "closed";
  const previousInspectorOpenRef = useRef(inspectorOpen);
  const lastInspectorModeRef = useRef(inspectorMode);

  useEffect(() => {
    if (inspectorOpen) {
      lastInspectorModeRef.current = inspectorMode;
    }
    if (
      previousInspectorOpenRef.current &&
      !inspectorOpen &&
      lastInspectorModeRef.current === "create"
    ) {
      createProviderButtonRef.current?.focus();
    }
    previousInspectorOpenRef.current = inspectorOpen;
  }, [inspectorMode, inspectorOpen]);

  return createProviderButtonRef;
}
