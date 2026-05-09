import { KindIcon, providerKindIconRegistry, type KindIconProps } from "@agh/ui";

type AgentIconTone = KindIconProps["tone"];

interface AgentIconProps extends Omit<KindIconProps, "kind" | "registry"> {
  provider: string;
}

const providerIconMap = providerKindIconRegistry;

function AgentIcon({
  provider,
  "data-slot": dataSlot = "agent-icon",
  "data-provider": dataProvider,
  ...props
}: AgentIconProps) {
  const key = provider.trim().toLowerCase();
  return (
    <KindIcon
      data-slot={dataSlot}
      data-provider={dataProvider ?? key}
      kind={key}
      registry={providerKindIconRegistry}
      {...props}
    />
  );
}

export { AgentIcon, providerIconMap };
export type { AgentIconProps, AgentIconTone };
