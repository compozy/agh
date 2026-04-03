import { createFileRoute } from "@tanstack/react-router";

import { DesignSystemShowcase } from "@/components/design-system";

export const Route = createFileRoute("/design-system")({
  component: DesignSystemPage,
});

function DesignSystemPage() {
  return <DesignSystemShowcase />;
}
