import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import type {
  WindowManagerLayoutResourceRecord,
  WindowManagerLayoutState,
  WindowManagerLayoutValidation,
} from "../../lib/window-manager-layout-types";
import { settingsKeys } from "../../lib/query-keys";
import { useWindowManagerLayoutEditor } from "../../hooks/use-window-manager-layout-editor";
import { useWindowManagerLayoutProfiles } from "../../hooks/use-window-manager-layout-profiles";
import { WindowManagerLayoutDocumentEditor } from "../window-manager-layout-document-editor";

const apiMocks = vi.hoisted(() => ({
  apply: vi.fn(),
  deleteProfile: vi.fn(),
  preview: vi.fn(),
  putProfile: vi.fn(),
  validate: vi.fn(),
}));

vi.mock("../../adapters/window-manager-layouts-api", () => ({
  applyWindowManagerLayout: apiMocks.apply,
  previewWindowManagerLayout: apiMocks.preview,
  validateWindowManagerLayout: apiMocks.validate,
  deleteWindowManagerLayoutProfile: apiMocks.deleteProfile,
  putWindowManagerLayoutProfile: apiMocks.putProfile,
}));

const initial: WindowManagerLayoutState = {
  revision: 7,
  document: {
    version: 1,
    workspaceId: "workspace-a",
    desktops: [
      {
        id: "desktop-a",
        name: "Primary",
        order: 0,
        purpose: "standard",
        focusOwner: null,
        groups: [
          {
            id: "group-a",
            frame: { x: 0, y: 0, w: 1, h: 1 },
            root: {
              id: "leaf-a",
              kind: "leaf",
              windowId: "window-a",
            },
          },
        ],
        floating: [],
      },
    ],
    windows: {
      "window-a": {
        id: "window-a",
        app: "agents",
        instanceKey: null,
        route: { pathname: "/agents", search: {} },
        placement: "tiled",
        desktopId: "desktop-a",
        floatingRect: { x: 0.1, y: 0.1, w: 0.5, h: 0.5 },
        minimized: false,
        returnAnchor: null,
      },
    },
    overrides: {},
  },
};

const PROFILE: WindowManagerLayoutResourceRecord = {
  kind: "window_layout",
  id: "primary-layout",
  version: 3,
  scope: { kind: "workspace", id: "workspace-a" },
  spec: {
    version: 1,
    id: "primary-layout",
    displayName: "Primary layout",
    aspectVariant: "any",
    participantSlots: ["window-a"],
    overflowPolicy: "stack",
    document: initial.document,
  },
  createdAt: "2026-07-22T00:00:00Z",
  updatedAt: "2026-07-22T00:00:00Z",
};

function EditorHarness({ profiles }: { profiles: readonly WindowManagerLayoutResourceRecord[] }) {
  const editor = useWindowManagerLayoutEditor("workspace-a", initial);
  const profilesEditor = useWindowManagerLayoutProfiles({
    workspaceId: "workspace-a",
    document: editor.draft,
    profiles,
    onLoad: editor.updateDraft,
  });
  return <WindowManagerLayoutDocumentEditor editor={editor} profilesEditor={profilesEditor} />;
}

function renderEditor(profiles: readonly WindowManagerLayoutResourceRecord[] = []) {
  const queryClient = new QueryClient({
    defaultOptions: {
      mutations: { retry: false },
      queries: { retry: false },
    },
  });
  queryClient.setQueryData(settingsKeys.windowManagerLayoutProfiles("workspace-a"), profiles);
  const rendered = render(
    <QueryClientProvider client={queryClient}>
      <EditorHarness profiles={profiles} />
    </QueryClientProvider>
  );
  return { ...rendered, queryClient };
}

function validation(valid: boolean): WindowManagerLayoutValidation {
  return {
    workspaceId: "workspace-a",
    valid,
    diagnostics: valid
      ? []
      : [
          {
            code: "invalid_split_weights",
            path: "desktops[0].groups[0].root.weights",
            message: "Split weights must be positive.",
          },
        ],
  };
}

afterEach(() => {
  vi.clearAllMocks();
});

describe("WindowManagerLayoutDocumentEditor", () => {
  it("Should apply only the exact document revision that passed validation and preview", async () => {
    apiMocks.validate.mockResolvedValue(validation(true));
    apiMocks.preview.mockResolvedValue({
      revision: 7,
      changed: true,
      changes: {
        desktopIds: ["desktop-a"],
        windowIds: [],
        groupIds: [],
        nodeIds: [],
        clientIds: [],
      },
      diagnostics: [],
    });
    apiMocks.apply.mockResolvedValue({
      revision: 8,
      applied: true,
      diagnostics: [],
    });
    renderEditor();

    const reviewButton = screen.getByRole("button", {
      name: "Validate and preview",
    });
    const applyButton = screen.getByRole("button", {
      name: "Apply reviewed layout",
    });
    expect(applyButton).toBeDisabled();

    fireEvent.click(reviewButton);

    expect(await screen.findByText("Daemon validation passed")).toBeInTheDocument();
    expect(apiMocks.preview).toHaveBeenCalledWith(
      "workspace-a",
      7,
      expect.objectContaining({ workspaceId: "workspace-a" })
    );
    expect(applyButton).toBeEnabled();

    fireEvent.change(screen.getByDisplayValue("Primary"), {
      target: { value: "Renamed" },
    });

    expect(applyButton).toBeDisabled();
    expect(screen.queryByTestId("window-manager-layout-review")).not.toBeInTheDocument();

    fireEvent.click(reviewButton);
    await waitFor(() => expect(apiMocks.preview).toHaveBeenCalledTimes(2));
    expect(applyButton).toBeEnabled();

    fireEvent.click(applyButton);

    await waitFor(() =>
      expect(apiMocks.apply).toHaveBeenCalledWith(
        "workspace-a",
        7,
        expect.objectContaining({
          desktops: [
            expect.objectContaining({
              id: "desktop-a",
              name: "Renamed",
            }),
          ],
        })
      )
    );
  });

  it("Should not preview or apply a document rejected by daemon validation", async () => {
    apiMocks.validate.mockResolvedValue(validation(false));
    renderEditor();

    fireEvent.click(
      screen.getByRole("button", {
        name: "Validate and preview",
      })
    );

    expect(await screen.findByText("Daemon validation failed")).toBeInTheDocument();
    expect(screen.getByTestId("window-manager-layout-review")).toHaveTextContent(
      "desktops[0].groups[0].root.weights: Split weights must be positive."
    );
    expect(apiMocks.preview).not.toHaveBeenCalled();
    expect(
      screen.getByRole("button", {
        name: "Apply reviewed layout",
      })
    ).toBeDisabled();
    expect(apiMocks.apply).not.toHaveBeenCalled();
  });

  it("Should move an existing profile scope with its current version and replace its cache identity", async () => {
    const moved: WindowManagerLayoutResourceRecord = {
      ...PROFILE,
      version: 4,
      scope: { kind: "global", id: "" },
      updatedAt: "2026-07-23T00:00:00Z",
    };
    apiMocks.putProfile.mockResolvedValue(moved);
    const { queryClient } = renderEditor([PROFILE]);

    fireEvent.click(screen.getByRole("radio", { name: /Primary layout/ }));
    fireEvent.change(screen.getByRole("combobox", { name: "Scope" }), {
      target: { value: "global" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Save profile" }));

    await waitFor(() =>
      expect(apiMocks.putProfile).toHaveBeenCalledWith(
        expect.objectContaining({ id: PROFILE.id }),
        "global",
        "workspace-a",
        PROFILE.version
      )
    );
    await waitFor(() =>
      expect(
        queryClient.getQueryData<WindowManagerLayoutResourceRecord[]>(
          settingsKeys.windowManagerLayoutProfiles("workspace-a")
        )
      ).toEqual([moved])
    );
  });
});
