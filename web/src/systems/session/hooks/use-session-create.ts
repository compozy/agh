import { useContext } from "react";

import {
  SessionCreateContext,
  type SessionCreateContextValue,
} from "../contexts/session-create-context";

export function useSessionCreate(): SessionCreateContextValue {
  const value = useContext(SessionCreateContext);
  if (!value) {
    throw new Error("useSessionCreate must be used inside <SessionCreateProvider>");
  }
  return value;
}
