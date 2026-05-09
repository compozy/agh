// @vitest-environment node

import { describe, expect, it } from "vitest";

import {
  assertNoSensitiveArtifactPayload,
  buildCoverageMatrix,
  captureBrowserTransportSnapshot,
  e2eScenarioContracts,
  nightlyScenarioContracts,
  requestBrowserRuntimeOperatorJSON,
  runBrowserRuntimeCLIJSON,
  validateCoverageMatrix,
  validateNightlySpecCoverage,
  validateScenarioContracts,
  type ScenarioContract,
} from "../scenario-contracts";

describe("scenario contracts", () => {
  it("validates the shipped E2E scenario contract matrix", () => {
    expect(validateScenarioContracts(e2eScenarioContracts)).toEqual([]);

    const matrix = buildCoverageMatrix(e2eScenarioContracts);
    expect(validateCoverageMatrix(matrix)).toEqual([]);
    expect(matrix.find(row => row.module === "runtime-harness-transport")).toMatchObject({
      artifacts: expect.arrayContaining(["browser_transport_snapshots"]),
      executionAuditIDs: expect.arrayContaining(["C1", "C6", "C13", "C16", "C17", "C18"]),
      lanes: expect.arrayContaining(["make test-e2e-runtime", "make test-e2e-web"]),
      module: "runtime-harness-transport",
    });
  });

  it("reports missing module, provider boundary, artifacts, audit IDs, and lane mapping", () => {
    const broken = [minimalContract({ artifacts: [], executionAuditIDs: ["C1"] })];

    expect(validateScenarioContracts(broken)).toEqual(
      expect.arrayContaining([
        "scenario TC-BROKEN-001 must declare artifact evidence",
        "scenario TC-BROKEN-001 is missing C16 provider boundary evidence",
        "scenario TC-BROKEN-001 is missing C17 release gate mapping",
        "scenario TC-BROKEN-001 is P0/P1 but lacks C12 viewport evidence",
      ])
    );

    const matrix = buildCoverageMatrix(broken);
    expect(
      validateCoverageMatrix(matrix, [
        {
          module: "missing-module",
          requiredArtifacts: ["browser_screenshots"],
          requiredExecutionAuditIDs: ["C1"],
          requiredLanes: ["make test-e2e-web"],
        },
      ])
    ).toEqual(["module missing-module has no scenario contract row"]);
  });

  it("distinguishes nightly expected cases from blocked provider-boundary cases", () => {
    const liveNightly = minimalContract({
      id: "TC-NIGHTLY-001",
      nightly: true,
      lanes: ["make test-e2e-nightly"],
      providerBoundary: "live_provider",
      title: "operator runs live nightly",
    });
    const blockedNightly = minimalContract({
      blockedReason: "credential intentionally absent",
      id: "TC-NIGHTLY-002",
      nightly: true,
      lanes: ["make test-e2e-nightly"],
      providerBoundary: "blocked",
      title: "operator waits for unavailable provider",
    });

    expect(nightlyScenarioContracts([liveNightly, blockedNightly])).toEqual([liveNightly]);
  });

  it("fails nightly coverage when the expected browser case is missing", () => {
    const nightly = minimalContract({
      grep: "operator runs live nightly",
      id: "TC-NIGHTLY-001",
      lanes: ["make test-e2e-nightly"],
      nightly: true,
      providerBoundary: "live_provider",
      specPath: "web/e2e/__tests__/nightly.spec.ts",
      title: "operator runs live nightly",
    });

    expect(validateNightlySpecCoverage([nightly], {})).toEqual([
      "TC-NIGHTLY-001 spec web/e2e/__tests__/nightly.spec.ts does not contain @nightly",
      "TC-NIGHTLY-001 expected nightly test text is absent: operator runs live nightly",
    ]);
    expect(
      validateNightlySpecCoverage([nightly], {
        "web/e2e/__tests__/nightly.spec.ts":
          'test("@nightly operator runs live nightly", async () => {})',
      })
    ).toEqual([]);
  });

  it("rejects sensitive token-shaped browser artifact payloads", () => {
    const sensitivePayloads = [
      { value: "agh_claim_abc123def456" },
      { value: "Authorization: Bearer browsersecretvalue" },
      { value: "123456:abcdefghijklmnopqrstuvwxyz" },
      { value: "oauth_access_token = browser-oauth-secret" },
    ];

    for (const payload of sensitivePayloads) {
      expect(() => assertNoSensitiveArtifactPayload([payload])).toThrow(
        /sensitive token-like value/
      );
    }
  });
});

describe("browser runtime parity helpers", () => {
  it("rejects CLI and UDS helpers when launch-mode runtime paths are absent", async () => {
    const runtime = {
      requestOperatorJSON: undefined,
    } as Parameters<typeof requestBrowserRuntimeOperatorJSON>[0];

    await expect(requestBrowserRuntimeOperatorJSON(runtime, "/api/daemon/status")).rejects.toThrow(
      /requires launch-mode runtime access/
    );
    await expect(runBrowserRuntimeCLIJSON(runtime, ["daemon", "status"])).rejects.toThrow(
      /requires launch-mode runtime paths/
    );
  });

  it("captures transport snapshots into the browser artifact collector", async () => {
    const captured: Array<{ kind: string; value: unknown }> = [];
    const runtime = {
      artifactCollector: {
        captureJSON: async (kind: string, value: unknown) => {
          captured.push({ kind, value });
        },
      },
    } as Parameters<typeof captureBrowserTransportSnapshot>[0];

    const snapshot = await captureBrowserTransportSnapshot(runtime, "TC-HARNESS-001", {
      cli: { status: "running" },
      http: { status: "running" },
      uds: { status: "running" },
    });

    expect(snapshot).toEqual({
      cli: { status: "running" },
      http: { status: "running" },
      scenarioID: "TC-HARNESS-001",
      uds: { status: "running" },
    });
    expect(captured).toEqual([
      {
        kind: "browser_transport_snapshots",
        value: snapshot,
      },
    ]);
  });
});

function minimalContract(overrides: Partial<ScenarioContract> = {}): ScenarioContract {
  return {
    artifacts: ["browser_screenshots"],
    auditIDs: ["A1"],
    evidenceLevel: "L4",
    executionAuditIDs: ["C1", "C12", "C16", "C17"],
    id: "TC-BROKEN-001",
    lanes: ["make test-e2e-web"],
    module: "broken",
    priority: "P1",
    providerBoundary: "bounded_fake",
    specPath: "web/e2e/__tests__/broken.spec.ts",
    surfaces: ["web"],
    title: "broken scenario",
    ...overrides,
  };
}
