import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useCallback, useState } from "react";

import { supportApi } from "../adapters/support-api";
import { supportKeys } from "../lib/query-keys";
import type { SupportBundleDownloadResult, SupportBundleOperation } from "../types";

interface SupportBundleDownloadInput {
  includeStatus?: boolean;
  yes: true;
}

function supportBundleFileName(operation: SupportBundleOperation): string {
  const fileName = operation.file_name?.trim();
  if (fileName) return fileName;
  return `agh-support-bundle-${operation.operation_id}.tar.gz`;
}

function delay(ms: number): Promise<void> {
  return new Promise(resolve => globalThis.setTimeout(resolve, ms));
}

async function waitForSupportBundle(operationId: string): Promise<SupportBundleOperation> {
  for (;;) {
    const operation = await supportApi.get(operationId);
    if (operation.status === "completed") {
      return operation;
    }
    if (operation.status === "failed") {
      throw new Error(operation.failure_reason?.trim() || "Support bundle failed");
    }
    await delay(750);
  }
}

function downloadBlob(blob: Blob, fileName: string) {
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement("a");
  anchor.href = url;
  anchor.download = fileName;
  document.body.append(anchor);
  anchor.click();
  anchor.remove();
  URL.revokeObjectURL(url);
}

export function useSupportBundleDownload() {
  const queryClient = useQueryClient();
  const [operation, setOperation] = useState<SupportBundleOperation | null>(null);

  const mutation = useMutation({
    mutationFn: async ({
      includeStatus = true,
      yes,
    }: SupportBundleDownloadInput): Promise<SupportBundleDownloadResult> => {
      const created = await supportApi.create({ include_status: includeStatus, yes });
      setOperation(created);
      const completed = await waitForSupportBundle(created.operation_id);
      setOperation(completed);
      const blob = await supportApi.download(completed.operation_id);
      const fileName = supportBundleFileName(completed);
      downloadBlob(blob, fileName);
      return { operation: completed, fileName };
    },
    onSuccess: result => {
      queryClient.setQueryData(supportKeys.bundle(result.operation.operation_id), result.operation);
    },
  });

  const create = useCallback(
    (input: SupportBundleDownloadInput) => mutation.mutateAsync(input),
    [mutation]
  );

  return {
    create,
    operation,
    isPending: mutation.isPending,
    error: mutation.error,
    reset: mutation.reset,
  };
}
