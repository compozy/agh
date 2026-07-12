"use client";

import { Children, createContext, isValidElement, use, useEffect, useRef, useState } from "react";
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
  const ctx = use(StepperContext);
  if (!ctx) throw new Error("useStepper must be used within a Stepper");
  return ctx;
}

export function useStepItem() {
  const ctx = use(StepItemContext);
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

  const registerTrigger = (node: HTMLButtonElement | null) => {
    if (!node) {
      return;
    }
    setTriggerNodes(prev => (prev.includes(node) ? prev : [...prev, node]));
  };

  const handleSetActiveStep = (step: number) => {
    if (value === undefined) {
      setActiveStep(step);
    }
    onValueChange?.(step);
  };

  const currentStep = value ?? activeStep;
  const focusTrigger = (idx: number) => {
    triggerNodes[idx]?.focus();
  };
  const focusNext = (currentIdx: number) => focusTrigger((currentIdx + 1) % triggerNodes.length);
  const focusPrev = (currentIdx: number) =>
    focusTrigger((currentIdx - 1 + triggerNodes.length) % triggerNodes.length);
  const focusFirst = () => focusTrigger(0);
  const focusLast = () => focusTrigger(triggerNodes.length - 1);
  const stepsCount = Children.toArray(children).filter(
    (child): child is ReactElement =>
      isValidElement(child) &&
      (child.type as { displayName?: string }).displayName === "StepperItem"
  ).length;

  return {
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
  };
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

  const triggerIndex = triggerNodes.findIndex(
    (node: HTMLButtonElement) => node === buttonRef.current
  );
  const selectStep = () => setActiveStep(step);
  const handleKeyDown = (event: KeyboardEvent<HTMLButtonElement>) => {
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
  };

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
