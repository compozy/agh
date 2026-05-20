import * as React from "react";

export type ToolCallSectionSlot = "input" | "output";

/**
 * Slot metadata stored in context. Intentionally minimal — only the parent
 * needs to know which slots exist (for chip rendering) and whether each one
 * wants to start open. Children/source/format stay in their declaring JSX
 * so React keeps reconciliation continuity across parent re-renders (which
 * is what makes async-loading children like `<CodeBlock>` survive).
 */
export type SlotRegistration = {
  defaultOpen: boolean;
};

export type ToolCallCardContextValue = {
  registerSlot: (slot: ToolCallSectionSlot, registration: SlotRegistration) => void;
  unregisterSlot: (slot: ToolCallSectionSlot) => void;
  registeredSlots: Partial<Record<ToolCallSectionSlot, true>>;
  openSlots: Record<ToolCallSectionSlot, boolean>;
  toggleSlot: (slot: ToolCallSectionSlot) => void;
  panelIds: Record<ToolCallSectionSlot, string>;
};

const SLOT_ORDER: ToolCallSectionSlot[] = ["input", "output"];

export const TOOL_CALL_INPUT_SLOT = Symbol("tool-call-input-slot");
export const TOOL_CALL_OUTPUT_SLOT = Symbol("tool-call-output-slot");

export const ToolCallCardContext = React.createContext<ToolCallCardContextValue | null>(null);

export function useToolCallCardContext(): ToolCallCardContextValue {
  const ctx = React.useContext(ToolCallCardContext);
  if (!ctx) {
    throw new Error("ToolCallCard compound components must be used within <ToolCallCard>");
  }
  return ctx;
}

function isSlotChild(child: React.ReactNode, marker: symbol): boolean {
  if (!React.isValidElement(child)) return false;
  const type = child.type as { slotMarker?: symbol };
  return type.slotMarker === marker;
}

function splitChildren(children: React.ReactNode | undefined): {
  slotChildren: React.ReactNode[];
  rawChildren: React.ReactNode[];
} {
  const slotChildren: React.ReactNode[] = [];
  const rawChildren: React.ReactNode[] = [];
  React.Children.forEach(children, child => {
    if (isSlotChild(child, TOOL_CALL_INPUT_SLOT) || isSlotChild(child, TOOL_CALL_OUTPUT_SLOT)) {
      slotChildren.push(child);
      return;
    }
    if (child !== undefined && child !== null && child !== false) {
      rawChildren.push(child);
    }
  });
  return { slotChildren, rawChildren };
}

export function useToolCallCardState(
  children: React.ReactNode | undefined,
  errorMessage: React.ReactNode | undefined
) {
  const inputPanelId = React.useId();
  const outputPanelId = React.useId();
  const [registeredSlots, setRegisteredSlots] = React.useState<
    Partial<Record<ToolCallSectionSlot, true>>
  >({});
  const [openSlots, setOpenSlots] = React.useState<Record<ToolCallSectionSlot, boolean>>({
    input: false,
    output: false,
  });

  const registerSlot = React.useCallback(
    (slot: ToolCallSectionSlot, registration: SlotRegistration) => {
      setRegisteredSlots(prev => (prev[slot] ? prev : { ...prev, [slot]: true }));
      if (registration.defaultOpen) {
        setOpenSlots(prev => (prev[slot] ? prev : { ...prev, [slot]: true }));
      }
    },
    []
  );

  const unregisterSlot = React.useCallback((slot: ToolCallSectionSlot) => {
    setRegisteredSlots(prev => {
      if (!prev[slot]) return prev;
      const next = { ...prev };
      delete next[slot];
      return next;
    });
    setOpenSlots(prev => (prev[slot] ? { ...prev, [slot]: false } : prev));
  }, []);

  const toggleSlot = React.useCallback((slot: ToolCallSectionSlot) => {
    setOpenSlots(prev => ({ ...prev, [slot]: !prev[slot] }));
  }, []);

  const panelIds = React.useMemo(
    () => ({ input: inputPanelId, output: outputPanelId }),
    [inputPanelId, outputPanelId]
  );

  const contextValue = React.useMemo<ToolCallCardContextValue>(
    () => ({
      registerSlot,
      unregisterSlot,
      registeredSlots,
      openSlots,
      toggleSlot,
      panelIds,
    }),
    [registerSlot, unregisterSlot, registeredSlots, openSlots, toggleSlot, panelIds]
  );

  const { slotChildren, rawChildren } = splitChildren(children);
  const hasRegisteredSlots = SLOT_ORDER.some(slot => registeredSlots[slot] === true);
  const hasOpenSlot = SLOT_ORDER.some(slot => registeredSlots[slot] === true && openSlots[slot]);
  const hasError = errorMessage !== undefined && errorMessage !== null && errorMessage !== false;
  const hasRawChildren = rawChildren.length > 0;
  const showBody = hasError || hasRawChildren || hasOpenSlot;

  return {
    contextValue,
    slotChildren,
    rawChildren,
    hasRegisteredSlots,
    hasError,
    hasRawChildren,
    showBody,
  };
}
