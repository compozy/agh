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

test("categorized agents surface on the fleet page and group inside the session-create command picker", async ({
  appPage,
}) => {
  const ui = sessionLifecycleSelectors(appPage);

  await useGlobalWorkspaceIfPrompted(ui);
  await expect(ui.appSidebar).toBeVisible();
  await appPage.getByTestId("nav-agents").click();
  await expect.poll(() => new URL(appPage.url()).pathname).toBe("/agents");

  await expect(ui.agentRow(categorizedAgent)).toBeVisible();
  await expect(ui.agentRow(flatAgent)).toBeVisible();
  await expect(appPage.getByTestId("agent-fleet-row-link-categorized-agent")).toContainText(
    "Marketing / Sales"
  );

  await ui.agentRow(flatAgent).click();
  await expect.poll(() => new URL(appPage.url()).pathname).toBe(`/agents/${flatAgent}`);
  await expect(ui.agentPageNewSession).toBeVisible();
  await ui.agentPageNewSession.click();

  const trigger = appPage.getByTestId("session-create-agent-select");
  await expect(trigger).toBeVisible();
  await trigger.click();
  const categorizedItem = appPage.getByTestId(`agent-command-item-${categorizedAgent}`);
  await expect(categorizedItem).toBeVisible();
  const folderSegments = categorizedAgentCategory.join("/");
  const groupedHeading = appPage.getByTestId(`agent-command-group-category:${folderSegments}`);
  await expect(groupedHeading).toBeVisible();
  await expect(groupedHeading).toContainText(categorizedAgentCategory.join(" / "));
  await categorizedItem.click();
  await expect(trigger).toContainText(categorizedAgent);

  await appPage.getByTestId("session-create-dialog-cancel").click();
  await appPage.getByTestId("nav-agents").click();
  await expect.poll(() => new URL(appPage.url()).pathname).toBe("/agents");
  await ui.agentRow(categorizedAgent).click();
  await expect.poll(() => new URL(appPage.url()).pathname).toBe(`/agents/${categorizedAgent}`);
});
