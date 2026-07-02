import { useCallback, useEffect, useMemo, useState, type FormEvent } from "react";

import type { FanOutTaskRunsRequest, FanOutTaskRunsResponse } from "../types";

const DEFAULT_DESIGNATIONS = [
  "Investigate runtime data path",
  "Validate UI impact",
  "Review operator docs",
].join("\n");

export interface UseTasksFanOutRunsCardParams {
  defaultNetworkChannel?: string | null;
  onFanOut: (data: FanOutTaskRunsRequest) => Promise<FanOutTaskRunsResponse | void>;
}

function parseDesignations(value: string): FanOutTaskRunsRequest["designations"] {
  return value
    .split(/\r?\n/u)
    .map(line => line.trim())
    .filter(line => line.length > 0)
    .map(brief => ({ brief }));
}

function trimOrUndefined(value: string): string | undefined {
  const trimmed = value.trim();
  return trimmed === "" ? undefined : trimmed;
}

export function useTasksFanOutRunsCard({
  defaultNetworkChannel,
  onFanOut,
}: UseTasksFanOutRunsCardParams) {
  const [open, setOpen] = useState(false);
  const [networkChannel, setNetworkChannel] = useState(defaultNetworkChannel ?? "");
  const [designationsText, setDesignationsText] = useState(DEFAULT_DESIGNATIONS);
  const [formError, setFormError] = useState<string | null>(null);
  const designations = useMemo(() => parseDesignations(designationsText), [designationsText]);

  useEffect(() => {
    if (open) {
      setNetworkChannel(defaultNetworkChannel ?? "");
    }
  }, [defaultNetworkChannel, open]);

  const handleOpenChange = useCallback((nextOpen: boolean) => {
    setOpen(nextOpen);
    if (!nextOpen) {
      setFormError(null);
      setDesignationsText(DEFAULT_DESIGNATIONS);
    }
  }, []);

  const openDialog = useCallback(() => handleOpenChange(true), [handleOpenChange]);
  const closeDialog = useCallback(() => handleOpenChange(false), [handleOpenChange]);

  const handleSubmit = useCallback(
    async (event: FormEvent<HTMLFormElement>) => {
      event.preventDefault();
      setFormError(null);
      if (designations.length === 0) {
        setFormError("Add at least one assignment.");
        return;
      }

      const payload: FanOutTaskRunsRequest = {
        designations,
        network_channel: trimOrUndefined(networkChannel),
      };

      try {
        await onFanOut(payload);
        handleOpenChange(false);
      } catch {
        // The route hook owns the toast; keep the dialog open for correction.
      }
    },
    [designations, handleOpenChange, networkChannel, onFanOut]
  );

  return {
    closeDialog,
    designationsText,
    formError,
    handleOpenChange,
    handleSubmit,
    networkChannel,
    open,
    openDialog,
    setDesignationsText,
    setNetworkChannel,
  };
}
