"use client";

import { useEffect, useId, useState } from "react";

let mermaidLoader: Promise<typeof import("mermaid").default> | null = null;

function loadMermaid() {
  if (!mermaidLoader) {
    mermaidLoader = import("mermaid").then(({ default: mermaid }) => {
      mermaid.initialize({
        startOnLoad: false,
        securityLevel: "strict",
        theme: "dark",
      });
      return mermaid;
    });
  }

  return mermaidLoader;
}

export function Mermaid({ chart, caption }: { chart: string; caption?: string }) {
  const reactId = useId();
  const diagramId = `agh-mermaid-${reactId.replace(/[^a-zA-Z0-9_-]/g, "")}`;
  const [svg, setSVG] = useState("");
  const [error, setError] = useState("");

  useEffect(() => {
    let active = true;

    setSVG("");
    setError("");

    void loadMermaid()
      .then(async mermaid => {
        const rendered = await mermaid.render(diagramId, chart);
        if (!active) return;
        setSVG(rendered.svg);
      })
      .catch(err => {
        if (!active) return;
        setError(err instanceof Error ? err.message : String(err));
      });

    return () => {
      active = false;
    };
  }, [chart, diagramId]);

  return (
    <figure className="not-prose my-6 overflow-x-auto rounded-lg border border-[var(--color-divider)] bg-[var(--color-surface)] p-4">
      {svg ? (
        <div
          aria-label="Mermaid diagram"
          className="[&_svg]:h-auto [&_svg]:max-w-full"
          dangerouslySetInnerHTML={{ __html: svg }}
        />
      ) : error ? (
        <div>
          <p className="font-mono text-xs font-semibold uppercase tracking-[0.14em] text-[var(--color-accent)]">
            Diagram source
          </p>
          <p className="mt-2 text-sm leading-6 text-[var(--color-text-secondary)]">
            Mermaid could not render this diagram in the current browser session.
          </p>
          <pre className="mt-4 overflow-x-auto rounded-md border border-[var(--color-divider)] bg-[rgba(255,255,255,0.03)] p-3 text-xs leading-6 text-[var(--color-text-secondary)]">
            <code>{chart}</code>
          </pre>
          <p className="mt-3 text-sm leading-6 text-[var(--color-text-tertiary)]">{error}</p>
        </div>
      ) : (
        <p className="text-sm leading-6 text-[var(--color-text-secondary)]">Rendering diagram...</p>
      )}

      {caption ? (
        <figcaption className="mt-3 text-sm leading-6 text-[var(--color-text-secondary)]">
          {caption}
        </figcaption>
      ) : null}
    </figure>
  );
}
