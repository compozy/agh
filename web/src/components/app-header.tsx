import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbList,
  BreadcrumbPage,
} from "@/components/ui/breadcrumb";
import { ConnectionStatus } from "@/systems/daemon/components/connection-status";
import { useDaemonHealth } from "@/systems/daemon/hooks/use-daemon-health";

function AppHeader() {
  const { connectionStatus } = useDaemonHealth();

  return (
    <header className="flex h-12 shrink-0 items-center gap-2 border-b border-[color:var(--ds-line-subtle)] px-4">
      <Breadcrumb>
        <BreadcrumbList>
          <BreadcrumbItem>
            <BreadcrumbPage className="font-mono text-[0.68rem] uppercase tracking-[0.2em] text-[color:var(--ds-text-mono)]">
              Dashboard
            </BreadcrumbPage>
          </BreadcrumbItem>
        </BreadcrumbList>
      </Breadcrumb>
      <div className="ml-auto">
        <ConnectionStatus status={connectionStatus} />
      </div>
    </header>
  );
}

export { AppHeader };
