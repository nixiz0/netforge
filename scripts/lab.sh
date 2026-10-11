#!/usr/bin/env bash
# Three "machines" inside one Linux box, to test the VPN (steps 6.4 and 7).
#
#   root namespace   = VPN client   veth0 192.168.100.1                     TUN nf0 10.10.0.1
#   namespace nfpeer = VPN server   veth1 192.168.100.2, veth2 203.0.113.1  TUN nf0 10.10.0.2
#   namespace nfnet  = a host "on the internet", behind the server: veth3 203.0.113.10
#   namespace nfc2   = a second VPN client (step 9): veth5 192.168.101.1 (server side: veth4 192.168.101.2), nf0 10.10.0.3
#
# Each namespace has its own interfaces, routes and settings, so both TUN interfaces can be nf0.
#
# Usage, as root, in this order:
#   lab.sh up        create the namespaces and the two veth cables
#   lab.sh tun       give both nf0 their addresses (after the netforge server and client started)
#   lab.sh route     client: send 203.0.113.0/24 into the tunnel; host: route back to 10.10.0.0/24
#   lab.sh forward   server: turn on IP forwarding, so it routes between nf0 and veth2
#   lab.sh nat       host: forget the return route; server: NAT the VPN subnet behind 203.0.113.1
#   lab.sh full      client: send ALL IPv4 traffic into the tunnel (WSL loses internet until full-off)
#   lab.sh full-off  client: back to normal routing
#   lab.sh dns       client: show where DNS queries go (through nf0, or leaking outside the VPN)
#   lab.sh up2       create the second client namespace and its cable to the server
#   lab.sh tun2      give the second client's nf0 its address (after its netforge client started)
#   lab.sh down      remove everything (stop the netforge processes first)
#
# Theory: _networking-lessons/4-advanced-network-concepts.md#linux-network-namespaces
# Theory: _networking-lessons/4-advanced-network-concepts.md#virtual-ethernet-pairs
set -euo pipefail

case "${1:-}" in
up)
	ip netns add nfpeer
	ip netns add nfnet
	# Cable 1: client <-> VPN server (the outer network the tunnel travels on).
	ip link add veth0 type veth peer name veth1
	ip link set veth1 netns nfpeer
	ip addr add 192.168.100.1/24 dev veth0
	ip link set veth0 up
	ip netns exec nfpeer ip addr add 192.168.100.2/24 dev veth1
	ip netns exec nfpeer ip link set veth1 up
	ip netns exec nfpeer ip link set lo up
	# Cable 2: VPN server <-> a host behind it.
	ip link add veth2 netns nfpeer type veth peer name veth3 netns nfnet
	ip netns exec nfpeer ip addr add 203.0.113.1/24 dev veth2
	ip netns exec nfpeer ip link set veth2 up
	ip netns exec nfnet ip addr add 203.0.113.10/24 dev veth3
	ip netns exec nfnet ip link set veth3 up
	ip netns exec nfnet ip link set lo up
	echo "lab up: client 192.168.100.1 <-> server 192.168.100.2 | server 203.0.113.1 <-> host 203.0.113.10"
	;;
tun)
	# "replace", not "add": safe to run again after restarting one end (its nf0 comes back empty).
	# MTU 1440 = 1500 - 60 bytes of overhead (IPv4 20 + UDP 8 + tunnel header 4 + nonce 12 + tag 16),
	# so every encrypted outer packet still fits one 1500-byte frame.
	# Theory: _networking-lessons/7-vpn-architecture.md#vpn-mtu
	ip addr replace 10.10.0.1/24 dev nf0
	ip link set nf0 up mtu 1440
	ip netns exec nfpeer ip addr replace 10.10.0.2/24 dev nf0
	ip netns exec nfpeer ip link set nf0 up mtu 1440
	echo "TUN up: client nf0 10.10.0.1 <-> server nf0 10.10.0.2 (MTU 1440)"
	;;
route)
	# Split tunnel: only this network goes through the VPN; everything else keeps the normal path.
	# Theory: _networking-lessons/7-vpn-architecture.md#split-tunnel
	ip route add 203.0.113.0/24 dev nf0
	# Return path without NAT: the host learns that 10.10.0.0/24 is behind the VPN server.
	# Theory: _networking-lessons/7-vpn-architecture.md#routing-instead-of-nat
	ip netns exec nfnet ip route add 10.10.0.0/24 via 203.0.113.1
	echo "routes: client 203.0.113.0/24 -> nf0 | host 10.10.0.0/24 -> via 203.0.113.1"
	;;
forward)
	# Off by default: a Linux box drops packets that are not addressed to itself.
	# Theory: _networking-lessons/7-vpn-architecture.md#ip-forwarding
	ip netns exec nfpeer sysctl -qw net.ipv4.ip_forward=1
	echo "server: IP forwarding on"
	;;
nat)
	command -v iptables >/dev/null || { echo "iptables is missing: sudo apt install -y iptables" >&2; exit 1; }
	# The host no longer knows 10.10.0.0/24, like any server on the real internet.
	ip netns exec nfnet ip route del 10.10.0.0/24 via 203.0.113.1 2>/dev/null || true
	# Source NAT: packets from the VPN subnet leaving through veth2 take the server's own address.
	# The kernel's connection tracking remembers each mapping and translates the replies back.
	# Theory: _networking-lessons/7-vpn-architecture.md#nat-on-a-vpn-server
	# Theory: _networking-lessons/4-advanced-network-concepts.md#connection-tracking
	ip netns exec nfpeer iptables -t nat -A POSTROUTING -s 10.10.0.0/24 -o veth2 -j MASQUERADE
	echo "NAT: 10.10.0.0/24 leaves the server as 203.0.113.1 | host has no route to 10.10.0.0/24"
	;;
full)
	# Two halves, 0.0.0.0/1 and 128.0.0.0/1, cover every IPv4 address. Each is more specific than the
	# default route (/0), so it wins by longest-prefix match without deleting it: easy to undo.
	# Theory: _networking-lessons/7-vpn-architecture.md#full-tunnel
	# Theory: _networking-lessons/1-network-fundamentals.md#longest-prefix-match
	ip route add 0.0.0.0/1 dev nf0
	ip route add 128.0.0.0/1 dev nf0
	# The tunnel's own packets to the server must NOT enter nf0, or they would loop forever. Here the
	# server (192.168.100.2) is on the connected /24 of veth0, more specific than /1, so it stays safe.
	# A real client adds a /32 route to the server via its old gateway before switching.
	# Theory: _networking-lessons/7-vpn-architecture.md#avoid-routing-the-tunnel-into-itself
	echo "full tunnel on: $(ip route get 1.1.1.1 | head -1)"
	echo "server stays outside: $(ip route get 192.168.100.2 | head -1)"
	;;
full-off)
	ip route del 0.0.0.0/1 dev nf0 2>/dev/null || true
	ip route del 128.0.0.0/1 dev nf0 2>/dev/null || true
	echo "full tunnel off: $(ip route get 1.1.1.1 | head -1)"
	;;
dns)
	# Where do this machine's DNS questions go? Ask the kernel the route to each resolver it uses.
	# Not through nf0 = the lookups bypass the VPN, even in full-tunnel mode: a DNS leak.
	# Theory: _networking-lessons/7-vpn-architecture.md#dns-through-a-vpn
	# Theory: _networking-lessons/1-network-fundamentals.md#18-dns
	for ns in $(awk '/^nameserver/ {print $2}' /etc/resolv.conf); do
		r=$(ip route get "$ns" | head -1 | tr -s ' ')
		case "$r" in
		*" dev nf0 "*) echo "VPN   $ns -> $r" ;;
		*) echo "LEAK  $ns -> $r" ;;
		esac
	done
	;;
up2)
	# A second VPN client with its own cable to the server: the server now has two peers.
	# Theory: _networking-lessons/9-multi-peer-networking.md#hub-and-spoke
	ip netns add nfc2
	ip link add veth4 netns nfpeer type veth peer name veth5 netns nfc2
	ip netns exec nfpeer ip addr add 192.168.101.2/24 dev veth4
	ip netns exec nfpeer ip link set veth4 up
	ip netns exec nfc2 ip addr add 192.168.101.1/24 dev veth5
	ip netns exec nfc2 ip link set veth5 up
	ip netns exec nfc2 ip link set lo up
	echo "client 2: 192.168.101.1 <-> server 192.168.101.2"
	;;
tun2)
	ip netns exec nfc2 ip addr replace 10.10.0.3/24 dev nf0
	ip netns exec nfc2 ip link set nf0 up mtu 1440
	echo "TUN up: client 2 nf0 10.10.0.3 (MTU 1440)"
	;;
down)
	ip netns del nfc2 2>/dev/null || true
	ip netns del nfnet 2>/dev/null || true
	ip netns del nfpeer 2>/dev/null || true
	ip link del veth0 2>/dev/null || true
	echo "lab down"
	;;
*)
	echo "usage: $0 up|tun|route|forward|nat|full|full-off|dns|up2|tun2|down" >&2
	exit 2
	;;
esac
