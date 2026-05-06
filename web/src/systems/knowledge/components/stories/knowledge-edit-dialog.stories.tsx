import type { Meta, StoryObj } from "@storybook/react-vite";
import { expect, fn, userEvent, within } from "storybook/test";

import { KnowledgeEditDialog } from "@/systems/knowledge/components/knowledge-edit-dialog";

const meta: Meta<typeof KnowledgeEditDialog> = {
  title: "systems/knowledge/KnowledgeEditDialog",
  component: KnowledgeEditDialog,
  parameters: {
    layout: "fullscreen",
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

const initialContent = [
  "# Operator Style",
  "",
  "Lead with the fact pattern, then the decision, then the next concrete action.",
].join("\n");

export const Default: Story = {
  render: () => (
    <KnowledgeEditDialog
      filename="operator-style.md"
      initialContent={initialContent}
      initialDescription="Northstar guidance for concise, accountable operator communication."
      isPending={false}
      onConfirm={async () => {}}
      onOpenChange={() => undefined}
      open
      scope="global"
    />
  ),
};

export const PendingSave: Story = {
  render: () => (
    <KnowledgeEditDialog
      filename="operator-style.md"
      initialContent={initialContent}
      initialDescription=""
      isPending
      onConfirm={async () => {}}
      onOpenChange={() => undefined}
      open
      scope="global"
    />
  ),
};

export const RejectedByPolicy: Story = {
  render: () => (
    <KnowledgeEditDialog
      error="Edit rejected by policy: invisible Unicode in content"
      filename="operator-style.md"
      initialContent={initialContent}
      initialDescription=""
      isPending={false}
      onConfirm={async () => {}}
      onOpenChange={() => undefined}
      open
      scope="global"
    />
  ),
};

export const ConfirmSubmits: Story = {
  tags: ["play-fn"],
  render: () => {
    const onConfirm = fn();
    return (
      <KnowledgeEditDialog
        filename="operator-style.md"
        initialContent={initialContent}
        initialDescription="Northstar guidance"
        isPending={false}
        onConfirm={onConfirm}
        onOpenChange={() => undefined}
        open
        scope="global"
      />
    );
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement.ownerDocument.body);
    const editor = await canvas.findByTestId("knowledge-edit-content");
    await userEvent.type(editor, "\n\nFollow up with the next concrete metric.");
    const confirm = await canvas.findByTestId("confirm-edit-memory-btn");
    await expect(confirm).toBeVisible();
  },
};
