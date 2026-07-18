import { Link } from "@tanstack/react-router";
import { MoreHorizontal, Puzzle } from "lucide-react";

import {
  Button,
  CatalogCard,
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
  Pill,
  Spinner,
  Switch,
} from "@agh/ui";

import { useToggleExtension, useUpdateExtension } from "../hooks/use-extension-actions";
import type { InstalledExtensionView } from "../types";

export interface ExtensionInventoryCardProps {
  item: InstalledExtensionView;
  onProvenance: () => void;
  onRemove: () => void;
}

export function ExtensionInventoryCard({
  item,
  onProvenance,
  onRemove,
}: ExtensionInventoryCardProps) {
  const toggle = useToggleExtension();
  const update = useUpdateExtension();
  const acting =
    (toggle.isPending && toggle.variables?.name === item.extension.name) ||
    (update.isPending && update.variables === item.extension.name);
  const description = item.listing?.description;

  return (
    <CatalogCard actionable data-testid={`extension-card-${item.extension.name}`}>
      <Link
        aria-label={`Open ${item.extension.name}`}
        className="flex min-w-0 flex-col gap-3"
        params={{ name: item.extension.name }}
        to="/extensions/$name"
      >
        <div className="flex items-start gap-3">
          <CatalogCard.Logo>
            <Puzzle className="size-4" />
          </CatalogCard.Logo>
          <div className="flex min-w-0 flex-1 flex-col gap-1">
            <CatalogCard.Title>{item.extension.name}</CatalogCard.Title>
            <CatalogCard.Meta>
              <span>{item.extension.type}</span>
              <span className="font-mono tracking-normal normal-case">
                v{item.extension.version}
              </span>
              <span>{item.extension.source}</span>
            </CatalogCard.Meta>
          </div>
        </div>
        {description ? <CatalogCard.Description>{description}</CatalogCard.Description> : null}
      </Link>
      <CatalogCard.Actions className="justify-between">
        <Pill mono size="sm" tone="neutral">
          {item.extension.type}
        </Pill>
        <div className="flex items-center gap-2">
          {acting ? (
            <Spinner className="size-3.5" />
          ) : (
            <Switch
              aria-label={`${item.extension.enabled ? "Disable" : "Enable"} ${item.extension.name}`}
              checked={item.extension.enabled}
              onCheckedChange={enabled => toggle.mutate({ name: item.extension.name, enabled })}
              size="sm"
            />
          )}
          {item.updateAvailable ? (
            <Button
              disabled={acting}
              onClick={() => update.mutate(item.extension.name)}
              size="sm"
              variant="outline"
            >
              Update
            </Button>
          ) : null}
          <DropdownMenu>
            <DropdownMenuTrigger
              render={
                <Button
                  aria-label={`Actions for ${item.extension.name}`}
                  size="icon-sm"
                  variant="ghost"
                />
              }
            >
              <MoreHorizontal className="size-4" />
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem onClick={onProvenance}>Provenance</DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem className="text-danger" onClick={onRemove}>
                Remove…
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </CatalogCard.Actions>
    </CatalogCard>
  );
}
