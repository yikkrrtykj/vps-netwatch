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

