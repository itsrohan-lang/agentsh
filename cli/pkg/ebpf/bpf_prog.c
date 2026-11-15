// +build ignore

#include <linux/bpf.h>
#include <linux/in.h>
#include <linux/in6.h>
#include <linux/if_ether.h>
#include <linux/ip.h>
#include <linux/ipv6.h>
#include <linux/tcp.h>
#include <linux/udp.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_endian.h>

// BPF Map to store blocked ports (e.g., 5432 for Postgres, 27017 for Mongo, 3306 for MySQL, 6379 for Redis)
struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(max_entries, 256);
    __type(key, __u16);
    __type(value, __u8);
} blocked_ports SEC(".maps");

// BPF Map to record actual dropped packets for telemetry
struct {
    __uint(type, BPF_MAP_TYPE_ARRAY);
    __uint(max_entries, 1);
    __type(key, __u32);
    __type(value, __u64);
} drop_counter SEC(".maps");

SEC("cgroup_skb/egress")
int block_outbound_db(struct __sk_buff *skb) {
    // agentsh Kernel Egress Interceptor
    // Attached directly to the container's cgroup v2 hierarchy
    // to physically block unauthorized database/service egress traffic.

    __u32 proto = 0;
    __u32 l4_offset = 0;

    // Read IP version (first nibble of IP header)
    __u8 ver_byte;
    if (bpf_skb_load_bytes(skb, 0, &ver_byte, sizeof(ver_byte)) < 0) {
        return 1; // Allow on packet read error
    }

    __u8 ip_version = (ver_byte >> 4) & 0x0F;

    if (ip_version == 4) {
        struct iphdr iph;
        if (bpf_skb_load_bytes(skb, 0, &iph, sizeof(iph)) < 0) {
            return 1;
        }

        proto = iph.protocol;
        l4_offset = iph.ihl * 4;
        if (l4_offset < sizeof(iph)) {
            return 1;
        }
    } else if (ip_version == 6) {
        struct ipv6hdr ip6h;
        if (bpf_skb_load_bytes(skb, 0, &ip6h, sizeof(ip6h)) < 0) {
            return 1;
        }

        proto = ip6h.nexthdr;
        l4_offset = sizeof(ip6h);
    } else {
        return 1; // Non-IP packet, allow
    }

    // Only inspect TCP and UDP egress traffic
    if (proto != IPPROTO_TCP && proto != IPPROTO_UDP) {
        return 1;
    }

    // Both TCP and UDP have the destination port at offset +2 in their L4 header
    __be16 dest_port_be;
    if (bpf_skb_load_bytes(skb, l4_offset + 2, &dest_port_be, sizeof(dest_port_be)) < 0) {
        return 1;
    }

    __u16 dest_port = bpf_ntohs(dest_port_be);

    // Check if the destination port is in our blocked map
    __u8 *is_blocked = bpf_map_lookup_elem(&blocked_ports, &dest_port);
    if (is_blocked && *is_blocked == 1) {
        // Increment the drop counter for telemetry
        __u32 key = 0;
        __u64 *count = bpf_map_lookup_elem(&drop_counter, &key);
        if (count) {
            __sync_fetch_and_add(count, 1);
        }

        bpf_printk("agentsh: BLOCKED outbound connection to port %d\n", dest_port);
        return 0; // DROP PACKET at the kernel level
    }

    return 1; // ALLOW PACKET
}

char __license[] SEC("license") = "Dual MIT/GPL";
