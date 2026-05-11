import {
  BellIcon,
  BoxesIcon,
  FolderIcon,
  GitBranchIcon,
  HomeIcon,
  InfoIcon,
  Layers3Icon,
  PlayIcon,
  PlusIcon,
  SearchIcon,
  SettingsIcon,
  SparklesIcon,
  SquareTerminalIcon,
  WaypointsIcon,
} from "lucide-react";
import { useState } from "react";
import type { ComponentType } from "react";

import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
  Alert,
  AlertDescription,
  AlertTitle,
  Avatar,
  AvatarFallback,
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
  Button,
  ButtonGroup,
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  ChatMessageBubble,
  CodeBlock,
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
  ConnectionIndicator,
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
  Empty,
  Eyebrow,
  Field,
  FieldDescription,
  FieldLabel,
  Input,
  InputGroup,
  InputGroupAddon,
  InputGroupInput,
  InputGroupText,
  Item,
  ItemContent,
  ItemDescription,
  ItemMedia,
  ItemTitle,
  Kbd,
  KbdGroup,
  Label,
  Metric,
  NativeSelect,
  NativeSelectOption,
  Pill,
  PillGroup,
  Popover,
  PopoverContent,
  PopoverDescription,
  PopoverHeader,
  PopoverTitle,
  PopoverTrigger,
  Progress,
  ProgressLabel,
  ProgressValue,
  ScrollArea,
  SearchInput,
  Section,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  Separator,
  Sheet,
  SheetClose,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
  Sidebar,
  Skeleton,
  Spinner,
  SplitPane,
  Switch,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
  Textarea,
  Toaster,
  ToggleGroup,
  ToggleGroupItem,
  ToolCallCard,
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
  toast,
} from "@agh/ui";
import { KindChip } from "@/systems/network";

const DESIGN_MD_BASE = "https://github.com/compozy/agh/blob/main/DESIGN.md";

type SwatchKind = "color" | "radius" | "duration" | "easing" | "tracking";

interface TokenSwatch {
  token: string;
  value: string;
  role?: string;
  kind: SwatchKind;
}

interface TokenGroup {
  id: string;
  label: string;
  caption: string;
  swatches: TokenSwatch[];
}

const TOKEN_GROUPS: TokenGroup[] = [
  {
    id: "backgrounds",
    label: "Surface ramp",
    caption: "Warm-dark layered backgrounds: rail → canvas → soft → tint → elevated.",
    swatches: [
      { token: "--rail", value: "#0c0b0b", role: "Workspace rail bg", kind: "color" },
      { token: "--canvas", value: "#131211", role: "Page bg", kind: "color" },
      {
        token: "--canvas-soft",
        value: "#1a1918",
        role: "Card / group / sidebar bg",
        kind: "color",
      },
      { token: "--canvas-tint", value: "#1c1b1a", role: "Kanban card baseline", kind: "color" },
      { token: "--sidebar", value: "#1a1918", role: "Sidebar panel", kind: "color" },
      {
        token: "--elevated",
        value: "#232220",
        role: "Active rows, segment-active",
        kind: "color",
      },
      {
        token: "--hover",
        value: "var(--row-hover)",
        role: "Generic hover (alias of --row-hover)",
        kind: "color",
      },
      { token: "--disabled", value: "#4a4847", role: "Disabled fill", kind: "color" },
    ],
  },
  {
    id: "hairlines",
    label: "Hairlines",
    caption: "Translucent rails derived from white. Soft → strong scales focus + dividers.",
    swatches: [
      {
        token: "--line",
        value: "rgba(255, 255, 255, 0.055)",
        role: "Generic 1 px hairline",
        kind: "color",
      },
      {
        token: "--line-soft",
        value: "rgba(255, 255, 255, 0.03)",
        role: "Group bottoms, popover ring",
        kind: "color",
      },
      {
        token: "--line-strong",
        value: "rgba(255, 255, 255, 0.09)",
        role: "Focus ring, scrollbar thumb hover",
        kind: "color",
      },
    ],
  },
  {
    id: "text",
    label: "Text",
    caption: "Five-step neutral text scale with explicit label/eyebrow roles.",
    swatches: [
      { token: "--fg", value: "#ececef", role: "Body", kind: "color" },
      { token: "--fg-strong", value: "#f6f6f8", role: "Titles, active labels", kind: "color" },
      { token: "--muted", value: "#9a9a9f", role: "Secondary copy", kind: "color" },
      { token: "--subtle", value: "#76767c", role: "Placeholders", kind: "color" },
      { token: "--faint", value: "#545458", role: "Mono ids, separators", kind: "color" },
    ],
  },
  {
    id: "accent",
    label: "Accent",
    caption: "Warm orange is the only non-neutral hue. Tints replace solid banners.",
    swatches: [
      { token: "--accent", value: "#e8572a", role: "Action / Primary", kind: "color" },
      { token: "--accent-hover", value: "#d14e25", role: "Accent pressed", kind: "color" },
      {
        token: "--accent-strong",
        value: "#f6874f",
        role: "Highlight accent",
        kind: "color",
      },
      { token: "--accent-ink", value: "#17110f", role: "Text on accent fill", kind: "color" },
      {
        token: "--accent-tint",
        value: "rgba(232, 87, 42, 0.1)",
        role: "Chip / pill tint",
        kind: "color",
      },
      {
        token: "--accent-tint-strong",
        value: "rgba(232, 87, 42, 0.16)",
        role: "Bar fill",
        kind: "color",
      },
      {
        token: "--accent-dim",
        value: "rgba(232, 87, 42, 0.24)",
        role: "Legacy focus ring",
        kind: "color",
      },
      {
        token: "--accent-glow",
        value: "rgba(232, 87, 42, 0.05)",
        role: "Pulse keyframe base",
        kind: "color",
      },
    ],
  },
  {
    id: "signal",
    label: "Signal palette",
    caption:
      "Desaturated signals. Tint backgrounds at 6–10% alpha; full-color text on tint surfaces.",
    swatches: [
      { token: "--success", value: "#5fbf85", role: "Stable / Live", kind: "color" },
      { token: "--warning", value: "#d6a647", role: "Caution / Pending", kind: "color" },
      { token: "--danger", value: "#e0635a", role: "Error / Destructive", kind: "color" },
      { token: "--info", value: "#8e8eb5", role: "Informational", kind: "color" },
      { token: "--neutral", value: "#7a7a80", role: "Idle / Cancelled", kind: "color" },
    ],
  },
  {
    id: "tints",
    label: "Signal tints",
    caption: "Background tints (6–10% alpha) for chips, pills, and kind dots.",
    swatches: [
      {
        token: "--success-tint",
        value: "rgba(95, 191, 133, 0.08)",
        role: "Success chip bg",
        kind: "color",
      },
      {
        token: "--warning-tint",
        value: "rgba(214, 166, 71, 0.08)",
        role: "Warning chip bg",
        kind: "color",
      },
      {
        token: "--danger-tint",
        value: "rgba(224, 99, 90, 0.09)",
        role: "Danger chip bg",
        kind: "color",
      },
      {
        token: "--info-tint",
        value: "rgba(142, 142, 181, 0.12)",
        role: "Info chip bg / Settings observability",
        kind: "color",
      },
      {
        token: "--neutral-tint",
        value: "rgba(150, 150, 155, 0.06)",
        role: "Neutral chip bg (warmed for ramp parity)",
        kind: "color",
      },
    ],
  },
  {
    id: "overlays",
    label: "Overlays",
    caption: "Modal scrim, ghost hover, text selection — all token-driven.",
    swatches: [
      {
        token: "--overlay-scrim",
        value: "rgba(0, 0, 0, 0.55)",
        role: "Modal / dialog backdrop",
        kind: "color",
      },
      {
        token: "--overlay-blur",
        value: "3px",
        role: "Dialog / sheet backdrop blur ONLY",
        kind: "radius",
      },
      {
        token: "--overlay-ghost-hover",
        value: "rgba(255, 255, 255, 0.06)",
        role: "Ghost hover on dark",
        kind: "color",
      },
    ],
  },
  {
    id: "glaze",
    label: "Surface glaze ladder",
    caption:
      "Translucent white tints layered on the warm ramp. Inline rgba literals are forbidden.",
    swatches: [
      {
        token: "--row-hover",
        value: "rgba(255, 255, 255, 0.022)",
        role: "List / nav hover (aliased as --hover)",
        kind: "color",
      },
      {
        token: "--row-selected",
        value: "rgba(255, 255, 255, 0.03)",
        role: "List / nav selected baseline",
        kind: "color",
      },
      {
        token: "--surface-glaze",
        value: "rgba(255, 255, 255, 0.04)",
        role: "RadioCard / panel head selected",
        kind: "color",
      },
      {
        token: "--bar-fill",
        value: "rgba(255, 255, 255, 0.085)",
        role: "Priority / progress / usage bars",
        kind: "color",
      },
      {
        token: "--input-fill",
        value: "rgba(255, 255, 255, 0.025)",
        role: "Composer / textarea / search input",
        kind: "color",
      },
      {
        token: "--btn-default-fill",
        value: "rgba(255, 255, 255, 0.04)",
        role: "Neutral Button default fill",
        kind: "color",
      },
      {
        token: "--btn-default-hover",
        value: "rgba(255, 255, 255, 0.07)",
        role: "Neutral Button hover fill",
        kind: "color",
      },
      {
        token: "--badge-fill",
        value: "rgba(255, 255, 255, 0.05)",
        role: "PillGroup count badge bg",
        kind: "color",
      },
    ],
  },
  {
    id: "avatars",
    label: "Owner avatar palette",
    caption:
      "Tokenised owner palette resolved via web/src/lib/owner-palette.ts colorsFor(). Storybook + design ref tools read from the same source.",
    swatches: [
      {
        token: "--avatar-agent-0-bg",
        value: "rgba(232, 144, 99, 0.18)",
        role: "Agent slot 0 — bg",
        kind: "color",
      },
      { token: "--avatar-agent-0-fg", value: "#f2b895", role: "Agent slot 0 — fg", kind: "color" },
      {
        token: "--avatar-agent-1-bg",
        value: "rgba(168, 178, 220, 0.16)",
        role: "Agent slot 1 — bg",
        kind: "color",
      },
      { token: "--avatar-agent-1-fg", value: "#c5cce7", role: "Agent slot 1 — fg", kind: "color" },
      {
        token: "--avatar-agent-2-bg",
        value: "rgba(143, 196, 178, 0.18)",
        role: "Agent slot 2 — bg",
        kind: "color",
      },
      { token: "--avatar-agent-2-fg", value: "#a9d9c7", role: "Agent slot 2 — fg", kind: "color" },
      {
        token: "--avatar-agent-3-bg",
        value: "rgba(214, 168, 192, 0.18)",
        role: "Agent slot 3 — bg",
        kind: "color",
      },
      { token: "--avatar-agent-3-fg", value: "#e0bcd0", role: "Agent slot 3 — fg", kind: "color" },
      {
        token: "--avatar-human-0-bg",
        value: "rgba(220, 192, 134, 0.2)",
        role: "Human slot 0 — bg",
        kind: "color",
      },
      { token: "--avatar-human-0-fg", value: "#e5cc9a", role: "Human slot 0 — fg", kind: "color" },
      {
        token: "--avatar-human-1-bg",
        value: "rgba(195, 178, 156, 0.2)",
        role: "Human slot 1 — bg",
        kind: "color",
      },
      { token: "--avatar-human-1-fg", value: "#d6c5aa", role: "Human slot 1 — fg", kind: "color" },
      {
        token: "--avatar-human-2-bg",
        value: "rgba(192, 173, 178, 0.2)",
        role: "Human slot 2 — bg",
        kind: "color",
      },
      { token: "--avatar-human-2-fg", value: "#d2bfc5", role: "Human slot 2 — fg", kind: "color" },
    ],
  },
  {
    id: "layout-grammar",
    label: "Layout grammar",
    caption:
      "Modal width ladder + logo well sizes. Inline arbitrary widths are forbidden in modal / catalog surfaces.",
    swatches: [
      {
        token: "--width-modal-sm",
        value: "560px",
        role: "Confirm / single-field editor",
        kind: "radius",
      },
      {
        token: "--width-modal-md",
        value: "720px",
        role: "Task editor / settings field editor",
        kind: "radius",
      },
      {
        token: "--width-modal-lg",
        value: "880px",
        role: "Bridges wizard / knowledge create dialog",
        kind: "radius",
      },
      {
        token: "--size-catalog-logo",
        value: "1.5rem",
        role: "CatalogCard logoSize='default' (24 px)",
        kind: "radius",
      },
      {
        token: "--size-provider-logo-well",
        value: "2.5rem",
        role: "CatalogCard logoSize='lg' (40 px) / settings provider card",
        kind: "radius",
      },
    ],
  },
  {
    id: "protocol-kinds",
    label: "Protocol Kind Colors",
    caption:
      "Kind-dot colors map onto the new palette: say/whois → neutral, greet/trace → info, direct → accent, receipt → success, capability → warning.",
    swatches: [
      { token: "--color-kind-say", value: "var(--neutral)", role: "say", kind: "color" },
      { token: "--color-kind-greet", value: "var(--info)", role: "greet", kind: "color" },
      {
        token: "--color-kind-direct",
        value: "var(--accent)",
        role: "direct",
        kind: "color",
      },
      {
        token: "--color-kind-receipt",
        value: "var(--success)",
        role: "receipt",
        kind: "color",
      },
      {
        token: "--color-kind-capability",
        value: "var(--warning)",
        role: "capability",
        kind: "color",
      },
      { token: "--color-kind-trace", value: "var(--info)", role: "trace", kind: "color" },
      { token: "--color-kind-whois", value: "var(--neutral)", role: "whois", kind: "color" },
    ],
  },
  {
    id: "radii",
    label: "Radii",
    caption: "Ladder: 4 / 5 / 6 / 8 / 10 / 14 / pill.",
    swatches: [
      { token: "--radius-xs", value: "4px", role: "Tightest chip", kind: "radius" },
      { token: "--radius-sm", value: "5px", role: "Kind chip", kind: "radius" },
      { token: "--radius", value: "6px", role: "Default", kind: "radius" },
      { token: "--radius-md", value: "8px", role: "Inputs / buttons", kind: "radius" },
      { token: "--radius-lg", value: "10px", role: "Cards / panels", kind: "radius" },
      { token: "--radius-xl", value: "14px", role: "Sheet / hero card", kind: "radius" },
      { token: "--radius-pill", value: "9999px", role: "Pill / search", kind: "radius" },
    ],
  },
  {
    id: "motion",
    label: "Motion",
    caption: "One fast tier (--dur 140ms) + one slow tier; reduced-motion zeroes everything.",
    swatches: [
      { token: "--dur", value: "140ms", role: "Default", kind: "duration" },
      {
        token: "--dur-slow",
        value: "200ms",
        role: "Panel / modal",
        kind: "duration",
      },
      {
        token: "--ease",
        value: "cubic-bezier(0.2, 0, 0, 1)",
        role: "Default easing",
        kind: "easing",
      },
    ],
  },
  {
    id: "tracking",
    label: "Tracking",
    caption: "Mono tracking used across eyebrows, badges, and protocol strings.",
    swatches: [
      {
        token: "--tracking-mono",
        value: "0.06em",
        role: "Mono eyebrow tracking",
        kind: "tracking",
      },
    ],
  },
];

interface ShowcaseSection {
  id: string;
  label: string;
  anchor: string;
}

const SECTIONS: ShowcaseSection[] = [
  { id: "foundations", label: "Foundations: Tokens", anchor: "#2-color-palette--roles" },
  { id: "typography", label: "Foundations: Typography", anchor: "#3-typography-rules" },
  { id: "buttons", label: "Buttons & Pills", anchor: "#buttons" },
  { id: "inputs", label: "Inputs & Search", anchor: "#inputs" },
  {
    id: "status",
    label: "Status, Metric, MonoBadge, KindChip",
    anchor: "#status-indicators",
  },
  { id: "feedback", label: "Feedback", anchor: "#empty-state" },
  { id: "overlays", label: "Dialog, Sheet, Popover, Tooltip", anchor: "#4-component-stylings" },
  { id: "code-chat", label: "Code & Chat", anchor: "#chat-components" },
  { id: "layout", label: "Sidebar & SplitPane", anchor: "#sidebar-operator-ui" },
];

const FILTERS = [
  { label: "All", value: "all" },
  { label: "Primitives", value: "primitives" },
  { label: "Surfaces", value: "surfaces" },
  { label: "Motion", value: "motion" },
] as const;

type FilterValue = (typeof FILTERS)[number]["value"];

const KINDS = ["greet", "whois", "say", "direct", "capability", "receipt", "trace"] as const;

function SectionLink({ section, children }: { section: ShowcaseSection; children?: string }) {
  return (
    <a
      data-testid={`section-link-${section.id}`}
      data-section-id={section.id}
      data-section-anchor={section.anchor}
      href={`${DESIGN_MD_BASE}${section.anchor}`}
      target="_blank"
      rel="noreferrer"
      className="inline-flex items-center gap-1.5 text-(--muted) transition-colors hover:text-accent"
    >
      <span>{children ?? section.label}</span>
      <span aria-hidden="true" className="font-mono text-badge tracking-(--tracking-mono)">
        {"↗"}
      </span>
    </a>
  );
}

function DesignSystemShowcase() {
  const [filter, setFilter] = useState<FilterValue>("all");
  const [searchValue, setSearchValue] = useState("");

  return (
    <TooltipProvider>
      <main
        data-testid="design-system-showcase"
        className="flex min-h-dvh flex-col bg-(--canvas) text-(--fg)"
      >
        <header
          data-slot="page-header"
          className="flex min-h-11 flex-col gap-2 border-b border-(--line) px-4 py-2.5"
        >
          <div
            data-slot="page-header-main"
            className="flex min-w-0 flex-wrap items-center gap-2 sm:gap-3"
          >
            <div data-slot="page-header-title" className="flex min-w-0 items-center gap-2">
              <span
                aria-hidden="true"
                data-slot="page-header-icon"
                className="inline-flex size-6 shrink-0 items-center justify-center rounded-(--radius-sm) bg-(--elevated) text-(--accent)"
              >
                <SparklesIcon className="size-3.5" />
              </span>
              <h1 className="truncate text-(length:--text-detail-h1) font-medium tracking-(--tracking-detail-h1) text-(--fg-strong)">
                AGH design system
              </h1>
              <span
                data-slot="page-header-count"
                className="inline-flex h-[19px] min-w-[19px] items-center justify-center rounded-(--radius-mono-badge) bg-(--canvas-soft) px-1.5 font-mono text-[10.5px] font-medium tabular-nums text-(--muted)"
              >
                v1
              </span>
            </div>
            <div
              data-slot="page-header-meta"
              className="ml-auto flex shrink-0 items-center gap-2 text-[13px] text-(--muted)"
            >
              <Button
                size="sm"
                variant="outline"
                render={
                  <a
                    data-testid="showcase-open-design-md"
                    href={DESIGN_MD_BASE}
                    target="_blank"
                    rel="noreferrer"
                  />
                }
              >
                Open DESIGN.md
              </Button>
            </div>
          </div>
        </header>

        <div
          role="toolbar"
          aria-label="Showcase filters"
          className="flex min-h-11 flex-wrap items-center gap-3 border-b border-(--line) bg-(--canvas-soft) px-4 py-2"
        >
          <PillGroup
            value={filter}
            onChange={(next: FilterValue) => setFilter(next)}
            items={FILTERS.map(item => ({ label: item.label, value: item.value }))}
          />
          <SearchInput
            value={searchValue}
            onChange={setSearchValue}
            placeholder="Search primitives…"
            containerClassName="ml-auto w-72"
            aria-label="Search primitives"
            kbd={
              <KbdGroup>
                <Kbd>⌘</Kbd>
                <Kbd>K</Kbd>
              </KbdGroup>
            }
          />
        </div>

        <div className="flex flex-col gap-10 px-6 py-8">
          <FoundationsTokenSection />
          <TypographySection />
          <ButtonsAndPillsSection />
          <InputsAndSearchSection />
          <StatusAndMetricSection />
          <FeedbackSection />
          <OverlaysSection />
          <CodeAndChatSection />
          <LayoutSection />
        </div>

        <Toaster />
      </main>
    </TooltipProvider>
  );
}

function FoundationsTokenSection() {
  return (
    <Section
      id="foundations"
      data-testid="section-foundations"
      label={<SectionLink section={SECTIONS[0]}>Foundations: Tokens</SectionLink>}
      right={<Pill mono>tokens.css</Pill>}
    >
      <div className="flex flex-col gap-6 pt-4">
        {TOKEN_GROUPS.map(group => (
          <div
            key={group.id}
            data-testid={`token-group-${group.id}`}
            data-group={group.id}
            className="flex flex-col gap-3"
          >
            <header className="flex items-end justify-between gap-4">
              <div>
                <h3 className="text-item-title font-medium text-(--fg)">{group.label}</h3>
                <p className="mt-0.5 text-small-body text-(--muted)">{group.caption}</p>
              </div>
              <Eyebrow className="text-(--subtle)">{group.swatches.length} tokens</Eyebrow>
            </header>
            <div className="grid grid-cols-2 gap-3 md:grid-cols-3 lg:grid-cols-4">
              {group.swatches.map(swatch => (
                <TokenCard key={swatch.token} swatch={swatch} />
              ))}
            </div>
          </div>
        ))}
      </div>
    </Section>
  );
}

function TokenCard({ swatch }: { swatch: TokenSwatch }) {
  return (
    <article
      data-testid={`token-${swatch.token}`}
      data-token={swatch.token}
      data-kind={swatch.kind}
      className="flex flex-col gap-3 rounded-lg border border-(--line) bg-(--canvas-soft) p-3"
    >
      <TokenPreview swatch={swatch} />
      <div className="flex flex-col gap-0.5">
        <Eyebrow className="text-(--subtle)">{swatch.token}</Eyebrow>
        <span className="font-mono text-eyebrow text-(--muted)">{swatch.value}</span>
        {swatch.role ? <span className="text-xs text-(--muted)">{swatch.role}</span> : null}
      </div>
    </article>
  );
}

function TokenPreview({ swatch }: { swatch: TokenSwatch }) {
  if (swatch.kind === "color") {
    return (
      <div
        aria-hidden="true"
        className="h-14 w-full rounded-md border border-(--line)"
        style={{ backgroundColor: `var(${swatch.token})` }}
      />
    );
  }
  if (swatch.kind === "radius") {
    return (
      <div
        aria-hidden="true"
        className="flex h-14 w-full items-center justify-center bg-(--elevated)"
        style={{ borderRadius: `var(${swatch.token})` }}
      >
        <span className="font-mono text-eyebrow text-(--muted)">{swatch.value}</span>
      </div>
    );
  }
  return (
    <div
      aria-hidden="true"
      className="flex h-14 w-full items-center justify-center rounded-md bg-(--elevated)"
    >
      <Eyebrow className="text-(--muted)">{swatch.value}</Eyebrow>
    </div>
  );
}

function TypographySection() {
  return (
    <Section
      id="typography"
      data-testid="section-typography"
      label={<SectionLink section={SECTIONS[1]}>Foundations: Typography</SectionLink>}
      right={<Pill mono>Inter · JetBrains Mono · NuixyberNext</Pill>}
    >
      <div className="grid gap-3 pt-4 md:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>Page title · Inter 20/700</CardTitle>
          </CardHeader>
          <CardContent className="flex flex-col gap-3">
            <p className="text-xl font-medium leading-7 tracking-tight" style={{ fontWeight: 510 }}>
              Runtime sessions overview
            </p>
            <p className="text-base leading-7 text-(--muted)">
              Body · Inter 16px regular, the default reading text for operator UI. Line-height
              1.5–1.7 keeps dense dashboards breathable without resorting to oversized padding.
            </p>
            <p className="text-small-body leading-small-body text-(--subtle)">
              Small body · Inter 13px, helper text, captions, meta rows.
            </p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle>Mono & wordmark</CardTitle>
          </CardHeader>
          <CardContent className="flex flex-col gap-3">
            <Eyebrow className="text-(--muted)">Eyebrow · JetBrains Mono 11/600 0.06em</Eyebrow>
            <p className="font-mono text-sm leading-7 text-(--fg)">
              agh-network/v0 · run_id_01hq8…
            </p>
            <div className="flex items-center gap-3">
              <span className="font-wordmark text-display-2xl leading-none tracking-tight text-(--fg)">
                agh
              </span>
              <Pill tone="neutral" size="sm" className="border-(--line)">
                Alpha
              </Pill>
            </div>
          </CardContent>
        </Card>
      </div>
    </Section>
  );
}

function ButtonsAndPillsSection() {
  const [pillFilter, setPillFilter] = useState("active");
  return (
    <Section
      id="buttons"
      data-testid="section-buttons"
      label={<SectionLink section={SECTIONS[2]}>Buttons & Pills</SectionLink>}
      right={
        <Pill mono tone="accent">
          action
        </Pill>
      }
    >
      <div className="flex flex-col gap-6 pt-4">
        <div className="flex flex-wrap items-center gap-3">
          <Button>Primary</Button>
          <Button variant="outline">Outline</Button>
          <Button variant="ghost">Ghost</Button>
          <Button variant="secondary">Secondary</Button>
          <Button variant="destructive">Destructive</Button>
          <Button variant="link">Link</Button>
        </div>
        <div className="flex flex-wrap items-center gap-3">
          <Button size="xs">XS</Button>
          <Button size="sm">Small</Button>
          <Button>Default</Button>
          <Button size="lg">Large</Button>
          <Button size="icon" aria-label="Icon action">
            <PlusIcon />
          </Button>
        </div>
        <div className="flex flex-wrap items-center gap-4">
          <ButtonGroup>
            <Button variant="outline">Day</Button>
            <Button variant="outline">Week</Button>
            <Button variant="outline">Month</Button>
          </ButtonGroup>
          <KbdGroup>
            <Kbd>⌘</Kbd>
            <Kbd>K</Kbd>
          </KbdGroup>
        </div>
        <div className="flex flex-wrap items-center gap-3">
          <Pill tone="neutral">Neutral</Pill>
          <Pill tone="accent">Action</Pill>
          <Pill tone="success">Stable</Pill>
          <Pill tone="warning">Pending</Pill>
          <Pill tone="danger">Error</Pill>
          <Pill tone="info">Info</Pill>
        </div>
        <PillGroup
          value={pillFilter}
          onChange={setPillFilter}
          items={[
            { label: "Active", value: "active", badge: 3 },
            { label: "Queued", value: "queued" },
            { label: "Done", value: "done", badge: 12 },
            { label: "Archived", value: "archived", disabled: true },
          ]}
        />
      </div>
    </Section>
  );
}

function InputsAndSearchSection() {
  const [textareaValue, setTextareaValue] = useState(
    "Multiline textarea — Inter 14px, 1.6 leading."
  );

  return (
    <Section
      id="inputs"
      data-testid="section-inputs"
      label={<SectionLink section={SECTIONS[3]}>Inputs & Search</SectionLink>}
      right={<Pill mono>form primitives</Pill>}
    >
      <div className="grid gap-6 pt-4 md:grid-cols-2">
        <Field>
          <FieldLabel htmlFor="showcase-name">Display name</FieldLabel>
          <Input id="showcase-name" placeholder="e.g. agh-core" />
          <FieldDescription>Used across session headers and agent metadata.</FieldDescription>
        </Field>
        <Field>
          <FieldLabel htmlFor="showcase-notes">Notes</FieldLabel>
          <Textarea
            id="showcase-notes"
            rows={3}
            value={textareaValue}
            onChange={event => setTextareaValue(event.target.value)}
          />
        </Field>
        <Field>
          <FieldLabel htmlFor="showcase-agent">Agent</FieldLabel>
          <Select defaultValue="claude">
            <SelectTrigger id="showcase-agent">
              <SelectValue placeholder="Select an agent" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="claude">Claude Code</SelectItem>
              <SelectItem value="codex">Codex CLI</SelectItem>
              <SelectItem value="gemini">Gemini CLI</SelectItem>
            </SelectContent>
          </Select>
        </Field>
        <Field>
          <FieldLabel htmlFor="showcase-env">Environment</FieldLabel>
          <NativeSelect id="showcase-env" defaultValue="dev">
            <NativeSelectOption value="dev">Development</NativeSelectOption>
            <NativeSelectOption value="stage">Staging</NativeSelectOption>
            <NativeSelectOption value="prod">Production</NativeSelectOption>
          </NativeSelect>
        </Field>
        <InputGroup>
          <InputGroupAddon align="inline-start">
            <SearchIcon className="size-3.5" />
          </InputGroupAddon>
          <InputGroupInput placeholder="Filter sessions…" />
          <InputGroupAddon align="inline-end">
            <InputGroupText>
              <Kbd>⌘</Kbd>
              <Kbd>F</Kbd>
            </InputGroupText>
          </InputGroupAddon>
        </InputGroup>
        <SearchInput
          placeholder="Search skills…"
          kbd={
            <KbdGroup>
              <Kbd>⌘</Kbd>
              <Kbd>K</Kbd>
            </KbdGroup>
          }
        />
        <div className="flex flex-col gap-3">
          <Label>Toggle group</Label>
          <ToggleGroup defaultValue={["tasks"]}>
            <ToggleGroupItem value="tasks" aria-label="Tasks">
              Tasks
            </ToggleGroupItem>
            <ToggleGroupItem value="sessions" aria-label="Sessions">
              Sessions
            </ToggleGroupItem>
            <ToggleGroupItem value="skills" aria-label="Skills">
              Skills
            </ToggleGroupItem>
          </ToggleGroup>
        </div>
        <div className="flex items-center gap-3">
          <Switch defaultChecked id="showcase-switch" />
          <Label htmlFor="showcase-switch">Autostart agents on boot</Label>
        </div>
      </div>
    </Section>
  );
}

function StatusAndMetricSection() {
  return (
    <Section
      id="status"
      data-testid="section-status"
      label={<SectionLink section={SECTIONS[4]}>Status, Metric, MonoBadge, KindChip</SectionLink>}
      right={
        <Pill mono tone="info">
          signal
        </Pill>
      }
    >
      <div className="grid gap-3 pt-4 md:grid-cols-3">
        <Metric label="Active sessions" value="12" detail="+3" tone="accent" />
        <Metric
          label="Throughput"
          value="248"
          subtext="Messages / minute across all workspaces."
          tone="success"
        />
        <Metric label="Error rate" value="0.4%" subtext="Last 24h · within SLO" tone="warning" />
      </div>
      <div className="flex flex-wrap items-center gap-4 pt-4">
        <div className="inline-flex items-center gap-2">
          <Pill.Dot tone="success" />
          <span className="text-sm text-(--muted)">Connected</span>
        </div>
        <div className="inline-flex items-center gap-2">
          <Pill.Dot tone="warning" pulse />
          <span className="text-sm text-(--muted)">Connecting</span>
        </div>
        <div className="inline-flex items-center gap-2">
          <Pill.Dot tone="danger" />
          <span className="text-sm text-(--muted)">Disconnected</span>
        </div>
        <ConnectionIndicator status="connected" />
        <ConnectionIndicator status="connecting" />
        <ConnectionIndicator status="disconnected" />
      </div>
      <div className="flex flex-wrap items-center gap-2 pt-4">
        <Pill mono tone="neutral">
          id_01HQ…
        </Pill>
        <Pill mono tone="neutral">
          idle
        </Pill>
        <Pill mono tone="accent">
          RUNNING
        </Pill>
        <Pill mono tone="success">
          DONE
        </Pill>
        <Pill mono tone="warning">
          PARTIAL
        </Pill>
        <Pill mono tone="danger">
          ERROR
        </Pill>
        <Pill mono tone="info">
          INFO
        </Pill>
      </div>
      <div className="flex flex-wrap items-center gap-2 pt-3">
        {KINDS.map(kind => (
          <KindChip key={kind} kind={kind} />
        ))}
      </div>
      <div className="flex flex-col gap-3 pt-6">
        <div className="flex flex-wrap items-center gap-4">
          <Spinner className="size-4 text-accent" />
          <Skeleton className="h-4 w-40" />
          <Separator orientation="vertical" className="h-6" />
          <Eyebrow className="text-(--subtle)">spinners · skeletons · separators</Eyebrow>
        </div>
        <Progress value={64} data-testid="showcase-progress">
          <ProgressLabel>Skill index rebuild</ProgressLabel>
          <ProgressValue />
        </Progress>
      </div>
    </Section>
  );
}

function FeedbackSection() {
  return (
    <Section
      id="feedback"
      data-testid="section-feedback"
      label={<SectionLink section={SECTIONS[5]}>Feedback (Alert, Empty, Toaster)</SectionLink>}
      right={
        <Pill mono tone="warning">
          state
        </Pill>
      }
    >
      <div className="grid gap-3 pt-4 md:grid-cols-2">
        <Alert>
          <InfoIcon />
          <AlertTitle>Scheduled maintenance</AlertTitle>
          <AlertDescription>
            Runtime will reload at 04:00 UTC to pick up skill index updates.
          </AlertDescription>
        </Alert>
        <Alert variant="danger">
          <InfoIcon />
          <AlertTitle>Connection lost</AlertTitle>
          <AlertDescription>
            Reconnecting to the daemon. Check the footer indicator for status.
          </AlertDescription>
        </Alert>
      </div>
      <div className="pt-4">
        <Empty
          icon={Layers3Icon}
          title="Ready for your first session"
          description="Pick an agent above or hit ⌘K to spawn a new run. Every session is replayable by default."
          action={
            <div className="flex items-center gap-2">
              <Button
                size="sm"
                onClick={() =>
                  toast("Toast fired", { description: "Sonner works out of the box." })
                }
              >
                Fire toast
              </Button>
              <Button size="sm" variant="outline">
                Read the docs
              </Button>
            </div>
          }
        />
      </div>
    </Section>
  );
}

function OverlaysSection() {
  return (
    <Section
      id="overlays"
      data-testid="section-overlays"
      label={<SectionLink section={SECTIONS[6]}>Dialog · Sheet · Popover · Tooltip</SectionLink>}
      right={
        <Pill mono tone="accent">
          motion
        </Pill>
      }
    >
      <div className="flex flex-wrap items-center gap-3 pt-4">
        <Dialog>
          <DialogTrigger
            render={<Button variant="outline" data-testid="showcase-dialog-trigger" />}
          >
            Open dialog
          </DialogTrigger>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>Delete session?</DialogTitle>
              <DialogDescription>
                Deleting removes the replay events. This cannot be undone.
              </DialogDescription>
            </DialogHeader>
            <DialogFooter>
              <DialogClose render={<Button variant="outline" />}>Cancel</DialogClose>
              <Button variant="destructive">Delete</Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>

        <Sheet>
          <SheetTrigger render={<Button variant="outline" data-testid="showcase-sheet-trigger" />}>
            Open sheet
          </SheetTrigger>
          <SheetContent>
            <SheetHeader>
              <SheetTitle>Session settings</SheetTitle>
              <SheetDescription>
                Configure per-session defaults without leaving the page.
              </SheetDescription>
            </SheetHeader>
            <SheetFooter>
              <SheetClose render={<Button variant="outline" />}>Close</SheetClose>
              <Button>Save changes</Button>
            </SheetFooter>
          </SheetContent>
        </Sheet>

        <Popover>
          <PopoverTrigger
            render={<Button variant="outline" data-testid="showcase-popover-trigger" />}
          >
            Open popover
          </PopoverTrigger>
          <PopoverContent>
            <PopoverHeader>
              <PopoverTitle>Quick tip</PopoverTitle>
              <PopoverDescription>
                Press <Kbd>⌘</Kbd> + <Kbd>K</Kbd> to open the command palette.
              </PopoverDescription>
            </PopoverHeader>
          </PopoverContent>
        </Popover>

        <Tooltip>
          <TooltipTrigger
            render={
              <Button variant="outline" data-testid="showcase-tooltip-trigger">
                Hover me
              </Button>
            }
          />
          <TooltipContent>Tooltip body copy</TooltipContent>
        </Tooltip>

        <DropdownMenu>
          <DropdownMenuTrigger
            render={<Button variant="outline" data-testid="showcase-menu-trigger" />}
          >
            Dropdown
          </DropdownMenuTrigger>
          <DropdownMenuContent>
            <DropdownMenuItem>
              <PlayIcon className="size-3.5" /> Start run
            </DropdownMenuItem>
            <DropdownMenuItem>
              <GitBranchIcon className="size-3.5" /> Fork session
            </DropdownMenuItem>
            <DropdownMenuItem>
              <BellIcon className="size-3.5" /> Notify on completion
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem variant="destructive">Delete</DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
      <div className="flex flex-col gap-4 pt-6">
        <Tabs defaultValue="overview">
          <TabsList>
            <TabsTrigger value="overview">Overview</TabsTrigger>
            <TabsTrigger value="events">Events</TabsTrigger>
            <TabsTrigger value="artifacts">Artifacts</TabsTrigger>
          </TabsList>
          <TabsContent value="overview">
            <p className="text-sm text-(--muted)">
              Tabs host section switches: Base UI driven, motion-free.
            </p>
          </TabsContent>
          <TabsContent value="events">
            <p className="text-sm text-(--muted)">
              Replayable event timeline lives under this tab in production.
            </p>
          </TabsContent>
          <TabsContent value="artifacts">
            <p className="text-sm text-(--muted)">Generated files with their provenance chain.</p>
          </TabsContent>
        </Tabs>
        <Accordion defaultValue={["item-1"]}>
          <AccordionItem value="item-1">
            <AccordionTrigger>Can I rewind a session?</AccordionTrigger>
            <AccordionContent>
              Yes, every session is fully replayable from its event store.
            </AccordionContent>
          </AccordionItem>
          <AccordionItem value="item-2">
            <AccordionTrigger>Does it work offline?</AccordionTrigger>
            <AccordionContent>
              The runtime is local-first. The network protocol only activates for peer flows.
            </AccordionContent>
          </AccordionItem>
        </Accordion>
        <Collapsible>
          <CollapsibleTrigger
            render={<Button variant="ghost" data-testid="showcase-collapsible-trigger" />}
          >
            Toggle diagnostics
          </CollapsibleTrigger>
          <CollapsibleContent>
            <p className="text-sm text-(--muted)">
              Collapsed content reveals with a CSS animation.
            </p>
          </CollapsibleContent>
        </Collapsible>
      </div>
    </Section>
  );
}

function CodeAndChatSection() {
  const sampleCode = `agh start --workspace agh-core
agh session list --active`;

  return (
    <Section
      id="code-chat"
      data-testid="section-code-chat"
      label={<SectionLink section={SECTIONS[7]}>Code & Chat</SectionLink>}
      right={
        <Pill mono tone="accent">
          session shells
        </Pill>
      }
    >
      <div className="grid gap-4 pt-4 md:grid-cols-2">
        <CodeBlock code={sampleCode} language="shell" />
        <div className="flex flex-col gap-3">
          <ChatMessageBubble role="user" meta={<span>YOU · 10:42</span>}>
            Spin up a new run against the research workspace.
          </ChatMessageBubble>
          <ChatMessageBubble
            role="agent"
            meta={
              <>
                <Pill.Dot tone="success" size="sm" />
                <span>CLAUDE · 10:42</span>
              </>
            }
          >
            Starting run_01HQ8… against agh-core. Streaming events to the inspector.
          </ChatMessageBubble>
          <ChatMessageBubble role="tool">
            <ToolCallCard
              toolName="read_file"
              filePath="internal/daemon/daemon.go"
              status="running"
            />
          </ChatMessageBubble>
          <ChatMessageBubble role="system">Session idle · 2m</ChatMessageBubble>
        </div>
      </div>
    </Section>
  );
}

function LayoutSection() {
  return (
    <Section
      id="layout"
      data-testid="section-layout"
      label={<SectionLink section={SECTIONS[8]}>Sidebar & SplitPane</SectionLink>}
      right={<Pill mono>layout</Pill>}
    >
      <div className="grid gap-4 pt-4 lg:grid-cols-2">
        <div className="h-[340px] overflow-hidden rounded-lg border border-(--line)">
          <Sidebar
            defaultCollapsed={false}
            rail={
              <>
                <button
                  type="button"
                  aria-label="Workspace agh-core"
                  className="inline-flex size-7 items-center justify-center rounded-full border border-(--accent) bg-(--elevated) font-mono text-eyebrow text-(--accent)"
                >
                  A
                </button>
                <button
                  type="button"
                  aria-label="Workspace research"
                  className="inline-flex size-7 items-center justify-center rounded-full border border-(--line) bg-(--canvas-soft) font-mono text-eyebrow text-(--muted)"
                >
                  R
                </button>
              </>
            }
            header={
              <>
                <FolderIcon className="size-3.5 text-(--subtle)" />
                <span className="text-small-body font-medium">agh-core</span>
              </>
            }
            nav={
              <div className="flex flex-col gap-0.5 px-2 py-3 text-sm">
                <SidebarRow icon={HomeIcon} label="Home" active />
                <SidebarRow icon={SquareTerminalIcon} label="Sessions" />
                <SidebarRow icon={BoxesIcon} label="Tasks" />
                <SidebarRow icon={WaypointsIcon} label="Network" />
                <SidebarRow icon={BellIcon} label="Automation" />
              </div>
            }
            footer={
              <div className="flex items-center justify-between gap-2 text-xs text-(--subtle)">
                <ConnectionIndicator status="connected" />
                <Button variant="ghost" size="icon-sm" aria-label="Settings">
                  <SettingsIcon className="size-3.5" />
                </Button>
              </div>
            }
          />
        </div>
        <div className="h-[340px] overflow-hidden rounded-lg border border-(--line)">
          <SplitPane
            list={
              <ScrollArea className="h-full">
                <ul className="flex flex-col divide-y divide-(--line)">
                  {[
                    "Skill: repo-refactor",
                    "Skill: ship-review",
                    "Skill: rebase-helper",
                    "Skill: test-writer",
                  ].map((entry, index) => (
                    <li key={entry}>
                      <Item data-active={index === 0 ? "true" : undefined}>
                        <ItemMedia>
                          <Avatar className="size-7">
                            <AvatarFallback>{entry.slice(7, 8).toUpperCase()}</AvatarFallback>
                          </Avatar>
                        </ItemMedia>
                        <ItemContent>
                          <ItemTitle>{entry}</ItemTitle>
                          <ItemDescription>Updated 2m ago</ItemDescription>
                        </ItemContent>
                      </Item>
                    </li>
                  ))}
                </ul>
              </ScrollArea>
            }
            detail={
              <div className="flex flex-1 flex-col gap-4 p-4">
                <Breadcrumb>
                  <BreadcrumbList>
                    <BreadcrumbItem>
                      <BreadcrumbLink href="#">Skills</BreadcrumbLink>
                    </BreadcrumbItem>
                    <BreadcrumbSeparator />
                    <BreadcrumbItem>
                      <BreadcrumbPage>repo-refactor</BreadcrumbPage>
                    </BreadcrumbItem>
                  </BreadcrumbList>
                </Breadcrumb>
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Key</TableHead>
                      <TableHead>Value</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    <TableRow>
                      <TableCell>Version</TableCell>
                      <TableCell>
                        <Pill mono tone="accent">
                          v0.4.2
                        </Pill>
                      </TableCell>
                    </TableRow>
                    <TableRow>
                      <TableCell>Last run</TableCell>
                      <TableCell>2h ago</TableCell>
                    </TableRow>
                  </TableBody>
                </Table>
              </div>
            }
          />
        </div>
      </div>
    </Section>
  );
}

function SidebarRow({
  icon: Icon,
  label,
  active,
}: {
  icon: ComponentType<{ className?: string }>;
  label: string;
  active?: boolean;
}) {
  return (
    <button
      type="button"
      data-active={active ? "true" : undefined}
      className="group flex items-center gap-2 rounded-md px-2 py-1.5 text-left text-small-body text-(--muted) transition-colors hover:bg-(--hover) hover:text-(--fg) data-[active=true]:bg-(--elevated) data-[active=true]:text-(--fg)"
    >
      <Icon className="size-3.5 text-(--subtle) group-data-[active=true]:text-accent" />
      <span>{label}</span>
    </button>
  );
}

export { DesignSystemShowcase, SECTIONS, TOKEN_GROUPS };
export type { TokenSwatch, TokenGroup, ShowcaseSection };
