import { Link } from "@tanstack/react-router";
import { MoreHorizontal, Puzzle } from "lucide-react";

import {
  Button,
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
  ListingRow,
  Pill,
  Spinner,
  Switch,
  Time,
} from "@agh/ui";

import { useToggleExtension, useUpdateExtension } from "../hooks/use-extension-actions";
import type { InstalledExtensionView } from "../types";

export interface ExtensionInventoryRowProps {
  item: InstalledExtensionView;
  onProvenance: () => void;
  onRemove: () => void;
}

export function ExtensionInventoryRow({
  item,
  onProvenance,
  onRemove,
}: ExtensionInventoryRowProps) {
  const toggle = useToggleExtension();
  const update = useUpdateExtension();
  const acting =
    (toggle.isPending && toggle.variables?.name === item.extension.name) ||
    (update.isPending && update.variables === item.extension.name);
  const description = item.listing?.description;

  return (
    <ListingRow data-testid={`extension-row-${item.extension.name}`}>
      <ListingRow.Link
        render={
          <Link
            aria-label={`Open ${item.extension.name}`}
            params={{ name: item.extension.name }}
            to="/extensions/$name"
          />
        }
      >
        <ListingRow.Icon>
          <Puzzle className="size-4" />
        </ListingRow.Icon>
        <ListingRow.Main>
          <ListingRow.Name>
            <ListingRow.Title>{item.extension.name}</ListingRow.Title>
            <Pill mono size="xs" tone="neutral">
              {item.extension.type}
            </Pill>
            <ListingRow.Slug>v{item.extension.version}</ListingRow.Slug>
          </ListingRow.Name>
          {description ? <ListingRow.Description>{description}</ListingRow.Description> : null}
          <ListingRow.Meta>
            <span>{item.extension.source}</span>
            {item.extension.provenance?.installed_at ? (
              <>
                <ListingRow.MetaDot />
                <span>
                  installed <Time iso={item.extension.provenance.installed_at} />
                </span>
              </>
            ) : null}
          </ListingRow.Meta>
        </ListingRow.Main>
      </ListingRow.Link>
      <ListingRow.Trail className="gap-3">
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
        <ExtensionOverflowMenu
          name={item.extension.name}
          onProvenance={onProvenance}
          onRemove={onRemove}
        />
      </ListingRow.Trail>
    </ListingRow>
  );
}

function ExtensionOverflowMenu({
  name,
  onProvenance,
  onRemove,
}: {
  name: string;
  onProvenance: () => void;
  onRemove: () => void;
}) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        render={<Button aria-label={`Actions for ${name}`} size="icon-sm" variant="ghost" />}
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
  );
}
