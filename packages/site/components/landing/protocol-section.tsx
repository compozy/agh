const messageKinds = [
  {
    name: "request",
    description: "Ask a peer to perform work and expect a response.",
  },
  {
    name: "response",
    description: "Return the result of a completed request.",
  },
  {
    name: "notify",
    description: "Fire-and-forget event broadcast to interested peers.",
  },
  {
    name: "discover",
    description: "Announce capabilities and find peers on the network.",
  },
  {
    name: "subscribe",
    description: "Register interest in a topic or event stream.",
  },
  {
    name: "cancel",
    description: "Abort an in-flight request or interaction.",
  },
  {
    name: "error",
    description: "Signal failure with structured error metadata.",
  },
];

const lifecycleSteps = [
  { step: "1", label: "Discover", description: "Peers announce capabilities" },
  { step: "2", label: "Request", description: "Sender issues a request envelope" },
  { step: "3", label: "Process", description: "Receiver handles the request" },
  { step: "4", label: "Respond", description: "Result returned to sender" },
];

export function ProtocolSection() {
  return (
    <section className="px-4 py-16 md:py-24">
      <div className="mx-auto max-w-5xl">
        <p className="text-center font-mono text-xs font-semibold uppercase tracking-[0.08em] text-[var(--color-text-tertiary)]">
          PROTOCOL
        </p>
        <h2 className="mt-3 text-center text-3xl font-bold tracking-tight text-[var(--color-text-primary)] md:text-4xl">
          An open wire format for agent coordination
        </h2>
        <p className="mx-auto mt-4 max-w-2xl text-center text-sm leading-relaxed text-[var(--color-text-secondary)]">
          Keep your runtime, map to AGH envelopes, implement the smallest core first. Seven message
          kinds cover the full interaction lifecycle.
        </p>

        {/* Message Kinds */}
        <div className="mt-12">
          <h3 className="font-mono text-xs font-semibold uppercase tracking-[0.08em] text-[var(--color-text-tertiary)]">
            MESSAGE KINDS
          </h3>
          <div className="mt-4 grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
            {messageKinds.map(kind => (
              <div
                key={kind.name}
                className="rounded-lg border border-[var(--color-divider)] bg-[var(--color-surface)] p-4"
              >
                <span className="font-mono text-sm font-semibold text-[var(--color-accent)]">
                  {kind.name}
                </span>
                <p className="mt-1.5 text-xs leading-relaxed text-[var(--color-text-secondary)]">
                  {kind.description}
                </p>
              </div>
            ))}
          </div>
        </div>

        {/* Interaction Lifecycle */}
        <div className="mt-12">
          <h3 className="font-mono text-xs font-semibold uppercase tracking-[0.08em] text-[var(--color-text-tertiary)]">
            INTERACTION LIFECYCLE
          </h3>
          <div className="mt-4 flex flex-col gap-0 sm:flex-row sm:items-start sm:gap-0">
            {lifecycleSteps.map((item, i) => (
              <div
                key={item.step}
                className="flex flex-1 items-start gap-3 sm:flex-col sm:items-center sm:text-center"
              >
                <div className="flex shrink-0 items-center gap-3 sm:flex-col sm:gap-2">
                  <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-[var(--color-surface-elevated)] font-mono text-sm font-semibold text-[var(--color-text-primary)]">
                    {item.step}
                  </div>
                  {i < lifecycleSteps.length - 1 && (
                    <div className="hidden h-px w-full bg-[var(--color-divider)] sm:block" />
                  )}
                </div>
                <div className="sm:mt-3">
                  <p className="text-sm font-semibold text-[var(--color-text-primary)]">
                    {item.label}
                  </p>
                  <p className="mt-0.5 text-xs text-[var(--color-text-secondary)]">
                    {item.description}
                  </p>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
    </section>
  );
}
