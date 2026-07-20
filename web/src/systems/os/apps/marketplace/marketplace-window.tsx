import { BundleActivationDetail } from "@/systems/extensions";
import {
  isMarketplaceKind,
  isMarketplaceRouteKind,
  marketplaceApiKindFor,
  MarketplaceKindPage,
  validateMarketplaceKindSearch,
} from "@/systems/marketplace";
import { useTopbarSlot } from "@agh/ui";
import { useDesktop } from "../../hooks/use-desktop";
import { MarketplaceDetailLocation } from "./marketplace-detail-location";
import { validateMarketplaceDetailSearch } from "./marketplace-detail-search";
import { MarketplaceFrame } from "./marketplace-frame";

function decodePathSegment(value: string): string {
  try {
    return decodeURIComponent(value);
  } catch {
    return value;
  }
}

function BundleActivationLocation({ id }: { id: string }) {
  useTopbarSlot({ crumb: `Marketplace / Bundles / ${id}` });
  return <BundleActivationDetail id={id} />;
}

/** Marketplace app controller driven exclusively by the logical window's WM location. */
export function MarketplaceWindow({ windowId }: { windowId: string }) {
  const location = useDesktop(
    state => state.windows[windowId]?.location ?? { pathname: "/marketplace/skills", search: {} }
  );
  const segments = location.pathname.split("/").filter(Boolean);

  if (segments[1] === "bundles" && segments[2] === "activations" && segments[3]) {
    return (
      <MarketplaceFrame deep pathname={location.pathname}>
        <BundleActivationLocation id={decodePathSegment(segments[3])} />
      </MarketplaceFrame>
    );
  }

  if (isMarketplaceKind(segments[1]) && segments[2]) {
    return (
      <MarketplaceFrame deep pathname={location.pathname}>
        <MarketplaceDetailLocation
          entryId={decodePathSegment(segments[2])}
          kind={segments[1]}
          search={validateMarketplaceDetailSearch(location.search)}
        />
      </MarketplaceFrame>
    );
  }

  const routeKind = isMarketplaceRouteKind(segments[1]) ? segments[1] : "skills";
  return (
    <MarketplaceFrame deep={false} pathname={location.pathname}>
      <MarketplaceKindPage
        kind={marketplaceApiKindFor(routeKind)}
        search={validateMarketplaceKindSearch(location.search)}
      />
    </MarketplaceFrame>
  );
}
