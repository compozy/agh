import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { TasksBridgeNotificationsCard } from "../tasks-bridge-notifications-card";
import {
  buildBridgeNotificationCursorFixture,
  buildTaskBridgeNotificationSubscriptionFixture,
} from "../../mocks/fixtures";

describe("TasksBridgeNotificationsCard", () => {
  it("renders empty state when no subscriptions exist", () => {
    render(
      <TasksBridgeNotificationsCard
        onCreate={async () => undefined}
        onDelete={async () => undefined}
        subscriptions={[]}
      />
    );
    expect(screen.getByTestId("tasks-bridge-notifications-empty")).toBeInTheDocument();
  });

  it("renders error state when fetch fails", () => {
    render(
      <TasksBridgeNotificationsCard
        errorMessage="boom"
        onCreate={async () => undefined}
        onDelete={async () => undefined}
        subscriptions={[]}
      />
    );
    expect(screen.getByTestId("tasks-bridge-notifications-error")).toHaveTextContent("boom");
  });

  it("renders zero-state cursor diagnostics for new subscriptions", () => {
    const sub = buildTaskBridgeNotificationSubscriptionFixture({
      subscription_id: "bsub_zero",
      cursor: buildBridgeNotificationCursorFixture({
        consumer_id: "bridge_task_subscription:bsub_zero",
        last_sequence: 0,
        last_delivery_id: undefined,
        last_delivered_at: null,
        updated_at: null,
      }),
    });
    render(
      <TasksBridgeNotificationsCard
        onCreate={async () => undefined}
        onDelete={async () => undefined}
        subscriptions={[sub]}
      />
    );
    expect(
      screen.getByTestId("tasks-bridge-notifications-row-bsub_zero-cursor-zero")
    ).toBeInTheDocument();
  });

  it("renders populated cursor diagnostics with sequence + delivery id", () => {
    const sub = buildTaskBridgeNotificationSubscriptionFixture({
      subscription_id: "bsub_filled",
    });
    render(
      <TasksBridgeNotificationsCard
        onCreate={async () => undefined}
        onDelete={async () => undefined}
        subscriptions={[sub]}
      />
    );
    expect(
      screen.getByTestId("tasks-bridge-notifications-row-bsub_filled-cursor-seq")
    ).toHaveTextContent("seq 14");
  });

  it("calls onCreate with the form payload", async () => {
    const onCreate = vi.fn().mockResolvedValue(undefined);
    render(
      <TasksBridgeNotificationsCard
        onCreate={onCreate}
        onDelete={async () => undefined}
        subscriptions={[]}
      />
    );

    fireEvent.click(screen.getByTestId("tasks-bridge-notifications-create-trigger"));
    fireEvent.change(screen.getByTestId("tasks-bridge-notifications-create-bridge-instance-id"), {
      target: { value: "bridge_alpha" },
    });
    fireEvent.change(screen.getByTestId("tasks-bridge-notifications-create-workspace-id"), {
      target: { value: "ws_default" },
    });
    fireEvent.click(screen.getByTestId("tasks-bridge-notifications-create-submit"));

    await waitFor(() => expect(onCreate).toHaveBeenCalledTimes(1));
    const [payload] = onCreate.mock.calls[0]!;
    expect(payload).toMatchObject({
      bridge_instance_id: "bridge_alpha",
      delivery_mode: "direct-send",
      scope: "workspace",
      workspace_id: "ws_default",
    });
  });

  it("blocks submission when bridge instance id is missing", async () => {
    const onCreate = vi.fn().mockResolvedValue(undefined);
    render(
      <TasksBridgeNotificationsCard
        onCreate={onCreate}
        onDelete={async () => undefined}
        subscriptions={[]}
      />
    );
    fireEvent.click(screen.getByTestId("tasks-bridge-notifications-create-trigger"));
    fireEvent.click(screen.getByTestId("tasks-bridge-notifications-create-submit"));
    await waitFor(() =>
      expect(screen.getByTestId("tasks-bridge-notifications-create-error")).toBeInTheDocument()
    );
    expect(onCreate).not.toHaveBeenCalled();
  });

  it("delegates delete to onDelete with the subscription id", async () => {
    const onDelete = vi.fn().mockResolvedValue(undefined);
    const sub = buildTaskBridgeNotificationSubscriptionFixture({ subscription_id: "bsub_a" });
    render(
      <TasksBridgeNotificationsCard
        onCreate={async () => undefined}
        onDelete={onDelete}
        subscriptions={[sub]}
      />
    );
    fireEvent.click(screen.getByTestId("tasks-bridge-notifications-row-bsub_a-delete"));
    fireEvent.click(screen.getByTestId("tasks-bridge-notifications-delete-confirm"));
    await waitFor(() => expect(onDelete).toHaveBeenCalledWith("bsub_a"));
  });
});
