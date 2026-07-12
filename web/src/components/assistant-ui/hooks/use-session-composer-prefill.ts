import { useContext } from "react";

import {
  SessionComposerPrefillContext,
  type SessionComposerPrefill,
} from "../session-composer-prefill-context";

export function useSessionComposerPrefill(): SessionComposerPrefill | null {
  return useContext(SessionComposerPrefillContext);
}
