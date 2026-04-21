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
  Badge,
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
  KindChip,
  Label,
  Metric,
  MonoBadge,
  NativeSelect,
  NativeSelectOption,
  PageHeader,
  Pill,
  Pills,
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
  SplitPane,
  Spinner,
  StatusDot,
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
  toast,
  ToggleGroup,
  ToggleGroupItem,
  Toolbar,
  ToolCallCard,
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@agh/ui";

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
    label: "Backgrounds",
    caption: "Flat depth steps — canvas → surface → elevated, never shadows.",
    swatches: [
      { token: "--color-canvas", value: "#141312", role: "App background", kind: "color" },
      {
        token: "--color-canvas-deep",
        value: "#0E0E0F",
        role: "Code blocks, deep panels",
        kind: "color",
      },
      {
        token: "--color-surface",
        value: "#1E1C1B",
        role: "Cards, sidebar, modals",
        kind: "color",
      },
      {
        token: "--color-surface-panel",
        value: "#181716",
        role: "Alt panel fill",
        kind: "color",
      },
      {
        token: "--color-surface-elevated",
        value: "#2E2C2B",
        role: "Popovers, icon wells, inputs",
        kind: "color",
      },
      {
        token: "--color-divider",
        value: "#3C3A39",
        role: "1px hairline dividers",
        kind: "color",
      },
      { token: "--color-line", value: "#3C3A39", role: "Diagram lines", kind: "color" },
      { token: "--color-hover", value: "#353332", role: "Neutral hover fill", kind: "color" },
      {
        token: "--color-disabled",
        value: "#4A4847",
        role: "Disabled background",
        kind: "color",
      },
    ],
  },
  {
    id: "text",
    label: "Text",
    caption: "Apple-derived neutral scale with explicit label/eyebrow role.",
    swatches: [
      { token: "--color-text-primary", value: "#E5E5E7", role: "Titles", kind: "color" },
      { token: "--color-text-secondary", value: "#8E8E93", role: "Body", kind: "color" },
      {
        token: "--color-text-tertiary",
        value: "#636366",
        role: "Placeholders",
        kind: "color",
      },
      { token: "--color-text-label", value: "#98989D", role: "Eyebrows", kind: "color" },
    ],
  },
  {
    id: "accent",
    label: "Accent & Semantic",
    caption: "Warm orange is the only non-neutral hue. Semantic = signal, never decoration.",
    swatches: [
      {
        token: "--color-accent",
        value: "#E8572A",
        role: "Action / Primary",
        kind: "color",
      },
      { token: "--color-accent-ink", value: "#17110F", role: "Text on accent", kind: "color" },
      {
        token: "--color-accent-hover",
        value: "#D14E25",
        role: "Accent pressed",
        kind: "color",
      },
      {
        token: "--color-accent-strong",
        value: "#F6874F",
        role: "Highlight accent",
        kind: "color",
      },
      { token: "--color-accent-dim", value: "#E8572A59", role: "~35% alpha", kind: "color" },
      {
        token: "--color-accent-tint-strong",
        value: "#E8572A3D",
        role: "~24% alpha hover",
        kind: "color",
      },
      {
        token: "--color-success",
        value: "#30D158",
        role: "Stable / Live",
        kind: "color",
      },
      { token: "--color-danger", value: "#FF453A", role: "Destructive", kind: "color" },
      { token: "--color-warning", value: "#FFD60A", role: "Pending", kind: "color" },
      { token: "--color-info", value: "#BF5AF2", role: "Informational", kind: "color" },
    ],
  },
  {
    id: "tints",
    label: "15% Tints",
    caption: "Badges and kind chips use 15% opacity of the semantic color.",
    swatches: [
      { token: "--color-accent-tint", value: "#E8572A26", role: "Accent chip", kind: "color" },
      {
        token: "--color-success-tint",
        value: "#30D15826",
        role: "Success chip",
        kind: "color",
      },
      {
        token: "--color-danger-tint",
        value: "#FF453A26",
        role: "Danger chip",
        kind: "color",
      },
      {
        token: "--color-warning-tint",
        value: "#FFD60A26",
        role: "Warning chip",
        kind: "color",
      },
      { token: "--color-info-tint", value: "#BF5AF226", role: "Info chip", kind: "color" },
      {
        token: "--color-neutral-tint",
        value: "#63636626",
        role: "Neutral chip",
        kind: "color",
      },
    ],
  },
  {
    id: "radii",
    label: "Radii",
    caption: "Small chips at 5/6px, inputs at 8px, cards/code at 12px.",
    swatches: [
      { token: "--radius-diagram", value: "12px", role: "Cards + code", kind: "radius" },
      { token: "--radius-chip", value: "5px", role: "Kind chips", kind: "radius" },
      {
        token: "--radius-mono-badge",
        value: "6px",
        role: "Status / mono badges",
        kind: "radius",
      },
    ],
  },
  {
    id: "motion",
    label: "Motion",
    caption: "Three durations, two easings. Reduced motion zeroes them globally.",
    swatches: [
      {
        token: "--duration-fast",
        value: "100ms",
        role: "Tooltip / fast hover",
        kind: "duration",
      },
      { token: "--duration-base", value: "150ms", role: "Default", kind: "duration" },
      {
        token: "--duration-slow",
        value: "200ms",
        role: "Panel / modal / sidebar",
        kind: "duration",
      },
      {
        token: "--ease-out",
        value: "cubic-bezier(0.2, 0, 0, 1)",
        role: "Default easing",
        kind: "easing",
      },
      {
        token: "--ease-in-out",
        value: "cubic-bezier(0.4, 0, 0.2, 1)",
        role: "Symmetric",
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
  { id: "foundations", label: "Foundations — Tokens", anchor: "#2-color-palette--roles" },
  { id: "typography", label: "Foundations — Typography", anchor: "#3-typography-rules" },
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
      className="inline-flex items-center gap-1.5 text-[color:var(--color-text-label)] transition-colors hover:text-[color:var(--color-accent)]"
    >
      <span>{children ?? section.label}</span>
      <span aria-hidden="true" className="font-mono text-[10px] tracking-[0.08em]">
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
        className="flex min-h-dvh flex-col bg-[color:var(--color-canvas)] text-[color:var(--color-text-primary)]"
      >
        <PageHeader
          title="AGH design system"
          icon={SparklesIcon}
          count="v1"
          meta={
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
          }
        />

        <Toolbar aria-label="Showcase filters" className="gap-3">
          <Pills
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
        </Toolbar>

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
      label={<SectionLink section={SECTIONS[0]}>Foundations — Tokens</SectionLink>}
      right={<MonoBadge>tokens.css</MonoBadge>}
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
                <h3 className="text-[15px] font-semibold text-[color:var(--color-text-primary)]">
                  {group.label}
                </h3>
                <p className="mt-0.5 text-[13px] text-[color:var(--color-text-secondary)]">
                  {group.caption}
                </p>
              </div>
              <span className="font-mono text-[10px] uppercase tracking-[0.08em] text-[color:var(--color-text-tertiary)]">
                {group.swatches.length} tokens
              </span>
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
      className="flex flex-col gap-3 rounded-[var(--radius-diagram)] border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] p-3"
    >
      <TokenPreview swatch={swatch} />
      <div className="flex flex-col gap-0.5">
        <span className="font-mono text-[11px] font-semibold uppercase tracking-[0.06em] text-[color:var(--color-text-tertiary)]">
          {swatch.token}
        </span>
        <span className="font-mono text-[11px] text-[color:var(--color-text-label)]">
          {swatch.value}
        </span>
        {swatch.role ? (
          <span className="text-[12px] text-[color:var(--color-text-secondary)]">
            {swatch.role}
          </span>
        ) : null}
      </div>
    </article>
  );
}

function TokenPreview({ swatch }: { swatch: TokenSwatch }) {
  if (swatch.kind === "color") {
    return (
      <div
        aria-hidden="true"
        className="h-14 w-full rounded-md border border-[color:var(--color-divider)]"
        style={{ backgroundColor: `var(${swatch.token})` }}
      />
    );
  }
  if (swatch.kind === "radius") {
    return (
      <div
        aria-hidden="true"
        className="flex h-14 w-full items-center justify-center bg-[color:var(--color-surface-elevated)]"
        style={{ borderRadius: `var(${swatch.token})` }}
      >
        <span className="font-mono text-[11px] text-[color:var(--color-text-label)]">
          {swatch.value}
        </span>
      </div>
    );
  }
  return (
    <div
      aria-hidden="true"
      className="flex h-14 w-full items-center justify-center rounded-md bg-[color:var(--color-surface-elevated)]"
    >
      <span className="font-mono text-[11px] uppercase tracking-[0.08em] text-[color:var(--color-text-label)]">
        {swatch.value}
      </span>
    </div>
  );
}

function TypographySection() {
  return (
    <Section
      id="typography"
      data-testid="section-typography"
      label={<SectionLink section={SECTIONS[1]}>Foundations — Typography</SectionLink>}
      right={<MonoBadge>Inter · JetBrains Mono · NuixyberNext</MonoBadge>}
    >
      <div className="grid gap-3 pt-4 md:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>Page title · Inter 20/700</CardTitle>
          </CardHeader>
          <CardContent className="flex flex-col gap-3">
            <p className="text-[20px] font-bold leading-[28px] tracking-[-0.01em]">
              Runtime sessions overview
            </p>
            <p className="text-[16px] leading-[1.6] text-[color:var(--color-text-secondary)]">
              Body · Inter 16px regular — the default reading text for operator UI. Line-height
              1.5–1.7 keeps dense dashboards breathable without resorting to oversized padding.
            </p>
            <p className="text-[13px] leading-[18px] text-[color:var(--color-text-tertiary)]">
              Small body · Inter 13px — helper text, captions, meta rows.
            </p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle>Mono & wordmark</CardTitle>
          </CardHeader>
          <CardContent className="flex flex-col gap-3">
            <p className="font-mono text-[11px] font-semibold uppercase tracking-[0.06em] text-[color:var(--color-text-label)]">
              Eyebrow · JetBrains Mono 11/600 0.06em
            </p>
            <p className="font-mono text-[14px] leading-[1.6] text-[color:var(--color-text-primary)]">
              agh-network/v0 · run_id_01hq8…
            </p>
            <div className="flex items-center gap-3">
              <span className="font-wordmark text-[28px] leading-none tracking-[-0.02em] text-[color:var(--color-text-primary)]">
                agh
              </span>
              <Pill variant="default" size="sm" className="border-[color:var(--color-divider)]">
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
      right={<MonoBadge tone="accent">action</MonoBadge>}
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
          <Badge>Default badge</Badge>
          <Badge variant="secondary">Secondary</Badge>
          <Badge variant="destructive">Destructive</Badge>
          <Badge variant="outline">Outline</Badge>
        </div>
        <div className="flex flex-wrap items-center gap-3">
          <Pill variant="default">Neutral</Pill>
          <Pill variant="accent">Action</Pill>
          <Pill variant="success">Stable</Pill>
          <Pill variant="warning">Pending</Pill>
          <Pill variant="danger">Error</Pill>
          <Pill variant="info">Info</Pill>
        </div>
        <Pills
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
      right={<MonoBadge>form primitives</MonoBadge>}
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
      right={<MonoBadge tone="info">signal</MonoBadge>}
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
          <StatusDot tone="success" />
          <span className="text-sm text-[color:var(--color-text-secondary)]">Connected</span>
        </div>
        <div className="inline-flex items-center gap-2">
          <StatusDot tone="warning" pulse />
          <span className="text-sm text-[color:var(--color-text-secondary)]">Reconnecting</span>
        </div>
        <div className="inline-flex items-center gap-2">
          <StatusDot tone="danger" />
          <span className="text-sm text-[color:var(--color-text-secondary)]">Disconnected</span>
        </div>
        <ConnectionIndicator status="connected" />
        <ConnectionIndicator status="reconnecting" />
        <ConnectionIndicator status="disconnected" />
      </div>
      <div className="flex flex-wrap items-center gap-2 pt-4">
        <MonoBadge tone="default">id_01HQ…</MonoBadge>
        <MonoBadge tone="neutral">idle</MonoBadge>
        <MonoBadge tone="accent">RUNNING</MonoBadge>
        <MonoBadge tone="success">DONE</MonoBadge>
        <MonoBadge tone="warning">PARTIAL</MonoBadge>
        <MonoBadge tone="danger">ERROR</MonoBadge>
        <MonoBadge tone="info">INFO</MonoBadge>
      </div>
      <div className="flex flex-wrap items-center gap-2 pt-3">
        {KINDS.map(kind => (
          <KindChip key={kind} kind={kind} />
        ))}
      </div>
      <div className="flex flex-col gap-3 pt-6">
        <div className="flex flex-wrap items-center gap-4">
          <Spinner className="size-4 text-[color:var(--color-accent)]" />
          <Skeleton className="h-4 w-40" />
          <Separator orientation="vertical" className="h-6" />
          <span className="font-mono text-[11px] uppercase tracking-[0.06em] text-[color:var(--color-text-tertiary)]">
            spinners · skeletons · separators
          </span>
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
      right={<MonoBadge tone="warning">state</MonoBadge>}
    >
      <div className="grid gap-3 pt-4 md:grid-cols-2">
        <Alert>
          <InfoIcon />
          <AlertTitle>Scheduled maintenance</AlertTitle>
          <AlertDescription>
            Runtime will reload at 04:00 UTC to pick up skill index updates.
          </AlertDescription>
        </Alert>
        <Alert variant="destructive">
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
      right={<MonoBadge tone="accent">motion</MonoBadge>}
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
            <p className="text-sm text-[color:var(--color-text-secondary)]">
              Tabs host section switches — Base UI driven, motion-free.
            </p>
          </TabsContent>
          <TabsContent value="events">
            <p className="text-sm text-[color:var(--color-text-secondary)]">
              Replayable event timeline lives under this tab in production.
            </p>
          </TabsContent>
          <TabsContent value="artifacts">
            <p className="text-sm text-[color:var(--color-text-secondary)]">
              Generated files with their provenance chain.
            </p>
          </TabsContent>
        </Tabs>
        <Accordion defaultValue={["item-1"]}>
          <AccordionItem value="item-1">
            <AccordionTrigger>Can I rewind a session?</AccordionTrigger>
            <AccordionContent>
              Yes — every session is fully replayable from its event store.
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
            <p className="text-sm text-[color:var(--color-text-secondary)]">
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
      right={<MonoBadge tone="accent">session shells</MonoBadge>}
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
                <StatusDot tone="success" size="sm" />
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
      right={<MonoBadge>layout</MonoBadge>}
    >
      <div className="grid gap-4 pt-4 lg:grid-cols-2">
        <div className="h-[340px] overflow-hidden rounded-[var(--radius-diagram)] border border-[color:var(--color-divider)]">
          <Sidebar
            defaultCollapsed={false}
            rail={
              <>
                <button
                  type="button"
                  aria-label="Workspace agh-core"
                  className="inline-flex size-7 items-center justify-center rounded-full border border-[color:var(--color-accent)] bg-[color:var(--color-surface-elevated)] font-mono text-[11px] text-[color:var(--color-accent)]"
                >
                  A
                </button>
                <button
                  type="button"
                  aria-label="Workspace research"
                  className="inline-flex size-7 items-center justify-center rounded-full border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] font-mono text-[11px] text-[color:var(--color-text-secondary)]"
                >
                  R
                </button>
              </>
            }
            header={
              <>
                <FolderIcon className="size-3.5 text-[color:var(--color-text-tertiary)]" />
                <span className="text-[13px] font-medium">agh-core</span>
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
              <div className="flex items-center justify-between gap-2 text-xs text-[color:var(--color-text-tertiary)]">
                <ConnectionIndicator status="connected" />
                <Button variant="ghost" size="icon-sm" aria-label="Settings">
                  <SettingsIcon className="size-3.5" />
                </Button>
              </div>
            }
          />
        </div>
        <div className="h-[340px] overflow-hidden rounded-[var(--radius-diagram)] border border-[color:var(--color-divider)]">
          <SplitPane
            list={
              <ScrollArea className="h-full">
                <ul className="flex flex-col divide-y divide-[color:var(--color-divider)]">
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
                        <MonoBadge tone="accent">v0.4.2</MonoBadge>
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
      className="group flex items-center gap-2 rounded-md px-2 py-1.5 text-left text-[13px] text-[color:var(--color-text-secondary)] transition-colors hover:bg-[color:var(--color-hover)] hover:text-[color:var(--color-text-primary)] data-[active=true]:bg-[color:var(--color-surface-elevated)] data-[active=true]:text-[color:var(--color-text-primary)]"
    >
      <Icon className="size-3.5 text-[color:var(--color-text-tertiary)] group-data-[active=true]:text-[color:var(--color-accent)]" />
      <span>{label}</span>
    </button>
  );
}

export { DesignSystemShowcase, SECTIONS, TOKEN_GROUPS };
export type { TokenSwatch, TokenGroup, ShowcaseSection };
