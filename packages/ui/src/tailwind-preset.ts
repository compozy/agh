/**
 * Tailwind CSS v4 preset encoding DESIGN.md scales and token references.
 *
 * In Tailwind v4 presets are plain config objects consumed via `@config` or
 * programmatic config. The actual CSS custom properties live in tokens.css —
 * this preset encodes the spacing scale, border radius scale, and font
 * families so consumers get consistent utility classes.
 */

export const aghPreset = {
  theme: {
    fontFamily: {
      sans: ['"Inter Variable"', "-apple-system", '"BlinkMacSystemFont"', "sans-serif"],
      mono: ['"JetBrains Mono"', '"Courier New"', "monospace"],
    },
    spacing: {
      0: "0px",
      px: "1px",
      0.5: "2px",
      1: "4px",
      1.5: "6px",
      2: "8px",
      2.5: "10px",
      3: "12px",
      3.5: "14px",
      4: "16px",
      5: "20px",
      6: "24px",
      7: "28px",
      8: "32px",
      9: "36px",
      10: "40px",
      11: "44px",
      12: "48px",
      14: "56px",
      16: "64px",
      20: "80px",
      24: "96px",
      28: "112px",
      32: "128px",
      36: "144px",
      40: "160px",
      44: "176px",
      48: "192px",
      52: "208px",
      56: "224px",
      60: "240px",
      64: "256px",
      72: "288px",
      80: "320px",
      96: "384px",
    },
    borderRadius: {
      none: "0px",
      sm: "calc(var(--radius) - 2px)",
      DEFAULT: "var(--radius)",
      md: "var(--radius)",
      lg: "calc(var(--radius) + 4px)",
      xl: "calc(var(--radius) + 12px)",
      "2xl": "1rem",
      "3xl": "1.5rem",
      pill: "20px",
      full: "9999px",
    },
  },
} as const;
