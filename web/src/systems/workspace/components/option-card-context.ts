import * as React from "react";

export type OptionCardDensity = "compact" | "comfortable";

export interface OptionCardContextValue {
  density: OptionCardDensity;
}

export const OptionCardContext = React.createContext<OptionCardContextValue | null>(null);
