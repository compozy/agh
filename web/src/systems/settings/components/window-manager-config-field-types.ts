import type { Dispatch, SetStateAction } from "react";

import type { WindowManagerConfig } from "@/systems/os";

export interface ConfigFieldsProps {
  draft: WindowManagerConfig;
  setDraft: Dispatch<SetStateAction<WindowManagerConfig>>;
}

export interface SelectOption<V extends string> {
  value: V;
  label: string;
}
