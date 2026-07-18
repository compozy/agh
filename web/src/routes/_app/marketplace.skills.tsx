import { createFileRoute } from "@tanstack/react-router";

import {
  MarketplaceKindPage,
  validateMarketplaceKindSearch,
  type MarketplaceKindSearch,
} from "@/systems/marketplace";
import type { TopbarRouteContext } from "@/types/topbar";

export type { MarketplaceKindSearch };

const MARKETPLACE_SKILLS_TOPBAR_CONTEXT: { topbar: TopbarRouteContext } = {
  topbar: { crumb: { label: "Skills" } },
};

export const Route = createFileRoute("/_app/marketplace/skills")({
  beforeLoad: (): { topbar: TopbarRouteContext } => MARKETPLACE_SKILLS_TOPBAR_CONTEXT,
  validateSearch: validateMarketplaceKindSearch,
  component: MarketplaceSkillsRoute,
});

function MarketplaceSkillsRoute() {
  return <MarketplaceKindPage kind="skill" search={Route.useSearch()} />;
}
