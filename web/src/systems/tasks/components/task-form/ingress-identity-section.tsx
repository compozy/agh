import { Field, FieldLabel, Input } from "@agh/ui";

interface IngressIdentitySectionProps {
  networkChannel: string;
  identifier: string;
  onNetworkChannel: (value: string) => void;
  onIdentifier: (value: string) => void;
}

/**
 * Ingress & identity — optional peer-ingress channel and a stable identifier
 * override. Both inputs are monospace and unbound by default.
 */
export function IngressIdentitySection({
  networkChannel,
  identifier,
  onNetworkChannel,
  onIdentifier,
}: IngressIdentitySectionProps) {
  return (
    <div className="grid gap-4 md:grid-cols-2">
      <Field>
        <FieldLabel htmlFor="task-network-input">
          Network channel
          <span className="font-normal text-faint"> (peer ingress)</span>
        </FieldLabel>
        <Input
          className="font-mono"
          data-testid="task-network-input"
          id="task-network-input"
          onChange={event => onNetworkChannel(event.target.value)}
          placeholder="ingress channel id"
          value={networkChannel}
        />
      </Field>

      <Field>
        <FieldLabel htmlFor="task-identifier-input">
          Identifier
          <span className="font-normal text-faint"> (override)</span>
        </FieldLabel>
        <Input
          className="font-mono"
          data-testid="task-identifier-input"
          id="task-identifier-input"
          onChange={event => onIdentifier(event.target.value)}
          placeholder="TASK-123"
          value={identifier}
        />
      </Field>
    </div>
  );
}
