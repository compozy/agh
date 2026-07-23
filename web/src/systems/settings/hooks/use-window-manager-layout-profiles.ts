import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";

import {
  deleteWindowManagerLayoutProfile,
  putWindowManagerLayoutProfile,
} from "../adapters/window-manager-layouts-api";
import { settingsKeys } from "../lib/query-keys";
import type {
  WindowManagerLayoutAspect,
  WindowManagerLayoutDocument,
  WindowManagerLayoutOverflow,
  WindowManagerLayoutResourceRecord,
  WindowManagerLayoutScopeKind,
} from "../lib/window-manager-layout-types";

function resourceKey(record: WindowManagerLayoutResourceRecord): string {
  return `${record.scope.kind}:${record.scope.id}:${record.id}`;
}

export function useWindowManagerLayoutProfiles({
  workspaceId,
  document,
  profiles,
  onLoad,
}: {
  workspaceId: string;
  document: WindowManagerLayoutDocument;
  profiles: readonly WindowManagerLayoutResourceRecord[];
  onLoad: (document: WindowManagerLayoutDocument) => void;
}) {
  const queryClient = useQueryClient();
  const [selectedKey, setSelectedKey] = useState<string | null>(null);
  const selected = profiles.find(profile => resourceKey(profile) === selectedKey) ?? null;
  const [id, setId] = useState("");
  const [displayName, setDisplayName] = useState("");
  const [aspect, setAspect] = useState<WindowManagerLayoutAspect>("any");
  const [overflow, setOverflow] = useState<WindowManagerLayoutOverflow>("stack");
  const [scope, setScope] = useState<WindowManagerLayoutScopeKind>("workspace");

  const save = useMutation({
    mutationFn: () => {
      const normalizedId = id.trim();
      return putWindowManagerLayoutProfile(
        {
          version: 1,
          id: normalizedId,
          displayName: displayName.trim(),
          aspectVariant: aspect,
          participantSlots: Object.keys(document.windows),
          overflowPolicy: overflow,
          document,
        },
        scope,
        workspaceId,
        selected !== null && selected.id === normalizedId ? selected.version : 0
      );
    },
    onSuccess: record => {
      const previousKey = selected === null ? null : resourceKey(selected);
      setSelectedKey(resourceKey(record));
      queryClient.setQueryData<WindowManagerLayoutResourceRecord[]>(
        settingsKeys.windowManagerLayoutProfiles(),
        current => [
          ...(current ?? []).filter(item => {
            const key = resourceKey(item);
            return key !== resourceKey(record) && key !== previousKey;
          }),
          record,
        ]
      );
    },
  });

  const remove = useMutation({
    mutationFn: () => {
      if (selected === null) throw new Error("Select a saved profile first.");
      return deleteWindowManagerLayoutProfile(selected.id, selected.version);
    },
    onSuccess: () => {
      if (selected === null) return;
      queryClient.setQueryData<WindowManagerLayoutResourceRecord[]>(
        settingsKeys.windowManagerLayoutProfiles(),
        current => (current ?? []).filter(item => resourceKey(item) !== resourceKey(selected))
      );
      setSelectedKey(null);
      setId("");
      setDisplayName("");
    },
  });

  const selectProfile = (record: WindowManagerLayoutResourceRecord) => {
    setSelectedKey(resourceKey(record));
    setId(record.id);
    setDisplayName(record.spec.displayName);
    setAspect(record.spec.aspectVariant);
    setOverflow(record.spec.overflowPolicy);
    setScope(record.scope.kind);
    onLoad({
      ...structuredClone(record.spec.document),
      workspaceId,
    });
  };

  const startNew = () => {
    setSelectedKey(null);
    setId("");
    setDisplayName("");
    setAspect("any");
    setOverflow("stack");
    setScope("workspace");
  };

  return {
    aspect,
    displayName,
    error: save.error ?? remove.error,
    id,
    overflow,
    profiles,
    remove,
    save,
    scope,
    selectProfile,
    selected,
    selectedKey,
    setAspect,
    setDisplayName,
    setId,
    setOverflow,
    setScope,
    startNew,
  };
}
