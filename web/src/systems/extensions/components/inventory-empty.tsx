import { Link } from "@tanstack/react-router";
import { PackageOpen } from "lucide-react";

import { Button, Empty } from "@agh/ui";

export function InventoryEmpty({ kind }: { kind: "extensions" | "bundles" }) {
  const bundles = kind === "bundles";
  return (
    <Empty
      action={
        <Button
          render={<Link search={{ kind: bundles ? "bundles" : "extensions" }} to="/marketplace" />}
          nativeButton={false}
          size="sm"
        >
          Browse marketplace
        </Button>
      }
      description={
        bundles
          ? "A bundle activates a curated set of capabilities in one step."
          : "Install one from the marketplace or run agh extension install."
      }
      icon={PackageOpen}
      title={bundles ? "No bundles activated" : "No extensions installed"}
    />
  );
}
