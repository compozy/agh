import { MotionConfig } from "motion/react";
import type { ReactNode } from "react";

export interface UIProviderProps {
  children: ReactNode;
  reducedMotion?: "user" | "always" | "never";
}

export function UIProvider({ children, reducedMotion = "user" }: UIProviderProps) {
  return (
    <MotionConfig reducedMotion={reducedMotion} transition={{ duration: 0.15, ease: "easeOut" }}>
      {children}
    </MotionConfig>
  );
}
