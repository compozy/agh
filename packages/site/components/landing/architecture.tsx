export function Architecture() {
  return (
    <section className="bg-[var(--color-surface)] px-4 py-16 md:py-24">
      <div className="mx-auto max-w-5xl">
        <p className="text-center font-mono text-xs font-semibold uppercase tracking-[0.08em] text-[var(--color-text-tertiary)]">
          ARCHITECTURE
        </p>
        <h2 className="mt-3 text-center text-3xl font-bold tracking-tight text-[var(--color-text-primary)] md:text-4xl">
          How it all fits together
        </h2>

        <div className="mt-12 overflow-x-auto">
          <svg
            viewBox="0 0 800 480"
            className="mx-auto w-full max-w-3xl"
            aria-label="AGH architecture diagram showing CLI, HTTP, and UDS clients connecting to the daemon, which manages sessions, memory, skills, and the agent network protocol"
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
                <path d="M0,0 L8,3 L0,6" fill="#3A3A3C" />
              </marker>
            </defs>

            {/* Client layer */}
            <g>
              <text
                x="400"
                y="24"
                textAnchor="middle"
                fill="#636366"
                fontSize="10"
                fontFamily="JetBrains Mono, monospace"
                letterSpacing="0.08em"
              >
                CLIENTS
              </text>
              {[
                { label: "CLI", x: 160 },
                { label: "Web UI", x: 400 },
                { label: "Agents", x: 640 },
              ].map(client => (
                <g key={client.label}>
                  <rect x={client.x - 60} y={36} width={120} height={36} rx={8} fill="#2C2C2E" />
                  <text
                    x={client.x}
                    y={58}
                    textAnchor="middle"
                    fill="#E5E5E7"
                    fontSize="13"
                    fontFamily="Inter, sans-serif"
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
                stroke="#3A3A3C"
                strokeWidth={1}
                markerEnd="url(#arrowhead)"
              />
            ))}

            {/* Transport labels */}
            <text
              x={160}
              y={96}
              textAnchor="middle"
              fill="#636366"
              fontSize="9"
              fontFamily="JetBrains Mono, monospace"
              letterSpacing="0.06em"
            >
              UDS
            </text>
            <text
              x={400}
              y={96}
              textAnchor="middle"
              fill="#636366"
              fontSize="9"
              fontFamily="JetBrains Mono, monospace"
              letterSpacing="0.06em"
            >
              HTTP/SSE
            </text>
            <text
              x={640}
              y={96}
              textAnchor="middle"
              fill="#636366"
              fontSize="9"
              fontFamily="JetBrains Mono, monospace"
              letterSpacing="0.06em"
            >
              JSON-RPC
            </text>

            {/* Daemon box */}
            <rect
              x={60}
              y={110}
              width={680}
              height={240}
              rx={12}
              fill="none"
              stroke="#3A3A3C"
              strokeWidth={1}
            />
            <text
              x={80}
              y={134}
              fill="#E8572A"
              fontSize="11"
              fontFamily="JetBrains Mono, monospace"
              fontWeight="600"
              letterSpacing="0.06em"
            >
              AGH DAEMON
            </text>

            {/* API Layer */}
            <rect x={80} y={148} width={300} height={36} rx={8} fill="#2C2C2E" />
            <text
              x={230}
              y={170}
              textAnchor="middle"
              fill="#E5E5E7"
              fontSize="12"
              fontFamily="Inter, sans-serif"
              fontWeight="500"
            >
              API Layer (HTTP + UDS)
            </text>

            <rect x={420} y={148} width={300} height={36} rx={8} fill="#2C2C2E" />
            <text
              x={570}
              y={170}
              textAnchor="middle"
              fill="#E5E5E7"
              fontSize="12"
              fontFamily="Inter, sans-serif"
              fontWeight="500"
            >
              ACP Client (JSON-RPC / stdio)
            </text>

            {/* Arrow from API to Session Manager */}
            <line
              x1={230}
              y1={184}
              x2={230}
              y2={210}
              stroke="#3A3A3C"
              strokeWidth={1}
              markerEnd="url(#arrowhead)"
            />
            <line
              x1={570}
              y1={184}
              x2={570}
              y2={210}
              stroke="#3A3A3C"
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
              fill="#1C1C1E"
              stroke="#3A3A3C"
              strokeWidth={1}
            />
            <text
              x={400}
              y={236}
              textAnchor="middle"
              fill="#E5E5E7"
              fontSize="13"
              fontFamily="Inter, sans-serif"
              fontWeight="600"
            >
              Session Manager
            </text>

            {/* Bottom row: subsystems */}
            {[
              { label: "Memory", x: 120 },
              { label: "Skills", x: 268 },
              { label: "Store", x: 400 },
              { label: "Observe", x: 532 },
              { label: "Config", x: 680 },
            ].map(mod => (
              <g key={mod.label}>
                <line
                  x1={mod.x}
                  y1={252}
                  x2={mod.x}
                  y2={276}
                  stroke="#3A3A3C"
                  strokeWidth={1}
                  markerEnd="url(#arrowhead)"
                />
                <rect x={mod.x - 56} y={278} width={112} height={32} rx={6} fill="#2C2C2E" />
                <text
                  x={mod.x}
                  y={298}
                  textAnchor="middle"
                  fill="#8E8E93"
                  fontSize="12"
                  fontFamily="Inter, sans-serif"
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
              fill="#636366"
              fontSize="10"
              fontFamily="JetBrains Mono, monospace"
              letterSpacing="0.08em"
            >
              AGENT NETWORK
            </text>

            {/* Arrow from daemon to protocol */}
            <line
              x1={400}
              y1={350}
              x2={400}
              y2={362}
              stroke="#3A3A3C"
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
              stroke="#E8572A"
              strokeWidth={1}
              strokeDasharray="4 3"
            />
            <text
              x={400}
              y={406}
              textAnchor="middle"
              fill="#E8572A"
              fontSize="13"
              fontFamily="Inter, sans-serif"
              fontWeight="600"
            >
              AGH Network Protocol (Envelopes over NATS)
            </text>

            {/* External peers */}
            {[
              { label: "Peer A", x: 200 },
              { label: "Peer B", x: 400 },
              { label: "Peer C", x: 600 },
            ].map(peer => (
              <g key={peer.label}>
                <line
                  x1={peer.x}
                  y1={424}
                  x2={peer.x}
                  y2={444}
                  stroke="#3A3A3C"
                  strokeWidth={1}
                  markerEnd="url(#arrowhead)"
                />
                <rect x={peer.x - 48} y={446} width={96} height={28} rx={6} fill="#2C2C2E" />
                <text
                  x={peer.x}
                  y={464}
                  textAnchor="middle"
                  fill="#8E8E93"
                  fontSize="11"
                  fontFamily="Inter, sans-serif"
                  fontWeight="500"
                >
                  {peer.label}
                </text>
              </g>
            ))}
          </svg>
        </div>
      </div>
    </section>
  );
}
