package ebpf

import (
	"encoding/binary"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/asm"
)

// DefaultBlockedPorts contains common database and service ports blocked by default.
var DefaultBlockedPorts = []uint16{
	5432,  // PostgreSQL
	27017, // MongoDB
	3306,  // MySQL
	6379,  // Redis
}

// PortToKey converts a standard host port number to network byte order key for the eBPF map.
func PortToKey(port uint16) [2]byte {
	var buf [2]byte
	binary.BigEndian.PutUint16(buf[:], port)
	return buf
}

// KeyToPort converts a 2-byte network key back to host port.
func KeyToPort(key [2]byte) uint16 {
	return binary.BigEndian.Uint16(key[:])
}

// NewEgressFilterCollectionSpec builds the in-memory eBPF program specification for cgroup_skb/egress.
func NewEgressFilterCollectionSpec() *ebpf.CollectionSpec {
	return &ebpf.CollectionSpec{
		Maps: map[string]*ebpf.MapSpec{
			"blocked_ports": {
				Name:       "blocked_ports",
				Type:       ebpf.Hash,
				KeySize:    2, // __u16 port (big-endian)
				ValueSize:  1, // __u8 flag (1 = blocked)
				MaxEntries: 256,
			},
			"drop_counter": {
				Name:       "drop_counter",
				Type:       ebpf.Array,
				KeySize:    4, // __u32 index 0
				ValueSize:  8, // __u64 dropped packets count
				MaxEntries: 1,
			},
		},
		Programs: map[string]*ebpf.ProgramSpec{
			"block_outbound_db": {
				Name:        "block_outbound_db",
				Type:        ebpf.CGroupSKB,
				AttachType:  ebpf.AttachCGroupInetEgress,
				SectionName: "cgroup_skb/egress",
				License:     "Dual MIT/GPL",
				Instructions: asm.Instructions{
					// Callee-saved R6 holds skb pointer across helper calls
					asm.Mov.Reg(asm.R6, asm.R1),

					// Load 20 bytes of IPv4 header into stack at R10 - 24
					// bpf_skb_load_bytes(skb, 0, R10-24, 20)
					asm.Mov.Reg(asm.R1, asm.R6),
					asm.Mov.Imm(asm.R2, 0),
					asm.Mov.Reg(asm.R3, asm.R10),
					asm.Add.Imm(asm.R3, -24),
					asm.Mov.Imm(asm.R4, 20),
					asm.FnSkbLoadBytes.Call(),
					asm.JNE.Imm(asm.R0, 0, "allow"),

					// Check IP version (R10-24 byte 0 >> 4 == 4)
					asm.LoadMem(asm.R1, asm.R10, -24, asm.Byte),
					asm.RSh.Imm(asm.R1, 4),
					asm.JNE.Imm(asm.R1, 4, "allow"),

					// Check protocol (R10-24 byte 9: IPPROTO_TCP=6, IPPROTO_UDP=17)
					asm.LoadMem(asm.R1, asm.R10, -15, asm.Byte),
					asm.JEq.Imm(asm.R1, 6, "parse_l4"),
					asm.JEq.Imm(asm.R1, 17, "parse_l4"),
					asm.Ja.Label("allow"),

					// Parse L4: IHL is (R10-24 byte 0 & 0x0F) * 4
					asm.LoadMem(asm.R2, asm.R10, -24, asm.Byte).Sym("parse_l4"),
					asm.And.Imm(asm.R2, 0x0F),
					asm.LSh.Imm(asm.R2, 2),
					// Destination port is at IHL + 2
					asm.Add.Imm(asm.R2, 2),

					// Read 2-byte destination port into stack at R10 - 28
					// bpf_skb_load_bytes(skb, IHL+2, R10-28, 2)
					asm.Mov.Reg(asm.R1, asm.R6),
					asm.Mov.Reg(asm.R3, asm.R10),
					asm.Add.Imm(asm.R3, -28),
					asm.Mov.Imm(asm.R4, 2),
					asm.FnSkbLoadBytes.Call(),
					asm.JNE.Imm(asm.R0, 0, "allow"),

					// Lookup destination port (R10-28) in blocked_ports map
					// bpf_map_lookup_elem(blocked_ports, R10-28)
					asm.LoadMapPtr(asm.R1, 0).WithReference("blocked_ports"),
					asm.Mov.Reg(asm.R2, asm.R10),
					asm.Add.Imm(asm.R2, -28),
					asm.FnMapLookupElem.Call(),
					asm.JEq.Imm(asm.R0, 0, "allow"),

					// Check if map value is 1
					asm.LoadMem(asm.R1, asm.R0, 0, asm.Byte),
					asm.JNE.Imm(asm.R1, 1, "allow"),

					// Port is blocked! Increment drop_counter[0]
					asm.LoadMapPtr(asm.R1, 0).WithReference("drop_counter"),
					asm.StoreImm(asm.R10, -32, 0, asm.Word), // key = 0
					asm.Mov.Reg(asm.R2, asm.R10),
					asm.Add.Imm(asm.R2, -32),
					asm.FnMapLookupElem.Call(),
					asm.JEq.Imm(asm.R0, 0, "drop_packet"),
					// Atomic 64-bit increment: *R0 += 1
					asm.Mov.Imm(asm.R1, 1),
					asm.StoreXAdd(asm.R0, asm.R1, asm.DWord),

					// Return 0: DROP packet at kernel level
					asm.Mov.Imm(asm.R0, 0).Sym("drop_packet"),
					asm.Return(),

					// Return 1: ALLOW packet
					asm.Mov.Imm(asm.R0, 1).Sym("allow"),
					asm.Return(),
				},
			},
		},
	}
}
