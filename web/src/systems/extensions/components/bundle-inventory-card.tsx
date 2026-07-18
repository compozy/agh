import { Link } from "@tanstack/react-router";
import { Box, MoreHorizontal } from "lucide-react";

import {
  Button,
  CatalogCard,
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
  Pill,
  Spinner,
} from "@agh/ui";

import type { BundleActivation } from "../types";

export interface BundleInventoryCardProps {
  activation: BundleActivation;
  pending: boolean;
  onUpdate: () => void;
  onDeactivate: () => void;
}

export function BundleInventoryCard({
  activation,
  pending,
  onUpdate,
  onDeactivate,
}: BundleInventoryCardProps) {
  const contentCount = (activation.inventory ?? []).length;
  return (
    <CatalogCard actionable data-testid={`bundle-card-${activation.id}`}>
      <Link
        aria-label={`Open ${activation.bundle_name}`}
        className="flex min-w-0 flex-col gap-3"
        params={{ id: activation.id }}
        to="/extensions/bundles/$id"
      >
        <div className="flex items-start gap-3">
          <CatalogCard.Logo>
            <Box className="size-4" />
          </CatalogCard.Logo>
          <div className="flex min-w-0 flex-1 flex-col gap-1">
            <CatalogCard.Title>{activation.bundle_name}</CatalogCard.Title>
            <CatalogCard.Meta>
              <span>{activation.profile_name}</span>
              <span>
                {activation.scope}
                {activation.workspace_id ? ` · ${activation.workspace_id}` : ""}
              </span>
              <span>{contentCount} capabilities</span>
            </CatalogCard.Meta>
          </div>
        </div>
      </Link>
      <CatalogCard.Actions className="justify-between">
        <Pill tone="success">active</Pill>
        <div className="flex items-center gap-2">
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
        </div>
      </CatalogCard.Actions>
    </CatalogCard>
  );
}
