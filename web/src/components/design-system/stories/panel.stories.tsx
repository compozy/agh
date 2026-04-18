import type { Meta, StoryObj } from "@storybook/react-vite";

import { Panel, PanelBody, PanelDescription, PanelFooter, PanelHeader, PanelTitle } from "../panel";
import { Pill } from "../pill";

import { StoryFrame } from "./story-frame";

const meta: Meta<typeof Panel> = {
  title: "components/design-system/Panel",
  component: Panel,
  parameters: {
    layout: "centered",
    docs: {
      description: {
        component:
          "The primary AGH surface container, supporting default, elevated, and accented depths for dashboards and trays.",
      },
    },
  },
  decorators: [
    Story => (
      <StoryFrame className="max-w-6xl">
        <Story />
      </StoryFrame>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof meta>;

const toneStories = [
  {
    label: "Default",
    text: "General-purpose surface for rows, summaries, and lists.",
    tone: "default",
  },
  {
    label: "Elevated",
    text: "Slightly brighter tier for stacked trays and supportive sections.",
    tone: "elevated",
  },
  {
    label: "Accented",
    text: "Warm-accented tier for first-pass highlights and guided entry points.",
    tone: "accented",
  },
] as const;

/**
 * Default panel with its header, body, and footer composition.
 */
export const Default: Story = {
  args: {},
  render: () => (
    <Panel>
      <PanelHeader>
        <PanelTitle>Dense surfaces still need calm internal hierarchy.</PanelTitle>
        <PanelDescription>
          The panel primitive carries the shared shell language so future views do not need to
          rebuild it from scratch.
        </PanelDescription>
      </PanelHeader>
      <PanelBody>
        <p className="text-sm leading-6 text-[color:var(--color-text-secondary)]">
          Matte background, fine border, and inner highlight are all handled at the primitive level.
        </p>
      </PanelBody>
      <PanelFooter>
        <Pill emphasis="strong" kind="state" tone="green">
          Ready
        </Pill>
        <span className="font-mono text-[0.625rem] uppercase tracking-[0.14em] text-[color:var(--color-text-label)]">
          panel / default
        </span>
      </PanelFooter>
    </Panel>
  ),
};

/**
 * Tone variants for different depth and emphasis levels.
 */
export const Tones: Story = {
  args: {},
  render: () => (
    <div className="grid w-full gap-4 lg:grid-cols-3">
      {toneStories.map(item => (
        <Panel key={item.label} tone={item.tone}>
          <PanelHeader>
            <PanelTitle>{item.label}</PanelTitle>
          </PanelHeader>
          <PanelBody>
            <p className="text-sm leading-6 text-[color:var(--color-text-secondary)]">
              {item.text}
            </p>
          </PanelBody>
        </Panel>
      ))}
    </div>
  ),
};
