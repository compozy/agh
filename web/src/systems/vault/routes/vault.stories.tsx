import type { Meta, StoryObj } from "@storybook/react-vite";
import { delay, HttpResponse } from "msw";

import { aghApiMock } from "@/storybook/openapi-msw";
import { storybookMswParameters } from "@/storybook/msw";
import {
  StorybookRouteCanvas,
  StorybookWorkspaceSetup,
  appRouteParameters,
} from "@/storybook/route-story-meta";

const meta: Meta<typeof StorybookRouteCanvas> = {
  title: "systems/vault/routes/Vault",
  component: StorybookRouteCanvas,
  parameters: {
    layout: "fullscreen",
    docs: {
      description: {
        component:
          "Full app-shell route stories for the vault secrets route, backed by OpenAPI-typed MSW handlers.",
      },
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {},
  parameters: appRouteParameters("/vault"),
  render: () => <StorybookWorkspaceSetup />,
};

export const Cards: Story = {
  args: {},
  parameters: appRouteParameters("/vault?view=cards"),
  render: () => <StorybookWorkspaceSetup />,
};

export const Empty: Story = {
  args: {},
  parameters: {
    ...appRouteParameters("/vault"),
    ...storybookMswParameters({
      vault: [aghApiMock.get("/api/vault/secrets", () => HttpResponse.json({ secrets: [] }))],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
};

export const Loading: Story = {
  args: {},
  parameters: {
    ...appRouteParameters("/vault"),
    ...storybookMswParameters({
      vault: [
        aghApiMock.get("/api/vault/secrets", async () => {
          await delay("infinite");
          return HttpResponse.json({ secrets: [] });
        }),
      ],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
};

export const Error: Story = {
  args: {},
  parameters: {
    ...appRouteParameters("/vault"),
    ...storybookMswParameters({
      vault: [
        aghApiMock.get("/api/vault/secrets", () =>
          HttpResponse.json({ error: "vault unavailable" }, { status: 503 })
        ),
      ],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
};
