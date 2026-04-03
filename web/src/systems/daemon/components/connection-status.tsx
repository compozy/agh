import {
  ConnectionIndicator,
  type ConnectionStatus as ConnectionStatusType,
} from "@/components/design-system/connection-indicator";

interface ConnectionStatusProps {
  status: ConnectionStatusType;
}

function ConnectionStatus({ status }: ConnectionStatusProps) {
  return <ConnectionIndicator status={status} />;
}

export { ConnectionStatus };
