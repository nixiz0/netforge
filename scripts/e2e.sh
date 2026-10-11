#!/usr/bin/env bash
# End-to-end test of every netforge stage, on a real Linux network stack (WSL2 Ubuntu works).
# It builds the binary, runs each feature, and checks what really happens on the wire and in logs.
#
#   sudo bash scripts/e2e.sh
#
# Needs: go, curl, dig (dnsutils), tcpdump, iptables, python3, and internet access (the proxies
# fetch example.com). It creates and removes the lab namespaces, so stop any running lab first.
# Logs stay in /tmp/netforge-e2e for inspection.
set -u
cd "$(dirname "$0")/.."
[ "$(id -u)" = 0 ] || { echo "run as root: sudo bash scripts/e2e.sh"; exit 1; }
for t in go curl dig tcpdump iptables python3 ip ping; do
	command -v $t >/dev/null || { echo "missing: $t (apt install -y dnsutils tcpdump iptables)"; exit 1; }
done

T=/tmp/netforge-e2e; rm -rf $T; mkdir -p $T
NF=$T/netforge; LAB="bash scripts/lab.sh"
CGO_ENABLED=0 go build -buildvcs=false -o $NF ./cmd/netforge || { echo "BUILD FAILED"; exit 1; }

PIDS=()
start() { local name=$1; shift; "$@" >$T/$name.log 2>&1 & PIDS+=($!); echo $! >$T/$name.pid; }
stop() { kill "$(cat $T/$1.pid)" 2>/dev/null; sleep 0.5; }
cleanup() { kill "${PIDS[@]}" 2>/dev/null; wait 2>/dev/null; $LAB full-off >/dev/null 2>&1; $LAB down >/dev/null 2>&1; }
trap cleanup EXIT
pass=0; fail=0
check() { if eval "$2"; then echo "PASS  $1"; pass=$((pass+1)); else echo "FAIL  $1"; fail=$((fail+1)); fi; }
waitfor() { for _ in $(seq 1 $(($3 * 4))); do grep -q "$2" "$1" 2>/dev/null && return 0; sleep 0.25; done; return 1; }
lost0() { grep -q " 0% packet loss" "$1"; }

echo "== CLI"
$NF >$T/usage.txt 2>&1; code=$?
check "no command: usage, exit code 2" '[ $code = 2 ] && grep -q "vpn server" $T/usage.txt'

echo "== 1 TCP: echo + dial"
start echo $NF echo; sleep 0.5
printf 'hello\nworld\n' | timeout 5 $NF dial >$T/dial.out 2>$T/dial.err
check "dial: both lines echoed back" 'grep -q hello $T/dial.out && grep -q world $T/dial.out'
check "dial: stdin end sends FIN, server closes" 'grep -q "stdin closed, sending FIN" $T/dial.err && grep -q "server closed the connection" $T/dial.err'
timeout 7 $NF dial 127.0.0.1:9001 </dev/null >/dev/null 2>$T/refused.err
check "dial to a closed port: connection refused (RST)" 'grep -qi refused $T/refused.err'
timeout 8 $NF dial 10.255.255.1:9000 </dev/null >/dev/null 2>$T/timeout.err
check "dial to nobody: i/o timeout (silence)" 'grep -qi timeout $T/timeout.err'

echo "== 2 HTTP proxy"
start proxy $NF proxy; sleep 0.5
curl -s -m 10 -x http://127.0.0.1:8080 http://example.com/ -o $T/http.html
check "plain HTTP through the proxy" 'grep -qi "<html" $T/http.html'
curl -s -m 10 -x http://127.0.0.1:8080 https://example.com/ -o $T/https.html
check "HTTPS through the proxy (CONNECT)" 'grep -qi "<html" $T/https.html'
code=$(curl -s -m 10 -o /dev/null -w "%{http_code}" -x http://127.0.0.1:8080 http://nonexistent.invalid/)
check "unknown host: 502 Bad Gateway" '[ "$code" = 502 ]'
check "explicit DNS lookup logged" 'grep -q "resolved example.com" $T/proxy.log'

echo "== 3 SOCKS5"
start socks $NF socks; sleep 0.5
curl -s -m 10 --socks5-hostname 127.0.0.1:1080 https://example.com/ -o $T/socks1.html
check "SOCKS5 with proxy-side DNS" 'grep -qi "<html" $T/socks1.html && grep -q "domain name: the proxy resolves it" $T/socks.log'
curl -s -m 10 --socks5 127.0.0.1:1080 https://example.com/ -o $T/socks2.html
check "SOCKS5 with client-side DNS" 'grep -qi "<html" $T/socks2.html'

echo "== 4 UDP"
start udpecho $NF udpecho; sleep 0.5
printf 'a\nb\n+1472\n' | timeout 8 $NF udpdial >$T/udp.out 2>&1
check "3 datagrams sent, 3 replies" 'grep -q "sent 3 datagrams, got 3 replies, 0 lost" $T/udp.out'
printf '+65508\n' | timeout 8 $NF udpdial >$T/udpbig.out 2>&1
check "65508 bytes: message too long" 'grep -qi "too long" $T/udpbig.out'
printf 'x\n' | timeout 8 $NF udpdial 10.255.255.1:9000 >$T/udploss.out 2>&1
check "datagram to nobody: silent loss, counted" 'grep -q "got 0 replies, 1 lost" $T/udploss.out'
printf 'x\n' | timeout 8 $NF udpdial 127.0.0.1:9009 >$T/udpicmp.out 2>&1
check "closed UDP port: ICMP error on a later read" 'grep -q "refused (ICMP error for an earlier datagram)" $T/udpicmp.out'

echo "== 5 Tunnel"
start tserver $NF tunnel server; start tclient $NF tunnel client; sleep 1
printf 'through the tunnel\n+1468\n+1469\n' | timeout 8 $NF udpdial 127.0.0.1:5300 >$T/tunnel.out 2>&1
check "through the tunnel: 1468 bytes pass, 1469 dropped (MTU)" 'grep -q "sent 3 datagrams, got 2 replies, 1 lost" $T/tunnel.out && grep -q "drop 1469-byte datagram" $T/tclient.log'
check "1468 inner bytes = exactly 1500 on the wire" 'grep -q "encapsulated 1472 bytes (1500 on the wire)" $T/tclient.log'
check "keepalives: tunnel up" 'grep -q "tunnel up" $T/tclient.log && grep -q "keepalive <- .*answered" $T/tserver.log'

echo "== 7.6 DNS server (local)"
start dns $NF dns; sleep 0.5
check "dig host.lab: 203.0.113.10" '[ "$(dig +short -p 5353 @127.0.0.1 host.lab)" = 203.0.113.10 ]'
dig -p 5353 @127.0.0.1 nope.lab >$T/dig-nx.out
check "dig nope.lab: NXDOMAIN, authoritative" 'grep -q "status: NXDOMAIN" $T/dig-nx.out && grep -q "flags: qr aa rd" $T/dig-nx.out'
dig -p 5353 @127.0.0.1 host.lab AAAA >$T/dig-aaaa.out
check "dig host.lab AAAA: NOERROR, no answer (NODATA)" 'grep -q "status: NOERROR" $T/dig-aaaa.out && grep -q "ANSWER: 0" $T/dig-aaaa.out'
stop echo; stop proxy; stop socks; stop udpecho; stop tserver; stop tclient; stop dns

echo "== 8.3 keys"
$LAB down >/dev/null 2>&1
for k in server client client2 bad; do $NF genkey >$T/$k.key; $NF pubkey $T/$k.key >$T/$k.pub; done
check "genkey/pubkey: 64 hex characters each" '[ $(tr -d "\n" <$T/server.key | wc -c) = 64 ] && [ $(tr -d "\n" <$T/server.pub | wc -c) = 64 ]'

echo "== 6 TUN: netforge answers ping from userspace"
start tun $NF tun; sleep 1
ip addr add 10.10.0.1/24 dev nf0 && ip link set nf0 up mtu 1468
ping -c 2 -W 2 10.10.0.2 >$T/ping-tun.out 2>&1
check "ping 10.10.0.2 answered by netforge" 'lost0 $T/ping-tun.out && grep -q "answered ping" $T/tun.log'
check "IPv4 header decoded" 'grep -q "IPv4 10.10.0.1 -> 10.10.0.2 ICMP ttl=64, 84 bytes" $T/tun.log'
stop tun

echo "== 7 + 8 encrypted VPN"
$LAB up >/dev/null && $LAB up2 >/dev/null
start server ip netns exec nfpeer $NF vpn server $T/server.key $T/client.pub=10.10.0.1 $T/client2.pub=10.10.0.3
sleep 0.5
start client $NF vpn client 192.168.100.2:7000 $T/client.key $T/server.pub
check "handshake: both ends derive session keys" 'waitfor $T/client.log "handshake done" 5 && waitfor $T/server.log "(peer 10.10.0.1): new session keys" 5'
$LAB tun >/dev/null
ping -c 2 -W 2 10.10.0.2 >$T/ping-vpn.out 2>&1
check "ping the server through the VPN" 'lost0 $T/ping-vpn.out'
check "84-byte packet travels as 116 encrypted bytes" 'grep -q "84-byte IP packet 10.10.0.1 -> 10.10.0.2, encrypted 116 bytes" $T/client.log'
ping -c 1 -W 2 -s 1412 -M do 10.10.0.2 >$T/mtu-ok.out 2>&1
ping -c 1 -W 2 -s 1413 -M do 10.10.0.2 >$T/mtu-big.out 2>&1
check "VPN MTU 1440: 1440 bytes pass, 1441 refused" 'lost0 $T/mtu-ok.out && grep -qi "mtu\|too long" $T/mtu-big.out'
timeout 6 tcpdump -i veth0 -w $T/wire.pcap -c 4 udp port 7000 >/dev/null 2>&1 &
sleep 1; ping -c 2 -W 2 10.10.0.2 >/dev/null 2>&1; sleep 2
python3 - $T/wire.pcap >$T/wire.out <<'EOF'
import sys, struct
d = open(sys.argv[1], "rb").read(); off = 24; n = clear = 0
while off + 16 <= len(d):
    incl = struct.unpack("<I", d[off+8:off+12])[0]; pkt = d[off+16:off+16+incl]; off += 16 + incl
    ip = pkt[14:]; payload = ip[(ip[0] & 15) * 4 + 8:]
    n += 1; clear += bytes([10, 10, 0, 1]) in payload or bytes([10, 10, 0, 2]) in payload
print(f"packets={n} inner_addresses_visible={clear}")
EOF
check "on the wire: inner addresses invisible (ciphertext)" 'grep -q "packets=[1-9]" $T/wire.out && grep -q "inner_addresses_visible=0" $T/wire.out'

echo "== 7.2 server as router"
$LAB route >/dev/null && $LAB forward >/dev/null
timeout 5 ip netns exec nfnet tcpdump -ni veth3 -c 1 icmp >$T/host-route.out 2>/dev/null &
sleep 1; ping -c 2 -W 2 203.0.113.10 >$T/ping-host.out 2>&1; sleep 1
check "reach the host behind the server, TTL 63 (one hop)" 'lost0 $T/ping-host.out && grep -q "ttl=63" $T/ping-host.out'
check "host sees 10.10.0.1 (routing, no NAT)" 'grep -q "10.10.0.1 > 203.0.113.10" $T/host-route.out'

echo "== 7.3 NAT"
$LAB nat >/dev/null
timeout 5 ip netns exec nfnet tcpdump -ni veth3 -c 1 icmp >$T/host-nat.out 2>/dev/null &
sleep 1; ping -c 2 -W 2 203.0.113.10 >$T/ping-nat.out 2>&1; sleep 1
check "host sees 203.0.113.1 (NAT), ping still works" 'lost0 $T/ping-nat.out && grep -q "203.0.113.1 > 203.0.113.10" $T/host-nat.out'

echo "== 7.4 full tunnel"
$LAB full >/dev/null
check "1.1.1.1 via nf0, server endpoint via veth0" 'ip route get 1.1.1.1 | grep -q "dev nf0" && ip route get 192.168.100.2 | grep -q "dev veth0"'
ping -c 1 -W 2 1.1.1.1 >$T/ping-full.out 2>&1
check "1.1.1.1: Net Unreachable from 10.10.0.2, back through the tunnel" 'grep -q "From 10.10.0.2.*Destination Net Unreachable" $T/ping-full.out'

echo "== 7.5 DNS leak"
$LAB dns >$T/dnsleak.out 2>&1
check "lab.sh dns: LEAK even in full-tunnel mode" 'grep -q "^LEAK" $T/dnsleak.out'
$LAB full-off >/dev/null
check "full-off: default route back" '! ip route get 1.1.1.1 | grep -q "dev nf0"'

echo "== 7.6 DNS inside the VPN"
start vpndns ip netns exec nfpeer $NF dns :53; sleep 0.5
check "dig @10.10.0.2 host.lab through the tunnel" '[ "$(dig +short +time=2 +tries=1 @10.10.0.2 host.lab)" = 203.0.113.10 ]'
dig +short +time=2 +tries=1 @192.168.100.2 host.lab >/dev/null
check "server sees 10.10.0.1 (inside) vs 192.168.100.1 (beside)" 'grep -q "^.* 10.10.0.1:[0-9]* asked A host.lab" $T/vpndns.log && grep -q "192.168.100.1:[0-9]* asked A host.lab" $T/vpndns.log'

echo "== 8.2 replay attack"
python3 scripts/replay.py >$T/replay.out 2>&1 &
RP=$!; sleep 1; ping -c 1 -W 2 10.10.0.2 >/dev/null 2>&1; wait $RP
check "replayed packet dropped" 'grep -q "dropped the copy" $T/replay.out && grep -q "replayed packet" $T/server.log'

echo "== 8.3 wrong key, server restart"
start bad ip netns exec nfc2 $NF vpn client 192.168.101.2:7000 $T/bad.key $T/server.pub
check "unknown key: 72-byte handshake dropped" 'waitfor $T/server.log "drop 72 bytes from 192.168.101.1.*authentication failed" 5'
stop bad; stop server
start server ip netns exec nfpeer $NF vpn server $T/server.key $T/client.pub=10.10.0.1 $T/client2.pub=10.10.0.3
check "server restarted: client sees tunnel down, re-handshakes" 'waitfor $T/client.log "tunnel down" 25 && waitfor $T/client.log "tunnel up" 10'
$LAB tun >/dev/null
ping -c 2 -W 2 10.10.0.2 >$T/ping-restart.out 2>&1
check "ping works with the new session keys" 'lost0 $T/ping-restart.out'

echo "== 9.1 two clients"
start client2 ip netns exec nfc2 $NF vpn client 192.168.101.2:7000 $T/client2.key $T/server.pub
check "client 2 handshake" 'waitfor $T/client2.log "handshake done" 5'
$LAB tun2 >/dev/null && $LAB forward >/dev/null
ping -c 2 -W 2 10.10.0.3 >$T/ping-c2.out 2>&1
check "client 1 -> server -> client 2, no ICMP redirect" 'lost0 $T/ping-c2.out && ! grep -q Redirect $T/ping-c2.out'
check "server re-encrypts for client 2's endpoint" 'grep -q "10.10.0.1 -> 10.10.0.3, encrypted 116 bytes -> 192.168.101.1" $T/server.log'

echo "== 9.2 allowed IPs"
$LAB spoof >$T/spoof.out 2>&1
check "client 2 claiming 10.10.0.1: dropped" 'grep -q "100% packet loss" $T/spoof.out && grep -q "from peer 10.10.0.3: source not allowed" $T/server.log'

echo
echo "== RESULT: $pass passed, $fail failed (logs in $T)"
[ $fail = 0 ]
