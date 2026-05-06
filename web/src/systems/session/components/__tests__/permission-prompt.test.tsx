import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { describe, expect, it, vi, beforeEach, afterEach } from "vitest";

vi.mock("sonner", () => ({
  toast: { error: vi.fn() },
}));

vi.mock("@/lib/utils", () => ({
  cn: (...args: unknown[]) => args.filter(Boolean).join(" "),
}));

vi.mock("@agh/ui", () => ({
  Button: ({
    children,
    onClick,
    disabled,
    ...props
  }: {
    children: React.ReactNode;
    onClick?: () => void;
    disabled?: boolean;
    [key: string]: unknown;
  }) => (
    <button onClick={onClick} disabled={disabled} {...props}>
      {children}
    </button>
  ),
  Card: ({ children, ...props }: Record<string, unknown>) => (
    <div {...props}>{children as React.ReactNode}</div>
  ),
  CardHeader: ({ children }: Record<string, unknown>) => <div>{children as React.ReactNode}</div>,
  CardTitle: ({ children }: Record<string, unknown>) => <h3>{children as React.ReactNode}</h3>,
  CardContent: ({ children }: Record<string, unknown>) => <div>{children as React.ReactNode}</div>,
  CardFooter: ({ children }: Record<string, unknown>) => <div>{children as React.ReactNode}</div>,
}));

vi.mock("../../adapters/session-api", () => ({
  approveSession: vi.fn(),
}));

import { toast } from "sonner";
import { PermissionPrompt } from "../permission-prompt";
import { approveSession } from "../../adapters/session-api";
import type { PermissionRequest } from "../../types";

const mockPermission: PermissionRequest = {
  requestId: "req-123",
  toolName: "Bash",
  action: "execute",
  resource: "rm -rf /tmp/test",
  toolInput: { command: "rm -rf /tmp/test" },
};

describe("PermissionPrompt", () => {
  beforeEach(() => {
    vi.mocked(approveSession).mockResolvedValue(undefined);
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("renders tool name, action, and resource from PermissionRequest", () => {
    render(
      <PermissionPrompt permission={mockPermission} sessionId="sess-001" onResolved={vi.fn()} />
    );

    expect(screen.getByText("Bash")).toBeInTheDocument();
    expect(screen.getByText("execute")).toBeInTheDocument();
    expect(screen.getByText("rm -rf /tmp/test")).toBeInTheDocument();
  });

  it("renders all 4 action buttons", () => {
    render(
      <PermissionPrompt permission={mockPermission} sessionId="sess-001" onResolved={vi.fn()} />
    );

    expect(screen.getByTestId("permission-allow-once")).toBeInTheDocument();
    expect(screen.getByTestId("permission-allow-always")).toBeInTheDocument();
    expect(screen.getByTestId("permission-reject-once")).toBeInTheDocument();
    expect(screen.getByTestId("permission-reject-always")).toBeInTheDocument();

    expect(screen.getByText("Allow Once")).toBeInTheDocument();
    expect(screen.getByText("Allow Always")).toBeInTheDocument();
    expect(screen.getByText("Reject Once")).toBeInTheDocument();
    expect(screen.getByText("Reject Always")).toBeInTheDocument();
  });

  it("calls approve API and onResolved on Allow Once click", async () => {
    const onResolved = vi.fn();
    render(
      <PermissionPrompt permission={mockPermission} sessionId="sess-001" onResolved={onResolved} />
    );

    fireEvent.click(screen.getByTestId("permission-allow-once"));

    await waitFor(() => {
      expect(approveSession).toHaveBeenCalledWith("sess-001", {
        request_id: "req-123",
        turn_id: "",
        decision: "allow-once",
      });
    });

    expect(onResolved).toHaveBeenCalled();
  });

  it("calls approve API with allow-always on Allow Always click", async () => {
    const onResolved = vi.fn();
    render(
      <PermissionPrompt permission={mockPermission} sessionId="sess-001" onResolved={onResolved} />
    );

    fireEvent.click(screen.getByTestId("permission-allow-always"));

    await waitFor(() => {
      expect(approveSession).toHaveBeenCalledWith("sess-001", {
        request_id: "req-123",
        turn_id: "",
        decision: "allow-always",
      });
    });

    expect(onResolved).toHaveBeenCalled();
  });

  it("calls approve API with reject-once on Reject Once click", async () => {
    const onResolved = vi.fn();
    render(
      <PermissionPrompt permission={mockPermission} sessionId="sess-001" onResolved={onResolved} />
    );

    fireEvent.click(screen.getByTestId("permission-reject-once"));

    await waitFor(() => {
      expect(approveSession).toHaveBeenCalledWith("sess-001", {
        request_id: "req-123",
        turn_id: "",
        decision: "reject-once",
      });
    });

    expect(onResolved).toHaveBeenCalled();
  });

  it("calls approve API with reject-always on Reject Always click", async () => {
    const onResolved = vi.fn();
    render(
      <PermissionPrompt permission={mockPermission} sessionId="sess-001" onResolved={onResolved} />
    );

    fireEvent.click(screen.getByTestId("permission-reject-always"));

    await waitFor(() => {
      expect(approveSession).toHaveBeenCalledWith("sess-001", {
        request_id: "req-123",
        turn_id: "",
        decision: "reject-always",
      });
    });

    expect(onResolved).toHaveBeenCalled();
  });

  it("handles approve API error gracefully (shows toast, clears state)", async () => {
    vi.mocked(approveSession).mockRejectedValue(new Error("Network error"));
    const onResolved = vi.fn();

    render(
      <PermissionPrompt permission={mockPermission} sessionId="sess-001" onResolved={onResolved} />
    );

    fireEvent.click(screen.getByTestId("permission-allow-once"));

    await waitFor(() => {
      expect(toast.error).toHaveBeenCalledWith(
        "Failed to send permission response. The agent may continue waiting."
      );
    });

    // Still clears state even on error
    expect(onResolved).toHaveBeenCalled();
  });

  it("renders tool input as formatted JSON", () => {
    render(
      <PermissionPrompt permission={mockPermission} sessionId="sess-001" onResolved={vi.fn()} />
    );

    const inputEl = screen.getByTestId("permission-tool-input");
    expect(inputEl).toBeInTheDocument();
    expect(inputEl.textContent).toContain("rm -rf /tmp/test");
  });

  it("does not render tool input when empty", () => {
    const emptyInputPermission: PermissionRequest = {
      ...mockPermission,
      toolInput: {},
    };

    render(
      <PermissionPrompt
        permission={emptyInputPermission}
        sessionId="sess-001"
        onResolved={vi.fn()}
      />
    );

    expect(screen.queryByTestId("permission-tool-input")).not.toBeInTheDocument();
  });

  it("renders Permission Required title", () => {
    render(
      <PermissionPrompt permission={mockPermission} sessionId="sess-001" onResolved={vi.fn()} />
    );

    expect(screen.getByText("Permission Required")).toBeInTheDocument();
  });
});
