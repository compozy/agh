import type { Meta, StoryObj } from "@storybook/react-vite";

import { Button } from "@agh/ui";

import { PageContent } from "../page-content";
import { Panel, PanelBody, PanelHeader, PanelTitle } from "../panel";
import { SectionHeading } from "../section-heading";

import { TexturedStoryFrame } from "./story-frame";

const meta: Meta<typeof PageContent> = {
  title: "components/design-system/PageContent",
  component: PageContent,
  parameters: {
    layout: "fullscreen",
    docs: {
      description: {
        component:
          "A constrained page wrapper for AGH command surfaces. It sets the outer rhythm and breathing room for dense operator layouts.",
      },
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Default page content composition inside the textured canvas.
 */
export const Default: Story = {
  args: {},
  render: () => (
    <TexturedStoryFrame>
      <PageContent className="min-h-0 gap-5 px-0 py-0">
        <SectionHeading
          action={<Button variant="default">Open workspace</Button>}
          description="The shell constrains wide layouts without flattening the denser interior rhythm."
          eyebrow="AGH shell"
          title="A centered command surface with room for dense panels."
        />
        <Panel className="max-w-4xl">
          <PanelHeader>
            <PanelTitle>Shell content stays readable even when the viewport grows.</PanelTitle>
          </PanelHeader>
          <PanelBody>
            <p className="text-sm leading-6 text-[color:var(--color-text-secondary)]">
              This is the outer frame future routes can reuse before composing panels, toolbars, and
              system rows.
            </p>
          </PanelBody>
        </Panel>
      </PageContent>
    </TexturedStoryFrame>
  ),
};
