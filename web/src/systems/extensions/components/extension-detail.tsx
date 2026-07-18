import { Link } from "@tanstack/react-router";
import { MoreHorizontal, Puzzle } from "lucide-react";

import {
  Button,
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
  Empty,
  Switch,
  useTopbarSlot,
} from "@agh/ui";

import { ExtensionDetailSkeleton } from "./extension-detail-sections";
import { ExtensionDetailBody } from "./extension-detail-body";
import { useExtensionDetailState } from "../hooks/use-extension-detail-state";

export function ExtensionDetail({ name }: { name: string }) {
  const state = useExtensionDetailState(name);
  const { detail, setProvenanceOpen, setRemoveOpen, toggle, update } = state;
  const data = detail.data;
  const acting = toggle.isPending || update.isPending;
  useTopbarSlot(
    data
      ? {
          actions: (
            <>
              <span className="flex items-center gap-2 text-xs text-muted">
                {data.extension.enabled ? "Enabled" : "Disabled"}
                <Switch
                  aria-label={`${data.extension.enabled ? "Disable" : "Enable"} ${name}`}
                  checked={data.extension.enabled}
                  disabled={acting}
                  onCheckedChange={enabled => toggle.mutate({ name, enabled })}
                  size="sm"
                />
              </span>
              {data.updateAvailable ? (
                <Button
                  disabled={acting}
                  onClick={() => update.mutate(name)}
                  size="sm"
                  variant="outline"
                >
                  Update
                </Button>
              ) : null}
              {data.listing ? (
                <Button
                  render={
                    <Link
                      params={{ entryId: data.listing.entry_id, kind: "extension" }}
                      to="/marketplace/$kind/$entryId"
                    />
                  }
                  nativeButton={false}
                  size="sm"
                  variant="ghost"
                >
                  View in marketplace →
                </Button>
              ) : null}
              <DropdownMenu>
                <DropdownMenuTrigger
                  render={
                    <Button aria-label={`Actions for ${name}`} size="icon-sm" variant="ghost" />
                  }
                >
                  <MoreHorizontal className="size-4" />
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end">
                  <DropdownMenuItem onClick={() => setProvenanceOpen(true)}>
                    Provenance
                  </DropdownMenuItem>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem className="text-danger" onClick={() => setRemoveOpen(true)}>
                    Remove…
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </>
          ),
        }
      : null
  );
  if (detail.isLoading) return <ExtensionDetailSkeleton />;
  if (detail.error)
    return (
      <Empty description={detail.error.message} icon={Puzzle} title="Unable to load extension" />
    );
  if (!detail.data)
    return (
      <Empty
        description={`No installed extension is named ${name}.`}
        icon={Puzzle}
        title="Extension not found"
      />
    );

  return <ExtensionDetailBody name={name} state={state} />;
}
