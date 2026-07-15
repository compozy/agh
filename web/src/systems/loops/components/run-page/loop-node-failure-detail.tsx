import type { LoopActionFailure } from "../../lib/action-failure";
import { LoopFailureDetail } from "./loop-failure-detail";

interface LoopNodeFailureDetailProps {
  failure: LoopActionFailure;
}

export function LoopNodeFailureDetail({ failure }: LoopNodeFailureDetailProps) {
  return <LoopFailureDetail className="mt-3" failure={failure} />;
}
