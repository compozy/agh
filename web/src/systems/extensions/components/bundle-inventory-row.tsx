import { Link } from "@tanstack/react-router";
import { Box, MoreHorizontal } from "lucide-react";

import {
  Button,
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
  ListingRow,
  Pill,
  Spinner,
  Time,
} from "@agh/ui";

import type { BundleActivation } from "../types";

export interface BundleInventoryRowProps {
  activation: BundleActivation;
  pending: boolean;
  onUpdate: () => void;
  onDeactivate: () => void;
}

export function BundleInventoryRow({
  activation,
  pending,
  onUpdate,
  onDeactivate,
}: BundleInventoryRowProps) {
  const contentCount = (activation.inventory ?? []).length;
  return (
    <ListingRow data-testid={`bundle-row-${activation.id}`}>
      <ListingRow.Link
        render={
          <Link
            aria-label={`Open ${activation.bundle_name}`}
            params={{ id: activation.id }}
            to="/extensions/bundles/$id"
          />
        }
      >
        <ListingRow.Icon>
          <Box className="size-4" />
        </ListingRow.Icon>
        <ListingRow.Main>
          <ListingRow.Name>
            <ListingRow.Title>{activation.bundle_name}</ListingRow.Title>
            <Pill mono size="xs" tone="neutral">
              {activation.profile_name}
            </Pill>
          </ListingRow.Name>
          <ListingRow.Meta>
            <span>{contentCount} capabilities</span>
            <ListingRow.MetaDot />
            <span>
              {activation.scope}
              {activation.workspace_id ? ` · ${activation.workspace_id}` : ""}
            </span>
            <ListingRow.MetaDot />
            <span>
              activated <Time iso={activation.created_at} />
            </span>
          </ListingRow.Meta>
        </ListingRow.Main>
      </ListingRow.Link>
      <ListingRow.Trail className="gap-3">
        <Pill tone="success">active</Pill>
        {activation.spec_drift ? (
          <Button disabled={pending} onClick={onUpdate} size="sm" variant="outline">
            {pending ? <Spinner className="size-3" /> : null}Update
          </Button>
        ) : null}
        <DropdownMenu>
          <DropdownMenuTrigger
            render={
              <Button
                aria-label={`Actions for ${activation.bundle_name}`}
                size="icon-sm"
                variant="ghost"
              />
            }
          >
            <MoreHorizontal className="size-4" />
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem className="text-danger" onClick={onDeactivate}>
              Deactivate…
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </ListingRow.Trail>
    </ListingRow>
  );
}
