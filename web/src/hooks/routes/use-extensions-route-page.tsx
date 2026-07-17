import { Link, useChildMatches, useNavigate } from "@tanstack/react-router";

import { Button, PillGroup, useTopbarSlot } from "@agh/ui";
import { useBundleActivations, useExtensionInventory } from "@/systems/extensions";

interface ExtensionsRouteSearch {
  tab?: "bundles";
}

export function useExtensionsRoutePage(search: ExtensionsRouteSearch) {
  const navigate = useNavigate({ from: "/extensions" });
  const child = useChildMatches().length > 0;
  const extensions = useExtensionInventory();
  const bundles = useBundleActivations();
  const tab: "extensions" | "bundles" = search.tab ?? "extensions";
  useTopbarSlot({
    count: child ? undefined : tab === "bundles" ? bundles.data?.length : extensions.data.length,
    tabs: child ? undefined : (
      <PillGroup
        aria-label="Extensions section"
        items={[
          { label: "Extensions", value: "extensions" },
          { label: "Bundles", value: "bundles" },
        ]}
        onChange={next =>
          void navigate({ search: next === "bundles" ? { tab: "bundles" } : {}, to: "/extensions" })
        }
        value={tab}
      />
    ),
    actions: child ? undefined : (
      <Button
        render={
          <Link search={{ kind: tab === "bundles" ? "bundles" : "extensions" }} to="/marketplace" />
        }
        nativeButton={false}
        size="sm"
        variant="ghost"
      >
        Browse marketplace
      </Button>
    ),
  });
  return { child, tab };
}
