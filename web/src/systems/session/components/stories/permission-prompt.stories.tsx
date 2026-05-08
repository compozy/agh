import { useEffect, useRef, useState } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";

import { CenteredSurface } from "@/storybook/story-layout";
import { permissionRequestFixture, primarySessionFixture } from "@/systems/session/mocks";

import { PermissionPrompt } from "../permission-prompt";

const meta: Meta<typeof PermissionPrompt> = {
  title: "systems/session/PermissionPrompt",
  component: PermissionPrompt,
  parameters: {
    layout: "centered",
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

type AutoDecision = "allow-once" | "reject-once" | null;

function PermissionPromptHarness({ autoDecision }: { autoDecision: AutoDecision }) {
  const [resolvedState, setResolvedState] = useState<AutoDecision>(null);
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!autoDecision) {
      return;
    }

    const frame = window.requestAnimationFrame(() => {
      const testId =
        autoDecision === "allow-once"
          ? "[data-testid='permission-allow-once']"
          : "[data-testid='permission-reject-once']";
      const button = containerRef.current?.querySelector<HTMLButtonElement>(testId);
      button?.click();
    });

    return () => window.cancelAnimationFrame(frame);
  }, [autoDecision]);

  return (
    <CenteredSurface className="flex-col gap-4">
      {resolvedState ? (
        <div className="w-full max-w-xl rounded-xl border border-(--color-divider) bg-(--color-surface) px-4 py-3 text-sm text-(--color-text-primary)">
          {resolvedState === "allow-once"
            ? "Permission approved for this turn."
            : "Permission rejected for this turn."}
        </div>
      ) : null}
      <div ref={containerRef} className="w-full max-w-xl">
        {resolvedState ? null : (
          <PermissionPrompt
            onResolved={() => setResolvedState(autoDecision ?? "allow-once")}
            permission={permissionRequestFixture}
            sessionId={primarySessionFixture.id}
          />
        )}
      </div>
    </CenteredSurface>
  );
}

export const Pending: Story = {
  render: () => <PermissionPromptHarness autoDecision={null} />,
};

export const Accepted: Story = {
  render: () => <PermissionPromptHarness autoDecision="allow-once" />,
};

export const Rejected: Story = {
  render: () => <PermissionPromptHarness autoDecision="reject-once" />,
};
