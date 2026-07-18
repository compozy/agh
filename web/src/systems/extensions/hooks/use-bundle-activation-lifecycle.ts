import { useNavigate } from "@tanstack/react-router";
import { useState } from "react";

import { useDeactivateBundle, useUpdateBundleActivation } from "./use-extension-actions";
import type { BundleActivation } from "../types";

export function useBundleActivationLifecycle(id: string, activation: BundleActivation | undefined) {
  const update = useUpdateBundleActivation();
  const deactivate = useDeactivateBundle();
  const navigate = useNavigate();
  const [dialogOpen, setDialogOpen] = useState(false);
  const [confirmNetworkRequirement, setConfirmNetworkRequirement] = useState(false);

  const applyUpdate = () => {
    if (!activation) return;
    update.mutate({
      id,
      body: {
        expected_version: activation.version,
        ...(confirmNetworkRequirement ? { confirm_network_requirement: true } : {}),
      },
    });
  };

  const confirmDeactivation = async () => {
    await deactivate.mutateAsync(id);
    setDialogOpen(false);
    void navigate({ search: { tab: "bundles" }, to: "/extensions" });
  };

  return {
    applyUpdate,
    confirmDeactivation,
    confirmNetworkRequirement,
    deactivate,
    dialogOpen,
    setConfirmNetworkRequirement,
    setDialogOpen,
    update,
  };
}
