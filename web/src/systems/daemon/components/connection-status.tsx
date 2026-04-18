import { ConnectionIndicator, type ConnectionStatus as ConnectionStatusType } from "@agh/ui";

interface ConnectionStatusProps {
  status: ConnectionStatusType;
}

function ConnectionStatus({ status }: ConnectionStatusProps) {
  return <ConnectionIndicator status={status} />;
}

export { ConnectionStatus };
