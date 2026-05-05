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
        className="h-px flex-1 bg-[color:var(--color-accent)]"
        data-testid="network-timeline-new-divider-line"
      />
      <span className="text-[11px] font-semibold tracking-[0.06em] text-[color:var(--color-accent)]">
        {label}
      </span>
      <span aria-hidden="true" className="h-px flex-1 bg-[color:var(--color-accent)]" />
    </div>
  );
}
