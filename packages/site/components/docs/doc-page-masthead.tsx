interface DocPageMastheadProps {
  kind: "runtime" | "protocol";
  slug: string[];
  title: string;
  description?: string;
}

function toLabel(value?: string) {
  if (!value) {
    return "Overview";
  }

  return value
    .split("-")
    .map(part => part[0]?.toUpperCase() + part.slice(1))
    .join(" ");
}

function resolveMeta(kind: "runtime" | "protocol", slug: string[]) {
  const family = slug[0];

  if (kind === "runtime") {
    return {
      eyebrow: "AGH Runtime",
      audience: "Operators running durable agent work",
      section: toLabel(family ?? "core"),
    };
  }

  return {
    eyebrow: family === "specification" ? "AGH Network Protocol" : "AGH Network",
    audience: "Implementers designing interoperable agents",
    section: toLabel(family ?? "overview"),
  };
}

export function DocPageMasthead({ kind, slug, title, description }: DocPageMastheadProps) {
  const meta = resolveMeta(kind, slug);

  return (
    <header className="not-prose border-b border-[var(--color-divider)] pb-8">
      <div className="flex flex-wrap items-center gap-3 font-mono text-[10px] font-semibold uppercase tracking-[0.16em] text-[var(--color-text-tertiary)]">
        <span className="text-[var(--color-accent)]">{meta.eyebrow}</span>
        <span className="h-px w-8 bg-[var(--color-divider)]" />
        <span>{meta.section}</span>
      </div>

      <h1 className="mt-5 max-w-[12ch] text-[clamp(2.55rem,4.7vw,4rem)] leading-[0.94] font-semibold tracking-[-0.05em] text-[var(--color-text-primary)]">
        {title}
      </h1>

      {description && (
        <p className="mt-4 max-w-[68ch] text-[1.02rem] leading-8 text-[var(--color-text-secondary)]">
          {description}
        </p>
      )}

      <dl className="mt-6 grid gap-5 border-t border-[var(--color-divider)] pt-4 md:grid-cols-2 xl:max-w-[48rem]">
        <div>
          <dt className="font-mono text-[10px] font-semibold uppercase tracking-[0.14em] text-[var(--color-text-tertiary)]">
            Audience
          </dt>
          <dd className="mt-2 text-sm leading-6 text-[var(--color-text-secondary)]">
            {meta.audience}
          </dd>
        </div>
        <div>
          <dt className="font-mono text-[10px] font-semibold uppercase tracking-[0.14em] text-[var(--color-text-tertiary)]">
            Focus
          </dt>
          <dd className="mt-2 text-sm leading-6 text-[var(--color-text-secondary)]">
            {meta.section} guidance shaped for scanability, day-two clarity, and operator context.
          </dd>
        </div>
      </dl>
    </header>
  );
}
