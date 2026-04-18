import type { Meta, StoryObj } from "@storybook/react-vite";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@agh/ui";

import { Tabs, TabsContent, TabsList, TabsTrigger } from "../tabs";

const meta: Meta<typeof Tabs> = {
  title: "components/ui/Tabs",
  component: Tabs,
  parameters: {
    layout: "centered",
    docs: {
      description: {
        component:
          "Tabbed content switcher with a default segmented style and a `line` variant for dashboards. Pairs with @agh/ui Card for panel framing.",
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

export const Default: Story = {
  args: {},
  render: () => (
    <Tabs defaultValue={panels[0].value} className="w-[28rem]">
      <TabsList>
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

export const LineVariant: Story = {
  args: {},
  render: () => (
    <Tabs defaultValue={panels[1].value} className="w-[28rem]">
      <TabsList variant="line">
        {panels.map(panel => (
          <TabsTrigger key={panel.value} value={panel.value}>
            {panel.label}
          </TabsTrigger>
        ))}
      </TabsList>
      {panels.map(panel => (
        <TabsContent key={panel.value} value={panel.value} className="pt-3">
          <p className="text-sm text-muted-foreground">{panel.body}</p>
        </TabsContent>
      ))}
    </Tabs>
  ),
};
