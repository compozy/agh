import { fileURLToPath } from "node:url";
import path from "node:path";

import { sessionLifecycleSelectors } from "../fixtures/selectors";
import { expect, test } from "../fixtures/test";
import { useGlobalWorkspaceIfPrompted } from "../fixtures/workspace";

const browserLifecycleFixture = path.resolve(
  path.dirname(fileURLToPath(import.meta.url)),
  "..",
  "..",
  "..",
  "internal",
  "testutil",
  "acpmock",
  "testdata",
  "browser_session_lifecycle_fixture.json"
);

const categorizedAgent = "categorized-agent";
const categorizedAgentCategory = ["Marketing", "Sales"] as const;
const flatAgent = "flat-agent";

test.use({
  runtimeOptions: {
    seed: {
      mockAgents: [
        {
          fixturePath: browserLifecycleFixture,
          fixtureAgent: "browser-lifecycle-agent",
          agentName: categorizedAgent,
          category_path: [...categorizedAgentCategory],
        },
        {
          fixturePath: browserLifecycleFixture,
          fixtureAgent: "browser-lifecycle-agent",
          agentName: flatAgent,
        },
      ],
    },
  },
});

test("categorized agents render through the sidebar tree, group inside the session-create command picker, and route on click", async ({
  appPage,
}) => {
  const ui = sessionLifecycleSelectors(appPage);

  await useGlobalWorkspaceIfPrompted(ui);
  await expect(ui.appSidebar).toBeVisible();

  // Sidebar: categorized agent appears under its folder; flat agent stays root-level.
  const folderSegments = categorizedAgentCategory.join("/");
  const folderTopId = `agent-category-${categorizedAgentCategory[0]}`;
  await expect(appPage.getByTestId(folderTopId)).toBeVisible();
  await expect(appPage.getByTestId(`agent-category-${folderSegments}`)).toBeVisible();
  await appPage.getByTestId(`agent-category-${folderSegments}`).click();
  await expect(ui.agentRow(categorizedAgent)).toBeVisible();
  await expect(ui.agentRow(flatAgent)).toBeVisible();

  // Open the agent page for the flat agent so the session-create dialog has a valid workspace.
  await ui.agentRow(flatAgent).click();
  await expect.poll(() => new URL(appPage.url()).pathname).toBe(`/agents/${flatAgent}`);
  await expect(ui.agentPageNewSession).toBeVisible();
  await ui.agentPageNewSession.click();

  // Session-create command picker: open it, type the categorized agent's name to confirm it
  // routes through the AgentCommandSelect (the popover renders the grouped list inside it).
  const trigger = appPage.getByTestId("session-create-agent-select");
  await expect(trigger).toBeVisible();
  await trigger.click();
  const categorizedItem = appPage.getByTestId(`agent-command-item-${categorizedAgent}`);
  await expect(categorizedItem).toBeVisible();
  const groupedHeading = appPage.getByTestId(`agent-command-group-category:${folderSegments}`);
  await expect(groupedHeading).toBeVisible();
  await expect(groupedHeading).toContainText(categorizedAgentCategory.join(" / "));
  await categorizedItem.click();
  await expect(trigger).toContainText(categorizedAgent);

  // Cancel the dialog and click the categorized leaf in the sidebar to confirm routing.
  await appPage.getByTestId("session-create-dialog-cancel").click();
  await ui.agentRow(categorizedAgent).click();
  await expect.poll(() => new URL(appPage.url()).pathname).toBe(`/agents/${categorizedAgent}`);
});
