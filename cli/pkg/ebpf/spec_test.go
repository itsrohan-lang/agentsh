package ebpf

import (
	"testing"
)

func TestNewEgressFilterCollectionSpec(t *testing.T) {
	spec := NewEgressFilterCollectionSpec()
	if spec == nil {
		t.Fatal("expected non-nil CollectionSpec")
	}

	if _, ok := spec.Maps["blocked_ports"]; !ok {
		t.Error("missing blocked_ports map spec")
	}
	if _, ok := spec.Maps["drop_counter"]; !ok {
		t.Error("missing drop_counter map spec")
	}
	prog, ok := spec.Programs["block_outbound_db"]
	if !ok {
		t.Fatal("missing block_outbound_db program spec")
	}

	if len(prog.Instructions) == 0 {
		t.Error("program has no instructions")
	}

	// Test port key conversions
	key := PortToKey(5432)
	port := KeyToPort(key)
	if port != 5432 {
		t.Errorf("PortToKey/KeyToPort mismatch: got %d, want 5432", port)
	}
}
