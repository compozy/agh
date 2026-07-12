import type { ReactNode } from "react";

export function OnboardingGateFrame({ children, testId }: { children: ReactNode; testId: string }) {
  return (
    <main
      id="app-content"
      data-testid={testId}
      className="flex min-h-0 flex-1 items-center justify-center bg-canvas"
    >
      {children}
    </main>
  );
}
