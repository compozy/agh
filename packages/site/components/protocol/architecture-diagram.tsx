import { Eyebrow } from "@agh/ui";
const controlNotes = [
  {
    label: "Operator surfaces",
    description:
      "CLI, HTTP/SSE, and the web UI all converge on the same local daemon instead of competing control paths.",
  },
  {
    label: "Runtime core",
    description:
      "Sessions, memory, skills, workspaces, and observability sit inside one operator-owned control plane.",
  },
  {
    label: "Network edge",
    description:
      "AGH Network stays at the boundary so interoperable coordination does not collapse into one product silo.",
  },
];

/**
 * Deep-dive architecture diagram rendered on the /protocol route.
 * Moved out of the landing page because the content is reference-grade,
 * not marketing-grade. Compose directly or expose through MDX via
 * `mdx-components.tsx`.
 */
export function ArchitectureDiagram() {
  return renderArchitectureDiagram();
}

function renderArchitectureDiagram() {
  return (
    <section className="bg-canvas-soft px-4 py-16 md:py-20">
      <div className="mx-auto max-w-site-layout-width">
        <div className="flex flex-col gap-12 lg:flex-row lg:items-start lg:justify-between lg:gap-16">
          <div className="max-w-135">
            <Eyebrow className="text-subtle">CONTROL PLANE</Eyebrow>
            <h2 className="mt-5 text-site-protocol-title leading-none font-semibold tracking-tight text-fg">
              One runtime for operator control and open coordination.
            </h2>
            <p className="mt-6 text-lg leading-relaxed text-muted">
              AGH keeps the operator surface, durable runtime behavior, and open network boundary in
              one place so teams can run real agent work without assembling another pile of
              infrastructure.
            </p>
          </div>

          <div className="w-full lg:max-w-105">
            <div className="rounded-xl bg-canvas p-6 md:p-8">
              <Eyebrow className="text-accent">Reading guide</Eyebrow>
              <div className="mt-6 flex flex-col gap-6">
                {controlNotes.map(note => (
                  <div
                    key={note.label}
                    className="border-b border-line pb-6 last:border-b-0 last:pb-0"
                  >
                    <p className="text-lg font-medium text-fg">{note.label}</p>
                    <p className="mt-2 text-sm leading-relaxed text-muted">{note.description}</p>
                  </div>
                ))}
              </div>
            </div>
          </div>
        </div>

        <div className="mt-16 rounded-xl bg-canvas p-6 md:p-10">
          <div className="flex flex-col gap-4 border-b border-line pb-6 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <Eyebrow className="text-accent">Runtime map</Eyebrow>
              <p className="mt-2 text-lg font-medium leading-tight text-fg">
                Operator surfaces feed one local daemon, which exposes AGH Network at the edge
              </p>
            </div>
          </div>

          <div className="mt-6 overflow-x-auto">
            <svg
              viewBox="0 0 800 480"
              className="mx-auto w-full max-w-4xl"
              aria-label="AGH runtime diagram showing operator surfaces feeding one local control plane with sessions, memory, observability, and AGH Network at the boundary"
              role="img"
            >
              {/* Definitions */}
              <defs>
                <marker
                  id="arrowhead"
                  markerWidth="8"
                  markerHeight="6"
                  refX="8"
                  refY="3"
                  orient="auto"
                >
                  <path d="M0,0 L8,3 L0,6" fill="var(--color-line)" />
                </marker>
              </defs>

              {/* Client layer */}
              <g>
                <text
                  x="400"
                  y="24"
                  textAnchor="middle"
                  fill="var(--color-subtle)"
                  fontSize="10"
                  fontFamily="var(--font-mono)"
                  letterSpacing="0.08em"
                >
                  OPERATOR SURFACES
                </text>
                {[
                  { label: "CLI", x: 160 },
                  { label: "Web UI", x: 400 },
                  { label: "Automations", x: 640 },
                ].map(client => (
                  <g key={client.label}>
                    <rect
                      x={client.x - 60}
                      y={36}
                      width={120}
                      height={36}
                      rx={8}
                      fill="var(--color-canvas-soft)"
                    />
                    <text
                      x={client.x}
                      y={58}
                      textAnchor="middle"
                      fill="var(--color-fg)"
                      fontSize="13"
                      fontFamily="var(--font-sans)"
                      fontWeight="500"
                    >
                      {client.label}
                    </text>
                  </g>
                ))}
              </g>

              {/* Connection arrows */}
              {[160, 400, 640].map(x => (
                <line
                  key={x}
                  x1={x}
                  y1={72}
                  x2={x}
                  y2={108}
                  stroke="var(--color-line)"
                  strokeWidth={1}
                  markerEnd="url(#arrowhead)"
                />
              ))}

              <text
                x={160}
                y={96}
                textAnchor="middle"
                fill="var(--color-subtle)"
                fontSize="9"
                fontFamily="var(--font-mono)"
                letterSpacing="0.06em"
              >
                operators
              </text>
              <text
                x={400}
                y={96}
                textAnchor="middle"
                fill="var(--color-subtle)"
                fontSize="9"
                fontFamily="var(--font-mono)"
                letterSpacing="0.06em"
              >
                live state
              </text>
              <text
                x={640}
                y={96}
                textAnchor="middle"
                fill="var(--color-subtle)"
                fontSize="9"
                fontFamily="var(--font-mono)"
                letterSpacing="0.06em"
              >
                scheduled work
              </text>

              {/* Daemon box */}
              <rect
                x={60}
                y={110}
                width={680}
                height={240}
                rx={12}
                fill="none"
                stroke="var(--color-line)"
                strokeWidth={1}
              />
              <text
                x={80}
                y={134}
                fill="var(--color-accent)"
                fontSize="11"
                fontFamily="var(--font-mono)"
                fontWeight="600"
                letterSpacing="0.06em"
              >
                AGH DAEMON
              </text>

              {/* API Layer */}
              <rect x={80} y={148} width={300} height={36} rx={8} fill="var(--color-canvas-soft)" />
              <text
                x={230}
                y={170}
                textAnchor="middle"
                fill="var(--color-fg)"
                fontSize="12"
                fontFamily="var(--font-sans)"
                fontWeight="500"
              >
                Operator Surfaces
              </text>

              <rect
                x={420}
                y={148}
                width={300}
                height={36}
                rx={8}
                fill="var(--color-canvas-soft)"
              />
              <text
                x={570}
                y={170}
                textAnchor="middle"
                fill="var(--color-fg)"
                fontSize="12"
                fontFamily="var(--font-sans)"
                fontWeight="500"
              >
                Managed Agent Execution
              </text>

              {/* Arrow from API to Session Manager */}
              <line
                x1={230}
                y1={184}
                x2={230}
                y2={210}
                stroke="var(--color-line)"
                strokeWidth={1}
                markerEnd="url(#arrowhead)"
              />
              <line
                x1={570}
                y1={184}
                x2={570}
                y2={210}
                stroke="var(--color-line)"
                strokeWidth={1}
                markerEnd="url(#arrowhead)"
              />

              {/* Session Manager */}
              <rect
                x={80}
                y={212}
                width={640}
                height={40}
                rx={8}
                fill="var(--color-canvas-soft)"
                stroke="var(--color-line)"
                strokeWidth={1}
              />
              <text
                x={400}
                y={236}
                textAnchor="middle"
                fill="var(--color-fg)"
                fontSize="13"
                fontFamily="var(--font-sans)"
                fontWeight="600"
              >
                Session Manager
              </text>

              {/* Bottom row: subsystems */}
              {[
                { label: "Memory", x: 120 },
                { label: "Skills", x: 268 },
                { label: "Workspaces", x: 400 },
                { label: "Observe", x: 532 },
                { label: "Automation", x: 680 },
              ].map(mod => (
                <g key={mod.label}>
                  <line
                    x1={mod.x}
                    y1={252}
                    x2={mod.x}
                    y2={276}
                    stroke="var(--color-line)"
                    strokeWidth={1}
                    markerEnd="url(#arrowhead)"
                  />
                  <rect
                    x={mod.x - 56}
                    y={278}
                    width={112}
                    height={32}
                    rx={6}
                    fill="var(--color-canvas-soft)"
                  />
                  <text
                    x={mod.x}
                    y={298}
                    textAnchor="middle"
                    fill="var(--color-muted)"
                    fontSize="12"
                    fontFamily="var(--font-sans)"
                    fontWeight="500"
                  >
                    {mod.label}
                  </text>
                </g>
              ))}

              {/* Protocol layer */}
              <text
                x="400"
                y={370}
                textAnchor="middle"
                fill="var(--color-subtle)"
                fontSize="10"
                fontFamily="var(--font-mono)"
                letterSpacing="0.08em"
              >
                AGH NETWORK
              </text>

              {/* Arrow from daemon to protocol */}
              <line
                x1={400}
                y1={350}
                x2={400}
                y2={362}
                stroke="var(--color-line)"
                strokeWidth={1}
                markerEnd="url(#arrowhead)"
              />

              {/* Protocol box */}
              <rect
                x={140}
                y={380}
                width={520}
                height={44}
                rx={8}
                fill="none"
                stroke="var(--color-accent)"
                strokeWidth={1}
                strokeDasharray="4 3"
              />
              <text
                x={400}
                y={406}
                textAnchor="middle"
                fill="var(--color-accent)"
                fontSize="13"
                fontFamily="var(--font-sans)"
                fontWeight="600"
              >
                Open coordination layer for discovery, delegation, and updates
              </text>

              {/* External peers */}
              {[
                { label: "Runtime A", x: 200 },
                { label: "Runtime B", x: 400 },
                { label: "Runtime C", x: 600 },
              ].map(peer => (
                <g key={peer.label}>
                  <line
                    x1={peer.x}
                    y1={424}
                    x2={peer.x}
                    y2={444}
                    stroke="var(--color-line)"
                    strokeWidth={1}
                    markerEnd="url(#arrowhead)"
                  />
                  <rect
                    x={peer.x - 48}
                    y={446}
                    width={96}
                    height={28}
                    rx={6}
                    fill="var(--color-canvas-soft)"
                  />
                  <text
                    x={peer.x}
                    y={464}
                    textAnchor="middle"
                    fill="var(--color-muted)"
                    fontSize="11"
                    fontFamily="var(--font-sans)"
                    fontWeight="500"
                  >
                    {peer.label}
                  </text>
                </g>
              ))}
            </svg>
          </div>
        </div>
      </div>
    </section>
  );
}
