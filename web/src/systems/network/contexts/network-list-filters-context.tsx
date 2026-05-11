import { createContext, useContext } from "react";
import type { ReactNode } from "react";

import type { UseNetworkListFiltersResult } from "../hooks/use-network-list-filters";

const NetworkListFiltersContext = createContext<UseNetworkListFiltersResult | null>(null);

interface NetworkListFiltersProviderProps {
  value: UseNetworkListFiltersResult;
  children: ReactNode;
}

export function NetworkListFiltersProvider({ value, children }: NetworkListFiltersProviderProps) {
  return (
    <NetworkListFiltersContext.Provider value={value}>
      {children}
    </NetworkListFiltersContext.Provider>
  );
}

export function useNetworkListFiltersContext(): UseNetworkListFiltersResult {
  const ctx = useContext(NetworkListFiltersContext);
  if (!ctx) {
    throw new Error(
      "useNetworkListFiltersContext must be used inside <NetworkListFiltersProvider>"
    );
  }
  return ctx;
}
