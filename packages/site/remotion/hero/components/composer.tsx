import type { CSSProperties } from "react";
import { useCurrentFrame } from "remotion";
import { TOKENS } from "../tokens";

interface ComposerProps {
  staticMode?: boolean;
}

export function Composer({ staticMode = false }: ComposerProps) {
  const frame = useCurrentFrame();
  const caretOn = !staticMode && frame % 30 < 15;

  const wrap: CSSProperties = {
    padding: "10px 14px 14px 14px",
    backgroundColor: TOKENS.surfacePanel,
    borderTop: `1px solid ${TOKENS.divider}`,
    flexShrink: 0,
  };
  const box: CSSProperties = {
    display: "flex",
    alignItems: "center",
    gap: 10,
    padding: "7px 8px 7px 14px",
    border: `1px solid ${TOKENS.divider}`,
    backgroundColor: TOKENS.surface,
    borderRadius: TOKENS.radiusLg,
  };
  const input: CSSProperties = {
    flex: 1,
    fontSize: 12,
    color: TOKENS.textLabel,
    fontFamily: TOKENS.fontSans,
    display: "flex",
    alignItems: "center",
    gap: 2,
  };
  const caret: CSSProperties = {
    display: "inline-block",
    width: 1,
    height: 12,
    backgroundColor: caretOn ? TOKENS.accent : "transparent",
    marginLeft: 2,
  };
  const send: CSSProperties = {
    width: 30,
    height: 30,
    borderRadius: 999,
    backgroundColor: TOKENS.accent,
    color: "#fff",
    display: "inline-flex",
    alignItems: "center",
    justifyContent: "center",
    flexShrink: 0,
  };

  return (
    <div style={wrap}>
      <div style={box}>
        <div style={input}>
          <span>Send a message…</span>
          <span style={caret} />
        </div>
        <span style={send} aria-label="send">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" aria-hidden="true">
            <path
              d="M4 12h14M14 6l6 6-6 6"
              stroke="currentColor"
              strokeWidth="2"
              strokeLinecap="round"
              strokeLinejoin="round"
            />
          </svg>
        </span>
      </div>
    </div>
  );
}
