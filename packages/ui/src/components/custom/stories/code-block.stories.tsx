import type { Meta, StoryObj } from "@storybook/react-vite";
import { expect, fn, userEvent, waitFor, within } from "storybook/test";

import { CodeBlock, CopyIconButton } from "../code-block";

const meta: Meta<typeof CodeBlock> = {
  title: "components/custom/CodeBlock",
  component: CodeBlock,
  parameters: {
    layout: "padded",
    docs: {
      description: {
        component:
          "Terminal-style code block per DESIGN.md §4. Canvas-deep container, JetBrains Mono body at 14px/1.6, optional accent `$ ` prompt, optional language eyebrow, and a ghost copy button that swaps to a checkmark for 1.5s on copy success.",
      },
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

export const ShellCommand: Story = {
  args: {
    code: "agh start",
  },
  parameters: {
    docs: {
      description: {
        story: "Default shell prompt: single command with the accent `$ ` prompt.",
      },
    },
  },
};

export const MultilineWithoutPrompt: Story = {
  args: {
    showPrompt: false,
    code: `export function greet(name: string) {
  return \`Hello, \${name}\`;
}`,
  },
  parameters: {
    docs: {
      description: {
        story: "Source-code block with prompts disabled, rendered as plain code.",
      },
    },
  },
};

export const LanguageLabel: Story = {
  args: {
    language: "agh network",
    code: `# discover peers, send one task
agh network status
agh network peers
agh network send reviewer --kind direct \\
    --body '{"task":"review PR #482"}'`,
  },
  parameters: {
    docs: {
      description: {
        story:
          "Language eyebrow in the top-left. Comment (`#`) and indented continuation lines skip the prompt.",
      },
    },
  },
};

export const CopyDisabled: Story = {
  args: {
    copyable: false,
    code: "agh start",
  },
  parameters: {
    docs: {
      description: {
        story: "Static code block with the copy affordance hidden.",
      },
    },
  },
};

export const CopyInteraction: Story = {
  args: {
    code: "agh network status",
  },
  parameters: {
    docs: {
      description: {
        story: "Interaction test: click copy, assert checkmark appears, assert revert after 1.5s.",
      },
    },
  },
  play: async ({ canvasElement, step }) => {
    const canvas = within(canvasElement);
    const writeText = fn(async (_value: string) => undefined);
    const original = Object.getOwnPropertyDescriptor(navigator, "clipboard");
    Object.defineProperty(navigator, "clipboard", {
      configurable: true,
      value: { writeText },
    });
    try {
      await step("Copy button renders with the copy glyph", async () => {
        const root = canvasElement.querySelector<HTMLElement>('[data-slot="code-block"]');
        await expect(root).not.toBeNull();
        const button = await canvas.findByRole("button", { name: "Copy to clipboard" });
        await expect(button.querySelector("svg.lucide-copy")).not.toBeNull();
      });
      await step("Clicking copy invokes navigator.clipboard.writeText", async () => {
        const button = await canvas.findByRole("button", { name: "Copy to clipboard" });
        await userEvent.click(button);
        await waitFor(() => expect(writeText).toHaveBeenCalledWith("agh network status"));
      });
      await step("Button swaps to the check glyph", async () => {
        const success = await canvas.findByRole("button", { name: "Copied" });
        await expect(success.getAttribute("data-copied")).toBe("true");
        await expect(success.querySelector("svg.lucide-check")).not.toBeNull();
      });
      await step("Button reverts to copy glyph after ~1.5s", async () => {
        await waitFor(
          async () => {
            const reverted = await canvas.findByRole("button", { name: "Copy to clipboard" });
            await expect(reverted.getAttribute("data-copied")).toBeNull();
            await expect(reverted.querySelector("svg.lucide-copy")).not.toBeNull();
          },
          { timeout: 2500 }
        );
      });
    } finally {
      if (original) {
        Object.defineProperty(navigator, "clipboard", original);
      } else {
        Reflect.deleteProperty(navigator as unknown as { clipboard?: unknown }, "clipboard");
      }
    }
  },
};

export const WarningToneTruncated: Story = {
  args: {
    code: [
      "First warning line",
      "Second warning line",
      "Third warning line",
      "Fourth warning line",
    ].join("\n"),
    showPrompt: false,
    tone: "warning",
    truncateLines: 2,
  },
};

export const StandaloneCopyButton: Story = {
  args: {
    code: "agh network status",
  },
  render: args => (
    <div className="flex items-center gap-3 rounded-md border border-line bg-canvas-soft p-3">
      <span className="font-mono text-small-body text-muted">{args.code}</span>
      <CopyIconButton value={args.code} />
    </div>
  ),
};
