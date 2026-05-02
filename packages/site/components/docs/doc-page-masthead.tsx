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
    .filter(part => part.length > 0)
    .map(part => part[0].toUpperCase() + part.slice(1))
    .join(" ");
}

function resolveRuntimeSection(slug: string[]) {
  if (slug.length === 0) {
    return "Runtime Overview";
  }

  const [root, section] = slug;

  if (root === "core") {
    return section ? toLabel(section) : "Core Concepts";
  }

  if (root === "cli-reference") {
    return section ? toLabel(section) : "CLI Reference";
  }

  if (root === "api-reference") {
    return "API Reference";
  }

  return toLabel(root);
}

function resolveMeta(kind: "runtime" | "protocol", slug: string[]) {
  if (kind === "runtime") {
    return {
      eyebrow: "AGH Runtime",
      audience: "Operators running durable agent work",
      section: resolveRuntimeSection(slug),
    };
  }

  const family = slug[0];

  return {
    eyebrow: family === "specification" ? "AGH Network Protocol" : "AGH Network",
    audience: "Implementers designing interoperable agents",
    section: toLabel(family ?? "overview"),
  };
}

export function DocPageMasthead({ kind, slug, title, description }: DocPageMastheadProps) {
  const meta = resolveMeta(kind, slug);

  return (
    <header className="not-prose border-b border-(--color-divider) pb-8">
      <div className="flex flex-wrap items-center gap-3 font-mono text-[10px] font-semibold uppercase tracking-[0.16em] text-(--color-text-tertiary)">
        <span className="text-(--color-accent)">{meta.eyebrow}</span>
        <span className="h-px w-8 bg-(--color-divider)" />
        <span>{meta.section}</span>
      </div>

      <h1 className="mt-5 max-w-[12ch] font-display text-[clamp(2.55rem,4.7vw,4rem)] leading-[0.98] font-normal tracking-[-0.025em] text-(--color-text-primary)">
        {title}
      </h1>

      {description && (
        <p className="mt-4 max-w-[68ch] text-[1.02rem] leading-8 text-(--color-text-secondary)">
          {description}
        </p>
      )}

      <dl className="mt-6 grid gap-5 border-t border-(--color-divider) pt-4 md:grid-cols-2 xl:max-w-3xl">
        <div>
          <dt className="font-mono text-[10px] font-semibold uppercase tracking-[0.14em] text-(--color-text-tertiary)">
            Audience
          </dt>
          <dd className="mt-2 text-sm leading-6 text-(--color-text-secondary)">{meta.audience}</dd>
        </div>
        <div>
          <dt className="font-mono text-[10px] font-semibold uppercase tracking-[0.14em] text-(--color-text-tertiary)">
            Focus
          </dt>
          <dd className="mt-2 text-sm leading-6 text-(--color-text-secondary)">
            {meta.section} guidance shaped for scanability, day-two clarity, and operator context.
          </dd>
        </div>
      </dl>
    </header>
  );
}
