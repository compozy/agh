import { CircleAlert } from "lucide-react";

import { Alert, AlertDescription, AlertTitle } from "@agh/ui";

interface LoopFailureDetailData {
  cause: string;
  recovery: string;
}

interface LoopFailureDetailProps {
  failure: LoopFailureDetailData;
  className?: string;
}

export function LoopFailureDetail({ failure, className }: LoopFailureDetailProps) {
  return (
    <Alert className={className} variant="danger">
      <CircleAlert aria-hidden="true" />
      <AlertTitle>{failure.cause}</AlertTitle>
      <AlertDescription>{failure.recovery}</AlertDescription>
    </Alert>
  );
}

export type { LoopFailureDetailData, LoopFailureDetailProps };
