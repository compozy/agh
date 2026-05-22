package globaldb

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/compozy/agh/internal/bridges"
	"github.com/compozy/agh/internal/store"
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
