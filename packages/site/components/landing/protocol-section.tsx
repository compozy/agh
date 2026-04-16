import Link from "next/link";
import { ArrowRight } from "lucide-react";

export function ProtocolSection() {
  return (
    <section className="px-4 py-20 md:py-28">
      <div className="mx-auto max-w-[var(--site-layout-width)]">
        {/* Full-width header — centered, not the sticky-left pattern */}
        <div className="mx-auto max-w-[720px] text-center">
          <p className="font-mono text-[11px] font-medium uppercase tracking-[0.06em] text-[var(--color-accent)]">
            AGH Network — the differentiator
          </p>
          <h2 className="mt-5 text-[clamp(2.4rem,5vw,3.8rem)] leading-[1.0] font-normal tracking-[-0.03em] text-[var(--color-text-primary)]">
            The only agent runtime with a built-in coordination protocol.
          </h2>
          <p className="mx-auto mt-6 max-w-[58ch] text-base leading-relaxed text-[var(--color-text-secondary)]">
            Every other agent tool stops at the single-runtime boundary. AGH Network is an open
            protocol that lets agents discover peers, delegate work, and exchange structured updates
            across different runtimes — without forcing everyone onto one stack.
          </p>
        </div>

        {/* Architecture diagram — full width, visual proof */}
        <div className="mt-14 rounded-[12px] border border-[var(--color-divider)] bg-[var(--color-surface)] p-6 md:p-10">
          <div className="flex items-center justify-between border-b border-[var(--color-divider)] pb-5">
            <p className="font-mono text-[11px] font-medium uppercase tracking-[0.06em] text-[var(--color-text-tertiary)]">
              How agents coordinate through AGH Network
            </p>
            <span className="rounded-[6px] bg-[var(--color-accent-tint)] px-2 py-1 font-mono text-[10px] font-semibold uppercase tracking-[0.08em] text-[var(--color-accent)]">
              open protocol
            </span>
          </div>

          <div className="mt-6 overflow-x-auto">
            <svg
              viewBox="0 0 800 380"
              className="mx-auto w-full max-w-4xl"
              aria-label="AGH Network coordination flow: agents discover peers, delegate tasks, and exchange updates across runtimes"
              role="img"
            >
              <defs>
                <marker id="arrow" markerWidth="8" markerHeight="6" refX="8" refY="3" orient="auto">
                  <path d="M0,0 L8,3 L0,6" fill="#3A3A3C" />
                </marker>
                <marker
                  id="arrow-accent"
                  markerWidth="8"
                  markerHeight="6"
                  refX="8"
                  refY="3"
                  orient="auto"
                >
                  <path d="M0,0 L8,3 L0,6" fill="#E8572A" />
                </marker>
              </defs>

              {/* Runtime A */}
              <rect
                x={40}
                y={40}
                width={200}
                height={140}
                rx={8}
                fill="none"
                stroke="#3A3A3C"
                strokeWidth={1}
              />
              <text
                x={60}
                y={64}
                fill="#8E8E93"
                fontSize="10"
                fontFamily="JetBrains Mono, monospace"
                letterSpacing="0.06em"
              >
                RUNTIME A
              </text>
              <rect x={60} y={78} width={160} height={30} rx={6} fill="#2C2C2E" />
              <text
                x={140}
                y={97}
                textAnchor="middle"
                fill="#E5E5E7"
                fontSize="12"
                fontFamily="Inter, sans-serif"
                fontWeight="500"
              >
                Coder
              </text>
              <rect x={60} y={118} width={160} height={30} rx={6} fill="#2C2C2E" />
              <text
                x={140}
                y={137}
                textAnchor="middle"
                fill="#8E8E93"
                fontSize="11"
                fontFamily="Inter, sans-serif"
              >
                Sessions + Memory
              </text>
              <rect x={60} y={155} width={72} height={18} rx={4} fill="#30D15826" />
              <text
                x={96}
                y={167}
                textAnchor="middle"
                fill="#30D158"
                fontSize="9"
                fontFamily="JetBrains Mono, monospace"
                fontWeight="600"
              >
                ONLINE
              </text>

              {/* Runtime B */}
              <rect
                x={300}
                y={40}
                width={200}
                height={140}
                rx={8}
                fill="none"
                stroke="#3A3A3C"
                strokeWidth={1}
              />
              <text
                x={320}
                y={64}
                fill="#8E8E93"
                fontSize="10"
                fontFamily="JetBrains Mono, monospace"
                letterSpacing="0.06em"
              >
                RUNTIME B
              </text>
              <rect x={320} y={78} width={160} height={30} rx={6} fill="#2C2C2E" />
              <text
                x={400}
                y={97}
                textAnchor="middle"
                fill="#E5E5E7"
                fontSize="12"
                fontFamily="Inter, sans-serif"
                fontWeight="500"
              >
                Deployer
              </text>
              <rect x={320} y={118} width={160} height={30} rx={6} fill="#2C2C2E" />
              <text
                x={400}
                y={137}
                textAnchor="middle"
                fill="#8E8E93"
                fontSize="11"
                fontFamily="Inter, sans-serif"
              >
                Sessions + Memory
              </text>
              <rect x={320} y={155} width={72} height={18} rx={4} fill="#30D15826" />
              <text
                x={356}
                y={167}
                textAnchor="middle"
                fill="#30D158"
                fontSize="9"
                fontFamily="JetBrains Mono, monospace"
                fontWeight="600"
              >
                ONLINE
              </text>

              {/* Runtime C */}
              <rect
                x={560}
                y={40}
                width={200}
                height={140}
                rx={8}
                fill="none"
                stroke="#3A3A3C"
                strokeWidth={1}
              />
              <text
                x={580}
                y={64}
                fill="#8E8E93"
                fontSize="10"
                fontFamily="JetBrains Mono, monospace"
                letterSpacing="0.06em"
              >
                RUNTIME C
              </text>
              <rect x={580} y={78} width={160} height={30} rx={6} fill="#2C2C2E" />
              <text
                x={660}
                y={97}
                textAnchor="middle"
                fill="#E5E5E7"
                fontSize="12"
                fontFamily="Inter, sans-serif"
                fontWeight="500"
              >
                Reviewer
              </text>
              <rect x={580} y={118} width={160} height={30} rx={6} fill="#2C2C2E" />
              <text
                x={660}
                y={137}
                textAnchor="middle"
                fill="#8E8E93"
                fontSize="11"
                fontFamily="Inter, sans-serif"
              >
                Sessions + Memory
              </text>
              <rect x={580} y={155} width={72} height={18} rx={4} fill="#30D15826" />
              <text
                x={616}
                y={167}
                textAnchor="middle"
                fill="#30D158"
                fontSize="9"
                fontFamily="JetBrains Mono, monospace"
                fontWeight="600"
              >
                ONLINE
              </text>

              {/* Arrows down to protocol layer */}
              <line
                x1={140}
                y1={180}
                x2={140}
                y2={220}
                stroke="#E8572A"
                strokeWidth={1}
                markerEnd="url(#arrow-accent)"
              />
              <line
                x1={400}
                y1={180}
                x2={400}
                y2={220}
                stroke="#E8572A"
                strokeWidth={1}
                markerEnd="url(#arrow-accent)"
              />
              <line
                x1={660}
                y1={180}
                x2={660}
                y2={220}
                stroke="#E8572A"
                strokeWidth={1}
                markerEnd="url(#arrow-accent)"
              />

              {/* AGH Network protocol band */}
              <rect
                x={40}
                y={224}
                width={720}
                height={56}
                rx={8}
                fill="none"
                stroke="#E8572A"
                strokeWidth={1.5}
                strokeDasharray="6 4"
              />
              <text
                x={400}
                y={248}
                textAnchor="middle"
                fill="#E8572A"
                fontSize="12"
                fontFamily="JetBrains Mono, monospace"
                fontWeight="600"
                letterSpacing="0.06em"
              >
                AGH NETWORK
              </text>
              <text
                x={400}
                y={268}
                textAnchor="middle"
                fill="#8E8E93"
                fontSize="11"
                fontFamily="Inter, sans-serif"
              >
                discover &bull; delegate &bull; update &bull; receipt
              </text>

              {/* Horizontal coordination arrows */}
              <line
                x1={240}
                y1={200}
                x2={300}
                y2={200}
                stroke="#E8572A"
                strokeWidth={1}
                strokeDasharray="3 2"
                markerEnd="url(#arrow-accent)"
              />
              <line
                x1={500}
                y1={200}
                x2={560}
                y2={200}
                stroke="#E8572A"
                strokeWidth={1}
                strokeDasharray="3 2"
                markerEnd="url(#arrow-accent)"
              />

              {/* Bottom: what each capability does */}
              <g>
                {[
                  { label: "DISCOVER", desc: "Find peers by capability", x: 120 },
                  { label: "DELEGATE", desc: "Hand off structured tasks", x: 320 },
                  { label: "UPDATE", desc: "Stream progress across runtimes", x: 520 },
                  { label: "RECEIPT", desc: "Confirm completion with results", x: 700 },
                ].map(cap => (
                  <g key={cap.label}>
                    <line
                      x1={cap.x}
                      y1={280}
                      x2={cap.x}
                      y2={306}
                      stroke="#3A3A3C"
                      strokeWidth={1}
                      markerEnd="url(#arrow)"
                    />
                    <text
                      x={cap.x}
                      y={322}
                      textAnchor="middle"
                      fill="#E8572A"
                      fontSize="10"
                      fontFamily="JetBrains Mono, monospace"
                      fontWeight="600"
                      letterSpacing="0.06em"
                    >
                      {cap.label}
                    </text>
                    <text
                      x={cap.x}
                      y={340}
                      textAnchor="middle"
                      fill="#636366"
                      fontSize="10"
                      fontFamily="Inter, sans-serif"
                    >
                      {cap.desc}
                    </text>
                  </g>
                ))}
              </g>
            </svg>
          </div>
        </div>

        {/* Three-column grid — unique layout, not the sticky-left clone */}
        <div className="mt-12 grid gap-6 md:grid-cols-3">
          <div className="rounded-[12px] bg-[var(--color-surface)] p-6">
            <p className="font-mono text-[11px] font-medium uppercase tracking-[0.06em] text-[var(--color-accent)]">
              Step 1
            </p>
            <p className="mt-3 text-base font-medium text-[var(--color-text-primary)]">
              Keep your runtime
            </p>
            <p className="mt-2 text-sm leading-relaxed text-[var(--color-text-secondary)]">
              AGH Network is a protocol boundary, not a demand to replace your agent stack. Adopt it
              alongside your existing control plane.
            </p>
          </div>
          <div className="rounded-[12px] bg-[var(--color-surface)] p-6">
            <p className="font-mono text-[11px] font-medium uppercase tracking-[0.06em] text-[var(--color-accent)]">
              Step 2
            </p>
            <p className="mt-3 text-base font-medium text-[var(--color-text-primary)]">
              Map your agents
            </p>
            <p className="mt-2 text-sm leading-relaxed text-[var(--color-text-secondary)]">
              Connect your internal agent model to the shared envelope. Announce capabilities,
              register for delegation, start receiving work.
            </p>
          </div>
          <div className="rounded-[12px] bg-[var(--color-surface)] p-6">
            <p className="font-mono text-[11px] font-medium uppercase tracking-[0.06em] text-[var(--color-accent)]">
              Step 3
            </p>
            <p className="mt-3 text-base font-medium text-[var(--color-text-primary)]">
              Layer in trust and transport
            </p>
            <p className="mt-2 text-sm leading-relaxed text-[var(--color-text-secondary)]">
              Start with the core contract, then add authentication, transport profiles, and
              governance policies where they matter.
            </p>
          </div>
        </div>

        <div className="mt-8 text-center">
          <Link
            href="/protocol"
            className="inline-flex items-center gap-2 text-sm font-medium text-[var(--color-accent)] transition-colors hover:text-[var(--color-accent-hover)]"
          >
            Read the full AGH Network specification
            <ArrowRight className="h-4 w-4" />
          </Link>
        </div>
      </div>
    </section>
  );
}
