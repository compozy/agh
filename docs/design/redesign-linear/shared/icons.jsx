/* Shared SVG primitives consumed by all 3 variations.
 * Stroke 1.75 by default — matches the production @agh/ui density. Each
 * icon takes { size = 14, stroke = 1.75, className } props. */

const Icon = ({ size = 14, stroke = 1.75, className = "", children }) => (
  <svg
    xmlns="http://www.w3.org/2000/svg"
    width={size}
    height={size}
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    strokeWidth={stroke}
    strokeLinecap="round"
    strokeLinejoin="round"
    className={className}
    aria-hidden="true"
    focusable="false"
  >
    {children}
  </svg>
);

const IconListChecks = (props) => (
  <Icon {...props}>
    <path d="m3 17 2 2 4-4" />
    <path d="m3 7 2 2 4-4" />
    <path d="M13 6h8" />
    <path d="M13 12h8" />
    <path d="M13 18h8" />
  </Icon>
);

const IconClock = (props) => (
  <Icon {...props}>
    <circle cx="12" cy="12" r="9" />
    <polyline points="12 7 12 12 15.5 14" />
  </Icon>
);

const IconSearch = (props) => (
  <Icon {...props}>
    <circle cx="11" cy="11" r="7" />
    <path d="m20 20-3.5-3.5" />
  </Icon>
);

const IconPlus = (props) => (
  <Icon {...props}>
    <path d="M12 5v14" />
    <path d="M5 12h14" />
  </Icon>
);

const IconAlert = (props) => (
  <Icon {...props}>
    <circle cx="12" cy="12" r="9" />
    <line x1="12" y1="8" x2="12" y2="13" />
    <line x1="12" y1="16.5" x2="12" y2="16.51" />
  </Icon>
);

const IconNetwork = (props) => (
  <Icon {...props}>
    <rect x="9" y="3" width="6" height="6" rx="1" />
    <rect x="3" y="15" width="6" height="6" rx="1" />
    <rect x="15" y="15" width="6" height="6" rx="1" />
    <path d="M12 9v3" />
    <path d="M6 15v-1.5a1.5 1.5 0 0 1 1.5-1.5h9a1.5 1.5 0 0 1 1.5 1.5V15" />
  </Icon>
);

const IconBoxes = (props) => (
  <Icon {...props}>
    <path d="M2.97 12.92A2 2 0 0 0 2 14.63v3.24a2 2 0 0 0 .97 1.71l3 1.8a2 2 0 0 0 2.06 0L12 19v-5.5l-5-3-4.03 2.42Z" />
    <path d="M12 13.5V19l3.97 2.38a2 2 0 0 0 2.06 0l3-1.8a2 2 0 0 0 .97-1.71v-3.24a2 2 0 0 0-.97-1.71L17 10.5l-5 3Z" />
    <path d="M7.97 4.42A2 2 0 0 0 7 6.13v4.37l5 3 5-3V6.13a2 2 0 0 0-.97-1.71l-3-1.8a2 2 0 0 0-2.06 0l-3 1.8Z" />
  </Icon>
);

const IconActivity = (props) => (
  <Icon {...props}>
    <polyline points="22 12 18 12 15 21 9 3 6 12 2 12" />
  </Icon>
);

const IconPlug = (props) => (
  <Icon {...props}>
    <path d="M12 22v-5" />
    <path d="M9 7V2" />
    <path d="M15 7V2" />
    <path d="M5 12V7h14v5a4 4 0 0 1-4 4H9a4 4 0 0 1-4-4Z" />
  </Icon>
);

const IconSparkles = (props) => (
  <Icon {...props}>
    <path d="M9.94 13.06 12 19l2.06-5.94L20 11l-5.94-2.06L12 3l-2.06 5.94L4 11l5.94 2.06Z" />
    <path d="M19 4 18 6l-2 1 2 1 1 2 1-2 2-1-2-1-1-2Z" />
  </Icon>
);

const IconSettings = (props) => (
  <Icon {...props}>
    <circle cx="12" cy="12" r="3" />
    <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 1 1-4 0v-.09a1.65 1.65 0 0 0-1-1.51 1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 1 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1Z" />
  </Icon>
);

const IconChevronRight = (props) => (
  <Icon {...props}>
    <polyline points="9 18 15 12 9 6" />
  </Icon>
);

const IconCommand = (props) => (
  <Icon {...props}>
    <path d="M18 3a3 3 0 0 0-3 3v12a3 3 0 0 0 3 3 3 3 0 0 0 3-3 3 3 0 0 0-3-3H6a3 3 0 0 0-3 3 3 3 0 0 0 3 3 3 3 0 0 0 3-3V6a3 3 0 0 0-3-3 3 3 0 0 0-3 3 3 3 0 0 0 3 3h12a3 3 0 0 0 3-3 3 3 0 0 0-3-3Z" />
  </Icon>
);

const IconBook = (props) => (
  <Icon {...props}>
    <path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20" />
    <path d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2Z" />
  </Icon>
);

const IconArrowUpRight = (props) => (
  <Icon {...props}>
    <line x1="7" y1="17" x2="17" y2="7" />
    <polyline points="7 7 17 7 17 17" />
  </Icon>
);

const IconCorner = (props) => (
  <Icon {...props}>
    <path d="M5 4 v8 a2 2 0 0 0 2 2 h8" />
    <polyline points="14 11 17 14 14 17" />
  </Icon>
);

const IconRefresh = (props) => (
  <Icon {...props}>
    <polyline points="20 4 20 10 14 10" />
    <path d="M20 10A8 8 0 1 0 12 20" />
  </Icon>
);

const IconPlay = (props) => (
  <Icon {...props}>
    <polygon points="6 4 19 12 6 20 6 4" />
  </Icon>
);

const IconCopy = (props) => (
  <Icon {...props}>
    <rect x="9" y="9" width="11" height="11" rx="2" />
    <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" />
  </Icon>
);

const IconFilter = (props) => (
  <Icon {...props}>
    <polygon points="22 3 2 3 10 12.46 10 19 14 21 14 12.46 22 3" />
  </Icon>
);

const IconWorkspace = (props) => (
  <Icon {...props}>
    <rect x="3" y="3" width="18" height="18" rx="3" />
    <path d="M9 9h6v6H9z" />
  </Icon>
);

/* ─── PriorityIcon — Linear's signature 3-bar (cellular signal) ───────── */

const IconPriority = ({ level = "none", size = 12, className = "" }) => {
  const filled = { high: 3, medium: 2, low: 1 }[level] ?? 0;
  if (level === "urgent") {
    return (
      <svg width={size} height={size} viewBox="0 0 12 12" className={className} aria-hidden="true">
        <path
          d="M6 1.2 1 10.6 11 10.6Z"
          fill="currentColor"
        />
        <line x1="6" y1="5" x2="6" y2="7.5" stroke="var(--color-canvas)" strokeWidth="1.4" strokeLinecap="round" />
        <circle cx="6" cy="9" r="0.7" fill="var(--color-canvas)" />
      </svg>
    );
  }
  if (filled === 0) {
    return (
      <svg width={size} height={size} viewBox="0 0 12 12" className={className} aria-hidden="true">
        <line x1="2" y1="6" x2="10" y2="6" stroke="currentColor" strokeWidth="1.4" strokeLinecap="round" opacity="0.45" />
      </svg>
    );
  }
  return (
    <svg width={size} height={size} viewBox="0 0 12 12" className={className} aria-hidden="true">
      <rect x="1.5" y="7.5" width="2" height="3.2" rx="0.4" fill="currentColor" opacity={filled >= 1 ? 1 : 0.28} />
      <rect x="5" y="5.2" width="2" height="5.5" rx="0.4" fill="currentColor" opacity={filled >= 2 ? 1 : 0.28} />
      <rect x="8.5" y="2.8" width="2" height="7.9" rx="0.4" fill="currentColor" opacity={filled >= 3 ? 1 : 0.28} />
    </svg>
  );
};

/* ─── StatusIcon — ring family (open / dashed / half / full / X / bang) ─ */

const IconStatus = ({ status = "pending", size = 14, className = "" }) => {
  const tone = (window.AGH_DATA?.STATUS_TONE ?? {})[status] ?? "neutral";
  const color = (window.AGH_DATA?.TONE_COLOR ?? {})[tone] ?? "currentColor";
  const sw = 1.5;
  const inkOnFill = "var(--color-canvas-deep)";
  const common = {
    width: size,
    height: size,
    viewBox: "0 0 16 16",
    className,
    "aria-hidden": "true",
    style: { color },
  };

  switch (status) {
    case "draft":
      return (
        <svg {...common}>
          <circle cx="8" cy="8" r="6" fill="none" stroke="currentColor" strokeWidth={sw} strokeDasharray="2 1.6" opacity="0.75" />
        </svg>
      );
    case "pending":
      return (
        <svg {...common}>
          <circle cx="8" cy="8" r="6" fill="none" stroke="currentColor" strokeWidth={sw} opacity="0.55" />
        </svg>
      );
    case "ready":
      return (
        <svg {...common}>
          <circle cx="8" cy="8" r="6" fill="none" stroke="currentColor" strokeWidth={sw} />
        </svg>
      );
    case "in_progress":
      return (
        <svg {...common}>
          <circle cx="8" cy="8" r="6" fill="none" stroke="currentColor" strokeWidth={sw} opacity="0.32" />
          <path d="M8 2 a6 6 0 0 1 0 12" fill="currentColor" />
        </svg>
      );
    case "running":
      return (
        <svg {...common}>
          <circle cx="8" cy="8" r="6" fill="none" stroke="currentColor" strokeWidth={sw} opacity="0.32" />
          <circle cx="8" cy="8" r="6" fill="none" stroke="currentColor" strokeWidth={sw} strokeDasharray="22 16" strokeLinecap="round">
            <animateTransform attributeName="transform" type="rotate" from="0 8 8" to="360 8 8" dur="1.6s" repeatCount="indefinite" />
          </circle>
        </svg>
      );
    case "completed":
      return (
        <svg {...common}>
          <circle cx="8" cy="8" r="6" fill="currentColor" />
          <path d="M5 8.2 7.1 10.3 11 6.4" fill="none" stroke={inkOnFill} strokeWidth={sw} strokeLinecap="round" strokeLinejoin="round" />
        </svg>
      );
    case "failed":
      return (
        <svg {...common}>
          <circle cx="8" cy="8" r="6" fill="currentColor" />
          <line x1="8" y1="5" x2="8" y2="9" stroke={inkOnFill} strokeWidth={sw} strokeLinecap="round" />
          <circle cx="8" cy="11" r="0.85" fill={inkOnFill} />
        </svg>
      );
    case "blocked":
      return (
        <svg {...common}>
          <circle cx="8" cy="8" r="6" fill="currentColor" />
          <rect x="5.4" y="5" width="1.6" height="6" fill={inkOnFill} />
          <rect x="9" y="5" width="1.6" height="6" fill={inkOnFill} />
        </svg>
      );
    case "canceled":
      return (
        <svg {...common}>
          <circle cx="8" cy="8" r="6" fill="none" stroke="currentColor" strokeWidth={sw} opacity="0.55" />
          <line x1="5.6" y1="5.6" x2="10.4" y2="10.4" stroke="currentColor" strokeWidth={sw} strokeLinecap="round" opacity="0.85" />
        </svg>
      );
    default:
      return (
        <svg {...common}>
          <circle cx="8" cy="8" r="6" fill="none" stroke="currentColor" strokeWidth={sw} opacity="0.5" />
        </svg>
      );
  }
};

window.AGH_ICONS = {
  ListChecks: IconListChecks,
  Clock: IconClock,
  Search: IconSearch,
  Plus: IconPlus,
  Alert: IconAlert,
  Network: IconNetwork,
  Boxes: IconBoxes,
  Activity: IconActivity,
  Plug: IconPlug,
  Sparkles: IconSparkles,
  Settings: IconSettings,
  ChevronRight: IconChevronRight,
  Command: IconCommand,
  Book: IconBook,
  ArrowUpRight: IconArrowUpRight,
  Corner: IconCorner,
  Refresh: IconRefresh,
  Play: IconPlay,
  Copy: IconCopy,
  Filter: IconFilter,
  Workspace: IconWorkspace,
  Priority: IconPriority,
  Status: IconStatus,
};
