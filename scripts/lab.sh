#!/usr/bin/env bash
# Three "machines" inside one Linux box, to test the VPN (steps 6.4 and 7).
#
#   root namespace   = VPN client   veth0 192.168.100.1                     TUN nf0 10.10.0.1
#   namespace nfpeer = VPN server   veth1 192.168.100.2, veth2 203.0.113.1  TUN nf0 10.10.0.2
#   namespace nfnet  = a host "on the internet", behind the server: veth3 203.0.113.10
#
# Each namespace has its own interfaces, routes and settings, so both TUN interfaces can be nf0.
#
# Usage, as root, in this order:
#   lab.sh up        create the namespaces and the two veth cables
#   lab.sh tun       give both nf0 their addresses (after the netforge server and client started)
#   lab.sh route     client: send 203.0.113.0/24 into the tunnel; host: route back to 10.10.0.0/24
#   lab.sh forward   server: turn on IP forwarding, so it routes between nf0 and veth2
#   lab.sh nat       host: forget the return route; server: NAT the VPN subnet behind 203.0.113.1
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
	# MTU 1468 = 1500 - 32 bytes of tunnel overhead, so every outer packet fits one frame.
	ip addr add 10.10.0.1/24 dev nf0
	ip link set nf0 up mtu 1468
	ip netns exec nfpeer ip addr add 10.10.0.2/24 dev nf0
	ip netns exec nfpeer ip link set nf0 up mtu 1468
	echo "TUN up: client nf0 10.10.0.1 <-> server nf0 10.10.0.2 (MTU 1468)"
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
down)
	ip netns del nfnet 2>/dev/null || true
	ip netns del nfpeer 2>/dev/null || true
	ip link del veth0 2>/dev/null || true
	echo "lab down"
	;;
*)
	echo "usage: $0 up|tun|route|forward|nat|down" >&2
	exit 2
	;;
esac
