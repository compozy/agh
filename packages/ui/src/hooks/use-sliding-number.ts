"use client";

import * as React from "react";

import { useMotionValue, useSpring } from "motion/react";

import { useIsInView, type UseIsInViewOptions } from "./use-is-in-view";

export interface UseSlidingNumberParams {
  ref?: React.Ref<HTMLElement>;
  number: number;
  fromNumber?: number;
  onNumberChange?: (number: number) => void;
  inView: boolean;
  inViewMargin: UseIsInViewOptions["inViewMargin"];
  inViewOnce: boolean;
  padStart: boolean;
  decimalPlaces: number;
  initiallyStable: boolean;
  delay: number;
}

export function useSlidingNumber({
  ref,
  number,
  fromNumber,
  onNumberChange,
  inView,
  inViewMargin,
  inViewOnce,
  padStart,
  decimalPlaces,
  initiallyStable,
  delay,
}: UseSlidingNumberParams) {
  const { ref: localRef, isInView } = useIsInView(ref as React.Ref<HTMLElement>, {
    inView,
    inViewOnce,
    inViewMargin,
  });

  const initialNumeric = Math.abs(Number(number));
  const [prevNumber, setPrevNumber] = React.useState<number>(initiallyStable ? initialNumeric : 0);

  const hasAnimated = fromNumber !== undefined;

  const motionVal = useMotionValue(initiallyStable ? initialNumeric : (fromNumber ?? 0));
  const springVal = useSpring(motionVal, { stiffness: 90, damping: 50 });

  const skippedInitialWhenStable = React.useRef(false);

  React.useEffect(() => {
    if (!hasAnimated) return;
    if (initiallyStable && !skippedInitialWhenStable.current) {
      skippedInitialWhenStable.current = true;
      return;
    }
    const timeoutId = setTimeout(() => {
      if (isInView) motionVal.set(number);
    }, delay);
    return () => clearTimeout(timeoutId);
  }, [hasAnimated, initiallyStable, isInView, number, motionVal, delay]);

  const [effectiveNumber, setEffectiveNumber] = React.useState<number>(
    initiallyStable ? initialNumeric : 0
  );

  React.useEffect(() => {
    if (hasAnimated) {
      const inferredDecimals =
        typeof decimalPlaces === "number" && decimalPlaces >= 0
          ? decimalPlaces
          : (() => {
              const s = String(number);
              const idx = s.indexOf(".");
              return idx >= 0 ? s.length - idx - 1 : 0;
            })();

      const factor = Math.pow(10, inferredDecimals);

      const unsubscribe = springVal.on("change", (latest: number) => {
        const newValue =
          inferredDecimals > 0 ? Math.round(latest * factor) / factor : Math.round(latest);

        if (effectiveNumber !== newValue) {
          setEffectiveNumber(newValue);
          onNumberChange?.(newValue);
        }
      });
      return () => unsubscribe();
    } else {
      setEffectiveNumber(initiallyStable ? initialNumeric : !isInView ? 0 : initialNumeric);
      return undefined;
    }
  }, [
    hasAnimated,
    springVal,
    isInView,
    number,
    decimalPlaces,
    onNumberChange,
    effectiveNumber,
    initiallyStable,
    initialNumeric,
  ]);

  const formatNumber = React.useCallback(
    (num: number) => (decimalPlaces != null ? num.toFixed(decimalPlaces) : num.toString()),
    [decimalPlaces]
  );

  const numberStr = formatNumber(effectiveNumber);
  const [newIntStrRaw = "0", newDecStrRaw = ""] = numberStr.split(".");

  const finalIntLength = padStart
    ? Math.max(Math.floor(Math.abs(number)).toString().length, newIntStrRaw.length)
    : newIntStrRaw.length;

  const newIntStr = padStart ? newIntStrRaw.padStart(finalIntLength, "0") : newIntStrRaw;

  const prevFormatted = formatNumber(prevNumber);
  const [prevIntStrRaw = "", prevDecStrRaw = ""] = prevFormatted.split(".");
  const prevIntStr = padStart ? prevIntStrRaw.padStart(finalIntLength, "0") : prevIntStrRaw;

  const adjustedPrevInt = React.useMemo(() => {
    return prevIntStr.length > finalIntLength
      ? prevIntStr.slice(-finalIntLength)
      : prevIntStr.padStart(finalIntLength, "0");
  }, [prevIntStr, finalIntLength]);

  const adjustedPrevDec = React.useMemo(() => {
    if (!newDecStrRaw) return "";
    return prevDecStrRaw.length > newDecStrRaw.length
      ? prevDecStrRaw.slice(0, newDecStrRaw.length)
      : prevDecStrRaw.padEnd(newDecStrRaw.length, "0");
  }, [prevDecStrRaw, newDecStrRaw]);

  React.useEffect(() => {
    if (isInView || initiallyStable) {
      setPrevNumber(effectiveNumber);
    }
  }, [effectiveNumber, isInView, initiallyStable]);

  const intPlaces = React.useMemo(
    () => Array.from({ length: finalIntLength }, (_, i) => Math.pow(10, finalIntLength - i - 1)),
    [finalIntLength]
  );
  const decPlaces = React.useMemo(
    () =>
      newDecStrRaw
        ? Array.from({ length: newDecStrRaw.length }, (_, i) =>
            Math.pow(10, newDecStrRaw.length - i - 1)
          )
        : [],
    [newDecStrRaw]
  );

  const newDecValue = newDecStrRaw ? parseInt(newDecStrRaw, 10) : 0;
  const prevDecValue = adjustedPrevDec ? parseInt(adjustedPrevDec, 10) : 0;

  return {
    localRef,
    isInView,
    intPlaces,
    decPlaces,
    adjustedPrevInt: parseInt(adjustedPrevInt, 10),
    newIntValue: parseInt(newIntStr ?? "0", 10),
    newDecStrRaw,
    newDecValue,
    prevDecValue,
  };
}
