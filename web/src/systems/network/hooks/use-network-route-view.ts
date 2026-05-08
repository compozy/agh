import { useNetworkCreateChannelAction } from "./use-network-create-channel-action";
import { useNetworkInspectorView } from "./use-network-inspector-view";
import { useNetworkRailView } from "./use-network-rail-view";
import { useNetworkRouteShell } from "./use-network-route-shell";
import { useOpenWork } from "./use-work";
import { useThreadViewMode } from "./use-thread-view-mode";

export function useNetworkRouteView() {
  const route = useNetworkRouteShell();
  const viewMode = useThreadViewMode();
  const showOverlayInRightRail = route.activeThreadId != null && viewMode === "overlay";
  const containerSurface = route.activeThreadId
    ? ("thread" as const)
    : route.activeDirectId
      ? ("direct" as const)
      : null;
  const containerId = route.activeThreadId ?? route.activeDirectId ?? null;
  const channelKey = route.activeChannel?.channel ?? null;
  const openWork = useOpenWork({
    channel: channelKey,
    surface: containerSurface,
    containerId,
    enabled: Boolean(channelKey) && containerSurface != null,
  });
  const inspectorView = useNetworkInspectorView({
    channel: channelKey,
    enabled: Boolean(channelKey) && !showOverlayInRightRail,
  });
  const railView = useNetworkRailView({ channel: channelKey });
  const networkCreate = useNetworkCreateChannelAction({ enabled: route.page.isNetworkEnabled });

  return {
    ...route,
    inspectorView,
    networkCreate,
    openWork,
    railView,
    showOverlayInRightRail,
  };
}
