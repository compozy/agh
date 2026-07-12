import type { ReactNode } from "react";

export function AppRouteBoundaryFrame({
  children,
  routeId,
  testId,
}: {
  children: ReactNode;
  routeId?: string;
  testId: string;
}) {
  return (
    <main
      id="app-content"
      data-route-id={routeId}
      data-testid={testId}
      className="flex min-h-0 flex-1 items-center justify-center overflow-y-auto bg-background px-6 py-8"
    >
      {children}
    </main>
  );
}
