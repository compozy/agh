//go:build integration && !windows

package daemon

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	aghcontract "github.com/pedronauck/agh/internal/api/contract"
	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/testutil/acpmock"
	e2etest "github.com/pedronauck/agh/internal/testutil/e2e"
	"github.com/pedronauck/agh/internal/transcript"
)

func TestDaemonE2ENetworkDirectReplyLifecycleWithMockAgents(t *testing.T) {
	skipWithoutNode(t)

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

	mustSendNetworkCLI(t, ctx, harness, []string{
		"--session", opsSession.ID,
		"--channel", "builders",
		"--kind", "say",
		"--id", "msg_say_01",
		"--trace-id", "trace_ops_patch_42",
		"--body", `{"text":"Who can take the failing migration tests in internal/store/sessiondb?","intent":"request-help","artifacts":[]}`,
	})

	waitForRuntimeCondition(t, "builders say delivery", 10*time.Second, func() bool {
		return channelHasMessageID(ctx, harness, "builders", "msg_say_01") &&
			sessionTranscriptHasNeedle(ctx, harness, opsSession.ID, attributeNeedle("kind", "say")) &&
			sessionTranscriptHasNeedle(ctx, harness, patchSession.ID, attributeNeedle("kind", "say"))
	})

	mustSendNetworkCLI(t, ctx, harness, []string{
		"--session", patchSession.ID,
		"--channel", "builders",
		"--kind", "direct",
		"--to", opsPeerID,
		"--interaction-id", "int_patch_42",
		"--reply-to", "msg_say_01",
		"--trace-id", "trace_ops_patch_42",
		"--causation-id", "msg_say_01",
		"--id", "msg_direct_01",
		"--body", `{"text":"I can take the failing migration tests and send back a patch summary.","intent":"handoff","artifacts":[]}`,
	})

	waitForRuntimeCondition(t, "direct delivery", 10*time.Second, func() bool {
		audit, err := harness.NetworkAuditSnapshot()
		if err != nil {
			return false
		}
		return validateNetworkAuditEntry(audit, networkAuditExpectation{
			MessageID: "msg_direct_01",
			Direction: "delivered",
			Kind:      "direct",
		}) == nil && sessionTranscriptHasNeedle(ctx, harness, opsSession.ID, attributeNeedle("id", "msg_direct_01"))
	})

	mustSendNetworkCLI(t, ctx, harness, []string{
		"--session", opsSession.ID,
		"--channel", "builders",
		"--kind", "receipt",
		"--to", patchPeerID,
		"--interaction-id", "int_patch_42",
		"--reply-to", "msg_direct_01",
		"--trace-id", "trace_ops_patch_42",
		"--causation-id", "msg_direct_01",
		"--id", "msg_receipt_01",
		"--body", `{"for_id":"msg_direct_01","status":"accepted","detail":"Proceed and report progress with trace messages."}`,
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
		"--session", patchSession.ID,
		"--channel", "builders",
		"--kind", "trace",
		"--to", opsPeerID,
		"--interaction-id", "int_patch_42",
		"--reply-to", "msg_receipt_01",
		"--trace-id", "trace_ops_patch_42",
		"--causation-id", "msg_receipt_01",
		"--id", "msg_trace_02",
		"--body", `{"state":"completed","message":"Patch prepared and local tests now pass.","result":{"summary":"Fixed migration assertion mismatch in sessiondb tests."},"artifact_refs":[]}`,
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
		"--session", patchSession.ID,
		"--channel", "builders",
		"--kind", "direct",
		"--to", opsPeerID,
		"--interaction-id", "int_patch_42",
		"--reply-to", "msg_say_01",
		"--trace-id", "trace_ops_patch_42",
		"--causation-id", "msg_say_01",
		"--id", "msg_direct_01",
		"--body", `{"text":"I can take the failing migration tests and send back a patch summary.","intent":"handoff","artifacts":[]}`,
	})

	waitForRuntimeCondition(t, "duplicate rejection audit", 10*time.Second, func() bool {
		audit, err := harness.NetworkAuditSnapshot()
		if err != nil {
			return false
		}
		return validateNetworkAuditEntry(audit, networkAuditExpectation{
			MessageID: "msg_direct_01",
			Direction: "rejected",
			Kind:      "direct",
			Reason:    "duplicate",
		}) == nil
	})

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
	requireChannelMessage(t, channelMessages, "msg_say_01", "Who can take the failing migration tests in internal/store/sessiondb?")

	opsTranscript := mustSessionTranscript(t, ctx, harness, opsSession.ID)
	patchTranscript := mustSessionTranscript(t, ctx, harness, patchSession.ID)
	audit := mustNetworkAuditSnapshot(t, harness)

	if err := validateNetworkCorrelationSurfaces(opsTranscript.Messages, audit, networkCorrelationExpectation{
		MessageID:       "msg_direct_01",
		Kind:            "direct",
		InteractionID:   "int_patch_42",
		ReplyTo:         "msg_say_01",
		TraceID:         "trace_ops_patch_42",
		AuditDirections: []string{"sent", "delivered"},
	}); err != nil {
		t.Fatalf("validateNetworkCorrelationSurfaces(direct) error = %v", err)
	}
	if err := validateNetworkCorrelationSurfaces(patchTranscript.Messages, audit, networkCorrelationExpectation{
		MessageID:       "msg_receipt_01",
		Kind:            "receipt",
		InteractionID:   "int_patch_42",
		ReplyTo:         "msg_direct_01",
		TraceID:         "trace_ops_patch_42",
		AuditDirections: []string{"sent", "delivered"},
	}); err != nil {
		t.Fatalf("validateNetworkCorrelationSurfaces(receipt) error = %v", err)
	}
	if err := validateNetworkCorrelationSurfaces(opsTranscript.Messages, audit, networkCorrelationExpectation{
		MessageID:       "msg_trace_02",
		Kind:            "trace",
		InteractionID:   "int_patch_42",
		ReplyTo:         "msg_receipt_01",
		TraceID:         "trace_ops_patch_42",
		AuditDirections: []string{"sent", "delivered"},
	}); err != nil {
		t.Fatalf("validateNetworkCorrelationSurfaces(trace) error = %v", err)
	}
	if err := validateNetworkAuditEntry(audit, networkAuditExpectation{
		MessageID: "msg_direct_01",
		Direction: "rejected",
		Kind:      "direct",
		Reason:    "duplicate",
	}); err != nil {
		t.Fatalf("validateNetworkAuditEntry(duplicate) error = %v", err)
	}

	assertCLINetworkParity(t, ctx, harness, status, peers, channel)
}

func TestDaemonE2ENetworkWhoisAndRecipeExchange(t *testing.T) {
	skipWithoutNode(t)

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
				FixtureAgent: "recipe-curator",
				AgentName:    "mock-recipe-curator",
			},
		},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	channelDetail := mustCreateNetworkChannel(
		t,
		ctx,
		harness,
		"recipes",
		"mock-release-bot",
		"mock-recipe-curator",
	)
	releaseSession := requireChannelSession(t, channelDetail, "mock-release-bot")
	curatorSession := requireChannelSession(t, channelDetail, "mock-recipe-curator")

	regRelease, ok := harness.MockAgentRegistration("mock-release-bot")
	if !ok {
		t.Fatal("MockAgentRegistration(mock-release-bot) = missing, want present")
	}
	regCurator, ok := harness.MockAgentRegistration("mock-recipe-curator")
	if !ok {
		t.Fatal("MockAgentRegistration(mock-recipe-curator) = missing, want present")
	}

	registerNetworkScenarioArtifacts(
		t,
		harness,
		"recipes",
		[]aghcontract.SessionPayload{releaseSession, curatorSession},
		[]acpmock.Registration{regRelease, regCurator},
	)

	peers := waitForChannelPeerCount(t, ctx, harness, "recipes", 2)
	releasePeerID := requirePeerIDForSession(t, peers, releaseSession.ID)
	curatorPeerID := requirePeerIDForSession(t, peers, curatorSession.ID)
	if releasePeerID == curatorPeerID {
		t.Fatalf("peer IDs = %q and %q, want distinct values", releasePeerID, curatorPeerID)
	}

	mustSendNetworkCLI(t, ctx, harness, []string{
		"--session", releaseSession.ID,
		"--channel", "recipes",
		"--kind", "say",
		"--id", "msg_recipe_say_01",
		"--trace-id", "trace_recipe_apply_7",
		"--body", `{"text":"Does anyone have a reusable migration test repair recipe?","intent":"request-help","artifacts":[]}`,
	})

	waitForRuntimeCondition(t, "recipe say delivery", 10*time.Second, func() bool {
		return channelHasMessageID(ctx, harness, "recipes", "msg_recipe_say_01") &&
			sessionTranscriptHasNeedle(ctx, harness, releaseSession.ID, attributeNeedle("kind", "say"))
	})

	mustSendNetworkCLI(t, ctx, harness, []string{
		"--session", releaseSession.ID,
		"--channel", "recipes",
		"--kind", "whois",
		"--to", curatorPeerID,
		"--id", "msg_whois_01",
		"--body", `{"type":"request","query":"recipe-curator"}`,
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
		"--channel", "recipes",
		"--kind", "recipe",
		"--id", "msg_recipe_01",
		"--trace-id", "trace_recipe_apply_7",
		"--body", `{"recipe":{"recipe_id":"agh.recipe.fix-go-migration-tests","version":"1.0.0","title":"Repair Go migration test assertions","summary":"Align failing migration assertions with the normalized audit rows and rerun the package tests.","content_type":"text/markdown","digest":"sha256:1111111111111111111111111111111111111111111111111111111111111111","inline":"1. Re-run the failing migration tests.\n2. Compare the expected schema with the normalized audit rows.\n3. Update the migration assertion and rerun the package tests.","inputs":["package path","failing test output"],"outputs":["updated assertion","passing package tests"],"requirements":["go test","sessiondb fixtures"]}}`,
	})

	waitForRuntimeCondition(t, "recipe delivery", 10*time.Second, func() bool {
		audit, err := harness.NetworkAuditSnapshot()
		if err != nil {
			return false
		}
		return validateNetworkAuditEntry(audit, networkAuditExpectation{
			MessageID: "msg_recipe_01",
			Direction: "sent",
			Kind:      "recipe",
		}) == nil && sessionTranscriptHasNeedle(ctx, harness, releaseSession.ID, attributeNeedle("id", "msg_recipe_01"))
	})

	mustSendNetworkCLI(t, ctx, harness, []string{
		"--session", releaseSession.ID,
		"--channel", "recipes",
		"--kind", "direct",
		"--to", curatorPeerID,
		"--interaction-id", "int_recipe_apply_7",
		"--reply-to", "msg_recipe_01",
		"--trace-id", "trace_recipe_apply_7",
		"--causation-id", "msg_recipe_01",
		"--id", "msg_direct_20",
		"--body", `{"text":"Can you help adapt this recipe to a failure in internal/store/sessiondb?","intent":"request-guidance","artifacts":[]}`,
	})

	waitForRuntimeCondition(t, "recipe direct delivery", 10*time.Second, func() bool {
		audit, err := harness.NetworkAuditSnapshot()
		if err != nil {
			return false
		}
		return validateNetworkAuditEntry(audit, networkAuditExpectation{
			MessageID: "msg_direct_20",
			Direction: "delivered",
			Kind:      "direct",
		}) == nil && sessionTranscriptHasNeedle(ctx, harness, curatorSession.ID, attributeNeedle("id", "msg_direct_20"))
	})

	mustSendNetworkCLI(t, ctx, harness, []string{
		"--session", curatorSession.ID,
		"--channel", "recipes",
		"--kind", "trace",
		"--to", releasePeerID,
		"--interaction-id", "int_recipe_apply_7",
		"--reply-to", "msg_direct_20",
		"--trace-id", "trace_recipe_apply_7",
		"--causation-id", "msg_direct_20",
		"--id", "msg_trace_21",
		"--body", `{"state":"needs_input","message":"Send the exact package path and the failing test output so I can tailor the recipe.","result":{"recipe_id":"agh.recipe.fix-go-migration-tests"},"artifact_refs":[]}`,
	})

	waitForRuntimeCondition(t, "recipe trace delivery", 10*time.Second, func() bool {
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

	peers = mustHTTPNetworkPeers(t, ctx, harness, "recipes")
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
	channel, ok := findChannelPayload(channels, "recipes")
	if !ok {
		t.Fatalf("HTTP network channels = %#v, want recipes entry", channels)
	}
	if channel.PeerCount != 2 || channel.SessionCount != 2 {
		t.Fatalf("HTTP recipes channel = %#v, want peer_count=2 session_count=2", channel)
	}
	if channel.MessageCount < 1 {
		t.Fatalf("HTTP recipes channel message_count = %d, want >= 1", channel.MessageCount)
	}

	channelDetail = mustHTTPNetworkChannel(t, ctx, harness, "recipes")
	if channelDetail.Channel != "recipes" || channelDetail.PeerCount != 2 || len(channelDetail.Sessions) != 2 {
		t.Fatalf("HTTP channel detail = %#v, want recipes with 2 peers and 2 sessions", channelDetail)
	}

	channelMessages := mustHTTPNetworkChannelMessages(t, ctx, harness, "recipes")
	requireChannelMessage(t, channelMessages, "msg_recipe_say_01", "Does anyone have a reusable migration test repair recipe?")

	releaseTranscript := mustSessionTranscript(t, ctx, harness, releaseSession.ID)
	curatorTranscript := mustSessionTranscript(t, ctx, harness, curatorSession.ID)
	audit := mustNetworkAuditSnapshot(t, harness)

	releaseContent := transcriptContent(releaseTranscript.Messages)
	for _, needle := range []string{
		attributeNeedle("kind", "whois"),
		attributeNeedle("reply-to", "msg_whois_01"),
		attributeNeedle("id", "msg_recipe_01"),
		attributeNeedle("kind", "recipe"),
		attributeNeedle("id", "msg_trace_21"),
		attributeNeedle("trace-id", "trace_recipe_apply_7"),
	} {
		if !strings.Contains(releaseContent, needle) {
			t.Fatalf("release transcript missing %q in %s", needle, releaseContent)
		}
	}
	if err := validateNetworkCorrelationSurfaces(curatorTranscript.Messages, audit, networkCorrelationExpectation{
		MessageID:       "msg_direct_20",
		Kind:            "direct",
		InteractionID:   "int_recipe_apply_7",
		ReplyTo:         "msg_recipe_01",
		TraceID:         "trace_recipe_apply_7",
		AuditDirections: []string{"sent", "delivered"},
	}); err != nil {
		t.Fatalf("validateNetworkCorrelationSurfaces(recipe direct) error = %v", err)
	}
	if err := validateNetworkCorrelationSurfaces(releaseTranscript.Messages, audit, networkCorrelationExpectation{
		MessageID:       "msg_trace_21",
		Kind:            "trace",
		InteractionID:   "int_recipe_apply_7",
		ReplyTo:         "msg_direct_20",
		TraceID:         "trace_recipe_apply_7",
		AuditDirections: []string{"sent", "delivered"},
	}); err != nil {
		t.Fatalf("validateNetworkCorrelationSurfaces(recipe trace) error = %v", err)
	}

	assertCLINetworkParity(t, ctx, harness, status, peers, channel)
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
	messages []aghcontract.NetworkChannelMessagePayload,
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

func mustSendNetworkCLI(
	t testing.TB,
	ctx context.Context,
	harness *e2etest.RuntimeHarness,
	args []string,
) aghcontract.NetworkSendPayload {
	t.Helper()

	var payload aghcontract.NetworkSendPayload
	fullArgs := append([]string{"network", "send"}, append(args, "-o", "json")...)
	if err := harness.CLI.RunJSON(ctx, &payload, fullArgs...); err != nil {
		t.Fatalf("CLI %v error = %v", fullArgs, err)
	}
	return payload
}

func assertCLINetworkParity(
	t testing.TB,
	ctx context.Context,
	harness *e2etest.RuntimeHarness,
	httpStatus aghcontract.NetworkStatusPayload,
	httpPeers []aghcontract.NetworkPeerPayload,
	httpChannel aghcontract.NetworkChannelPayload,
) {
	t.Helper()

	var cliStatus aghcontract.NetworkStatusPayload
	if err := harness.CLI.RunJSON(ctx, &cliStatus, "network", "status", "-o", "json"); err != nil {
		t.Fatalf("CLI network status error = %v", err)
	}
	if cliStatus.Enabled != httpStatus.Enabled ||
		cliStatus.Status != httpStatus.Status ||
		cliStatus.LocalPeers != httpStatus.LocalPeers ||
		cliStatus.Channels != httpStatus.Channels {
		t.Fatalf("CLI network status = %#v, want parity with HTTP %#v", cliStatus, httpStatus)
	}

	var cliPeers []aghcontract.NetworkPeerPayload
	if err := harness.CLI.RunJSON(ctx, &cliPeers, "network", "peers", httpChannel.Channel, "-o", "json"); err != nil {
		t.Fatalf("CLI network peers error = %v", err)
	}
	if len(cliPeers) != len(httpPeers) {
		t.Fatalf("CLI network peers = %#v, want %d peers", cliPeers, len(httpPeers))
	}
	for _, peer := range httpPeers {
		if requirePeerIDForSession(t, cliPeers, derefString(peer.SessionID)) != strings.TrimSpace(peer.PeerID) {
			t.Fatalf("CLI peers = %#v, want peer %q for session %v", cliPeers, peer.PeerID, peer.SessionID)
		}
	}

	var cliChannels []aghcontract.NetworkChannelPayload
	if err := harness.CLI.RunJSON(ctx, &cliChannels, "network", "channels", "-o", "json"); err != nil {
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

func mustHTTPNetworkPeers(
	t testing.TB,
	ctx context.Context,
	harness *e2etest.RuntimeHarness,
	channel string,
) []aghcontract.NetworkPeerPayload {
	t.Helper()

	peers, err := mustHTTPNetworkPeersMaybe(ctx, harness, channel)
	if err != nil {
		t.Fatalf("HTTPJSON(/api/network/peers) error = %v", err)
	}
	return peers
}

func mustHTTPNetworkPeersMaybe(
	ctx context.Context,
	harness *e2etest.RuntimeHarness,
	channel string,
) ([]aghcontract.NetworkPeerPayload, error) {
	var response aghcontract.NetworkPeersResponse
	path := "/api/network/peers"
	if trimmed := strings.TrimSpace(channel); trimmed != "" {
		path += "?channel=" + trimmed
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
	if err := harness.HTTPJSON(ctx, http.MethodGet, "/api/network/channels", nil, &response); err != nil {
		t.Fatalf("HTTPJSON(/api/network/channels) error = %v", err)
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
	if err := harness.HTTPJSON(ctx, http.MethodGet, "/api/network/channels/"+channel, nil, &response); err != nil {
		t.Fatalf("HTTPJSON(/api/network/channels/%s) error = %v", channel, err)
	}
	return response.Channel
}

func mustHTTPNetworkChannelMessages(
	t testing.TB,
	ctx context.Context,
	harness *e2etest.RuntimeHarness,
	channel string,
) []aghcontract.NetworkChannelMessagePayload {
	t.Helper()

	var response aghcontract.NetworkChannelMessagesResponse
	if err := harness.HTTPJSON(
		ctx,
		http.MethodGet,
		"/api/network/channels/"+channel+"/messages",
		nil,
		&response,
	); err != nil {
		t.Fatalf("HTTPJSON(/api/network/channels/%s/messages) error = %v", channel, err)
	}
	return response.Messages
}

func channelHasMessageID(
	ctx context.Context,
	harness *e2etest.RuntimeHarness,
	channel string,
	messageID string,
) bool {
	var response aghcontract.NetworkChannelMessagesResponse
	if err := harness.HTTPJSON(
		ctx,
		http.MethodGet,
		"/api/network/channels/"+channel+"/messages",
		nil,
		&response,
	); err != nil {
		return false
	}
	target := strings.TrimSpace(messageID)
	for _, message := range response.Messages {
		if strings.TrimSpace(message.MessageID) == target {
			return true
		}
	}
	return false
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
	return strings.Contains(transcriptContent(response.Messages), needle)
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

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", label)
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

		transcripts := make(map[string][]transcript.Message, len(sessions))
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
