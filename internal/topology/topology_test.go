package topology

import "testing"

func TestDefaultTopologyIncludesExpectedPath(t *testing.T) {
	topo := Default()
	if len(topo.Nodes) < 8 {
		t.Fatalf("expected default topology nodes, got %d", len(topo.Nodes))
	}
	if topo.Nodes[0].ID != "terminal" || topo.Nodes[len(topo.Nodes)-1].ID != "game-server" {
		t.Fatalf("unexpected endpoints: %#v", topo.Nodes)
	}
}

