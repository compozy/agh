export interface NewDividerProps {
  label?: string;
}

export function NewDivider({ label = "NEW" }: NewDividerProps) {
  return (
    <div
      aria-label="New messages"
      className="my-4 flex items-center gap-3 px-5"
      data-testid="network-timeline-new-divider"
      role="separator"
    >
      <span
        aria-hidden="true"
        className="h-px flex-1 bg-accent"
        data-testid="network-timeline-new-divider-line"
      />
      <span className="text-eyebrow font-semibold tracking-mono text-accent">{label}</span>
      <span aria-hidden="true" className="h-px flex-1 bg-accent" />
    </div>
  );
}
