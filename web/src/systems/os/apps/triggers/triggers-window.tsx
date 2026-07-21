import { useDesktop } from "../../hooks/use-desktop";
import { validateTriggersSearch } from "../automation/use-automation-page";
import { TriggerDetailLocation } from "./trigger-detail-location";
import { TriggersCatalogLocation } from "./triggers-catalog-location";

function decodePathSegment(value: string): string {
  try {
    return decodeURIComponent(value);
  } catch {
    return value;
  }
}

/** Triggers app controller driven exclusively by the logical window's WM location. */
export function TriggersWindow({ windowId }: { windowId: string }) {
  const location = useDesktop(
    state => state.windows[windowId]?.location ?? { pathname: "/triggers", search: {} }
  );
  const detail = /^\/triggers\/([^/]+)$/.exec(location.pathname);
  if (detail) return <TriggerDetailLocation triggerId={decodePathSegment(detail[1])} />;
  return <TriggersCatalogLocation search={validateTriggersSearch(location.search)} />;
}
