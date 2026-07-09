package globaldb

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/compozy/agh/internal/bridges"
	hookspkg "github.com/compozy/agh/internal/hooks"
	looppkg "github.com/compozy/agh/internal/loop"
	"github.com/compozy/agh/internal/store"
	workspacepkg "github.com/compozy/agh/internal/workspace"
)

func BenchmarkReplaceBridgeInstances(b *testing.B) {
	b.ReportAllocs()

	globalDB := openBenchmarkGlobalDB(b)
	ctx := context.Background()
	instances := benchmarkBridgeInstances(128)

	if err := globalDB.ReplaceBridgeInstances(ctx, instances); err != nil {
		b.Fatalf("ReplaceBridgeInstances(seed) error = %v", err)
	}

	b.ResetTimer()
	for b.Loop() {
		if err := globalDB.ReplaceBridgeInstances(ctx, instances); err != nil {
			b.Fatalf("ReplaceBridgeInstances() error = %v", err)
		}
	}
}

func BenchmarkReadNetworkMessagePersistedCursor(b *testing.B) {
	globalDB := openBenchmarkGlobalDB(b)
	ctx := context.Background()
	now := time.Date(2026, 7, 9, 12, 0, 0, 0, time.UTC)
	workspaceID := "ws-network-cursor-bench"
	if err := globalDB.InsertWorkspace(ctx, workspacepkg.Workspace{
		ID:        workspaceID,
		RootDir:   b.TempDir(),
		Name:      "Network cursor benchmark",
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		b.Fatalf("InsertWorkspace() error = %v", err)
	}
	const rowCount = 4096
	for index := range rowCount {
		if err := globalDB.WriteNetworkMessage(ctx, store.NetworkMessageEntry{
			MessageID:   fmt.Sprintf("msg-bench-%04d", index),
			WorkspaceID: workspaceID,
			Channel:     "bench",
			Surface:     store.NetworkSurfaceThread,
			ThreadID:    "thread_bench",
			Direction:   "received",
			PeerFrom:    "peer.bench",
			Kind:        store.NetworkKindSay,
			Body:        []byte(`{}`),
			Timestamp:   now.Add(time.Duration(index) * time.Millisecond),
		}); err != nil {
			b.Fatalf("WriteNetworkMessage(%d) error = %v", index, err)
		}
	}
	query := normalizedWatchEventsQuery{
		workspaceID: workspaceID,
		streams:     map[string]int64{looppkg.WatchEventsNetworkStream: 0},
		kinds:       []string{string(hookspkg.HookNetworkMessagePersisted)},
		limit:       looppkg.LoopMaxFanoutWidth,
	}

	b.Run("max_rowid", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			cursor, err := globalDB.readNetworkWatchEventsCursor(ctx, query)
			if err != nil {
				b.Fatalf("readNetworkWatchEventsCursor() error = %v", err)
			}
			if cursor != rowCount {
				b.Fatalf("cursor = %d, want %d", cursor, rowCount)
			}
		}
	})

	b.Run("projected_replay", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			cursor, err := globalDB.readProjectedNetworkWatchEventsCursor(ctx, query)
			if err != nil {
				b.Fatalf("readProjectedNetworkWatchEventsCursor() error = %v", err)
			}
			if cursor != rowCount {
				b.Fatalf("cursor = %d, want %d", cursor, rowCount)
			}
		}
	})
}

func openBenchmarkGlobalDB(b *testing.B) *GlobalDB {
	b.Helper()

	globalDB, err := OpenGlobalDB(
		context.Background(),
		filepath.Join(b.TempDir(), store.GlobalDatabaseName),
	)
	if err != nil {
		b.Fatalf("OpenGlobalDB() error = %v", err)
	}
	b.Cleanup(func() {
		if err := globalDB.Close(context.Background()); err != nil {
			b.Fatalf("Close() error = %v", err)
		}
	})
	return globalDB
}

func benchmarkBridgeInstances(count int) []bridges.BridgeInstance {
	instances := make([]bridges.BridgeInstance, 0, count)
	createdAt := time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC)

	for idx := range count {
		status := benchmarkBridgeStatus(idx)
		instances = append(instances, bridges.BridgeInstance{
			ID:            fmt.Sprintf("brg-%03d", idx),
			Scope:         bridges.ScopeGlobal,
			Platform:      fmt.Sprintf("platform-%02d", idx%8),
			ExtensionName: fmt.Sprintf("extension-%02d", idx%16),
			DisplayName:   fmt.Sprintf("Bridge %03d", idx),
			Source:        bridges.BridgeInstanceSourceDynamic,
			Enabled:       status != bridges.BridgeStatusDisabled,
			Status:        status,
			DMPolicy:      bridges.BridgeDMPolicyOpen,
			RoutingPolicy: bridges.RoutingPolicy{
				IncludePeer:   true,
				IncludeThread: idx%2 == 0,
				IncludeGroup:  idx%3 == 0,
			},
			ProviderConfig:   fmt.Appendf(nil, `{"tenant":"tenant-%02d","token":"tok-%03d"}`, idx%8, idx),
			DeliveryDefaults: []byte(`{"mode":"reply"}`),
			CreatedAt:        createdAt,
			UpdatedAt:        createdAt.Add(time.Duration(idx) * time.Second),
		})
	}

	return instances
}

func benchmarkBridgeStatus(idx int) bridges.BridgeStatus {
	switch idx % 4 {
	case 0:
		return bridges.BridgeStatusReady
	case 1:
		return bridges.BridgeStatusStarting
	case 2:
		return bridges.BridgeStatusDisabled
	default:
		return bridges.BridgeStatusDegraded
	}
}
