import type { Meta, StoryObj } from "@storybook/react-vite";

import { Button } from "../../button";
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbSeparator,
} from "../../breadcrumb";
import { RouteNav } from "../route-nav";
import { Topbar, TopbarOverflowIcon, TopbarSlotProvider, useTopbarSlot } from "../topbar";

const meta: Meta<typeof Topbar> = {
  title: "components/custom/Topbar",
  component: Topbar,
  parameters: {
    layout: "fullscreen",
    docs: {
      description: {
        component:
          "Route chrome shell (§04): one 48px three-zone grid — leading ancestry plus the route H1, centered sister-route navigation, and trailing actions. Routes push title overrides, routeNav, actions, and overflow via `useTopbarSlot`.",
      },
    },
  },
  decorators: [
    Story => (
      <div className="w-full border border-line bg-background">
        <Story />
      </div>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof meta>;

function IndexBreadcrumb() {
  return (
    <Breadcrumb aria-label="Breadcrumb">
      <BreadcrumbList className="flex-nowrap whitespace-nowrap">
        <BreadcrumbItem>
          <BreadcrumbLink href="#home">Home</BreadcrumbLink>
        </BreadcrumbItem>
        <BreadcrumbSeparator />
      </BreadcrumbList>
    </Breadcrumb>
  );
}

function DetailBreadcrumb() {
  return (
    <Breadcrumb aria-label="Breadcrumb">
      <BreadcrumbList className="flex-nowrap whitespace-nowrap">
        <BreadcrumbItem>
          <BreadcrumbLink href="#loops">Loops</BreadcrumbLink>
        </BreadcrumbItem>
        <BreadcrumbSeparator />
      </BreadcrumbList>
    </Breadcrumb>
  );
}

/**
 * T1 · Breadcrumb only — the minimum viable topbar.
 */
export const BreadcrumbOnly: Story = {
  args: {},
  render: () => (
    <TopbarSlotProvider>
      <Topbar breadcrumb={<IndexBreadcrumb />} title="Runs" />
    </TopbarSlotProvider>
  ),
};

function DetailActionsSetup() {
  useTopbarSlot({
    actions: (
      <div className="flex items-center gap-2">
        <Button size="sm" variant="ghost">
          Edit
        </Button>
        <Button size="sm">Run loop</Button>
      </div>
    ),
    overflow: (
      <Button aria-label="More actions" size="sm" variant="ghost">
        <TopbarOverflowIcon className="size-3" />
      </Button>
    ),
  });
  return null;
}

/**
 * T2 · Detail — parent › entity breadcrumb plus trailing actions.
 */
export const DetailActions: Story = {
  args: {},
  render: () => (
    <TopbarSlotProvider>
      <DetailActionsSetup />
      <Topbar breadcrumb={<DetailBreadcrumb />} title="software-delivery" />
    </TopbarSlotProvider>
  ),
};

function FullCompositionSetup() {
  useTopbarSlot({
    routeNav: (
      <RouteNav aria-label="Tasks views">
        <RouteNav.Link aria-current="page" href="#list">
          List
        </RouteNav.Link>
        <RouteNav.Link href="#kanban">Kanban</RouteNav.Link>
        <RouteNav.Link href="#inbox">
          Inbox <RouteNav.Count>2</RouteNav.Count>
        </RouteNav.Link>
      </RouteNav>
    ),
    actions: <Button size="sm">New task</Button>,
  });
  return null;
}

/**
 * T4 · Breadcrumb + centered route navigation + actions.
 */
export const FullComposition: Story = {
  args: {},
  render: () => (
    <TopbarSlotProvider>
      <FullCompositionSetup />
      <Topbar
        breadcrumb={
          <Breadcrumb aria-label="Breadcrumb">
            <BreadcrumbList className="flex-nowrap whitespace-nowrap">
              <BreadcrumbItem>
                <BreadcrumbLink href="#operate">Operate</BreadcrumbLink>
              </BreadcrumbItem>
              <BreadcrumbSeparator />
            </BreadcrumbList>
          </Breadcrumb>
        }
        title="Tasks"
      />
    </TopbarSlotProvider>
  ),
};
