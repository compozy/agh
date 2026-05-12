import { createContext } from "react";
import type { ReactNode } from "react";

import type { UseNetworkListFiltersResult } from "../hooks/use-network-list-filters";

export const NetworkListFiltersContext = createContext<UseNetworkListFiltersResult | null>(null);

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
