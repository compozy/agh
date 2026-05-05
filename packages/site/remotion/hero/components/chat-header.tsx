import type { CSSProperties } from "react";
import { TOKENS } from "../tokens";
import type { AgentId } from "../data";

interface ChatHeaderProps {
  activeAgent: AgentId;
  sessionName: string;
}

export function ChatHeader({ activeAgent, sessionName }: ChatHeaderProps) {
  const wrap: CSSProperties = {
    display: "flex",
    alignItems: "center",
    justifyContent: "space-between",
    height: 48,
    padding: "0 16px",
    backgroundColor: TOKENS.surface,
    borderBottom: `1px solid ${TOKENS.divider}`,
    flexShrink: 0,
  };
  const left: CSSProperties = {
    display: "flex",
    alignItems: "center",
    gap: 10,
    minWidth: 0,
  };
  const dot: CSSProperties = {
    width: 9,
    height: 9,
    borderRadius: 999,
    backgroundColor: TOKENS.success,
    boxShadow: `0 0 0 3px ${TOKENS.success}1a`,
    flexShrink: 0,
  };
  const agent: CSSProperties = {
    color: TOKENS.textPrimary,
    fontFamily: TOKENS.fontMono,
    fontSize: 12,
    fontWeight: 600,
    letterSpacing: TOKENS.trackingMono,
    textTransform: "uppercase",
    whiteSpace: "nowrap",
  };
  const chevron: CSSProperties = {
    color: TOKENS.textLabel,
    display: "inline-flex",
    alignItems: "center",
    flexShrink: 0,
  };
  const session: CSSProperties = {
    color: TOKENS.textSecondary,
    fontSize: 13,
    fontWeight: 400,
    whiteSpace: "nowrap",
    overflow: "hidden",
    textOverflow: "ellipsis",
    minWidth: 0,
  };
  const right: CSSProperties = {
    display: "flex",
    alignItems: "center",
    gap: 6,
    color: TOKENS.textLabel,
    flexShrink: 0,
  };
  const iconBtn: CSSProperties = {
    width: 26,
    height: 26,
    borderRadius: TOKENS.radiusMd,
    display: "inline-flex",
    alignItems: "center",
    justifyContent: "center",
    color: TOKENS.textLabel,
  };

  return (
    <div style={wrap}>
      <div style={left}>
        <span style={dot} />
        <span style={agent}>{activeAgent}</span>
        <span style={chevron}>
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" aria-hidden="true">
            <path
              d="M9 6l6 6-6 6"
              stroke="currentColor"
              strokeWidth="2"
              strokeLinecap="round"
              strokeLinejoin="round"
            />
          </svg>
        </span>
        <span style={session}>{sessionName}</span>
      </div>
      <div style={right}>
        <span style={iconBtn} aria-label="stop">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
            <rect x="6" y="6" width="12" height="12" rx="2" />
          </svg>
        </span>
      </div>
    </div>
  );
}
