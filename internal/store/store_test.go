package store

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/yikkrrtykj/vps-netwatch/internal/model"
)

func TestSaveCollectorPushMergesVPSNodes(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ctx := context.Background()
	if err := db.SaveCollectorPush(ctx, model.CollectorPush{
		CollectorID: "vps-a",
		Timestamp:   time.Now().UTC(),
		VPSNodes: []model.VPSNodeStatus{
			{ID: "vps-a", Name: "A", UpdatedAt: time.Now().UTC()},
		},
	}); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveCollectorPush(ctx, model.CollectorPush{
		CollectorID: "vps-b",
		Timestamp:   time.Now().UTC(),
		VPSNodes: []model.VPSNodeStatus{
			{ID: "vps-b", Name: "B", UpdatedAt: time.Now().UTC()},
		},
	}); err != nil {
		t.Fatal(err)
	}

	var nodes []model.VPSNodeStatus
	ok, err := db.LoadState(ctx, "vps_nodes", &nodes)
	if err != nil {
		t.Fatal(err)
	}
	if !ok || len(nodes) != 2 {
		t.Fatalf("expected two merged nodes, ok=%v nodes=%#v", ok, nodes)
	}
}

func TestSaveCollectorPushReplacesConnectionSnapshot(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ctx := context.Background()
	if err := db.SaveCollectorPush(ctx, model.CollectorPush{
		CollectorID:            "lan",
		ConnectionControllers: []string{"proxy"},
		Connections: []model.Connection{
			{Controller: "proxy", ID: "a", DestIP: "203.0.113.1", UpdatedAt: time.Now().UTC()},
			{Controller: "proxy", ID: "b", DestIP: "203.0.113.2", UpdatedAt: time.Now().UTC()},
		},
	}); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveCollectorPush(ctx, model.CollectorPush{
		CollectorID:            "lan",
		ConnectionControllers: []string{"proxy"},
		Connections: []model.Connection{
			{Controller: "proxy", ID: "b", DestIP: "203.0.113.2", UpdatedAt: time.Now().UTC()},
		},
	}); err != nil {
		t.Fatal(err)
	}

	connections, err := db.LatestConnections(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(connections) != 1 || connections[0].ID != "b" {
		t.Fatalf("expected stale connection to be removed, got %#v", connections)
	}

	if err := db.SaveCollectorPush(ctx, model.CollectorPush{
		CollectorID:            "lan",
		ConnectionControllers: []string{"proxy"},
	}); err != nil {
		t.Fatal(err)
	}
	connections, err = db.LatestConnections(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(connections) != 0 {
		t.Fatalf("expected empty snapshot to clear controller, got %#v", connections)
	}
}

func TestSaveCollectorPushMergesEgressAndLatencyByCollector(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ctx := context.Background()
	if err := db.SaveCollectorPush(ctx, model.CollectorPush{
		CollectorID: "vps-a",
		Egress:      &model.EgressResult{IP: "198.51.100.1", CheckedAt: time.Now().UTC()},
		Latency: []model.ProbeResult{
			{Name: "dashboard", Host: "example.com", Port: 443, Protocol: "tcp", OK: true},
		},
	}); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveCollectorPush(ctx, model.CollectorPush{
		CollectorID: "vps-b",
		Egress:      &model.EgressResult{IP: "198.51.100.2", CheckedAt: time.Now().UTC()},
		Latency: []model.ProbeResult{
			{Name: "dashboard", Host: "example.com", Port: 443, Protocol: "tcp", OK: false},
		},
	}); err != nil {
		t.Fatal(err)
	}

	var egress []model.EgressResult
	ok, err := db.LoadState(ctx, "egress", &egress)
	if err != nil {
		t.Fatal(err)
	}
	if !ok || len(egress) != 2 {
		t.Fatalf("expected two egress results, ok=%v egress=%#v", ok, egress)
	}

	var latency []model.ProbeResult
	ok, err = db.LoadState(ctx, "latency", &latency)
	if err != nil {
		t.Fatal(err)
	}
	if !ok || len(latency) != 2 {
		t.Fatalf("expected two latency results, ok=%v latency=%#v", ok, latency)
	}
	if latency[0].CollectorID == "" || latency[1].CollectorID == "" {
		t.Fatalf("collector id was not applied to latency results: %#v", latency)
	}
}

func TestSaveCollectorPushReplacesEgressForSameCollector(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ctx := context.Background()
	if err := db.SaveCollectorPush(ctx, model.CollectorPush{
		CollectorID: "vps-a",
		Egress:      &model.EgressResult{IP: "198.51.100.1", CheckedAt: time.Now().UTC()},
	}); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveCollectorPush(ctx, model.CollectorPush{
		CollectorID: "vps-a",
		Egress:      &model.EgressResult{IP: "198.51.100.99", CheckedAt: time.Now().UTC()},
	}); err != nil {
		t.Fatal(err)
	}

	var egress []model.EgressResult
	ok, err := db.LoadState(ctx, "egress", &egress)
	if err != nil {
		t.Fatal(err)
	}
	if !ok || len(egress) != 1 || egress[0].IP != "198.51.100.99" {
		t.Fatalf("expected same collector egress to be replaced, ok=%v egress=%#v", ok, egress)
	}
}

func TestSaveCollectorPushMigratesSingleEgressState(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ctx := context.Background()
	if err := db.SaveState(ctx, "egress", model.EgressResult{
		CollectorID: "vps-a",
		IP:          "198.51.100.1",
		CheckedAt:   time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveCollectorPush(ctx, model.CollectorPush{
		CollectorID: "vps-b",
		Egress:      &model.EgressResult{IP: "198.51.100.2", CheckedAt: time.Now().UTC()},
	}); err != nil {
		t.Fatal(err)
	}

	var egress []model.EgressResult
	ok, err := db.LoadState(ctx, "egress", &egress)
	if err != nil {
		t.Fatal(err)
	}
	if !ok || len(egress) != 2 {
		t.Fatalf("expected old single egress state to migrate, ok=%v egress=%#v", ok, egress)
	}
}
