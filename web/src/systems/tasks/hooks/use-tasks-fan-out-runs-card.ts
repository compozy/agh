import { useState, type FormEvent } from "react";

import type { FanOutTaskRunsRequest, FanOutTaskRunsResponse } from "../types";

const DEFAULT_DESIGNATIONS = [
  "Investigate runtime data path",
  "Validate UI impact",
  "Review operator docs",
].join("\n");

export interface UseTasksFanOutRunsCardParams {
  onFanOut: (data: FanOutTaskRunsRequest) => Promise<FanOutTaskRunsResponse | void>;
}

function parseDesignations(value: string): FanOutTaskRunsRequest["designations"] {
  return value
    .split(/\r?\n/u)
    .map(line => line.trim())
    .filter(line => line.length > 0)
    .map(brief => ({ brief }));
}

export function useTasksFanOutRunsCard({ onFanOut }: UseTasksFanOutRunsCardParams) {
  const [open, setOpen] = useState(false);
  const [designationsText, setDesignationsText] = useState(DEFAULT_DESIGNATIONS);
  const [formError, setFormError] = useState<string | null>(null);
  const designations = parseDesignations(designationsText);

  const handleOpenChange = (nextOpen: boolean) => {
    setOpen(nextOpen);
    if (!nextOpen) {
      setFormError(null);
      setDesignationsText(DEFAULT_DESIGNATIONS);
    }
  };

  const openDialog = () => handleOpenChange(true);
  const closeDialog = () => handleOpenChange(false);

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setFormError(null);
    if (designations.length === 0) {
      setFormError("Add at least one assignment.");
      return;
    }

    const payload: FanOutTaskRunsRequest = {
      designations,
    };

    try {
      await onFanOut(payload);
      handleOpenChange(false);
    } catch {
      // The route hook owns the toast; keep the dialog open for correction.
    }
  };

  return {
    closeDialog,
    designationsText,
    formError,
    handleOpenChange,
    handleSubmit,
    open,
    openDialog,
    setDesignationsText,
  };
}
