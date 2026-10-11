#!/usr/bin/env bash
# Two "machines" inside one Linux box, to test the VPN (steps 6.4 and 7).
#
#   root namespace   = VPN client   veth0 192.168.100.1   TUN nf0 10.10.0.1   (netforge vpn client ...)
#   namespace nfpeer = VPN server   veth1 192.168.100.2   TUN nf0 10.10.0.2   (netforge vpn server)
#
# Each namespace has its own interfaces, so both TUN interfaces can be called nf0.
#
# Usage, as root:
#   lab.sh up     create the nfpeer namespace and the veth cable between the two
#   lab.sh tun    give both nf0 interfaces their addresses (after both netforge processes started)
#   lab.sh down   remove everything (stop the netforge processes first)
set -euo pipefail

case "${1:-}" in
up)
	ip netns add nfpeer
	ip link add veth0 type veth peer name veth1
	ip link set veth1 netns nfpeer
	ip addr add 192.168.100.1/24 dev veth0
	ip link set veth0 up
	ip netns exec nfpeer ip addr add 192.168.100.2/24 dev veth1
	ip netns exec nfpeer ip link set veth1 up
	ip netns exec nfpeer ip link set lo up
	echo "lab up: veth0 192.168.100.1 (client side) <-> veth1 192.168.100.2 (server side, namespace nfpeer)"
	;;
tun)
	# MTU 1468 = 1500 - 32 bytes of tunnel overhead, so every outer packet fits one frame.
	ip addr add 10.10.0.1/24 dev nf0
	ip link set nf0 up mtu 1468
	ip netns exec nfpeer ip addr add 10.10.0.2/24 dev nf0
	ip netns exec nfpeer ip link set nf0 up mtu 1468
	echo "TUN up: client nf0 10.10.0.1 <-> server nf0 10.10.0.2 (MTU 1468)"
	;;
down)
	ip netns del nfpeer 2>/dev/null || true
	ip link del veth0 2>/dev/null || true
	echo "lab down"
	;;
*)
	echo "usage: $0 up|tun|down" >&2
	exit 2
	;;
esac
