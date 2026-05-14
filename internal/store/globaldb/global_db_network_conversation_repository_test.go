package globaldb

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/testutil"
)

func TestGlobalDBResolveDirectRoom(t *testing.T) {
	t.Parallel()

	t.Run("Should return one durable room for concurrent resolves of the same pair", func(t *testing.T) {
		t.Parallel()

		globalDB := openTestGlobalDB(t)
		expectedID, expectedPeerA, expectedPeerB, err := store.NetworkDirectRoomIdentity(
			networkStoreTestWorkspaceID,
			"builders",
			"reviewer.sess-xyz",
			"coder.sess-abc",
		)
		if err != nil {
			t.Fatalf("NetworkDirectRoomIdentity() error = %v", err)
		}

		const workers = 16
		var waitGroup sync.WaitGroup
		errs := make(chan error, workers)
		for index := range workers {
			waitGroup.Add(1)
			go func(index int) {
				defer waitGroup.Done()

				peerA := "coder.sess-abc"
				peerB := "reviewer.sess-xyz"
				if index%2 == 0 {
					peerA, peerB = peerB, peerA
				}
				summary, resolveErr := globalDB.ResolveDirectRoom(testutil.Context(t), store.NetworkDirectRoomEntry{
					WorkspaceID: networkStoreTestWorkspaceID,
					Channel:     "builders",
					PeerA:       peerA,
					PeerB:       peerB,
				})
				if resolveErr != nil {
					errs <- resolveErr
					return
				}
				if summary.DirectID != expectedID || summary.PeerA != expectedPeerA || summary.PeerB != expectedPeerB {
					errs <- errors.New("resolved direct room summary did not match deterministic identity")
				}
			}(index)
		}
		waitGroup.Wait()
		close(errs)
		for err := range errs {
			if err != nil {
				t.Fatalf("ResolveDirectRoom(concurrent) error = %v", err)
			}
		}

		rooms, err := globalDB.ListDirectRooms(
			testutil.Context(t),
			store.NetworkChannelRef{WorkspaceID: networkStoreTestWorkspaceID, Channel: "builders"},
			store.NetworkDirectRoomQuery{Limit: 10},
		)
		if err != nil {
			t.Fatalf("ListDirectRooms() error = %v", err)
		}
		if got, want := len(rooms), 1; got != want {
			t.Fatalf("len(rooms) = %d, want %d", got, want)
		}
		if got, want := rooms[0].DirectID, expectedID; got != want {
			t.Fatalf("rooms[0].DirectID = %q, want %q", got, want)
		}
	})

	t.Run("Should fail closed when a deterministic direct id is already bound to another pair", func(t *testing.T) {
		t.Parallel()

		globalDB := openTestGlobalDB(t)
		directID, _, _, err := store.NetworkDirectRoomIdentity(
			networkStoreTestWorkspaceID,
			"builders",
			"coder.sess-abc",
			"reviewer.sess-xyz",
		)
		if err != nil {
			t.Fatalf("NetworkDirectRoomIdentity() error = %v", err)
		}
		insertDirectRoom(t, globalDB.db, networkStoreTestWorkspaceID, "builders", directID, "alpha.sess", "zulu.sess")

		_, err = globalDB.ResolveDirectRoom(testutil.Context(t), store.NetworkDirectRoomEntry{
			WorkspaceID: networkStoreTestWorkspaceID,
			Channel:     "builders",
			PeerA:       "coder.sess-abc",
			PeerB:       "reviewer.sess-xyz",
		})
		if !errors.Is(err, store.ErrNetworkDirectRoomCollision) {
			t.Fatalf("ResolveDirectRoom(collision) error = %v, want ErrNetworkDirectRoomCollision", err)
		}
	})
}

func TestGlobalDBWriteConversationMessageThreadSummaries(t *testing.T) {
	t.Parallel()

	t.Run("Should open a thread and derive message and participant counts from committed rows", func(t *testing.T) {
		t.Parallel()

		globalDB := openTestGlobalDB(t)
		startedAt := time.Date(2026, 5, 5, 14, 0, 0, 0, time.UTC)
		messages := []store.NetworkConversationMessage{
			threadMessage("msg_thread_root", "thread_store_counts", "coder.sess-abc", "hello builders", startedAt),
			threadMessage(
				"msg_thread_second",
				"thread_store_counts",
				"coder.sess-abc",
				"still here",
				startedAt.Add(time.Minute),
			),
			threadMessage(
				"msg_thread_third",
				"thread_store_counts",
				"reviewer.sess-xyz",
				"reviewing",
				startedAt.Add(2*time.Minute),
			),
		}

		firstResult, err := globalDB.WriteConversationMessage(testutil.Context(t), messages[0])
		if err != nil {
			t.Fatalf("WriteConversationMessage(root) error = %v", err)
		}
		if !firstResult.ConversationOpened {
			t.Fatal("WriteConversationMessage(root).ConversationOpened = false, want true")
		}
		for _, message := range messages[1:] {
			result, writeErr := globalDB.WriteConversationMessage(testutil.Context(t), message)
			if writeErr != nil {
				t.Fatalf("WriteConversationMessage(%q) error = %v", message.MessageID, writeErr)
			}
			if result.ConversationOpened {
				t.Fatalf("WriteConversationMessage(%q).ConversationOpened = true, want false", message.MessageID)
			}
		}

		thread, err := globalDB.GetThread(
			testutil.Context(t),
			store.NetworkChannelRef{WorkspaceID: networkStoreTestWorkspaceID, Channel: "builders"},
			"thread_store_counts",
		)
		if err != nil {
			t.Fatalf("GetThread() error = %v", err)
		}
		if got, want := thread.RootMessageID, "msg_thread_root"; got != want {
			t.Fatalf("thread.RootMessageID = %q, want %q", got, want)
		}
		if got, want := thread.MessageCount, 3; got != want {
			t.Fatalf("thread.MessageCount = %d, want %d", got, want)
		}
		if got, want := thread.ParticipantCount, 2; got != want {
			t.Fatalf("thread.ParticipantCount = %d, want %d", got, want)
		}
		if got, want := thread.LastMessagePreview, "reviewing"; got != want {
			t.Fatalf("thread.LastMessagePreview = %q, want %q", got, want)
		}

		threads, err := globalDB.ListThreads(
			testutil.Context(t),
			store.NetworkChannelRef{WorkspaceID: networkStoreTestWorkspaceID, Channel: "builders"},
			store.NetworkThreadQuery{Limit: 10},
		)
		if err != nil {
			t.Fatalf("ListThreads() error = %v", err)
		}
		if got, want := len(threads), 1; got != want {
			t.Fatalf("len(threads) = %d, want %d", got, want)
		}
		if got, want := threads[0].ThreadID, "thread_store_counts"; got != want {
			t.Fatalf("threads[0].ThreadID = %q, want %q", got, want)
		}

		entries, err := globalDB.ListConversationMessages(
			testutil.Context(t),
			store.NetworkConversationRef{
				WorkspaceID: networkStoreTestWorkspaceID,
				Channel:     "builders",
				Surface:     store.NetworkSurfaceThread,
				ThreadID:    "thread_store_counts",
			},
			store.NetworkConversationMessageQuery{Limit: 10},
		)
		if err != nil {
			t.Fatalf("ListConversationMessages(thread) error = %v", err)
		}
		if got, want := len(entries), 3; got != want {
			t.Fatalf("len(entries) = %d, want %d", got, want)
		}

		auditRows, err := globalDB.ListNetworkAudit(testutil.Context(t), store.NetworkAuditQuery{
			WorkspaceID: networkStoreTestWorkspaceID,
			MessageID:   "msg_thread_root",
			Limit:       10,
		})
		if err != nil {
			t.Fatalf("ListNetworkAudit(root) error = %v", err)
		}
		if got, want := len(auditRows), 1; got != want {
			t.Fatalf("len(auditRows) = %d, want %d", got, want)
		}
		if got, want := auditRows[0].ThreadID, "thread_store_counts"; got != want {
			t.Fatalf("auditRows[0].ThreadID = %q, want %q", got, want)
		}
	})
}

func TestGlobalDBWriteConversationMessageDirectIsolationAndWorkLookup(t *testing.T) {
	t.Parallel()

	t.Run("Should update only the matching direct room and isolate thread and direct queries", func(t *testing.T) {
		t.Parallel()

		globalDB := openTestGlobalDB(t)
		startedAt := time.Date(2026, 5, 5, 15, 0, 0, 0, time.UTC)
		directID, _, _, err := store.NetworkDirectRoomIdentity(
			networkStoreTestWorkspaceID,
			"builders",
			"coder.sess-abc",
			"reviewer.sess-xyz",
		)
		if err != nil {
			t.Fatalf("NetworkDirectRoomIdentity() error = %v", err)
		}
		direct := store.NetworkConversationMessage{
			MessageID:   "msg_direct_review",
			SessionID:   "sess-direct",
			WorkspaceID: networkStoreTestWorkspaceID,
			Channel:     "builders",
			Surface:     store.NetworkSurfaceDirect,
			DirectID:    directID,
			Direction:   "sent",
			PeerFrom:    "coder.sess-abc",
			PeerTo:      "reviewer.sess-xyz",
			Kind:        store.NetworkKindSay,
			WorkID:      "work_direct_review",
			Text:        "please review privately",
			PreviewText: "please review privately",
			Body:        []byte(`{"text":"please review privately"}`),
			Timestamp:   startedAt,
		}
		if _, err := globalDB.WriteConversationMessage(testutil.Context(t), direct); err != nil {
			t.Fatalf("WriteConversationMessage(direct) error = %v", err)
		}
		if _, err := globalDB.WriteConversationMessage(
			testutil.Context(t),
			threadMessage(
				"msg_thread_public",
				"thread_store_isolation",
				"founder.sess-main",
				"public update",
				startedAt,
			),
		); err != nil {
			t.Fatalf("WriteConversationMessage(thread) error = %v", err)
		}

		directSummary, err := globalDB.GetDirectRoom(
			testutil.Context(t),
			store.NetworkChannelRef{WorkspaceID: networkStoreTestWorkspaceID, Channel: "builders"},
			directID,
		)
		if err != nil {
			t.Fatalf("GetDirectRoom() error = %v", err)
		}
		if got, want := directSummary.MessageCount, 1; got != want {
			t.Fatalf("directSummary.MessageCount = %d, want %d", got, want)
		}
		if got, want := directSummary.OpenWorkCount, 1; got != want {
			t.Fatalf("directSummary.OpenWorkCount = %d, want %d", got, want)
		}

		directMessages, err := globalDB.ListConversationMessages(
			testutil.Context(t),
			store.NetworkConversationRef{
				WorkspaceID: networkStoreTestWorkspaceID,
				Channel:     "builders",
				Surface:     store.NetworkSurfaceDirect,
				DirectID:    directID,
			},
			store.NetworkConversationMessageQuery{Limit: 10},
		)
		if err != nil {
			t.Fatalf("ListConversationMessages(direct) error = %v", err)
		}
		if got, want := len(directMessages), 1; got != want {
			t.Fatalf("len(directMessages) = %d, want %d", got, want)
		}
		if got, want := directMessages[0].MessageID, "msg_direct_review"; got != want {
			t.Fatalf("directMessages[0].MessageID = %q, want %q", got, want)
		}

		threadMessages, err := globalDB.ListConversationMessages(
			testutil.Context(t),
			store.NetworkConversationRef{
				WorkspaceID: networkStoreTestWorkspaceID,
				Channel:     "builders",
				Surface:     store.NetworkSurfaceThread,
				ThreadID:    "thread_store_isolation",
			},
			store.NetworkConversationMessageQuery{Limit: 10},
		)
		if err != nil {
			t.Fatalf("ListConversationMessages(thread) error = %v", err)
		}
		if got, want := len(threadMessages), 1; got != want {
			t.Fatalf("len(threadMessages) = %d, want %d", got, want)
		}
		if got, want := threadMessages[0].MessageID, "msg_thread_public"; got != want {
			t.Fatalf("threadMessages[0].MessageID = %q, want %q", got, want)
		}

		work, err := globalDB.GetWork(testutil.Context(t), networkStoreTestWorkspaceID, "work_direct_review")
		if err != nil {
			t.Fatalf("GetWork() error = %v", err)
		}
		if got, want := work.Surface, store.NetworkSurfaceDirect; got != want {
			t.Fatalf("work.Surface = %q, want %q", got, want)
		}
		if got, want := work.DirectID, directID; got != want {
			t.Fatalf("work.DirectID = %q, want %q", got, want)
		}
	})
}

func TestGlobalDBConversationQueriesSupportCursorsAndFilters(t *testing.T) {
	t.Parallel()

	t.Run("Should page thread and direct summaries and filter conversation messages", func(t *testing.T) {
		t.Parallel()

		globalDB := openTestGlobalDB(t)
		startedAt := time.Date(2026, 5, 5, 18, 0, 0, 0, time.UTC)
		threadMessages := []store.NetworkConversationMessage{
			threadMessage("msg_query_one", "thread_query_cursors", "coder.sess-abc", "first", startedAt),
			threadMessage(
				"msg_query_two",
				"thread_query_cursors",
				"coder.sess-abc",
				"second",
				startedAt.Add(time.Minute),
			),
			threadMessage(
				"msg_query_three",
				"thread_query_cursors",
				"reviewer.sess-xyz",
				"third",
				startedAt.Add(2*time.Minute),
			),
			threadMessage(
				"msg_query_other",
				"thread_query_other",
				"founder.sess-main",
				"other",
				startedAt.Add(3*time.Minute),
			),
		}
		threadMessages[1].PeerTo = "reviewer.sess-xyz"
		threadMessages[1].WorkID = "work_query_filter"
		for _, message := range threadMessages {
			if _, err := globalDB.WriteConversationMessage(testutil.Context(t), message); err != nil {
				t.Fatalf("WriteConversationMessage(%q) error = %v", message.MessageID, err)
			}
		}

		firstThreadPage, err := globalDB.ListThreads(
			testutil.Context(t), store.NetworkChannelRef{WorkspaceID: networkStoreTestWorkspaceID, Channel: "builders"},
			store.NetworkThreadQuery{Limit: 1},
		)
		if err != nil {
			t.Fatalf("ListThreads(first page) error = %v", err)
		}
		if got, want := len(firstThreadPage), 1; got != want {
			t.Fatalf("len(firstThreadPage) = %d, want %d", got, want)
		}
		if got, want := firstThreadPage[0].ThreadID, "thread_query_other"; got != want {
			t.Fatalf("firstThreadPage[0].ThreadID = %q, want %q", got, want)
		}
		secondThreadPage, err := globalDB.ListThreads(
			testutil.Context(t),
			store.NetworkChannelRef{WorkspaceID: networkStoreTestWorkspaceID, Channel: "builders"},
			store.NetworkThreadQuery{
				Limit: 10,
				After: firstThreadPage[0].ThreadID,
			},
		)
		if err != nil {
			t.Fatalf("ListThreads(second page) error = %v", err)
		}
		if got, want := len(secondThreadPage), 1; got != want {
			t.Fatalf("len(secondThreadPage) = %d, want %d", got, want)
		}
		if got, want := secondThreadPage[0].ThreadID, "thread_query_cursors"; got != want {
			t.Fatalf("secondThreadPage[0].ThreadID = %q, want %q", got, want)
		}

		ref := store.NetworkConversationRef{
			WorkspaceID: networkStoreTestWorkspaceID,
			Channel:     "builders",
			Surface:     store.NetworkSurfaceThread,
			ThreadID:    "thread_query_cursors",
		}
		before, err := globalDB.ListConversationMessages(
			testutil.Context(t),
			ref,
			store.NetworkConversationMessageQuery{
				BeforeMessageID: "msg_query_three",
				Limit:           10,
			},
		)
		if err != nil {
			t.Fatalf("ListConversationMessages(before) error = %v", err)
		}
		if got, want := messageIDs(before), []string{"msg_query_one", "msg_query_two"}; !sameStrings(got, want) {
			t.Fatalf("before message IDs = %v, want %v", got, want)
		}
		after, err := globalDB.ListConversationMessages(testutil.Context(t), ref, store.NetworkConversationMessageQuery{
			AfterMessageID: "msg_query_one",
			Limit:          10,
		})
		if err != nil {
			t.Fatalf("ListConversationMessages(after) error = %v", err)
		}
		if got, want := messageIDs(after), []string{"msg_query_two", "msg_query_three"}; !sameStrings(got, want) {
			t.Fatalf("after message IDs = %v, want %v", got, want)
		}
		filtered, err := globalDB.ListConversationMessages(
			testutil.Context(t),
			ref,
			store.NetworkConversationMessageQuery{
				WorkID: "work_query_filter",
				Limit:  10,
			},
		)
		if err != nil {
			t.Fatalf("ListConversationMessages(work filter) error = %v", err)
		}
		if got, want := messageIDs(filtered), []string{"msg_query_two"}; !sameStrings(got, want) {
			t.Fatalf("filtered message IDs = %v, want %v", got, want)
		}

		firstDirectID, err := writeDirectMessage(
			t,
			globalDB,
			"msg_direct_cursor_one",
			"coder.sess-abc",
			"reviewer.sess-xyz",
			"first direct",
			startedAt.Add(4*time.Minute),
		)
		if err != nil {
			t.Fatalf("writeDirectMessage(first) error = %v", err)
		}
		secondDirectID, err := writeDirectMessage(
			t,
			globalDB,
			"msg_direct_cursor_two",
			"coder.sess-abc",
			"planner.sess-123",
			"second direct",
			startedAt.Add(5*time.Minute),
		)
		if err != nil {
			t.Fatalf("writeDirectMessage(second) error = %v", err)
		}

		firstDirectPage, err := globalDB.ListDirectRooms(
			testutil.Context(t),
			store.NetworkChannelRef{WorkspaceID: networkStoreTestWorkspaceID, Channel: "builders"},
			store.NetworkDirectRoomQuery{
				PeerID: "coder.sess-abc",
				Limit:  1,
			},
		)
		if err != nil {
			t.Fatalf("ListDirectRooms(first page) error = %v", err)
		}
		if got, want := len(firstDirectPage), 1; got != want {
			t.Fatalf("len(firstDirectPage) = %d, want %d", got, want)
		}
		if got, want := firstDirectPage[0].DirectID, secondDirectID; got != want {
			t.Fatalf("firstDirectPage[0].DirectID = %q, want %q", got, want)
		}
		secondDirectPage, err := globalDB.ListDirectRooms(
			testutil.Context(t),
			store.NetworkChannelRef{WorkspaceID: networkStoreTestWorkspaceID, Channel: "builders"},
			store.NetworkDirectRoomQuery{
				PeerID: "coder.sess-abc",
				Limit:  10,
				After:  firstDirectPage[0].DirectID,
			},
		)
		if err != nil {
			t.Fatalf("ListDirectRooms(second page) error = %v", err)
		}
		if got, want := len(secondDirectPage), 1; got != want {
			t.Fatalf("len(secondDirectPage) = %d, want %d", got, want)
		}
		if got, want := secondDirectPage[0].DirectID, firstDirectID; got != want {
			t.Fatalf("secondDirectPage[0].DirectID = %q, want %q", got, want)
		}
	})
}

func TestGlobalDBWriteConversationMessageWorkReceiptTransitions(t *testing.T) {
	t.Parallel()

	t.Run("Should apply receipt states and resume needs-input work from a say message", func(t *testing.T) {
		t.Parallel()

		globalDB := openTestGlobalDB(t)
		startedAt := time.Date(2026, 5, 5, 19, 0, 0, 0, time.UTC)
		opening := threadMessage(
			"msg_receipt_open",
			"thread_receipt_transitions",
			"coder.sess-abc",
			"please build",
			startedAt,
		)
		opening.PeerTo = "reviewer.sess-xyz"
		opening.WorkID = "work_receipt_failed"
		if _, err := globalDB.WriteConversationMessage(testutil.Context(t), opening); err != nil {
			t.Fatalf("WriteConversationMessage(opening) error = %v", err)
		}

		rejected := threadReceiptMessage(
			"msg_receipt_rejected",
			"thread_receipt_transitions",
			"reviewer.sess-xyz",
			"work_receipt_failed",
			"rejected",
			startedAt.Add(time.Minute),
		)
		result, err := globalDB.WriteConversationMessage(testutil.Context(t), rejected)
		if err != nil {
			t.Fatalf("WriteConversationMessage(rejected receipt) error = %v", err)
		}
		if !result.WorkTransitioned || result.WorkState != store.NetworkWorkStateFailed {
			t.Fatalf("rejected receipt result = %#v, want failed transition", result)
		}
		failedWork, err := globalDB.GetWork(testutil.Context(t), networkStoreTestWorkspaceID, "work_receipt_failed")
		if err != nil {
			t.Fatalf("GetWork(failed) error = %v", err)
		}
		if got, want := failedWork.State, store.NetworkWorkStateFailed; got != want {
			t.Fatalf("failedWork.State = %q, want %q", got, want)
		}

		needsInput := threadMessage(
			"msg_needs_input_open",
			"thread_receipt_transitions",
			"coder.sess-abc",
			"needs input work",
			startedAt.Add(2*time.Minute),
		)
		needsInput.PeerTo = "reviewer.sess-xyz"
		needsInput.WorkID = "work_needs_input"
		if _, err := globalDB.WriteConversationMessage(testutil.Context(t), needsInput); err != nil {
			t.Fatalf("WriteConversationMessage(needs input opening) error = %v", err)
		}
		traceNeedsInput := threadTraceMessage(
			"msg_needs_input_trace",
			"thread_receipt_transitions",
			"reviewer.sess-xyz",
			"work_needs_input",
			store.NetworkWorkStateNeedsInput,
			startedAt.Add(3*time.Minute),
		)
		if _, err := globalDB.WriteConversationMessage(testutil.Context(t), traceNeedsInput); err != nil {
			t.Fatalf("WriteConversationMessage(needs input trace) error = %v", err)
		}
		resume := threadMessage(
			"msg_needs_input_resume",
			"thread_receipt_transitions",
			"coder.sess-abc",
			"resume work",
			startedAt.Add(4*time.Minute),
		)
		resume.PeerTo = "reviewer.sess-xyz"
		resume.WorkID = "work_needs_input"
		resumeResult, err := globalDB.WriteConversationMessage(testutil.Context(t), resume)
		if err != nil {
			t.Fatalf("WriteConversationMessage(resume) error = %v", err)
		}
		if !resumeResult.WorkTransitioned || resumeResult.WorkState != store.NetworkWorkStateWorking {
			t.Fatalf("resume result = %#v, want working transition", resumeResult)
		}
		resumedWork, err := globalDB.GetWork(testutil.Context(t), networkStoreTestWorkspaceID, "work_needs_input")
		if err != nil {
			t.Fatalf("GetWork(resumed) error = %v", err)
		}
		if got, want := resumedWork.State, store.NetworkWorkStateWorking; got != want {
			t.Fatalf("resumedWork.State = %q, want %q", got, want)
		}
	})
}

func TestGlobalDBWriteConversationMessageRejectsInvalidLifecycleWrites(t *testing.T) {
	t.Parallel()

	t.Run("Should reject invalid work mutations without keeping timeline or audit side effects", func(t *testing.T) {
		t.Parallel()

		globalDB := openTestGlobalDB(t)
		startedAt := time.Date(2026, 5, 5, 20, 0, 0, 0, time.UTC)
		receiptWithoutWork := threadReceiptMessage(
			"msg_receipt_missing_work",
			"thread_invalid_lifecycle",
			"reviewer.sess-xyz",
			"work_missing",
			"accepted",
			startedAt,
		)
		_, err := globalDB.WriteConversationMessage(testutil.Context(t), receiptWithoutWork)
		if err == nil {
			t.Fatal("WriteConversationMessage(receipt without work) error = nil, want non-nil")
		}
		assertNoTimelineOrAuditRows(t, globalDB, receiptWithoutWork.MessageID)

		missingPeerTo := threadMessage(
			"msg_work_missing_peer",
			"thread_invalid_lifecycle",
			"coder.sess-abc",
			"missing target",
			startedAt.Add(time.Minute),
		)
		missingPeerTo.WorkID = "work_missing_peer"
		_, err = globalDB.WriteConversationMessage(testutil.Context(t), missingPeerTo)
		if err == nil {
			t.Fatal("WriteConversationMessage(work without peer_to) error = nil, want non-nil")
		}
		assertNoTimelineOrAuditRows(t, globalDB, missingPeerTo.MessageID)

		opening := threadMessage(
			"msg_invalid_work_open",
			"thread_invalid_lifecycle",
			"coder.sess-abc",
			"open work",
			startedAt.Add(2*time.Minute),
		)
		opening.PeerTo = "reviewer.sess-xyz"
		opening.WorkID = "work_invalid_lifecycle"
		if _, err := globalDB.WriteConversationMessage(testutil.Context(t), opening); err != nil {
			t.Fatalf("WriteConversationMessage(opening) error = %v", err)
		}

		accepted := threadReceiptMessage(
			"msg_receipt_accepted",
			"thread_invalid_lifecycle",
			"reviewer.sess-xyz",
			"work_invalid_lifecycle",
			"accepted",
			startedAt.Add(3*time.Minute),
		)
		result, err := globalDB.WriteConversationMessage(testutil.Context(t), accepted)
		if err != nil {
			t.Fatalf("WriteConversationMessage(accepted receipt) error = %v", err)
		}
		if result.WorkTransitioned || result.WorkState != store.NetworkWorkStateSubmitted {
			t.Fatalf("accepted receipt result = %#v, want no submitted transition", result)
		}

		invalidTrace := threadTraceMessage(
			"msg_trace_invalid_state",
			"thread_invalid_lifecycle",
			"reviewer.sess-xyz",
			"work_invalid_lifecycle",
			store.NetworkWorkStateSubmitted,
			startedAt.Add(4*time.Minute),
		)
		_, err = globalDB.WriteConversationMessage(testutil.Context(t), invalidTrace)
		if err == nil {
			t.Fatal("WriteConversationMessage(invalid trace) error = nil, want non-nil")
		}
		assertNoTimelineOrAuditRows(t, globalDB, invalidTrace.MessageID)

		canceledOpening := threadMessage(
			"msg_canceled_work_open",
			"thread_invalid_lifecycle",
			"coder.sess-abc",
			"cancel me",
			startedAt.Add(5*time.Minute),
		)
		canceledOpening.PeerTo = "reviewer.sess-xyz"
		canceledOpening.WorkID = "work_receipt_canceled"
		if _, err := globalDB.WriteConversationMessage(testutil.Context(t), canceledOpening); err != nil {
			t.Fatalf("WriteConversationMessage(canceled opening) error = %v", err)
		}
		canceled := threadReceiptMessage(
			"msg_receipt_canceled",
			"thread_invalid_lifecycle",
			"reviewer.sess-xyz",
			"work_receipt_canceled",
			"canceled",
			startedAt.Add(6*time.Minute),
		)
		result, err = globalDB.WriteConversationMessage(testutil.Context(t), canceled)
		if err != nil {
			t.Fatalf("WriteConversationMessage(canceled receipt) error = %v", err)
		}
		if !result.WorkTransitioned || result.WorkState != store.NetworkWorkStateCanceled {
			t.Fatalf("canceled receipt result = %#v, want canceled transition", result)
		}

		unsupportedOpening := threadMessage(
			"msg_unsupported_work_open",
			"thread_invalid_lifecycle",
			"coder.sess-abc",
			"unsupported receipt",
			startedAt.Add(7*time.Minute),
		)
		unsupportedOpening.PeerTo = "reviewer.sess-xyz"
		unsupportedOpening.WorkID = "work_receipt_unsupported"
		if _, err := globalDB.WriteConversationMessage(testutil.Context(t), unsupportedOpening); err != nil {
			t.Fatalf("WriteConversationMessage(unsupported opening) error = %v", err)
		}
		unsupported := threadReceiptMessage(
			"msg_receipt_unsupported",
			"thread_invalid_lifecycle",
			"reviewer.sess-xyz",
			"work_receipt_unsupported",
			"ignored",
			startedAt.Add(8*time.Minute),
		)
		_, err = globalDB.WriteConversationMessage(testutil.Context(t), unsupported)
		if err == nil {
			t.Fatal("WriteConversationMessage(unsupported receipt) error = nil, want non-nil")
		}
		assertNoTimelineOrAuditRows(t, globalDB, unsupported.MessageID)

		mismatched := threadTraceMessage(
			"msg_work_container_mismatch",
			"thread_other_container",
			"reviewer.sess-xyz",
			"work_invalid_lifecycle",
			store.NetworkWorkStateWorking,
			startedAt.Add(9*time.Minute),
		)
		_, err = globalDB.WriteConversationMessage(testutil.Context(t), mismatched)
		if !errors.Is(err, store.ErrNetworkWorkContainerMismatch) {
			t.Fatalf("WriteConversationMessage(container mismatch) error = %v, want mismatch", err)
		}
		assertNoTimelineOrAuditRows(t, globalDB, mismatched.MessageID)
	})
}

func TestGlobalDBConversationQueryErrors(t *testing.T) {
	t.Parallel()

	t.Run("Should reject invalid cursors and missing conversation rows", func(t *testing.T) {
		t.Parallel()

		globalDB := openTestGlobalDB(t)
		startedAt := time.Date(2026, 5, 5, 21, 0, 0, 0, time.UTC)
		if _, err := globalDB.WriteConversationMessage(
			testutil.Context(t),
			threadMessage("msg_query_error", "thread_query_error", "coder.sess-abc", "query errors", startedAt),
		); err != nil {
			t.Fatalf("WriteConversationMessage(thread) error = %v", err)
		}
		directID, err := writeDirectMessage(
			t,
			globalDB,
			"msg_direct_query_error",
			"coder.sess-abc",
			"reviewer.sess-xyz",
			"direct query errors",
			startedAt.Add(time.Minute),
		)
		if err != nil {
			t.Fatalf("writeDirectMessage() error = %v", err)
		}

		_, err = globalDB.GetThread(
			testutil.Context(t),
			store.NetworkChannelRef{WorkspaceID: networkStoreTestWorkspaceID, Channel: "builders"},
			"thread_missing",
		)
		if !errors.Is(err, store.ErrNetworkConversationNotFound) {
			t.Fatalf("GetThread(missing) error = %v, want ErrNetworkConversationNotFound", err)
		}
		_, err = globalDB.GetDirectRoom(
			testutil.Context(t),
			store.NetworkChannelRef{WorkspaceID: networkStoreTestWorkspaceID, Channel: "builders"},
			"direct_missing",
		)
		if err == nil {
			t.Fatal("GetDirectRoom(invalid id) error = nil, want non-nil")
		}
		_, err = globalDB.GetWork(testutil.Context(t), networkStoreTestWorkspaceID, "")
		if err == nil {
			t.Fatal("GetWork(empty) error = nil, want non-nil")
		}
		_, err = globalDB.GetWork(testutil.Context(t), networkStoreTestWorkspaceID, "work_missing")
		if !errors.Is(err, store.ErrNetworkConversationNotFound) {
			t.Fatalf("GetWork(missing) error = %v, want ErrNetworkConversationNotFound", err)
		}

		_, err = globalDB.ListThreads(
			testutil.Context(t),
			store.NetworkChannelRef{WorkspaceID: networkStoreTestWorkspaceID, Channel: "builders"},
			store.NetworkThreadQuery{
				Limit: 10,
				After: "thread_missing",
			},
		)
		if err == nil {
			t.Fatal("ListThreads(missing cursor) error = nil, want non-nil")
		}
		_, err = globalDB.ListDirectRooms(
			testutil.Context(t),
			store.NetworkChannelRef{WorkspaceID: networkStoreTestWorkspaceID, Channel: "builders"},
			store.NetworkDirectRoomQuery{
				PeerID: "other.sess-peer",
				Limit:  10,
				After:  directID,
			},
		)
		if err == nil {
			t.Fatal("ListDirectRooms(missing cursor for peer) error = nil, want non-nil")
		}
		ref := store.NetworkConversationRef{
			WorkspaceID: networkStoreTestWorkspaceID,
			Channel:     "builders",
			Surface:     store.NetworkSurfaceThread,
			ThreadID:    "thread_query_error",
		}
		_, err = globalDB.ListConversationMessages(testutil.Context(t), ref, store.NetworkConversationMessageQuery{
			BeforeMessageID: "msg_missing",
			Limit:           10,
		})
		if err == nil {
			t.Fatal("ListConversationMessages(missing cursor) error = nil, want non-nil")
		}
		_, err = globalDB.ListConversationMessages(testutil.Context(t), ref, store.NetworkConversationMessageQuery{
			BeforeMessageID: "msg_query_error",
			AfterMessageID:  "msg_query_error",
			Limit:           10,
		})
		if err == nil {
			t.Fatal("ListConversationMessages(conflicting cursors) error = nil, want non-nil")
		}
	})
}

func TestGlobalDBWriteConversationMessageIdempotencyAndRollback(t *testing.T) {
	t.Parallel()

	t.Run(
		"Should process duplicates before lifecycle mutation and reject terminal continuations atomically",
		func(t *testing.T) {
			t.Parallel()

			globalDB := openTestGlobalDB(t)
			startedAt := time.Date(2026, 5, 5, 16, 0, 0, 0, time.UTC)
			opening := threadMessage("msg_work_open", "thread_store_work", "coder.sess-abc", "please review", startedAt)
			opening.PeerTo = "reviewer.sess-xyz"
			opening.WorkID = "work_thread_review"

			result, err := globalDB.WriteConversationMessage(testutil.Context(t), opening)
			if err != nil {
				t.Fatalf("WriteConversationMessage(opening) error = %v", err)
			}
			if !result.WorkOpened || result.WorkState != store.NetworkWorkStateSubmitted {
				t.Fatalf("opening work result = %#v, want submitted open work", result)
			}

			duplicateOpening := opening
			duplicateOpening.Timestamp = startedAt.Add(time.Hour)
			duplicate, err := globalDB.WriteConversationMessage(testutil.Context(t), duplicateOpening)
			if err != nil {
				t.Fatalf("WriteConversationMessage(duplicate opening) error = %v", err)
			}
			if !duplicate.Duplicate || duplicate.WorkOpened || duplicate.WorkTransitioned {
				t.Fatalf("duplicate opening result = %#v, want duplicate without lifecycle mutation", duplicate)
			}

			completed := threadTraceMessage(
				"msg_work_complete",
				"thread_store_work",
				"reviewer.sess-xyz",
				"work_thread_review",
				store.NetworkWorkStateCompleted,
				startedAt.Add(time.Minute),
			)
			completedResult, err := globalDB.WriteConversationMessage(testutil.Context(t), completed)
			if err != nil {
				t.Fatalf("WriteConversationMessage(completed) error = %v", err)
			}
			if !completedResult.WorkTransitioned || completedResult.WorkState != store.NetworkWorkStateCompleted {
				t.Fatalf("completed result = %#v, want completed transition", completedResult)
			}

			duplicateCompleted, err := globalDB.WriteConversationMessage(testutil.Context(t), completed)
			if err != nil {
				t.Fatalf("WriteConversationMessage(duplicate completed) error = %v", err)
			}
			if !duplicateCompleted.Duplicate {
				t.Fatalf("duplicate completed result = %#v, want duplicate", duplicateCompleted)
			}

			rejected := threadTraceMessage(
				"msg_work_after_terminal",
				"thread_store_work",
				"reviewer.sess-xyz",
				"work_thread_review",
				store.NetworkWorkStateWorking,
				startedAt.Add(2*time.Minute),
			)
			_, err = globalDB.WriteConversationMessage(testutil.Context(t), rejected)
			if !errors.Is(err, store.ErrNetworkWorkClosed) {
				t.Fatalf("WriteConversationMessage(after terminal) error = %v, want ErrNetworkWorkClosed", err)
			}

			thread, err := globalDB.GetThread(
				testutil.Context(t),
				store.NetworkChannelRef{WorkspaceID: networkStoreTestWorkspaceID, Channel: "builders"},
				"thread_store_work",
			)
			if err != nil {
				t.Fatalf("GetThread() error = %v", err)
			}
			if got, want := thread.MessageCount, 2; got != want {
				t.Fatalf("thread.MessageCount = %d, want %d", got, want)
			}
			if got, want := thread.OpenWorkCount, 0; got != want {
				t.Fatalf("thread.OpenWorkCount = %d, want %d", got, want)
			}
			if got := networkRowCount(
				t,
				globalDB,
				"network_timeline_log",
				"message_id",
				"msg_work_after_terminal",
			); got != 0 {
				t.Fatalf("rolled-back message count = %d, want 0", got)
			}
			if got := networkRowCount(
				t,
				globalDB,
				"network_audit_log",
				"message_id",
				"msg_work_after_terminal",
			); got != 0 {
				t.Fatalf("rolled-back audit count = %d, want 0", got)
			}
		},
	)

	t.Run("Should roll back message and side effects when direct room binding fails", func(t *testing.T) {
		t.Parallel()

		globalDB := openTestGlobalDB(t)
		directID, _, _, err := store.NetworkDirectRoomIdentity(
			networkStoreTestWorkspaceID,
			"builders",
			"coder.sess-abc",
			"reviewer.sess-xyz",
		)
		if err != nil {
			t.Fatalf("NetworkDirectRoomIdentity() error = %v", err)
		}
		message := store.NetworkConversationMessage{
			MessageID:   "msg_direct_collision_rollback",
			SessionID:   "sess-direct-collision",
			WorkspaceID: networkStoreTestWorkspaceID,
			Channel:     "builders",
			Surface:     store.NetworkSurfaceDirect,
			DirectID:    directID,
			Direction:   "sent",
			PeerFrom:    "coder.sess-abc",
			PeerTo:      "other.sess-peer",
			Kind:        store.NetworkKindSay,
			Text:        "wrong room",
			PreviewText: "wrong room",
			Body:        []byte(`{"text":"wrong room"}`),
			Timestamp:   time.Date(2026, 5, 5, 16, 30, 0, 0, time.UTC),
		}

		_, err = globalDB.WriteConversationMessage(testutil.Context(t), message)
		if !errors.Is(err, store.ErrNetworkDirectRoomCollision) {
			t.Fatalf("WriteConversationMessage(direct collision) error = %v, want ErrNetworkDirectRoomCollision", err)
		}
		if got := networkRowCount(t, globalDB, "network_timeline_log", "message_id", message.MessageID); got != 0 {
			t.Fatalf("rolled-back direct message count = %d, want 0", got)
		}
		if got := networkRowCount(t, globalDB, "network_direct_rooms", "channel", "builders"); got != 0 {
			t.Fatalf("direct room count = %d, want 0", got)
		}
		if got := networkRowCount(t, globalDB, "network_audit_log", "message_id", message.MessageID); got != 0 {
			t.Fatalf("rolled-back direct audit count = %d, want 0", got)
		}
	})
}

func TestGlobalDBWriteConversationMessageRejectsRawClaimTokens(t *testing.T) {
	t.Parallel()

	t.Run("Should reject raw claim tokens before persisting message or audit rows", func(t *testing.T) {
		t.Parallel()

		globalDB := openTestGlobalDB(t)
		message := threadMessage(
			"msg_raw_claim_token",
			"thread_store_redaction",
			"coder.sess-abc",
			"agh_claim_NET05TOKEN123",
			time.Date(2026, 5, 5, 17, 0, 0, 0, time.UTC),
		)

		_, err := globalDB.WriteConversationMessage(testutil.Context(t), message)
		if err == nil {
			t.Fatal("WriteConversationMessage(raw claim token) error = nil, want non-nil")
		}
		if got := networkRowCount(t, globalDB, "network_timeline_log", "message_id", message.MessageID); got != 0 {
			t.Fatalf("raw-token message count = %d, want 0", got)
		}
		if got := networkRowCount(t, globalDB, "network_audit_log", "message_id", message.MessageID); got != 0 {
			t.Fatalf("raw-token audit count = %d, want 0", got)
		}
	})

	t.Run("Should reject raw claim tokens in timeline text fields even when body is redacted", func(t *testing.T) {
		t.Parallel()

		globalDB := openTestGlobalDB(t)
		message := threadMessage(
			"msg_raw_claim_token_text",
			"thread_store_redaction",
			"coder.sess-abc",
			"agh_claim_NET05TEXT123",
			time.Date(2026, 5, 5, 17, 30, 0, 0, time.UTC),
		)
		message.Body = []byte(`{"text":"[redacted]"}`)

		_, err := globalDB.WriteConversationMessage(testutil.Context(t), message)
		if err == nil {
			t.Fatal("WriteConversationMessage(raw text claim token) error = nil, want non-nil")
		}
		assertNoTimelineOrAuditRows(t, globalDB, message.MessageID)
	})

	t.Run("Should reject raw claim tokens in direct audit writes", func(t *testing.T) {
		t.Parallel()

		globalDB := openTestGlobalDB(t)
		err := globalDB.WriteNetworkAudit(testutil.Context(t), store.NetworkAuditEntry{
			WorkspaceID: networkStoreTestWorkspaceID,
			SessionID:   "sess-audit-token",
			Direction:   "rejected",
			Kind:        store.NetworkKindSay,
			Channel:     "builders",
			PeerFrom:    "coder.sess-abc",
			MessageID:   "msg_audit_token",
			Reason:      "agh_claim_NET05TOKEN123",
			Size:        1,
		})
		if err == nil {
			t.Fatal("WriteNetworkAudit(raw claim token) error = nil, want non-nil")
		}
		if got := networkRowCount(t, globalDB, "network_audit_log", "message_id", "msg_audit_token"); got != 0 {
			t.Fatalf("raw-token audit count = %d, want 0", got)
		}
	})
}

func threadMessage(
	messageID string,
	threadID string,
	peerFrom string,
	text string,
	timestamp time.Time,
) store.NetworkConversationMessage {
	return store.NetworkConversationMessage{
		MessageID:   messageID,
		SessionID:   "sess-" + messageID,
		WorkspaceID: networkStoreTestWorkspaceID,
		Channel:     "builders",
		Surface:     store.NetworkSurfaceThread,
		ThreadID:    threadID,
		Direction:   "sent",
		PeerFrom:    peerFrom,
		Kind:        store.NetworkKindSay,
		Text:        text,
		PreviewText: text,
		Body:        []byte(`{"text":"` + text + `"}`),
		Timestamp:   timestamp,
	}
}

func threadTraceMessage(
	messageID string,
	threadID string,
	peerFrom string,
	workID string,
	state string,
	timestamp time.Time,
) store.NetworkConversationMessage {
	return store.NetworkConversationMessage{
		MessageID:   messageID,
		SessionID:   "sess-" + messageID,
		WorkspaceID: networkStoreTestWorkspaceID,
		Channel:     "builders",
		Surface:     store.NetworkSurfaceThread,
		ThreadID:    threadID,
		Direction:   "received",
		PeerFrom:    peerFrom,
		Kind:        store.NetworkKindTrace,
		WorkID:      workID,
		PreviewText: state,
		Body:        []byte(`{"state":"` + state + `"}`),
		Timestamp:   timestamp,
	}
}

func threadReceiptMessage(
	messageID string,
	threadID string,
	peerFrom string,
	workID string,
	status string,
	timestamp time.Time,
) store.NetworkConversationMessage {
	return store.NetworkConversationMessage{
		MessageID:   messageID,
		SessionID:   "sess-" + messageID,
		WorkspaceID: networkStoreTestWorkspaceID,
		Channel:     "builders",
		Surface:     store.NetworkSurfaceThread,
		ThreadID:    threadID,
		Direction:   "received",
		PeerFrom:    peerFrom,
		Kind:        store.NetworkKindReceipt,
		WorkID:      workID,
		PreviewText: status,
		Body:        []byte(`{"status":"` + status + `"}`),
		Timestamp:   timestamp,
	}
}

func writeDirectMessage(
	t *testing.T,
	globalDB *GlobalDB,
	messageID string,
	peerFrom string,
	peerTo string,
	text string,
	timestamp time.Time,
) (string, error) {
	t.Helper()

	directID, _, _, err := store.NetworkDirectRoomIdentity(networkStoreTestWorkspaceID, "builders", peerFrom, peerTo)
	if err != nil {
		return "", err
	}
	_, err = globalDB.WriteConversationMessage(testutil.Context(t), store.NetworkConversationMessage{
		MessageID:   messageID,
		SessionID:   "sess-" + messageID,
		WorkspaceID: networkStoreTestWorkspaceID,
		Channel:     "builders",
		Surface:     store.NetworkSurfaceDirect,
		DirectID:    directID,
		Direction:   "sent",
		PeerFrom:    peerFrom,
		PeerTo:      peerTo,
		Kind:        store.NetworkKindSay,
		Text:        text,
		PreviewText: text,
		Body:        []byte(`{"text":"` + text + `"}`),
		Timestamp:   timestamp,
	})
	return directID, err
}

func messageIDs(messages []store.NetworkConversationMessage) []string {
	ids := make([]string, 0, len(messages))
	for _, message := range messages {
		ids = append(ids, message.MessageID)
	}
	return ids
}

func sameStrings(got []string, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for index := range got {
		if got[index] != want[index] {
			return false
		}
	}
	return true
}

func assertNoTimelineOrAuditRows(t *testing.T, globalDB *GlobalDB, messageID string) {
	t.Helper()

	if got := networkRowCount(t, globalDB, "network_timeline_log", "message_id", messageID); got != 0 {
		t.Fatalf("timeline count for %q = %d, want 0", messageID, got)
	}
	if got := networkRowCount(t, globalDB, "network_audit_log", "message_id", messageID); got != 0 {
		t.Fatalf("audit count for %q = %d, want 0", messageID, got)
	}
}

func networkRowCount(t *testing.T, globalDB *GlobalDB, table string, column string, value string) int {
	t.Helper()

	var count int
	if err := globalDB.db.QueryRowContext(
		testutil.Context(t),
		`SELECT COUNT(*) FROM `+table+` WHERE `+column+` = ?`,
		value,
	).Scan(&count); err != nil {
		t.Fatalf("count %s.%s error = %v", table, column, err)
	}
	return count
}
