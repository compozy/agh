import { useState, type FormEvent } from "react";
import { toast } from "sonner";

import { useDaemonStatus } from "@/systems/status";

import { useResolveWorkspace } from "./use-workspaces";

type SubmissionMode = "global" | "manual" | null;
export type WorkspaceSetupVariant = "dialog" | "onboarding";

interface UseWorkspaceSetupContentOptions {
  onSuccessClose?: () => void;
  onWorkspaceResolved: (workspaceId: string) => void;
}

function getErrorMessage(error: unknown, fallback: string): string {
  if (error instanceof Error && error.message.trim() !== "") {
    return error.message;
  }

  return fallback;
}

function isAbsoluteWorkspacePath(path: string): boolean {
  return path.startsWith("/") || /^[A-Za-z]:[\\/]/.test(path) || path.startsWith("\\\\");
}

function getGlobalUnavailableReason(isLoading: boolean, userHomeDir: string): string | null {
  if (isLoading) {
    return "Loading daemon status...";
  }

  if (!userHomeDir) {
    return "Daemon status unavailable. Connect AGH to use your global workspace.";
  }

  return null;
}

export function useWorkspaceSetupContent({
  onWorkspaceResolved,
  onSuccessClose,
}: UseWorkspaceSetupContentOptions) {
  const resolveWorkspace = useResolveWorkspace();
  const statusQuery = useDaemonStatus();
  const [manualPath, setManualPath] = useState("");
  const [submissionMode, setSubmissionMode] = useState<SubmissionMode>(null);
  const [manualError, setManualError] = useState<string | null>(null);

  const userHomeDir = statusQuery.data?.user_home_dir ?? "";

  const globalUnavailableReason = getGlobalUnavailableReason(statusQuery.isLoading, userHomeDir);

  const runResolve = async (path: string, mode: Exclude<SubmissionMode, null>) => {
    setSubmissionMode(mode);
    setManualError(null);

    try {
      const workspace = await resolveWorkspace.mutateAsync({ path });
      onWorkspaceResolved(workspace.id);

      if (mode === "manual") {
        setManualPath("");
      }

      toast.success(`Workspace ready: ${workspace.name}`);
      onSuccessClose?.();
    } catch (error) {
      const message = getErrorMessage(error, "Failed to register workspace");
      if (mode === "manual") {
        setManualError(message);
      }
      toast.error(message);
    }
    setSubmissionMode(null);
  };

  const handleManualSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();

    const trimmedPath = manualPath.trim();
    if (trimmedPath === "") {
      setManualError("Workspace path is required.");
      return;
    }

    if (!isAbsoluteWorkspacePath(trimmedPath)) {
      setManualError("Workspace path must be absolute.");
      return;
    }

    await runResolve(trimmedPath, "manual");
  };

  const handleUseGlobalWorkspace = () => {
    void runResolve(userHomeDir, "global");
  };

  return {
    globalUnavailableReason,
    handleManualSubmit,
    handleUseGlobalWorkspace,
    manualError,
    manualPath,
    setManualPath,
    submissionMode,
    userHomeDir,
  };
}
