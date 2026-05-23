//go:build integration && !windows

package daemon

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	aghcontract "github.com/compozy/agh/internal/api/contract"
	aghconfig "github.com/compozy/agh/internal/config"
	"github.com/compozy/agh/internal/store"
	"github.com/compozy/agh/internal/testutil/acpmock"
	e2etest "github.com/compozy/agh/internal/testutil/e2e"
	"github.com/compozy/agh/internal/transcript"
)

func TestDaemonE2ENetworkDirectReplyLifecycleWithMockAgents(t *testing.T) {
	t.Run("Should complete direct reply lifecycle with mock agents", func(t *testing.T) {
		acpmock.RequireDriver(t)

		fixturePath := mockFixturePath(t, "network_collaboration_fixture.json")
		harness := e2etest.StartRuntimeHarness(t, e2etest.RuntimeHarnessOptions{
			EnableNetwork: true,
			MockAgents: []e2etest.MockAgentSpec{
				{
					FixturePath:  fixturePath,
					FixtureAgent: "ops-coordinator",
					AgentName:    "mock-ops-coordinator",
				},
				{
					FixturePath:  fixturePath,
					FixtureAgent: "patch-worker",
					AgentName:    "mock-patch-worker",
				},
			},
		})

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		channelDetail := mustCreateNetworkChannel(
			t,
			ctx,
			harness,
			"builders",
			"mock-ops-coordinator",
			"mock-patch-worker",
		)
		opsSession := requireChannelSession(t, channelDetail, "mock-ops-coordinator")
		patchSession := requireChannelSession(t, channelDetail, "mock-patch-worker")

		regOps, ok := harness.MockAgentRegistration("mock-ops-coordinator")
		if !ok {
			t.Fatal("MockAgentRegistration(mock-ops-coordinator) = missing, want present")
		}
		regPatch, ok := harness.MockAgentRegistration("mock-patch-worker")
		if !ok {
			t.Fatal("MockAgentRegistration(mock-patch-worker) = missing, want present")
		}

		registerNetworkScenarioArtifacts(
			t,
			harness,
			"builders",
			[]aghcontract.SessionPayload{opsSession, patchSession},
			[]acpmock.Registration{regOps, regPatch},
		)

		peers := waitForChannelPeerCount(t, ctx, harness, "builders", 2)
		opsPeerID := requirePeerIDForSession(t, peers, opsSession.ID)
		patchPeerID := requirePeerIDForSession(t, peers, patchSession.ID)
		if opsPeerID == patchPeerID {
			t.Fatalf("peer IDs = %q and %q, want distinct values", opsPeerID, patchPeerID)
		}
		buildersThreadID := "thread_builders_main"
		patchWorkID := "work_patch_42"
		patchDirectID := requireDirectResolveRace(
			t,
			ctx,
			harness,
			"builders",
			opsSession.ID,
			opsPeerID,
			patchSession.ID,
			patchPeerID,
		)

		mustSendNetworkCLI(t, ctx, harness, []string{
			"--session",
			opsSession.ID,
			"--channel",
			"builders",
			"--kind",
			"say",
			"--surface",
			"thread",
			"--thread",
			buildersThreadID,
			"--id",
			"msg_say_01",
			"--trace-id",
			"trace_ops_patch_42",
			"--body",
			`{"text":"Who can take the failing migration tests in internal/store/sessiondb?","intent":"request-help","artifacts":[]}`,
		})

		waitForRuntimeCondition(t, "builders say delivery", 10*time.Second, func() bool {
			return channelHasMessageID(ctx, harness, "builders", "msg_say_01") &&
				sessionTranscriptHasNeedle(ctx, harness, patchSession.ID, attributeNeedle("id", "msg_say_01"))
		})

		mustSendNetworkCLI(t, ctx, harness, []string{
			"--session",
			patchSession.ID,
			"--channel",
			"builders",
			"--kind",
			"say",
			"--surface",
			"direct",
			"--direct",
			patchDirectID,
			"--work",
			patchWorkID,
			"--to",
			opsPeerID,
			"--reply-to",
			"msg_say_01",
			"--trace-id",
			"trace_ops_patch_42",
			"--causation-id",
			"msg_say_01",
			"--id",
			"msg_direct_01",
			"--body",
			`{"text":"I can take the failing migration tests and send back a patch summary.","intent":"handoff","artifacts":[]}`,
		})

		waitForRuntimeCondition(t, "direct delivery", 10*time.Second, func() bool {
			audit, err := harness.NetworkAuditSnapshot()
			if err != nil {
				return false
			}
			return validateNetworkAuditEntry(audit, networkAuditExpectation{
				MessageID: "msg_direct_01",
				Direction: "delivered",
				Kind:      "say",
			}) == nil && sessionTranscriptHasNeedle(ctx, harness, opsSession.ID, attributeNeedle("id", "msg_direct_01"))
		})

		mustSendNetworkCLI(t, ctx, harness, []string{
			"--session",
			opsSession.ID,
			"--channel",
			"builders",
			"--kind",
			"receipt",
			"--surface",
			"direct",
			"--direct",
			patchDirectID,
			"--work",
			patchWorkID,
			"--to",
			patchPeerID,
			"--reply-to",
			"msg_direct_01",
			"--trace-id",
			"trace_ops_patch_42",
			"--causation-id",
			"msg_direct_01",
			"--id",
			"msg_receipt_01",
			"--body",
			`{"for_id":"msg_direct_01","status":"accepted","detail":"Proceed and report progress with trace messages."}`,
		})

		waitForRuntimeCondition(t, "receipt delivery", 10*time.Second, func() bool {
			audit, err := harness.NetworkAuditSnapshot()
			if err != nil {
				return false
			}
			return validateNetworkAuditEntry(audit, networkAuditExpectation{
				MessageID: "msg_receipt_01",
				Direction: "delivered",
				Kind:      "receipt",
			}) == nil && sessionTranscriptHasNeedle(ctx, harness, patchSession.ID, attributeNeedle("id", "msg_receipt_01"))
		})

		mustSendNetworkCLI(t, ctx, harness, []string{
			"--session",
			patchSession.ID,
			"--channel",
			"builders",
			"--kind",
			"trace",
			"--surface",
			"direct",
			"--direct",
			patchDirectID,
			"--work",
			patchWorkID,
			"--to",
			opsPeerID,
			"--reply-to",
			"msg_receipt_01",
			"--trace-id",
			"trace_ops_patch_42",
			"--causation-id",
			"msg_receipt_01",
			"--id",
			"msg_trace_02",
			"--body",
			`{"state":"completed","message":"Patch prepared and local tests now pass.","result":{"summary":"Fixed migration assertion mismatch in sessiondb tests."},"artifact_refs":[]}`,
		})

		waitForRuntimeCondition(t, "trace delivery", 10*time.Second, func() bool {
			audit, err := harness.NetworkAuditSnapshot()
			if err != nil {
				return false
			}
			return validateNetworkAuditEntry(audit, networkAuditExpectation{
				MessageID: "msg_trace_02",
				Direction: "delivered",
				Kind:      "trace",
			}) == nil && sessionTranscriptHasNeedle(ctx, harness, opsSession.ID, attributeNeedle("id", "msg_trace_02"))
		})

		mustSendNetworkCLI(t, ctx, harness, []string{
			"--session",
			patchSession.ID,
			"--channel",
			"builders",
			"--kind",
			"say",
			"--surface",
			"thread",
			"--thread",
			buildersThreadID,
			"--reply-to",
			"msg_trace_02",
			"--trace-id",
			"trace_ops_patch_42",
			"--causation-id",
			"msg_trace_02",
			"--id",
			"msg_summary_01",
			"--body",
			`{"text":"Summary: patch prepared and migration assertions now pass locally.","intent":"summarize-back","artifacts":[]}`,
		})

		summarySurface := "thread"
		summaryThreadID := buildersThreadID
		waitForRuntimeCondition(t, "summary back to public thread", 10*time.Second, func() bool {
			audit, err := harness.NetworkAuditSnapshot()
			if err != nil {
				return false
			}
			return validateNetworkAuditEntry(audit, networkAuditExpectation{
				MessageID: "msg_summary_01",
				Direction: "delivered",
				Kind:      "say",
				Surface:   &summarySurface,
				ThreadID:  &summaryThreadID,
			}) == nil &&
				channelHasMessageID(ctx, harness, "builders", "msg_summary_01") &&
				sessionTranscriptHasNeedle(ctx, harness, opsSession.ID, attributeNeedle("id", "msg_summary_01"))
		})

		postTerminalDirectArgs := []string{
			"network",
			"--workspace",
			harness.WorkspaceID,
			"send",
			"--session",
			patchSession.ID,
			"--channel",
			"builders",
			"--kind",
			"say",
			"--surface",
			"direct",
			"--direct",
			patchDirectID,
			"--work",
			patchWorkID,
			"--to",
			opsPeerID,
			"--reply-to",
			"msg_say_01",
			"--trace-id",
			"trace_ops_patch_42",
			"--causation-id",
			"msg_say_01",
			"--id",
			"msg_direct_after_closed",
			"--body",
			`{"text":"I can take the failing migration tests and send back a patch summary.","intent":"handoff","artifacts":[]}`,
			"-o",
			"json",
		}
		_, stderr, err := harness.CLI.Run(ctx, postTerminalDirectArgs...)
		if err == nil {
			t.Fatalf("CLI %v error = nil, want work-closed rejection", postTerminalDirectArgs)
		}
		if !strings.Contains(stderr, "work closed") || !strings.Contains(stderr, patchWorkID) {
			t.Fatalf("CLI %v stderr = %q, want work-closed details", postTerminalDirectArgs, stderr)
		}

		status := mustHTTPNetworkStatus(t, ctx, harness)
		if !status.Enabled || status.Status != "running" {
			t.Fatalf("HTTP network status = %#v, want enabled running", status)
		}
		if status.LocalPeers != 2 {
			t.Fatalf("HTTP network local_peers = %d, want %d", status.LocalPeers, 2)
		}
		if status.MessagesDelivered < 3 {
			t.Fatalf("HTTP network messages_delivered = %d, want >= 3", status.MessagesDelivered)
		}

		peers = mustHTTPNetworkPeers(t, ctx, harness, "builders")
		if len(peers) != 2 {
			t.Fatalf("HTTP network peers = %#v, want 2 peers", peers)
		}
		if requirePeerIDForSession(t, peers, opsSession.ID) != opsPeerID {
			t.Fatalf("HTTP network peers missing ops peer %q", opsPeerID)
		}
		if requirePeerIDForSession(t, peers, patchSession.ID) != patchPeerID {
			t.Fatalf("HTTP network peers missing patch peer %q", patchPeerID)
		}

		channels := mustHTTPNetworkChannels(t, ctx, harness)
		channel, ok := findChannelPayload(channels, "builders")
		if !ok {
			t.Fatalf("HTTP network channels = %#v, want builders entry", channels)
		}
		if channel.PeerCount != 2 || channel.SessionCount != 2 {
			t.Fatalf("HTTP builders channel = %#v, want peer_count=2 session_count=2", channel)
		}
		if channel.MessageCount < 1 {
			t.Fatalf("HTTP builders channel message_count = %d, want >= 1", channel.MessageCount)
		}

		channelDetail = mustHTTPNetworkChannel(t, ctx, harness, "builders")
		if channelDetail.Channel != "builders" || channelDetail.PeerCount != 2 || len(channelDetail.Sessions) != 2 {
			t.Fatalf("HTTP channel detail = %#v, want builders with 2 peers and 2 sessions", channelDetail)
		}

		channelMessages := mustHTTPNetworkChannelMessages(t, ctx, harness, "builders")
		requireChannelMessage(
			t,
			channelMessages,
			"msg_say_01",
			"Who can take the failing migration tests in internal/store/sessiondb?",
		)
		requireChannelMessage(t, channelMessages, "msg_summary_01", "Summary: patch prepared")
		requireNoChannelMessage(t, channelMessages, "msg_direct_01")
		requireNoChannelMessage(t, channelMessages, "msg_trace_02")

		threads := mustHTTPNetworkThreads(t, ctx, harness, "builders")
		requireThreadSummary(t, threads, buildersThreadID, "msg_say_01")
		thread := mustHTTPNetworkThread(t, ctx, harness, "builders", buildersThreadID)
		if thread.ThreadID != buildersThreadID || thread.MessageCount < 2 {
			t.Fatalf("HTTP builders thread = %#v, want summary with >= 2 messages", thread)
		}
		threadMessages := mustHTTPNetworkThreadMessages(t, ctx, harness, "builders", buildersThreadID)
		requireConversationMessage(t, threadMessages, "msg_say_01", "thread", buildersThreadID, "")
		requireConversationMessage(t, threadMessages, "msg_summary_01", "thread", buildersThreadID, "")

		directs := mustHTTPNetworkDirectRooms(t, ctx, harness, "builders")
		requireDirectRoomSummary(t, directs, patchDirectID)
		direct := mustHTTPNetworkDirectRoom(t, ctx, harness, "builders", patchDirectID)
		if direct.DirectID != patchDirectID || direct.MessageCount < 3 {
			t.Fatalf("HTTP builders direct = %#v, want direct room with >= 3 messages", direct)
		}
		directMessages := mustHTTPNetworkDirectRoomMessages(t, ctx, harness, "builders", patchDirectID)
		requireConversationMessage(t, directMessages, "msg_direct_01", "direct", "", patchDirectID)
		requireConversationMessage(t, directMessages, "msg_trace_02", "direct", "", patchDirectID)

		work := mustHTTPNetworkWork(t, ctx, harness, patchWorkID)
		if work.WorkID != patchWorkID || work.Surface != "direct" || work.DirectID != patchDirectID {
			t.Fatalf("HTTP network work = %#v, want direct work %q in %q", work, patchWorkID, patchDirectID)
		}

		opsTranscript := mustSessionTranscript(t, ctx, harness, opsSession.ID)
		patchTranscript := mustSessionTranscript(t, ctx, harness, patchSession.ID)
		audit := mustNetworkAuditSnapshot(t, harness)

		if err := validateNetworkCorrelationSurfaces(opsTranscript.Messages, audit, networkCorrelationExpectation{
			MessageID:       "msg_direct_01",
			Kind:            "say",
			Surface:         "direct",
			DirectID:        patchDirectID,
			WorkID:          patchWorkID,
			ReplyTo:         "msg_say_01",
			TraceID:         "trace_ops_patch_42",
			CausationID:     "msg_say_01",
			Trust:           "untrusted",
			AuditDirections: []string{"delivered"},
		}); err != nil {
			t.Fatalf("validateNetworkCorrelationSurfaces(direct) error = %v", err)
		}
		if err := validateNetworkCorrelationSurfaces(patchTranscript.Messages, audit, networkCorrelationExpectation{
			MessageID:       "msg_receipt_01",
			Kind:            "receipt",
			Surface:         "direct",
			DirectID:        patchDirectID,
			WorkID:          patchWorkID,
			ReplyTo:         "msg_direct_01",
			TraceID:         "trace_ops_patch_42",
			CausationID:     "msg_direct_01",
			Trust:           "untrusted",
			AuditDirections: []string{"delivered"},
		}); err != nil {
			t.Fatalf("validateNetworkCorrelationSurfaces(receipt) error = %v", err)
		}
		if err := validateNetworkCorrelationSurfaces(opsTranscript.Messages, audit, networkCorrelationExpectation{
			MessageID:       "msg_trace_02",
			Kind:            "trace",
			Surface:         "direct",
			DirectID:        patchDirectID,
			WorkID:          patchWorkID,
			ReplyTo:         "msg_receipt_01",
			TraceID:         "trace_ops_patch_42",
			CausationID:     "msg_receipt_01",
			Trust:           "untrusted",
			AuditDirections: []string{"delivered"},
		}); err != nil {
			t.Fatalf("validateNetworkCorrelationSurfaces(trace) error = %v", err)
		}
		if err := validateNetworkCorrelationSurfaces(opsTranscript.Messages, audit, networkCorrelationExpectation{
			MessageID:       "msg_summary_01",
			Kind:            "say",
			Surface:         "thread",
			ThreadID:        buildersThreadID,
			ReplyTo:         "msg_trace_02",
			TraceID:         "trace_ops_patch_42",
			CausationID:     "msg_trace_02",
			Trust:           "untrusted",
			AuditDirections: []string{"delivered"},
		}); err != nil {
			t.Fatalf("validateNetworkCorrelationSurfaces(summary) error = %v", err)
		}
		assertCLINetworkParity(t, ctx, harness, status, peers, channel, channelDetail)
	})
}

func TestDaemonE2ENetworkWhoisAndCapabilityExchange(t *testing.T) {
	t.Run("Should complete whois and capability exchange", func(t *testing.T) {
		acpmock.RequireDriver(t)

		fixturePath := mockFixturePath(t, "network_collaboration_fixture.json")
		harness := e2etest.StartRuntimeHarness(t, e2etest.RuntimeHarnessOptions{
			EnableNetwork: true,
			MockAgents: []e2etest.MockAgentSpec{
				{
					FixturePath:  fixturePath,
					FixtureAgent: "release-bot",
					AgentName:    "mock-release-bot",
				},
				{
					FixturePath:  fixturePath,
					FixtureAgent: "capability-curator",
					AgentName:    "mock-capability-curator",
				},
			},
		})

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		channelDetail := mustCreateNetworkChannel(
			t,
			ctx,
			harness,
			"capabilities",
			"mock-release-bot",
			"mock-capability-curator",
		)
		releaseSession := requireChannelSession(t, channelDetail, "mock-release-bot")
		curatorSession := requireChannelSession(t, channelDetail, "mock-capability-curator")

		regRelease, ok := harness.MockAgentRegistration("mock-release-bot")
		if !ok {
			t.Fatal("MockAgentRegistration(mock-release-bot) = missing, want present")
		}
		regCurator, ok := harness.MockAgentRegistration("mock-capability-curator")
		if !ok {
			t.Fatal("MockAgentRegistration(mock-capability-curator) = missing, want present")
		}

		registerNetworkScenarioArtifacts(
			t,
			harness,
			"capabilities",
			[]aghcontract.SessionPayload{releaseSession, curatorSession},
			[]acpmock.Registration{regRelease, regCurator},
		)

		peers := waitForChannelPeerCount(t, ctx, harness, "capabilities", 2)
		releasePeerID := requirePeerIDForSession(t, peers, releaseSession.ID)
		curatorPeerID := requirePeerIDForSession(t, peers, curatorSession.ID)
		if releasePeerID == curatorPeerID {
			t.Fatalf("peer IDs = %q and %q, want distinct values", releasePeerID, curatorPeerID)
		}
		capabilitiesThreadID := "thread_capabilities_main"
		capabilityThreadWorkID := "work_capability_catalog_7"
		capabilityDirectWorkID := "work_capability_apply_7"
		capabilityDirectID := mustHTTPResolveNetworkDirectRoom(
			t,
			ctx,
			harness,
			"capabilities",
			releaseSession.ID,
			curatorPeerID,
		)

		capabilityBody := mustCapabilityBodyString(t, aghconfig.CapabilityDef{
			ID:                "fix-go-migration-tests",
			Summary:           "Repair failing Go migration test assertions and rerun the package verification lane.",
			Outcome:           "A patched migration test with passing package verification output.",
			Version:           "1.0.0",
			ContextNeeded:     []string{"package path", "failing test output"},
			ArtifactsExpected: []string{"updated assertion", "passing package tests"},
			ExecutionOutline: []string{
				"Re-run the failing migration tests.",
				"Compare the expected schema with the normalized audit rows.",
				"Update the migration assertion and rerun the package tests.",
			},
			Requirements: []string{"go-test", "sessiondb-fixtures"},
		})

		mustSendNetworkCLI(t, ctx, harness, []string{
			"--session",
			releaseSession.ID,
			"--channel",
			"capabilities",
			"--kind",
			"say",
			"--surface",
			"thread",
			"--thread",
			capabilitiesThreadID,
			"--id",
			"msg_capability_say_01",
			"--trace-id",
			"trace_capability_apply_7",
			"--body",
			`{"text":"Does anyone have a reusable migration test repair capability?","intent":"request-help","artifacts":[]}`,
		})

		waitForRuntimeCondition(t, "capability say delivery", 10*time.Second, func() bool {
			return channelHasMessageID(ctx, harness, "capabilities", "msg_capability_say_01") &&
				sessionTranscriptHasNeedle(
					ctx,
					harness,
					curatorSession.ID,
					attributeNeedle("id", "msg_capability_say_01"),
				)
		})

		mustSendNetworkCLI(t, ctx, harness, []string{
			"--session", releaseSession.ID,
			"--channel", "capabilities",
			"--kind", "whois",
			"--to", curatorPeerID,
			"--id", "msg_whois_01",
			"--body", `{"type":"request","query":"capability-curator"}`,
		})

		waitForRuntimeCondition(t, "whois response delivery", 10*time.Second, func() bool {
			audit, err := harness.NetworkAuditSnapshot()
			if err != nil {
				return false
			}
			return validateNetworkAuditEntry(audit, networkAuditExpectation{
				MessageID: "msg_whois_01",
				Direction: "sent",
				Kind:      "whois",
			}) == nil && sessionTranscriptHasNeedle(ctx, harness, releaseSession.ID, attributeNeedle("reply-to", "msg_whois_01"))
		})

		mustSendNetworkCLI(t, ctx, harness, []string{
			"--session", curatorSession.ID,
			"--channel", "capabilities",
			"--kind", "capability",
			"--surface", "thread",
			"--thread", capabilitiesThreadID,
			"--work", capabilityThreadWorkID,
			"--to", releasePeerID,
			"--id", "msg_capability_01",
			"--trace-id", "trace_capability_apply_7",
			"--body", capabilityBody,
		})

		waitForRuntimeCondition(t, "capability delivery", 10*time.Second, func() bool {
			audit, err := harness.NetworkAuditSnapshot()
			if err != nil {
				return false
			}
			return validateNetworkAuditEntry(audit, networkAuditExpectation{
				MessageID: "msg_capability_01",
				Direction: "delivered",
				Kind:      "capability",
			}) == nil && sessionTranscriptHasNeedle(ctx, harness, releaseSession.ID, attributeNeedle("id", "msg_capability_01"))
		})

		mustSendNetworkCLI(t, ctx, harness, []string{
			"--session",
			releaseSession.ID,
			"--channel",
			"capabilities",
			"--kind",
			"say",
			"--surface",
			"direct",
			"--direct",
			capabilityDirectID,
			"--work",
			capabilityDirectWorkID,
			"--to",
			curatorPeerID,
			"--reply-to",
			"msg_capability_01",
			"--trace-id",
			"trace_capability_apply_7",
			"--causation-id",
			"msg_capability_01",
			"--id",
			"msg_direct_20",
			"--body",
			`{"text":"Can you adapt this capability to a failure in internal/store/sessiondb?","intent":"request-guidance","artifacts":[]}`,
		})

		waitForRuntimeCondition(t, "capability direct delivery", 10*time.Second, func() bool {
			audit, err := harness.NetworkAuditSnapshot()
			if err != nil {
				return false
			}
			return validateNetworkAuditEntry(audit, networkAuditExpectation{
				MessageID: "msg_direct_20",
				Direction: "delivered",
				Kind:      "say",
			}) == nil && sessionTranscriptHasNeedle(ctx, harness, curatorSession.ID, attributeNeedle("id", "msg_direct_20"))
		})

		mustSendNetworkCLI(t, ctx, harness, []string{
			"--session",
			curatorSession.ID,
			"--channel",
			"capabilities",
			"--kind",
			"trace",
			"--surface",
			"direct",
			"--direct",
			capabilityDirectID,
			"--work",
			capabilityDirectWorkID,
			"--to",
			releasePeerID,
			"--reply-to",
			"msg_direct_20",
			"--trace-id",
			"trace_capability_apply_7",
			"--causation-id",
			"msg_direct_20",
			"--id",
			"msg_trace_21",
			"--body",
			`{"state":"needs_input","message":"Send the exact package path and failing test output so I can tailor the capability.","result":{"capability_id":"fix-go-migration-tests"},"artifact_refs":[]}`,
		})

		waitForRuntimeCondition(t, "capability trace delivery", 10*time.Second, func() bool {
			audit, err := harness.NetworkAuditSnapshot()
			if err != nil {
				return false
			}
			return validateNetworkAuditEntry(audit, networkAuditExpectation{
				MessageID: "msg_trace_21",
				Direction: "delivered",
				Kind:      "trace",
			}) == nil && sessionTranscriptHasNeedle(ctx, harness, releaseSession.ID, attributeNeedle("id", "msg_trace_21"))
		})

		status := mustHTTPNetworkStatus(t, ctx, harness)
		if !status.Enabled || status.Status != "running" {
			t.Fatalf("HTTP network status = %#v, want enabled running", status)
		}
		if status.LocalPeers != 2 {
			t.Fatalf("HTTP network local_peers = %d, want %d", status.LocalPeers, 2)
		}

		peers = mustHTTPNetworkPeers(t, ctx, harness, "capabilities")
		if len(peers) != 2 {
			t.Fatalf("HTTP network peers = %#v, want 2 peers", peers)
		}
		if requirePeerIDForSession(t, peers, releaseSession.ID) != releasePeerID {
			t.Fatalf("HTTP network peers missing release peer %q", releasePeerID)
		}
		if requirePeerIDForSession(t, peers, curatorSession.ID) != curatorPeerID {
			t.Fatalf("HTTP network peers missing curator peer %q", curatorPeerID)
		}

		channels := mustHTTPNetworkChannels(t, ctx, harness)
		channel, ok := findChannelPayload(channels, "capabilities")
		if !ok {
			t.Fatalf("HTTP network channels = %#v, want capabilities entry", channels)
		}
		if channel.PeerCount != 2 || channel.SessionCount != 2 {
			t.Fatalf("HTTP capabilities channel = %#v, want peer_count=2 session_count=2", channel)
		}
		if channel.MessageCount < 1 {
			t.Fatalf("HTTP capabilities channel message_count = %d, want >= 1", channel.MessageCount)
		}

		channelDetail = mustHTTPNetworkChannel(t, ctx, harness, "capabilities")
		if channelDetail.Channel != "capabilities" || channelDetail.PeerCount != 2 || len(channelDetail.Sessions) != 2 {
			t.Fatalf("HTTP channel detail = %#v, want capabilities with 2 peers and 2 sessions", channelDetail)
		}

		channelMessages := mustHTTPNetworkChannelMessages(t, ctx, harness, "capabilities")
		requireChannelMessage(
			t,
			channelMessages,
			"msg_capability_say_01",
			"Does anyone have a reusable migration test repair capability?",
		)

		releaseTranscript := mustSessionTranscript(t, ctx, harness, releaseSession.ID)
		curatorTranscript := mustSessionTranscript(t, ctx, harness, curatorSession.ID)
		audit := mustNetworkAuditSnapshot(t, harness)

		releaseContent := joinTranscriptContent(releaseTranscript.Messages)
		for _, needle := range []string{
			attributeNeedle("kind", "whois"),
			attributeNeedle("reply-to", "msg_whois_01"),
			attributeNeedle("id", "msg_capability_01"),
			attributeNeedle("kind", "capability"),
			attributeNeedle("id", "msg_trace_21"),
			attributeNeedle("trace-id", "trace_capability_apply_7"),
		} {
			if !strings.Contains(releaseContent, needle) {
				t.Fatalf("release transcript missing %q in %s", needle, releaseContent)
			}
		}
		if err := validateNetworkCorrelationSurfaces(curatorTranscript.Messages, audit, networkCorrelationExpectation{
			MessageID:       "msg_direct_20",
			Kind:            "say",
			Surface:         "direct",
			DirectID:        capabilityDirectID,
			WorkID:          capabilityDirectWorkID,
			ReplyTo:         "msg_capability_01",
			TraceID:         "trace_capability_apply_7",
			CausationID:     "msg_capability_01",
			Trust:           "untrusted",
			AuditDirections: []string{"delivered"},
		}); err != nil {
			t.Fatalf("validateNetworkCorrelationSurfaces(capability direct) error = %v", err)
		}
		if err := validateNetworkCorrelationSurfaces(releaseTranscript.Messages, audit, networkCorrelationExpectation{
			MessageID:       "msg_trace_21",
			Kind:            "trace",
			Surface:         "direct",
			DirectID:        capabilityDirectID,
			WorkID:          capabilityDirectWorkID,
			ReplyTo:         "msg_direct_20",
			TraceID:         "trace_capability_apply_7",
			CausationID:     "msg_direct_20",
			Trust:           "untrusted",
			AuditDirections: []string{"delivered"},
		}); err != nil {
			t.Fatalf("validateNetworkCorrelationSurfaces(capability trace) error = %v", err)
		}

		assertCLINetworkParity(t, ctx, harness, status, peers, channel, channelDetail)
	})
}

func mustCreateNetworkChannel(
	t testing.TB,
	ctx context.Context,
	harness *e2etest.RuntimeHarness,
	channel string,
	agentNames ...string,
) aghcontract.NetworkChannelDetailPayload {
	t.Helper()

	detail, err := harness.CreateNetworkChannel(ctx, aghcontract.CreateNetworkChannelRequest{
		Channel:     channel,
		Purpose:     "Release validation channel for " + channel,
		WorkspaceID: harness.WorkspaceID,
		AgentNames:  append([]string(nil), agentNames...),
	})
	if err != nil {
		t.Fatalf("CreateNetworkChannel(%q) error = %v", channel, err)
	}
	return detail
}

func requireChannelSession(
	t testing.TB,
	detail aghcontract.NetworkChannelDetailPayload,
	agentName string,
) aghcontract.SessionPayload {
	t.Helper()

	target := strings.TrimSpace(agentName)
	for _, session := range detail.Sessions {
		if strings.TrimSpace(session.AgentName) == target {
			return session
		}
	}
	t.Fatalf("channel sessions = %#v, want agent %q", detail.Sessions, agentName)
	return aghcontract.SessionPayload{}
}

func waitForChannelPeerCount(
	t testing.TB,
	ctx context.Context,
	harness *e2etest.RuntimeHarness,
	channel string,
	want int,
) []aghcontract.NetworkPeerPayload {
	t.Helper()

	var peers []aghcontract.NetworkPeerPayload
	waitForRuntimeCondition(t, "network peers for "+channel, 10*time.Second, func() bool {
		var err error
		peers, err = mustHTTPNetworkPeersMaybe(ctx, harness, channel)
		return err == nil && len(peers) == want
	})
	return peers
}

func requirePeerIDForSession(
	t testing.TB,
	peers []aghcontract.NetworkPeerPayload,
	sessionID string,
) string {
	t.Helper()

	target := strings.TrimSpace(sessionID)
	for _, peer := range peers {
		if peer.SessionID != nil && strings.TrimSpace(*peer.SessionID) == target {
			return strings.TrimSpace(peer.PeerID)
		}
	}
	t.Fatalf("network peers = %#v, want session %q", peers, sessionID)
	return ""
}

func requireChannelMessage(
	t testing.TB,
	messages []aghcontract.NetworkConversationMessagePayload,
	messageID string,
	text string,
) {
	t.Helper()

	for _, message := range messages {
		if strings.TrimSpace(message.MessageID) != strings.TrimSpace(messageID) {
			continue
		}
		if !strings.Contains(message.Text, text) {
			t.Fatalf("channel message = %#v, want text containing %q", message, text)
		}
		return
	}
	t.Fatalf("channel messages = %#v, want message_id %q", messages, messageID)
}

func requireNoChannelMessage(
	t testing.TB,
	messages []aghcontract.NetworkConversationMessagePayload,
	messageID string,
) {
	t.Helper()

	target := strings.TrimSpace(messageID)
	for _, message := range messages {
		if strings.TrimSpace(message.MessageID) == target {
			t.Fatalf("channel messages = %#v, want no message_id %q", messages, messageID)
		}
	}
}

func requireThreadSummary(
	t testing.TB,
	threads []aghcontract.NetworkThreadSummaryPayload,
	threadID string,
	rootMessageID string,
) {
	t.Helper()

	for _, thread := range threads {
		if strings.TrimSpace(thread.ThreadID) != strings.TrimSpace(threadID) {
			continue
		}
		if strings.TrimSpace(thread.RootMessageID) != strings.TrimSpace(rootMessageID) {
			t.Fatalf("thread summary = %#v, want root_message_id %q", thread, rootMessageID)
		}
		return
	}
	t.Fatalf("threads = %#v, want thread_id %q", threads, threadID)
}

func requireDirectRoomSummary(
	t testing.TB,
	directs []aghcontract.NetworkDirectRoomPayload,
	directID string,
) {
	t.Helper()

	for _, direct := range directs {
		if strings.TrimSpace(direct.DirectID) == strings.TrimSpace(directID) {
			return
		}
	}
	t.Fatalf("direct rooms = %#v, want direct_id %q", directs, directID)
}

func requireConversationMessage(
	t testing.TB,
	messages []aghcontract.NetworkConversationMessagePayload,
	messageID string,
	surface string,
	threadID string,
	directID string,
) {
	t.Helper()

	for _, message := range messages {
		if strings.TrimSpace(message.MessageID) != strings.TrimSpace(messageID) {
			continue
		}
		if strings.TrimSpace(message.Surface) != strings.TrimSpace(surface) {
			t.Fatalf("message = %#v, want surface %q", message, surface)
		}
		if strings.TrimSpace(threadID) != "" && strings.TrimSpace(message.ThreadID) != strings.TrimSpace(threadID) {
			t.Fatalf("message = %#v, want thread_id %q", message, threadID)
		}
		if strings.TrimSpace(directID) != "" && strings.TrimSpace(message.DirectID) != strings.TrimSpace(directID) {
			t.Fatalf("message = %#v, want direct_id %q", message, directID)
		}
		return
	}
	t.Fatalf("conversation messages = %#v, want message_id %q", messages, messageID)
}

func mustSendNetworkCLI(
	t testing.TB,
	ctx context.Context,
	harness *e2etest.RuntimeHarness,
	args []string,
) aghcontract.NetworkSendPayload {
	t.Helper()

	var payload aghcontract.NetworkSendPayload
	fullArgs := append(
		[]string{"network", "--workspace", harness.WorkspaceID, "send"},
		append(args, "-o", "json")...,
	)
	if err := harness.CLI.RunJSON(ctx, &payload, fullArgs...); err != nil {
		t.Fatalf("CLI %v error = %v", fullArgs, err)
	}
	return payload
}

func mustCapabilityBodyString(t testing.TB, def aghconfig.CapabilityDef) string {
	t.Helper()

	digest, err := aghconfig.CanonicalCapabilityDigest(def)
	if err != nil {
		t.Fatalf("CanonicalCapabilityDigest(%q) error = %v", def.ID, err)
	}

	raw, err := json.Marshal(map[string]any{
		"capability": map[string]any{
			"id":                 def.ID,
			"summary":            def.Summary,
			"outcome":            def.Outcome,
			"version":            def.Version,
			"digest":             digest,
			"context_needed":     append([]string(nil), def.ContextNeeded...),
			"artifacts_expected": append([]string(nil), def.ArtifactsExpected...),
			"execution_outline":  append([]string(nil), def.ExecutionOutline...),
			"constraints":        append([]string(nil), def.Constraints...),
			"examples":           append([]string(nil), def.Examples...),
			"requirements":       append([]string(nil), def.Requirements...),
		},
	})
	if err != nil {
		t.Fatalf("json.Marshal(capability body %q) error = %v", def.ID, err)
	}
	return string(raw)
}

func assertCLINetworkParity(
	t testing.TB,
	ctx context.Context,
	harness *e2etest.RuntimeHarness,
	httpStatus aghcontract.NetworkStatusPayload,
	httpPeers []aghcontract.NetworkPeerPayload,
	httpChannel aghcontract.NetworkChannelPayload,
	httpDetail aghcontract.NetworkChannelDetailPayload,
) {
	t.Helper()

	var cliStatus aghcontract.NetworkStatusPayload
	if err := harness.CLI.RunJSON(
		ctx,
		&cliStatus,
		"network",
		"--workspace",
		harness.WorkspaceID,
		"status",
		"-o",
		"json",
	); err != nil {
		t.Fatalf("CLI network status error = %v", err)
	}
	if cliStatus.Enabled != httpStatus.Enabled ||
		cliStatus.Status != httpStatus.Status ||
		cliStatus.LocalPeers != httpStatus.LocalPeers ||
		cliStatus.Channels != httpStatus.Channels {
		t.Fatalf("CLI network status = %#v, want parity with HTTP %#v", cliStatus, httpStatus)
	}

	var cliPeers []aghcontract.NetworkPeerPayload
	if err := harness.CLI.RunJSON(
		ctx,
		&cliPeers,
		"network",
		"--workspace",
		harness.WorkspaceID,
		"peers",
		httpChannel.Channel,
		"-o",
		"json",
	); err != nil {
		t.Fatalf("CLI network peers error = %v", err)
	}
	assertNetworkPeerOrdering(t, "HTTP peer list", httpPeers)
	assertNetworkPeerOrdering(t, "HTTP channel detail peers", httpDetail.Peers)
	assertMatchingPeerOrder(t, "HTTP peer list", httpPeers, "HTTP channel detail peers", httpDetail.Peers)
	if len(cliPeers) != len(httpPeers) {
		t.Fatalf("CLI network peers = %#v, want %d peers", cliPeers, len(httpPeers))
	}
	assertNetworkPeerOrdering(t, "CLI peer list", cliPeers)
	assertMatchingPeerOrder(t, "HTTP peer list", httpPeers, "CLI peer list", cliPeers)
	for _, peer := range httpPeers {
		if requirePeerIDForSession(t, cliPeers, derefString(peer.SessionID)) != strings.TrimSpace(peer.PeerID) {
			t.Fatalf("CLI peers = %#v, want peer %q for session %v", cliPeers, peer.PeerID, peer.SessionID)
		}
	}

	var cliChannels []aghcontract.NetworkChannelPayload
	if err := harness.CLI.RunJSON(
		ctx,
		&cliChannels,
		"network",
		"--workspace",
		harness.WorkspaceID,
		"channels",
		"-o",
		"json",
	); err != nil {
		t.Fatalf("CLI network channels error = %v", err)
	}
	cliChannel, ok := findChannelPayload(cliChannels, httpChannel.Channel)
	if !ok {
		t.Fatalf("CLI network channels = %#v, want channel %q", cliChannels, httpChannel.Channel)
	}
	if cliChannel.PeerCount != httpChannel.PeerCount ||
		cliChannel.SessionCount != httpChannel.SessionCount ||
		cliChannel.MessageCount != httpChannel.MessageCount {
		t.Fatalf("CLI channel = %#v, want parity with HTTP %#v", cliChannel, httpChannel)
	}
}

func assertNetworkPeerOrdering(
	t testing.TB,
	label string,
	peers []aghcontract.NetworkPeerPayload,
) {
	t.Helper()

	for index := 1; index < len(peers); index++ {
		left := peers[index-1]
		right := peers[index]
		if networkPeerShouldSortBefore(left, right) {
			continue
		}
		t.Fatalf("%s ordering mismatch at index %d: %#v should not come before %#v", label, index, left, right)
	}
}

func assertMatchingPeerOrder(
	t testing.TB,
	leftLabel string,
	left []aghcontract.NetworkPeerPayload,
	rightLabel string,
	right []aghcontract.NetworkPeerPayload,
) {
	t.Helper()

	if len(left) != len(right) {
		t.Fatalf("%s len = %d, %s len = %d, want equal", leftLabel, len(left), rightLabel, len(right))
	}
	for index := range left {
		if got, want := strings.TrimSpace(right[index].PeerID), strings.TrimSpace(left[index].PeerID); got != want {
			t.Fatalf("%s[%d].peer_id = %q, want %q from %s", rightLabel, index, got, want, leftLabel)
		}
	}
}

func networkPeerShouldSortBefore(
	left aghcontract.NetworkPeerPayload,
	right aghcontract.NetworkPeerPayload,
) bool {
	if left.Local != right.Local {
		return left.Local
	}

	leftRecency := networkPeerEffectiveRecency(left)
	rightRecency := networkPeerEffectiveRecency(right)
	switch {
	case leftRecency != nil && rightRecency != nil && !leftRecency.Equal(*rightRecency):
		return leftRecency.After(*rightRecency)
	case leftRecency != nil && rightRecency == nil:
		return true
	case leftRecency == nil && rightRecency != nil:
		return false
	}

	leftName := networkPeerSortName(left)
	rightName := networkPeerSortName(right)
	if leftName != rightName {
		return leftName < rightName
	}
	if strings.TrimSpace(left.PeerID) != strings.TrimSpace(right.PeerID) {
		return strings.TrimSpace(left.PeerID) < strings.TrimSpace(right.PeerID)
	}
	return strings.TrimSpace(left.Channel) <= strings.TrimSpace(right.Channel)
}

func networkPeerEffectiveRecency(peer aghcontract.NetworkPeerPayload) *time.Time {
	if peer.LastSeen != nil {
		return peer.LastSeen
	}
	return peer.JoinedAt
}

func networkPeerSortName(peer aghcontract.NetworkPeerPayload) string {
	if value := strings.TrimSpace(peer.DisplayName); value != "" {
		return value
	}
	return strings.TrimSpace(peer.PeerID)
}

func findChannelPayload(
	channels []aghcontract.NetworkChannelPayload,
	channel string,
) (aghcontract.NetworkChannelPayload, bool) {
	target := strings.TrimSpace(channel)
	for _, item := range channels {
		if strings.TrimSpace(item.Channel) == target {
			return item, true
		}
	}
	return aghcontract.NetworkChannelPayload{}, false
}

func mustHTTPNetworkStatus(
	t testing.TB,
	ctx context.Context,
	harness *e2etest.RuntimeHarness,
) aghcontract.NetworkStatusPayload {
	t.Helper()

	var response aghcontract.NetworkStatusResponse
	if err := harness.HTTPJSON(ctx, http.MethodGet, "/api/network/status", nil, &response); err != nil {
		t.Fatalf("HTTPJSON(/api/network/status) error = %v", err)
	}
	return response.Network
}

func httpWorkspaceNetworkPath(harness *e2etest.RuntimeHarness, suffix string) string {
	if !strings.HasPrefix(suffix, "/") {
		suffix = "/" + suffix
	}
	return "/api/workspaces/" + url.PathEscape(strings.TrimSpace(harness.WorkspaceID)) + "/network" + suffix
}

func mustHTTPNetworkPeers(
	t testing.TB,
	ctx context.Context,
	harness *e2etest.RuntimeHarness,
	channel string,
) []aghcontract.NetworkPeerPayload {
	t.Helper()

	peers, err := mustHTTPNetworkPeersMaybe(ctx, harness, channel)
	if err != nil {
		t.Fatalf("HTTPJSON(%s) error = %v", httpWorkspaceNetworkPath(harness, "/peers"), err)
	}
	return peers
}

func mustHTTPNetworkPeersMaybe(
	ctx context.Context,
	harness *e2etest.RuntimeHarness,
	channel string,
) ([]aghcontract.NetworkPeerPayload, error) {
	var response aghcontract.NetworkPeersResponse
	path := httpWorkspaceNetworkPath(harness, "/peers")
	if trimmed := strings.TrimSpace(channel); trimmed != "" {
		path += "?channel=" + url.QueryEscape(trimmed)
	}
	if err := harness.HTTPJSON(ctx, http.MethodGet, path, nil, &response); err != nil {
		return nil, err
	}
	return response.Peers, nil
}

func mustHTTPNetworkChannels(
	t testing.TB,
	ctx context.Context,
	harness *e2etest.RuntimeHarness,
) []aghcontract.NetworkChannelPayload {
	t.Helper()

	var response aghcontract.NetworkChannelsResponse
	path := httpWorkspaceNetworkPath(harness, "/channels")
	if err := harness.HTTPJSON(ctx, http.MethodGet, path, nil, &response); err != nil {
		t.Fatalf("HTTPJSON(%s) error = %v", path, err)
	}
	return response.Channels
}

func mustHTTPNetworkChannel(
	t testing.TB,
	ctx context.Context,
	harness *e2etest.RuntimeHarness,
	channel string,
) aghcontract.NetworkChannelDetailPayload {
	t.Helper()

	var response aghcontract.NetworkChannelResponse
	escapedChannel := url.PathEscape(channel)
	path := httpWorkspaceNetworkPath(harness, "/channels/"+escapedChannel)
	if err := harness.HTTPJSON(ctx, http.MethodGet, path, nil, &response); err != nil {
		t.Fatalf("HTTPJSON(%s) error = %v", path, err)
	}
	return response.Channel
}

func mustHTTPNetworkChannelMessages(
	t testing.TB,
	ctx context.Context,
	harness *e2etest.RuntimeHarness,
	channel string,
) []aghcontract.NetworkConversationMessagePayload {
	t.Helper()

	var threadsResponse aghcontract.NetworkThreadsResponse
	escapedChannel := url.PathEscape(channel)
	threadsPath := httpWorkspaceNetworkPath(harness, "/channels/"+escapedChannel+"/threads")
	if err := harness.HTTPJSON(ctx, http.MethodGet, threadsPath, nil, &threadsResponse); err != nil {
		t.Fatalf("HTTPJSON(%s) error = %v", threadsPath, err)
	}
	messages := make([]aghcontract.NetworkConversationMessagePayload, 0)
	for _, thread := range threadsResponse.Threads {
		var response aghcontract.NetworkThreadMessagesResponse
		messagesPath := httpWorkspaceNetworkPath(
			harness,
			"/channels/"+escapedChannel+"/threads/"+url.PathEscape(thread.ThreadID)+"/messages",
		)
		if err := harness.HTTPJSON(ctx, http.MethodGet, messagesPath, nil, &response); err != nil {
			t.Fatalf("HTTPJSON(%s) error = %v", messagesPath, err)
		}
		messages = append(messages, response.Messages...)
	}
	return messages
}

func mustHTTPNetworkThreads(
	t testing.TB,
	ctx context.Context,
	harness *e2etest.RuntimeHarness,
	channel string,
) []aghcontract.NetworkThreadSummaryPayload {
	t.Helper()

	var response aghcontract.NetworkThreadsResponse
	path := httpWorkspaceNetworkPath(harness, "/channels/"+url.PathEscape(channel)+"/threads")
	if err := harness.HTTPJSON(ctx, http.MethodGet, path, nil, &response); err != nil {
		t.Fatalf("HTTPJSON(%s) error = %v", path, err)
	}
	return response.Threads
}

func mustHTTPNetworkThread(
	t testing.TB,
	ctx context.Context,
	harness *e2etest.RuntimeHarness,
	channel string,
	threadID string,
) aghcontract.NetworkThreadSummaryPayload {
	t.Helper()

	var response aghcontract.NetworkThreadResponse
	path := httpWorkspaceNetworkPath(
		harness,
		"/channels/"+url.PathEscape(channel)+"/threads/"+url.PathEscape(threadID),
	)
	if err := harness.HTTPJSON(ctx, http.MethodGet, path, nil, &response); err != nil {
		t.Fatalf("HTTPJSON(%s) error = %v", path, err)
	}
	return response.Thread
}

func mustHTTPNetworkThreadMessages(
	t testing.TB,
	ctx context.Context,
	harness *e2etest.RuntimeHarness,
	channel string,
	threadID string,
) []aghcontract.NetworkConversationMessagePayload {
	t.Helper()

	var response aghcontract.NetworkThreadMessagesResponse
	path := httpWorkspaceNetworkPath(
		harness,
		"/channels/"+url.PathEscape(channel)+"/threads/"+url.PathEscape(threadID)+"/messages",
	)
	if err := harness.HTTPJSON(ctx, http.MethodGet, path, nil, &response); err != nil {
		t.Fatalf("HTTPJSON(%s) error = %v", path, err)
	}
	return response.Messages
}

func mustHTTPNetworkDirectRooms(
	t testing.TB,
	ctx context.Context,
	harness *e2etest.RuntimeHarness,
	channel string,
) []aghcontract.NetworkDirectRoomPayload {
	t.Helper()

	var response aghcontract.NetworkDirectRoomsResponse
	path := httpWorkspaceNetworkPath(harness, "/channels/"+url.PathEscape(channel)+"/directs")
	if err := harness.HTTPJSON(ctx, http.MethodGet, path, nil, &response); err != nil {
		t.Fatalf("HTTPJSON(%s) error = %v", path, err)
	}
	return response.Directs
}

func mustHTTPNetworkDirectRoom(
	t testing.TB,
	ctx context.Context,
	harness *e2etest.RuntimeHarness,
	channel string,
	directID string,
) aghcontract.NetworkDirectRoomPayload {
	t.Helper()

	var response aghcontract.NetworkDirectRoomResponse
	path := httpWorkspaceNetworkPath(
		harness,
		"/channels/"+url.PathEscape(channel)+"/directs/"+url.PathEscape(directID),
	)
	if err := harness.HTTPJSON(ctx, http.MethodGet, path, nil, &response); err != nil {
		t.Fatalf("HTTPJSON(%s) error = %v", path, err)
	}
	return response.Direct
}

func mustHTTPNetworkDirectRoomMessages(
	t testing.TB,
	ctx context.Context,
	harness *e2etest.RuntimeHarness,
	channel string,
	directID string,
) []aghcontract.NetworkConversationMessagePayload {
	t.Helper()

	var response aghcontract.NetworkDirectRoomMessagesResponse
	path := httpWorkspaceNetworkPath(
		harness,
		"/channels/"+url.PathEscape(channel)+"/directs/"+url.PathEscape(directID)+"/messages",
	)
	if err := harness.HTTPJSON(ctx, http.MethodGet, path, nil, &response); err != nil {
		t.Fatalf("HTTPJSON(%s) error = %v", path, err)
	}
	return response.Messages
}

func mustHTTPNetworkWork(
	t testing.TB,
	ctx context.Context,
	harness *e2etest.RuntimeHarness,
	workID string,
) aghcontract.NetworkWorkPayload {
	t.Helper()

	var response aghcontract.NetworkWorkResponse
	path := httpWorkspaceNetworkPath(harness, "/work/"+url.PathEscape(workID))
	if err := harness.HTTPJSON(ctx, http.MethodGet, path, nil, &response); err != nil {
		t.Fatalf("HTTPJSON(%s) error = %v", path, err)
	}
	return response.Work
}

func channelHasMessageID(
	ctx context.Context,
	harness *e2etest.RuntimeHarness,
	channel string,
	messageID string,
) bool {
	var threadsResponse aghcontract.NetworkThreadsResponse
	escapedChannel := url.PathEscape(channel)
	threadsPath := httpWorkspaceNetworkPath(harness, "/channels/"+escapedChannel+"/threads")
	if err := harness.HTTPJSON(ctx, http.MethodGet, threadsPath, nil, &threadsResponse); err != nil {
		return false
	}
	target := strings.TrimSpace(messageID)
	for _, thread := range threadsResponse.Threads {
		var response aghcontract.NetworkThreadMessagesResponse
		messagesPath := httpWorkspaceNetworkPath(
			harness,
			"/channels/"+escapedChannel+"/threads/"+url.PathEscape(thread.ThreadID)+"/messages",
		)
		if err := harness.HTTPJSON(ctx, http.MethodGet, messagesPath, nil, &response); err != nil {
			return false
		}
		for _, message := range response.Messages {
			if strings.TrimSpace(message.MessageID) == target {
				return true
			}
		}
	}
	return false
}

func requireDirectResolveRace(
	t testing.TB,
	ctx context.Context,
	harness *e2etest.RuntimeHarness,
	channel string,
	leftSessionID string,
	leftPeerID string,
	rightSessionID string,
	rightPeerID string,
) string {
	t.Helper()

	type resolveResult struct {
		directID string
		err      error
	}
	results := make(chan resolveResult, 2)
	var wg sync.WaitGroup
	for _, request := range []struct {
		sessionID string
		peerID    string
	}{
		{sessionID: leftSessionID, peerID: rightPeerID},
		{sessionID: rightSessionID, peerID: leftPeerID},
	} {
		request := request
		wg.Add(1)
		go func() {
			defer wg.Done()
			directID, err := httpResolveNetworkDirectRoomMaybe(ctx, harness, channel, request.sessionID, request.peerID)
			results <- resolveResult{directID: directID, err: err}
		}()
	}
	wg.Wait()
	close(results)

	var directID string
	for result := range results {
		if result.err != nil {
			t.Fatalf("direct resolve race error = %v", result.err)
		}
		if strings.TrimSpace(result.directID) == "" {
			t.Fatal("direct resolve race returned empty direct_id")
		}
		if directID == "" {
			directID = result.directID
			continue
		}
		if result.directID != directID {
			t.Fatalf("direct resolve race returned %q and %q, want same direct_id", directID, result.directID)
		}
	}
	return directID
}

func mustHTTPResolveNetworkDirectRoom(
	t testing.TB,
	ctx context.Context,
	harness *e2etest.RuntimeHarness,
	channel string,
	sessionID string,
	peerID string,
) string {
	t.Helper()

	directID, err := httpResolveNetworkDirectRoomMaybe(ctx, harness, channel, sessionID, peerID)
	if err != nil {
		t.Fatalf("resolve network direct room error = %v", err)
	}
	if directID == "" {
		t.Fatal("resolve network direct room direct_id = empty")
	}
	return directID
}

func httpResolveNetworkDirectRoomMaybe(
	ctx context.Context,
	harness *e2etest.RuntimeHarness,
	channel string,
	sessionID string,
	peerID string,
) (string, error) {
	var response aghcontract.NetworkDirectRoomResponse
	escapedChannel := url.PathEscape(channel)
	path := httpWorkspaceNetworkPath(harness, "/channels/"+escapedChannel+"/directs/resolve")
	request := aghcontract.NetworkDirectResolveRequest{
		SessionID: sessionID,
		PeerID:    peerID,
	}
	if err := harness.HTTPJSON(ctx, http.MethodPost, path, request, &response); err != nil {
		return "", err
	}
	directID := strings.TrimSpace(response.Direct.DirectID)
	return directID, nil
}

func mustSessionTranscript(
	t testing.TB,
	ctx context.Context,
	harness *e2etest.RuntimeHarness,
	sessionID string,
) aghcontract.SessionTranscriptResponse {
	t.Helper()

	response, err := harness.SessionTranscript(ctx, sessionID)
	if err != nil {
		t.Fatalf("SessionTranscript(%q) error = %v", sessionID, err)
	}
	return response
}

func sessionTranscriptHasNeedle(
	ctx context.Context,
	harness *e2etest.RuntimeHarness,
	sessionID string,
	needle string,
) bool {
	response, err := harness.SessionTranscript(ctx, sessionID)
	if err != nil {
		return false
	}
	return strings.Contains(joinTranscriptContent(response.Messages), needle)
}

func mustNetworkAuditSnapshot(
	t testing.TB,
	harness *e2etest.RuntimeHarness,
) []store.NetworkAuditEntry {
	t.Helper()

	entries, err := harness.NetworkAuditSnapshot()
	if err != nil {
		t.Fatalf("NetworkAuditSnapshot() error = %v", err)
	}
	return entries
}

func waitForRuntimeCondition(
	t testing.TB,
	label string,
	timeout time.Duration,
	fn func() bool,
) {
	t.Helper()

	timer := time.NewTimer(timeout)
	defer timer.Stop()
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()

	for {
		if fn() {
			return
		}
		select {
		case <-timer.C:
			t.Fatalf("timed out waiting for %s", label)
		case <-ticker.C:
		}
	}
}

func registerNetworkScenarioArtifacts(
	t testing.TB,
	harness *e2etest.RuntimeHarness,
	channel string,
	sessions []aghcontract.SessionPayload,
	registrations []acpmock.Registration,
) {
	t.Helper()

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := harness.CaptureNetworkArtifacts(ctx, channel); err != nil {
			t.Logf("CaptureNetworkArtifacts(%q) error = %v", channel, err)
		}

		transcripts := make(map[string][]transcript.UIMessage, len(sessions))
		events := make(map[string][]aghcontract.SessionEventPayload, len(sessions))
		for _, session := range sessions {
			transcriptResp, err := harness.SessionTranscript(ctx, session.ID)
			if err != nil {
				t.Logf("SessionTranscript(%q) artifact error = %v", session.ID, err)
				continue
			}
			eventResp, err := harness.SessionEvents(ctx, session.ID)
			if err != nil {
				t.Logf("SessionEvents(%q) artifact error = %v", session.ID, err)
				continue
			}
			transcripts[session.AgentName] = transcriptResp.Messages
			events[session.AgentName] = eventResp.Events
		}
		if len(transcripts) > 0 {
			if err := harness.Artifacts.CaptureJSON(e2etest.ArtifactKindTranscript, transcripts); err != nil {
				t.Logf("CaptureJSON(transcript) error = %v", err)
			}
		}
		if len(events) > 0 {
			if err := harness.Artifacts.CaptureJSON(e2etest.ArtifactKindEvents, events); err != nil {
				t.Logf("CaptureJSON(events) error = %v", err)
			}
		}

		diagnostics := make(map[string][]acpmock.DiagnosticsRecord, len(registrations))
		for _, registration := range registrations {
			records, err := acpmock.ReadDiagnostics(registration.DiagnosticsPath)
			if err != nil {
				t.Logf("ReadDiagnostics(%q) error = %v", registration.AgentName, err)
				continue
			}
			diagnostics[registration.AgentName] = records
		}
		if len(diagnostics) > 0 {
			if err := harness.CaptureProviderCallsJSON(diagnostics); err != nil {
				t.Logf("CaptureProviderCallsJSON() error = %v", err)
			}
		}
	})
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}
