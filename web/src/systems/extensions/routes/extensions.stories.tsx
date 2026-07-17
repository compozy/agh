import type { Meta, StoryObj } from "@storybook/react-vite";
import { delay, HttpResponse } from "msw";
import { expect, userEvent, within } from "storybook/test";

import { aghApiMock } from "@/storybook/openapi-msw";
import { storybookMswParameters } from "@/storybook/msw";
import {
  StorybookRouteCanvas,
  StorybookWorkspaceSetup,
  appRouteParameters,
} from "@/storybook/route-story-meta";
import { extensionFixtures } from "@/systems/extensions/mocks";
import { marketplaceKindFixture } from "@/systems/marketplace/mocks";

const extensionFeed = {
  ...marketplaceKindFixture("extension"),
  items: marketplaceKindFixture("extension").items.map(item =>
    item.entry_id === "otel-bridge"
      ? {
          ...item,
          installed: true,
          installed_version: "0.5.2",
          manage_path: "/extensions/otel-bridge",
          update_available: true,
        }
      : item.entry_id === "slack-notify"
        ? {
            ...item,
            installed: true,
            installed_version: "1.1.4",
            manage_path: "/extensions/slack-notify",
          }
        : item
  ),
};

const populatedParameters = {
  ...appRouteParameters("/extensions"),
  ...storybookMswParameters({
    marketplace: [
      aghApiMock.get("/api/marketplace/{kind}", () => HttpResponse.json(extensionFeed)),
    ],
  }),
};

const meta: Meta<typeof StorybookRouteCanvas> = {
  title: "systems/extensions/routes/Extensions",
  component: StorybookRouteCanvas,
  parameters: {
    layout: "fullscreen",
    docs: {
      description: {
        component:
          "Route-connected inventory stories for installed extensions and active bundle activations.",
      },
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  parameters: populatedParameters,
  render: () => <StorybookWorkspaceSetup />,
};

export const Bundles: Story = {
  parameters: appRouteParameters("/extensions?tab=bundles"),
  render: () => <StorybookWorkspaceSetup />,
};

export const Loading: Story = {
  parameters: {
    ...appRouteParameters("/extensions"),
    ...storybookMswParameters({
      extensions: [
        aghApiMock.get("/api/extensions", async () => {
          await delay("infinite");
          return HttpResponse.json({ extensions: extensionFixtures });
        }),
      ],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
};

export const BundlesLoading: Story = {
  parameters: {
    ...appRouteParameters("/extensions?tab=bundles"),
    ...storybookMswParameters({
      extensions: [
        aghApiMock.get("/api/bundles/activations", async () => {
          await delay("infinite");
          return HttpResponse.json({ activations: [] });
        }),
      ],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
};

export const ProvenanceOpen: Story = {
  parameters: populatedParameters,
  render: () => <StorybookWorkspaceSetup />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await userEvent.click(await canvas.findByRole("button", { name: "Actions for otel-bridge" }));
    await userEvent.click(
      await within(document.body).findByRole("menuitem", { name: "Provenance" })
    );
    await expect(
      within(document.body).findByTestId("extension-provenance-content")
    ).resolves.toBeDefined();
  },
};

export const RemoveOpen: Story = {
  parameters: populatedParameters,
  render: () => <StorybookWorkspaceSetup />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await userEvent.click(await canvas.findByRole("button", { name: "Actions for otel-bridge" }));
    await userEvent.click(await within(document.body).findByRole("menuitem", { name: "Remove…" }));
    await expect(
      within(document.body).findByTestId("remove-extension-dialog")
    ).resolves.toBeDefined();
  },
};

export const RemoveDependencyLoading: Story = {
  tags: ["play-fn"],
  parameters: {
    ...populatedParameters,
    ...storybookMswParameters({
      marketplace: [
        aghApiMock.get("/api/marketplace/{kind}", () => HttpResponse.json(extensionFeed)),
      ],
      extensions: [
        aghApiMock.get("/api/bundles/activations", async () => {
          await delay("infinite");
          return HttpResponse.json({ activations: [] });
        }),
      ],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await userEvent.click(await canvas.findByRole("button", { name: "Actions for otel-bridge" }));
    await userEvent.click(await within(document.body).findByRole("menuitem", { name: "Remove…" }));
    await within(document.body).findByText("Checking active bundles before removal.");
  },
};

export const RemoveDependencyError: Story = {
  tags: ["play-fn"],
  parameters: {
    ...populatedParameters,
    ...storybookMswParameters({
      marketplace: [
        aghApiMock.get("/api/marketplace/{kind}", () => HttpResponse.json(extensionFeed)),
      ],
      extensions: [
        aghApiMock.get("/api/bundles/activations", () =>
          HttpResponse.json({ error: "Bundle inventory unavailable" }, { status: 503 })
        ),
      ],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await userEvent.click(await canvas.findByRole("button", { name: "Actions for otel-bridge" }));
    await userEvent.click(await within(document.body).findByRole("menuitem", { name: "Remove…" }));
    await within(document.body).findByRole("button", { name: "Retry bundle activity" });
  },
};

export const DeactivateOpen: Story = {
  parameters: appRouteParameters("/extensions?tab=bundles"),
  render: () => <StorybookWorkspaceSetup />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await userEvent.click(await canvas.findByRole("button", { name: "Actions for ops-starter" }));
    await userEvent.click(
      await within(document.body).findByRole("menuitem", { name: "Deactivate…" })
    );
    await expect(
      within(document.body).findByTestId("deactivate-bundle-dialog")
    ).resolves.toBeDefined();
  },
};
