"use client";

import {
  Children,
  createContext,
  isValidElement,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import type { KeyboardEvent, ReactElement, ReactNode } from "react";

export type StepperOrientation = "horizontal" | "vertical";
export type StepState = "active" | "completed" | "inactive" | "loading";
export type StepIndicators = {
  active?: ReactNode;
  completed?: ReactNode;
  inactive?: ReactNode;
  loading?: ReactNode;
};

export interface StepperContextValue {
  activeStep: number;
  setActiveStep: (step: number) => void;
  stepsCount: number;
  orientation: StepperOrientation;
  registerTrigger: (node: HTMLButtonElement | null) => void;
  triggerNodes: HTMLButtonElement[];
  focusNext: (currentIdx: number) => void;
  focusPrev: (currentIdx: number) => void;
  focusFirst: () => void;
  focusLast: () => void;
  indicators: StepIndicators;
}

export interface StepItemContextValue {
  step: number;
  state: StepState;
  isDisabled: boolean;
  isLoading: boolean;
}

interface UseStepperStateOptions {
  defaultValue: number;
  value?: number;
  onValueChange?: (value: number) => void;
  orientation: StepperOrientation;
  indicators: StepIndicators;
  children: ReactNode;
}

export const StepperContext = createContext<StepperContextValue | undefined>(undefined);
export const StepItemContext = createContext<StepItemContextValue | undefined>(undefined);

export function useStepper() {
  const ctx = useContext(StepperContext);
  if (!ctx) throw new Error("useStepper must be used within a Stepper");
  return ctx;
}

export function useStepItem() {
  const ctx = useContext(StepItemContext);
  if (!ctx) throw new Error("useStepItem must be used within a StepperItem");
  return ctx;
}

export function useStepperState({
  defaultValue,
  value,
  onValueChange,
  orientation,
  indicators,
  children,
}: UseStepperStateOptions): StepperContextValue {
  const [activeStep, setActiveStep] = useState(defaultValue);
  const [triggerNodes, setTriggerNodes] = useState<HTMLButtonElement[]>([]);

  const registerTrigger = useCallback((node: HTMLButtonElement | null) => {
    if (!node) {
      return;
    }
    setTriggerNodes(prev => (prev.includes(node) ? prev : [...prev, node]));
  }, []);

  const handleSetActiveStep = useCallback(
    (step: number) => {
      if (value === undefined) {
        setActiveStep(step);
      }
      onValueChange?.(step);
    },
    [value, onValueChange]
  );

  const currentStep = value ?? activeStep;
  const focusTrigger = useCallback(
    (idx: number) => {
      triggerNodes[idx]?.focus();
    },
    [triggerNodes]
  );
  const focusNext = useCallback(
    (currentIdx: number) => focusTrigger((currentIdx + 1) % triggerNodes.length),
    [focusTrigger, triggerNodes.length]
  );
  const focusPrev = useCallback(
    (currentIdx: number) =>
      focusTrigger((currentIdx - 1 + triggerNodes.length) % triggerNodes.length),
    [focusTrigger, triggerNodes.length]
  );
  const focusFirst = useCallback(() => focusTrigger(0), [focusTrigger]);
  const focusLast = useCallback(
    () => focusTrigger(triggerNodes.length - 1),
    [focusTrigger, triggerNodes.length]
  );
  const stepsCount = useMemo(
    () =>
      Children.toArray(children).filter(
        (child): child is ReactElement =>
          isValidElement(child) &&
          (child.type as { displayName?: string }).displayName === "StepperItem"
      ).length,
    [children]
  );

  return useMemo<StepperContextValue>(
    () => ({
      activeStep: currentStep,
      setActiveStep: handleSetActiveStep,
      stepsCount,
      orientation,
      registerTrigger,
      focusNext,
      focusPrev,
      focusFirst,
      focusLast,
      triggerNodes,
      indicators,
    }),
    [
      currentStep,
      focusFirst,
      focusLast,
      focusNext,
      focusPrev,
      handleSetActiveStep,
      indicators,
      orientation,
      registerTrigger,
      stepsCount,
      triggerNodes,
    ]
  );
}

export function useStepperTrigger() {
  const { state, isLoading, step, isDisabled } = useStepItem();
  const {
    setActiveStep,
    activeStep,
    registerTrigger,
    triggerNodes,
    focusNext,
    focusPrev,
    focusFirst,
    focusLast,
  } = useStepper();
  const buttonRef = useRef<HTMLButtonElement>(null);
  const isSelected = activeStep === step;
  const id = `stepper-tab-${step}`;
  const panelId = `stepper-panel-${step}`;

  useEffect(() => {
    registerTrigger(buttonRef.current);
  }, [registerTrigger]);

  const triggerIndex = useMemo(
    () => triggerNodes.findIndex((node: HTMLButtonElement) => node === buttonRef.current),
    [triggerNodes]
  );
  const selectStep = useCallback(() => setActiveStep(step), [setActiveStep, step]);
  const handleKeyDown = useCallback(
    (event: KeyboardEvent<HTMLButtonElement>) => {
      switch (event.key) {
        case "ArrowRight":
        case "ArrowDown":
          event.preventDefault();
          if (triggerIndex !== -1) focusNext(triggerIndex);
          break;
        case "ArrowLeft":
        case "ArrowUp":
          event.preventDefault();
          if (triggerIndex !== -1) focusPrev(triggerIndex);
          break;
        case "Home":
          event.preventDefault();
          focusFirst();
          break;
        case "End":
          event.preventDefault();
          focusLast();
          break;
        case "Enter":
        case " ":
          event.preventDefault();
          selectStep();
          break;
      }
    },
    [focusFirst, focusLast, focusNext, focusPrev, selectStep, triggerIndex]
  );

  return {
    buttonRef,
    handleKeyDown,
    id,
    isDisabled,
    isLoading,
    isSelected,
    panelId,
    selectStep,
    state,
  };
}
