import { render, screen, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { TopbarSlotProvider } from "@agh/ui";

import { SettingsPageFrame } from "../settings-page-frame";
import { SettingsSaveBar } from "../settings-save-bar";

function renderFrame(saveBar: React.ReactNode) {
  return render(
    <TopbarSlotProvider>
      <SettingsPageFrame
        description="Defaults and day-to-day behavior."
        meta={[{ key: "sessions", content: <span>3 active sessions</span> }]}
        saveBar={saveBar}
        slug="general"
      >
        <div data-testid="frame-body-content">content</div>
      </SettingsPageFrame>
    </TopbarSlotProvider>
  );
}

describe("SettingsPageFrame", () => {
  it("renders the subhead sentence, quiet meta, and body content", () => {
    renderFrame(null);

    const subhead = screen.getByTestId("settings-page-general-subhead");
    expect(subhead).toHaveTextContent("Defaults and day-to-day behavior.");
    expect(subhead).toHaveTextContent("3 active sessions");
    expect(screen.getByTestId("settings-page-general-body")).toContainElement(
      screen.getByTestId("frame-body-content")
    );
  });

  it("keeps the floating save bar inside the scroll body when dirty", () => {
    renderFrame(
      <SettingsSaveBar
        slug="general"
        isDirty={true}
        isSaving={false}
        onSave={() => {}}
        onReset={() => {}}
      />
    );

    const body = screen.getByTestId("settings-page-general-body");
    expect(within(body).getByTestId("settings-page-general-save-bar")).toBeInTheDocument();
  });

  it("renders no save bar band when the page is clean", () => {
    renderFrame(
      <SettingsSaveBar
        slug="general"
        isDirty={false}
        isSaving={false}
        onSave={() => {}}
        onReset={() => {}}
      />
    );

    expect(screen.queryByTestId("settings-page-general-save-bar")).not.toBeInTheDocument();
  });

  it("does not own the route title (the window topbar does)", () => {
    renderFrame(null);

    expect(screen.queryByRole("heading", { level: 1 })).not.toBeInTheDocument();
  });
});
