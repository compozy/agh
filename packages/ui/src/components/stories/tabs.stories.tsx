import type { Meta, StoryObj } from "@storybook/react-vite";

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../card";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "../tabs";

const meta: Meta<typeof Tabs> = {
  title: "components/ui/Tabs",
  component: Tabs,
  parameters: {
    layout: "centered",
    docs: {
      description: {
        component:
          "Tabbed content switcher with two variants — `line` (canonical content tabs, 1.5px `--fg-strong` underline) and `lane` (page-head filter row with `·` separators and bare mono counts). The deprecated chipped `default` variant is removed; segmented-control surfaces use `<PillGroup>` instead.",
      },
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

const panels = [
  {
    value: "events",
    label: "Events",
    title: "Session events",
    body: "Append-only ACP event stream replayed from sessiondb.",
  },
  {
    value: "metrics",
    label: "Metrics",
    title: "Health metrics",
    body: "Latency, error rate, and token counters per driver.",
  },
  {
    value: "artifacts",
    label: "Artifacts",
    title: "Files & snapshots",
    body: "Captured file snapshots grouped by turn.",
  },
] as const;

export const LineDefault: Story = {
  args: {},
  parameters: {
    docs: {
      description: {
        story:
          "Default `line` variant — 1.5px `--fg-strong` underline (formerly 2px `--accent`), neutral chrome, sentence-case labels.",
      },
    },
  },
  render: () => (
    <Tabs defaultValue={panels[0].value} className="w-[28rem]">
      <TabsList variant="line">
        {panels.map(panel => (
          <TabsTrigger key={panel.value} value={panel.value}>
            {panel.label}
          </TabsTrigger>
        ))}
      </TabsList>
      {panels.map(panel => (
        <TabsContent key={panel.value} value={panel.value}>
          <Card>
            <CardHeader>
              <CardTitle>{panel.title}</CardTitle>
              <CardDescription>{panel.body}</CardDescription>
            </CardHeader>
            <CardContent className="text-sm text-muted-foreground">
              Switch tabs to inspect each slice of the session.
            </CardContent>
          </Card>
        </TabsContent>
      ))}
    </Tabs>
  ),
};

export const LineWithCounts: Story = {
  args: {},
  parameters: {
    docs: {
      description: {
        story:
          "`line` variant with count chips — inactive chip is neutral `--canvas-tint`/`--muted`; the active count switches to a 0.07 white glaze on `--fg` (no accent fill).",
      },
    },
  },
  render: () => (
    <Tabs defaultValue="runs" className="w-[32rem]">
      <TabsList variant="line">
        <TabsTrigger count={6} value="children">
          Children
        </TabsTrigger>
        <TabsTrigger count={2} value="dependencies">
          Dependencies
        </TabsTrigger>
        <TabsTrigger count={1} liveLabel="Live" value="runs">
          Runs
        </TabsTrigger>
      </TabsList>
      <TabsContent value="children" className="pt-3">
        <p className="text-sm text-muted-foreground">Six child tasks reference this parent.</p>
      </TabsContent>
      <TabsContent value="dependencies" className="pt-3">
        <p className="text-sm text-muted-foreground">Two dependencies gate execution.</p>
      </TabsContent>
      <TabsContent value="runs" className="pt-3">
        <p className="text-sm text-muted-foreground">One active run is streaming status updates.</p>
      </TabsContent>
    </Tabs>
  ),
};

export const LaneDefault: Story = {
  args: {},
  parameters: {
    docs: {
      description: {
        story:
          "`lane` variant — page-head filter row introduced by Triggers separated by a `·` glyph; counts render inline as bare 10.5px mono `--faint` (no chip).",
      },
    },
  },
  render: () => (
    <Tabs defaultValue="all" className="w-[36rem]">
      <TabsList variant="lane">
        <TabsTrigger count={42} value="all">
          All
        </TabsTrigger>
        <TabsTrigger count={6} value="active">
          Active
        </TabsTrigger>
        <TabsTrigger count={3} value="waiting">
          Waiting
        </TabsTrigger>
        <TabsTrigger count={11} value="done">
          Done
        </TabsTrigger>
      </TabsList>
      <TabsContent value="all" className="pt-3">
        <p className="text-sm text-muted-foreground">Showing 42 tasks across every status.</p>
      </TabsContent>
      <TabsContent value="active" className="pt-3">
        <p className="text-sm text-muted-foreground">6 tasks in flight right now.</p>
      </TabsContent>
      <TabsContent value="waiting" className="pt-3">
        <p className="text-sm text-muted-foreground">3 tasks waiting on dependencies.</p>
      </TabsContent>
      <TabsContent value="done" className="pt-3">
        <p className="text-sm text-muted-foreground">11 tasks completed in the last 24h.</p>
      </TabsContent>
    </Tabs>
  ),
};

export const LaneWithoutCounts: Story = {
  args: {},
  parameters: {
    docs: {
      description: {
        story: "`lane` variant without counts — separator + active underline still apply.",
      },
    },
  },
  render: () => (
    <Tabs defaultValue="overview" className="w-[36rem]">
      <TabsList variant="lane">
        <TabsTrigger value="overview">Overview</TabsTrigger>
        <TabsTrigger value="usage">Usage</TabsTrigger>
        <TabsTrigger value="audit">Audit log</TabsTrigger>
      </TabsList>
      <TabsContent value="overview" className="pt-3">
        <p className="text-sm text-muted-foreground">Workspace summary.</p>
      </TabsContent>
      <TabsContent value="usage" className="pt-3">
        <p className="text-sm text-muted-foreground">Session and token usage.</p>
      </TabsContent>
      <TabsContent value="audit" className="pt-3">
        <p className="text-sm text-muted-foreground">Recent operator and agent actions.</p>
      </TabsContent>
    </Tabs>
  ),
};

export const Vertical: Story = {
  args: {},
  parameters: {
    docs: {
      description: {
        story:
          "`orientation='vertical'` stacks the list down the left edge and aligns triggers to the start.",
      },
    },
  },
  render: () => (
    <Tabs
      defaultValue={panels[0].value}
      orientation="vertical"
      className="w-[32rem] flex-row items-start gap-4"
    >
      <TabsList variant="line">
        {panels.map(panel => (
          <TabsTrigger key={panel.value} value={panel.value}>
            {panel.label}
          </TabsTrigger>
        ))}
      </TabsList>
      {panels.map(panel => (
        <TabsContent key={panel.value} value={panel.value}>
          <p className="text-sm text-muted-foreground">{panel.body}</p>
        </TabsContent>
      ))}
    </Tabs>
  ),
};
