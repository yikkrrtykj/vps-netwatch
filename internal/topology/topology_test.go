package topology

import "testing"

func TestDefaultTopologyIncludesExpectedPath(t *testing.T) {
	topo := Default()
	if len(topo.Nodes) < 5 {
		t.Fatalf("expected default topology nodes, got %d", len(topo.Nodes))
	}
	if topo.Nodes[0].ID != "browser" || topo.Nodes[1].ID != "dashboard" {
		t.Fatalf("unexpected endpoints: %#v", topo.Nodes)
	}
}
