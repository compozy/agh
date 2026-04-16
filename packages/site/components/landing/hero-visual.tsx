"use client";

import { cn } from "@agh/ui/utils";
import { useReducedMotion } from "./primitives/use-reduced-motion";

type Point = { x: number; y: number };

const CENTER: Point = { x: 280, y: 280 };
const VIEWBOX = 560;
const CORE_R = 36;

const SESSIONS = [
  { id: "coder", driver: "claude-code", pos: { x: 188, y: 188 }, corner: "nw" },
  { id: "reviewer", driver: "codex-cli", pos: { x: 372, y: 188 }, corner: "ne" },
  { id: "analyst", driver: "gemini-cli", pos: { x: 372, y: 372 }, corner: "se" },
  { id: "deployer", driver: "opencode", pos: { x: 188, y: 372 }, corner: "sw" },
] as const;

const PEERS = [
  { id: "peer·eu-west", pos: { x: 280, y: 60 }, mobileHide: false },
  { id: "peer·ci", pos: { x: 470, y: 390 }, mobileHide: true },
  { id: "peer·desk-02", pos: { x: 90, y: 390 }, mobileHide: true },
] as const;

type PacketRoute = {
  id: string;
  d: string;
  label: "greet" | "whois" | "direct" | "receipt";
  delay: number;
  duration: number;
  mobileHide?: boolean;
};

/** Paths below are bezier/straight segments used both for visual lines and <animateMotion>. */
const ROUTES: PacketRoute[] = [
  {
    id: "coder-to-core",
    d: `M 188 188 L ${CENTER.x - 22} ${CENTER.y - 22}`,
    label: "greet",
    delay: 0,
    duration: 2.4,
  },
  {
    id: "core-to-eu",
    d: `M ${CENTER.x} ${CENTER.y - 16} C ${CENTER.x - 40} 180, ${CENTER.x - 60} 120, 280 70`,
    label: "direct",
    delay: 0.6,
    duration: 2.8,
  },
  {
    id: "ci-to-core",
    d: `M 460 380 C 430 340, 360 320, ${CENTER.x + 24} ${CENTER.y + 12}`,
    label: "receipt",
    delay: 1.4,
    duration: 2.6,
    mobileHide: true,
  },
  {
    id: "core-to-reviewer",
    d: `M ${CENTER.x + 20} ${CENTER.y - 20} L 360 200`,
    label: "whois",
    delay: 2.0,
    duration: 2.2,
  },
];

export function HeroVisual({ className }: { className?: string }) {
  const reducedMotion = useReducedMotion();

  return (
    <div
      className={cn("relative mx-auto aspect-square w-full max-w-[560px]", className)}
      style={{
        // Subtle accent glow that blends with the page background, no card frame.
        background:
          "radial-gradient(circle at 50% 50%, color-mix(in srgb, var(--color-accent) 10%, transparent) 0%, transparent 58%)",
      }}
    >
      <svg
        viewBox={`0 0 ${VIEWBOX} ${VIEWBOX}`}
        role="img"
        aria-labelledby="hero-visual-title hero-visual-desc"
        preserveAspectRatio="xMidYMid meet"
        className="h-full w-full"
      >
        <title id="hero-visual-title">
          AGH runtime map: one daemon with four active sessions and three peers across machines
        </title>
        <desc id="hero-visual-desc">
          The daemon sits at the center. Four session nodes named coder, reviewer, analyst, and
          deployer surround it, each labeled with its driver CLI. Three peer nodes connect to the
          daemon via dashed arcs, representing agh-network over NATS. Small orange packets flow
          along the connections carrying kinds like greet, whois, direct, and receipt.
        </desc>

        <defs>
          <radialGradient id="core-glow" cx="0.5" cy="0.5" r="0.5">
            <stop offset="0%" stopColor="var(--color-accent-strong)" stopOpacity="0.35" />
            <stop offset="60%" stopColor="var(--color-accent)" stopOpacity="0.1" />
            <stop offset="100%" stopColor="var(--color-accent)" stopOpacity="0" />
          </radialGradient>
          {ROUTES.map(route => (
            <path key={`def-${route.id}`} id={`path-${route.id}`} d={route.d} />
          ))}
        </defs>

        {/* Session-to-core connections */}
        {SESSIONS.map(session => (
          <line
            key={`link-${session.id}`}
            x1={session.pos.x}
            y1={session.pos.y}
            x2={CENTER.x}
            y2={CENTER.y}
            stroke="var(--color-line)"
            strokeWidth={1}
            opacity={0.55}
          />
        ))}

        {/* Peer arcs (dashed, curved) */}
        {ROUTES.filter(r => r.id === "core-to-eu" || r.id === "ci-to-core").map(r => (
          <path
            key={`arc-${r.id}`}
            d={r.d}
            fill="none"
            stroke="var(--color-accent-dim)"
            strokeWidth={1}
            strokeDasharray="4 4"
            className={r.mobileHide ? "max-[639px]:hidden" : undefined}
          />
        ))}
        {/* Third peer (desk-02) arc — outbound leg, dashed */}
        <path
          d={`M ${CENTER.x - 24} ${CENTER.y + 12} C 220 340, 150 360, 100 380`}
          fill="none"
          stroke="var(--color-accent-dim)"
          strokeWidth={1}
          strokeDasharray="4 4"
          className="max-[639px]:hidden"
        />

        {/* Core halo — pulsing */}
        <circle cx={CENTER.x} cy={CENTER.y} r={CORE_R + 26} fill="url(#core-glow)">
          {!reducedMotion && (
            <animate
              attributeName="r"
              values={`${CORE_R + 22};${CORE_R + 40};${CORE_R + 22}`}
              dur="4s"
              repeatCount="indefinite"
            />
          )}
          {!reducedMotion && (
            <animate
              attributeName="opacity"
              values="0.9;0.5;0.9"
              dur="4s"
              repeatCount="indefinite"
            />
          )}
        </circle>

        {/* Core disc */}
        <circle cx={CENTER.x} cy={CENTER.y} r={CORE_R} fill="var(--color-accent)" opacity={0.95} />
        <circle
          cx={CENTER.x}
          cy={CENTER.y}
          r={CORE_R - 10}
          fill="var(--color-accent-strong)"
          opacity={0.85}
        />
        <text
          x={CENTER.x}
          y={CENTER.y - 2}
          textAnchor="middle"
          fontFamily="var(--font-mono)"
          fontSize="11"
          fontWeight={600}
          fill="#17110f"
          letterSpacing="0.04em"
        >
          DAEMON
        </text>
        <text
          x={CENTER.x}
          y={CENTER.y + 11}
          textAnchor="middle"
          fontFamily="var(--font-mono)"
          fontSize="8.5"
          fill="#17110f"
          opacity={0.75}
        >
          pid 42871
        </text>
        <text
          x={CENTER.x}
          y={CENTER.y + CORE_R + 22}
          textAnchor="middle"
          fontFamily="var(--font-mono)"
          fontSize="9.5"
          fill="var(--color-text-tertiary)"
          letterSpacing="0.12em"
        >
          sqlite · sse · acp
        </text>

        {/* Sessions */}
        {SESSIONS.map(session => {
          const { x, y } = session.pos;
          const width = 112;
          const height = 44;
          const nameY = y - 3;
          const driverY = y + 11;
          const rectX = x - width / 2;
          const rectY = y - height / 2;
          return (
            <g key={session.id}>
              <rect
                x={rectX}
                y={rectY}
                width={width}
                height={height}
                rx={8}
                fill="var(--color-surface)"
                stroke="var(--color-divider)"
                strokeWidth={1}
              />
              <circle cx={rectX + 12} cy={y} r={3} fill="var(--color-success)" />
              <text
                x={rectX + 22}
                y={nameY}
                fontFamily="var(--font-sans)"
                fontSize="12"
                fontWeight={500}
                fill="var(--color-text-primary)"
              >
                {session.id}
              </text>
              <text
                x={rectX + 22}
                y={driverY}
                fontFamily="var(--font-mono)"
                fontSize="9.5"
                fill="var(--color-text-tertiary)"
                letterSpacing="0.04em"
              >
                {session.driver}
              </text>
            </g>
          );
        })}

        {/* Peers */}
        {PEERS.map(peer => {
          const { x, y } = peer.pos;
          const width = 120;
          const height = 36;
          const rectX = x - width / 2;
          const rectY = y - height / 2;
          return (
            <g key={peer.id} className={peer.mobileHide ? "max-[639px]:hidden" : undefined}>
              <rect
                x={rectX}
                y={rectY}
                width={width}
                height={height}
                rx={8}
                fill="var(--color-canvas)"
                stroke="var(--color-accent-dim)"
                strokeWidth={1}
                strokeDasharray="3 3"
              />
              <circle cx={rectX + 12} cy={y} r={3} fill="var(--color-accent)" opacity={0.7} />
              <text
                x={rectX + 22}
                y={y + 4}
                fontFamily="var(--font-mono)"
                fontSize="10.5"
                fontWeight={500}
                fill="var(--color-text-secondary)"
                letterSpacing="0.03em"
              >
                {peer.id}
              </text>
            </g>
          );
        })}

        {/* Packets — orange dots moving along routes */}
        <g aria-hidden="true">
          {ROUTES.map(route => (
            <g
              key={`packet-${route.id}`}
              className={route.mobileHide ? "max-[639px]:hidden" : undefined}
            >
              <circle r={3.5} fill="var(--color-accent)">
                {!reducedMotion && (
                  <animateMotion
                    dur={`${route.duration}s`}
                    begin={`${route.delay}s`}
                    repeatCount="indefinite"
                    rotate="auto"
                  >
                    <mpath href={`#path-${route.id}`} />
                  </animateMotion>
                )}
              </circle>
              <circle r={5.5} fill="var(--color-accent)" opacity={0.25}>
                {!reducedMotion && (
                  <animateMotion
                    dur={`${route.duration}s`}
                    begin={`${route.delay}s`}
                    repeatCount="indefinite"
                  >
                    <mpath href={`#path-${route.id}`} />
                  </animateMotion>
                )}
              </circle>
            </g>
          ))}
        </g>

        {/* Footer — protocol tag, floating without a box */}
        <text
          x={VIEWBOX / 2}
          y={VIEWBOX - 14}
          textAnchor="middle"
          fontFamily="var(--font-mono)"
          fontSize="10"
          fill="var(--color-text-tertiary)"
          letterSpacing="0.08em"
        >
          agh-network/v0 · nats
        </text>
      </svg>
    </div>
  );
}
